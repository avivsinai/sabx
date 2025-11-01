package extensions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type InstalledExtension struct {
	Name       string `json:"name"`
	Binary     string `json:"binary"`
	Source     string `json:"source"`
	Kind       string `json:"kind"`
	InstallDir string `json:"install_dir,omitempty"`
}

var (
	errBinaryNotFound = errors.New("extension binary not found")
)

// List returns installed extensions (metadata + PATH discovery).
func List() ([]InstalledExtension, error) {
	meta, err := loadMetadata()
	if err != nil {
		return nil, err
	}

	result := make([]InstalledExtension, 0, len(meta.Extensions)+4)
	seen := map[string]struct{}{}

	for name, ext := range meta.Extensions {
		if ext.Kind == "git" || ext.Kind == "local" {
			if _, err := os.Stat(ext.Binary); errors.Is(err, fs.ErrNotExist) {
				// skip missing binary but keep metadata for debugging
				continue
			}
		}
		ext.Name = name
		result = append(result, ext)
		seen[name] = struct{}{}
	}

	pathExts := discoverPATH()
	for name, bin := range pathExts {
		if _, exists := seen[name]; exists {
			continue
		}
		result = append(result, InstalledExtension{
			Name:   name,
			Binary: bin,
			Kind:   "path",
			Source: "PATH",
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// Install clones or links an extension into the sabx extension dir.
func Install(source string, overwrite bool) (InstalledExtension, error) {
	name, repoURL, installKind, err := deriveSource(source)
	if err != nil {
		return InstalledExtension{}, err
	}

	dirs, err := ensureDirs()
	if err != nil {
		return InstalledExtension{}, err
	}

	targetDir := filepath.Join(dirs.extensionsDir, name)
	if _, err := os.Stat(targetDir); err == nil {
		if !overwrite {
			return InstalledExtension{}, fmt.Errorf("extension %q already installed", name)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return InstalledExtension{}, err
		}
	}

	switch installKind {
	case "git":
		if err := cloneRepo(repoURL, targetDir); err != nil {
			return InstalledExtension{}, err
		}
	case "local":
		if err := copyLocalDirectory(repoURL, targetDir); err != nil {
			return InstalledExtension{}, err
		}
	default:
		return InstalledExtension{}, fmt.Errorf("unsupported source kind %q", installKind)
	}

	binaryPath, err := findBinary(targetDir, name)
	if err != nil {
		return InstalledExtension{}, err
	}

	if err := os.Chmod(binaryPath, 0o755); err != nil && !errors.Is(err, fs.ErrPermission) {
		return InstalledExtension{}, err
	}

	meta, err := loadMetadata()
	if err != nil {
		return InstalledExtension{}, err
	}

	meta.Extensions[name] = InstalledExtension{
		Name:       name,
		Binary:     binaryPath,
		Source:     source,
		Kind:       installKind,
		InstallDir: targetDir,
	}

	if err := saveMetadata(meta); err != nil {
		return InstalledExtension{}, err
	}

	return meta.Extensions[name], nil
}

// Remove deletes an installed extension and its metadata entry.
func Remove(name string) error {
	meta, err := loadMetadata()
	if err != nil {
		return err
	}
	ext, ok := meta.Extensions[name]
	if !ok {
		return fmt.Errorf("extension %q not installed", name)
	}

	if ext.InstallDir != "" {
		_ = os.RemoveAll(ext.InstallDir)
	}

	delete(meta.Extensions, name)
	return saveMetadata(meta)
}

// Exec delegates to an installed extension binary with passthrough stdio.
func Exec(name string, args []string) error {
	ext, err := Resolve(name)
	if err != nil {
		return err
	}
	cmd := exec.Command(ext.Binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

// Resolve locates an extension by name (installed metadata or PATH).
func Resolve(name string) (InstalledExtension, error) {
	meta, err := loadMetadata()
	if err != nil {
		return InstalledExtension{}, err
	}

	if ext, ok := meta.Extensions[name]; ok {
		if _, err := os.Stat(ext.Binary); err == nil {
			return ext, nil
		}
	}

	if pathExts := discoverPATH(); pathExts[name] != "" {
		return InstalledExtension{
			Name:   name,
			Binary: pathExts[name],
			Kind:   "path",
			Source: "PATH",
		}, nil
	}

	return InstalledExtension{}, fmt.Errorf("extension %q not found", name)
}

// ExtractExtensionCommand identifies the extension command from CLI args.
func ExtractExtensionCommand(args []string) (name string, extArgs []string, ok bool) {
	// Skip global flags (long form only).
	skipNext := false
	for i := 0; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				return args[i+1], args[i+2:], true
			}
			return "", nil, false
		}
		if strings.HasPrefix(arg, "--") {
			if arg == "--profile" || arg == "--base-url" || arg == "--api-key" {
				skipNext = true
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg, args[i+1:], true
	}
	return "", nil, false
}

// helper structures & functions

type metadata struct {
	Extensions map[string]InstalledExtension `json:"extensions"`
}

type dirs struct {
	configDir     string
	extensionsDir string
}

func ensureDirs() (dirs, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return dirs{}, err
	}
	configDir = filepath.Join(configDir, "sabx")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return dirs{}, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return dirs{}, err
	}
	extDir := filepath.Join(home, ".sabx", "extensions")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		return dirs{}, err
	}

	return dirs{configDir: configDir, extensionsDir: extDir}, nil
}

func metadataPath() (string, error) {
	dirs, err := ensureDirs()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirs.configDir, "extensions.json"), nil
}

func loadMetadata() (metadata, error) {
	path, err := metadataPath()
	if err != nil {
		return metadata{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return metadata{Extensions: map[string]InstalledExtension{}}, nil
		}
		return metadata{}, err
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return metadata{}, err
	}
	if meta.Extensions == nil {
		meta.Extensions = map[string]InstalledExtension{}
	}
	return meta, nil
}

func saveMetadata(meta metadata) error {
	path, err := metadataPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func deriveSource(source string) (name, repo string, kind string, err error) {
	if source == "" {
		return "", "", "", errors.New("source is required")
	}

	if strings.Contains(source, string(os.PathSeparator)) || strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") || strings.HasPrefix(source, "/") {
		abs, err := filepath.Abs(source)
		if err != nil {
			return "", "", "", err
		}
		base := filepath.Base(abs)
		name = strings.TrimPrefix(base, "sabx-")
		if name == "" {
			name = base
		}
		return name, abs, "local", nil
	}

	repo = source
	if !strings.HasSuffix(repo, ".git") && !strings.HasPrefix(repo, "http://") && !strings.HasPrefix(repo, "https://") {
		repo = fmt.Sprintf("https://github.com/%s.git", source)
	}
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		base := filepath.Base(strings.TrimSuffix(repo, ".git"))
		name = strings.TrimPrefix(base, "sabx-")
		if name == "" {
			name = base
		}
		return name, repo, "git", nil
	}

	return "", "", "", fmt.Errorf("unsupported source format: %s", source)
}

func cloneRepo(url, target string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyLocalDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyLocalDirectory(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func findBinary(dir, name string) (string, error) {
	expected := fmt.Sprintf("sabx-%s", name)
	var found string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == expected {
			found = path
			return fs.SkipDir
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipDir) {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%w: expected %s", errBinaryNotFound, expected)
	}
	return found, nil
}

func discoverPATH() map[string]string {
	result := map[string]string{}
	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, "sabx-") {
				continue
			}
			extName := strings.TrimPrefix(name, "sabx-")
			full := filepath.Join(dir, name)
			result[extName] = full
		}
	}
	return result
}
