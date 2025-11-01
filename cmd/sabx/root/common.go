package root

import (
	"context"
	"strings"
	"time"
)

const requestTimeout = 15 * time.Second
const jsonHelpSuffix = " (supports --json output)"
const jsonLongNote = "Supports the global --json flag for machine-readable output. Errors return a non-zero exit code."

func timeoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, requestTimeout)
}

func jsonShort(text string) string {
	if strings.Contains(strings.ToLower(text), "--json") {
		return text
	}
	return text + jsonHelpSuffix
}

func appendJSONLong(base string) string {
	if strings.Contains(strings.ToLower(base), "--json") {
		return base
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return jsonLongNote
	}
	return base + "\n\n" + jsonLongNote
}
