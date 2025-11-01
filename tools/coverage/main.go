package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type entry struct {
	Mode      string   `json:"mode"`
	Name      string   `json:"name,omitempty"`
	Functions []string `json:"functions"`
}

type combo struct {
	mode string
	name string
}

func main() {
	source := flag.String("source", "internal/sabapi/client.go", "path to sabapi client source")
	format := flag.String("format", "table", "output format: table|json")
	flag.Parse()

	entries, err := collectCoverage(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch strings.ToLower(*format) {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(entries); err != nil {
			fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
			os.Exit(1)
		}
	default:
		printTable(entries)
	}
}

func collectCoverage(path string) ([]entry, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, abs, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	entries := map[combo]*entry{}

	manualExtras := map[string][]combo{
		"AddFile": {
			{mode: "addfile"},
		},
		"ServerControl": {
			{mode: "restart"},
			{mode: "shutdown"},
		},
		"TestNotification": {
			{mode: "test_email"},
			{mode: "test_windows"},
			{mode: "test_notif"},
			{mode: "test_osd"},
			{mode: "test_pushover"},
			{mode: "test_pushbullet"},
			{mode: "test_apprise"},
			{mode: "test_prowl"},
			{mode: "test_nscript"},
		},
	}

	globalExtras := map[combo]string{
		{mode: "queue", name: "purge"}: "queuePurgeCmd",
		{mode: "queue", name: "move"}:  "queueItemMoveCmd",
	}

	ast.Inspect(file, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Body == nil {
			return true
		}

		modes := map[string]struct{}{}
		names := map[string]struct{}{}
		queueNames := map[string]struct{}{}

		ast.Inspect(fd.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				receiver, _ := sel.X.(*ast.Ident)
				switch {
				case receiver != nil && receiver.Name == "c" && (sel.Sel.Name == "call" || sel.Sel.Name == "do"):
					if len(call.Args) > 1 {
						if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							mode := strings.Trim(lit.Value, "`\"")
							if mode != "" {
								modes[mode] = struct{}{}
							}
						}
					}
				case receiver != nil && receiver.Name == "c" && sel.Sel.Name == "QueueAction":
					if len(call.Args) > 1 {
						if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							name := strings.Trim(lit.Value, "`\"")
							if name != "" {
								queueNames[name] = struct{}{}
							}
						}
					}
				case sel.Sel.Name == "Set":
					if len(call.Args) >= 2 {
						if key, ok := call.Args[0].(*ast.BasicLit); ok && key.Kind == token.STRING {
							if strings.Trim(key.Value, "`\"") == "name" {
								if val, ok := call.Args[1].(*ast.BasicLit); ok && val.Kind == token.STRING {
									name := strings.Trim(val.Value, "`\"")
									if name != "" {
										names[name] = struct{}{}
									}
								}
							}
						}
					}
				}
			}
			return true
		})

		for mode := range modes {
			if len(names) == 0 {
				addEntry(entries, combo{mode: mode}, fd.Name.Name)
				continue
			}
			for name := range names {
				addEntry(entries, combo{mode: mode, name: name}, fd.Name.Name)
			}
		}

		for name := range queueNames {
			addEntry(entries, combo{mode: "queue", name: name}, fd.Name.Name)
		}

		if extras, ok := manualExtras[fd.Name.Name]; ok {
			for _, extra := range extras {
				addEntry(entries, extra, fd.Name.Name)
			}
		}

		return false
	})

	for combo, fn := range globalExtras {
		addEntry(entries, combo, fn)
	}

	out := make([]entry, 0, len(entries))
	for _, val := range entries {
		sort.Strings(val.Functions)
		out = append(out, *val)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mode == out[j].Mode {
			return out[i].Name < out[j].Name
		}
		return out[i].Mode < out[j].Mode
	})
	return out, nil
}

func addEntry(entries map[combo]*entry, key combo, fn string) {
	e, ok := entries[key]
	if !ok {
		e = &entry{Mode: key.mode, Name: key.name}
		entries[key] = e
	}
	if fn != "" {
		if !contains(e.Functions, fn) {
			e.Functions = append(e.Functions, fn)
		}
	}
}

func contains(slice []string, candidate string) bool {
	for _, v := range slice {
		if v == candidate {
			return true
		}
	}
	return false
}

func printTable(entries []entry) {
	fmt.Printf("| Mode | Name | Functions |\n")
	fmt.Printf("| --- | --- | --- |\n")
	for _, e := range entries {
		name := e.Name
		if name == "" {
			name = "—"
		}
		fmt.Printf("| %s | %s | %s |\n", e.Mode, name, strings.Join(e.Functions, ", "))
	}
	fmt.Printf("\nTotal operations: %d\n", len(entries))
}
