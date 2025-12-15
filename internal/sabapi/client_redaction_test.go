package sabapi

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClientRedactsAPIKeyFromHTTPError(t *testing.T) {
	t.Helper()

	client, err := NewClient("http://127.0.0.1:0", "super-secret-key")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Status(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), "super-secret-key") {
		t.Fatalf("error should redact API key, got: %v", err)
	}
}
