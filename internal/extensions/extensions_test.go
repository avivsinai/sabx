package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveSourceGitHubShorthand(t *testing.T) {
	name, repo, kind, err := deriveSource("owner/sabx-foo")
	if err != nil {
		t.Fatalf("deriveSource returned error: %v", err)
	}
	if kind != "git" {
		t.Fatalf("expected kind=git, got %q", kind)
	}
	if name != "foo" {
		t.Fatalf("expected name=foo, got %q", name)
	}
	if repo != "https://github.com/owner/sabx-foo.git" {
		t.Fatalf("expected repo to be github clone URL, got %q", repo)
	}
}

func TestDeriveSourceGitHubShorthandWithDotGit(t *testing.T) {
	name, repo, kind, err := deriveSource("owner/sabx-foo.git")
	if err != nil {
		t.Fatalf("deriveSource returned error: %v", err)
	}
	if kind != "git" {
		t.Fatalf("expected kind=git, got %q", kind)
	}
	if name != "foo" {
		t.Fatalf("expected name=foo, got %q", name)
	}
	if repo != "https://github.com/owner/sabx-foo.git" {
		t.Fatalf("expected repo to be github clone URL, got %q", repo)
	}
}

func TestDeriveSourceHTTPURL(t *testing.T) {
	name, repo, kind, err := deriveSource("https://github.com/owner/sabx-bar")
	if err != nil {
		t.Fatalf("deriveSource returned error: %v", err)
	}
	if kind != "git" {
		t.Fatalf("expected kind=git, got %q", kind)
	}
	if name != "bar" {
		t.Fatalf("expected name=bar, got %q", name)
	}
	if repo != "https://github.com/owner/sabx-bar" {
		t.Fatalf("expected repo to be preserved, got %q", repo)
	}
}

func TestDeriveSourceLocalPath(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sabx-localext")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	name, repo, kind, err := deriveSource(srcDir)
	if err != nil {
		t.Fatalf("deriveSource returned error: %v", err)
	}
	if kind != "local" {
		t.Fatalf("expected kind=local, got %q", kind)
	}
	if name != "localext" {
		t.Fatalf("expected name=localext, got %q", name)
	}

	wantAbs, err := filepath.Abs(srcDir)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	if repo != wantAbs {
		t.Fatalf("expected repo to be absolute path %q, got %q", wantAbs, repo)
	}
}

func TestDeriveSourceMissingExplicitPathErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, _, _, err := deriveSource(missing); err == nil {
		t.Fatalf("expected error for missing local path, got nil")
	}
}
