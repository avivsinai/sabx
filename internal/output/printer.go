package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Printer renders human or machine output.
type Printer struct {
	JSON  bool
	Quiet bool
	Out   io.Writer
	Err   io.Writer
}

// New returns a Printer with sensible defaults.
func New() *Printer {
	return &Printer{Out: os.Stdout, Err: os.Stderr}
}

// Print writes data respecting the configured format.
func (p *Printer) Print(data any) error {
	if p.Quiet {
		return nil
	}
	if p.JSON {
		enc := json.NewEncoder(p.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	switch v := data.(type) {
	case string:
		_, err := fmt.Fprintln(p.Out, v)
		return err
	case fmt.Stringer:
		_, err := fmt.Fprintln(p.Out, v.String())
		return err
	default:
		enc := json.NewEncoder(p.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}

// Table renders a simple tabular view.
func (p *Printer) Table(headers []string, rows [][]string) error {
	if p.Quiet {
		return nil
	}
	if p.JSON {
		data := map[string]any{"headers": headers, "rows": rows}
		return p.Print(data)
	}
	tw := tabwriter.NewWriter(p.Out, 2, 4, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
	}
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

// Error writes an error message.
func (p *Printer) Error(format string, args ...any) {
	if p.Err == nil || p.Quiet {
		return
	}
	fmt.Fprintf(p.Err, format+"\n", args...)
}
