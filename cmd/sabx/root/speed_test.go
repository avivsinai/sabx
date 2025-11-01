package root

import "testing"

func TestNormalizeSpeedLimitInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{input: "50%", expected: "50"},
		{input: "800K", expected: "800K"},
		{input: "4MB/s", expected: "4M"},
		{input: "4MiB/s", expected: "4.194M"},
		{input: "10Mbps", expected: "1.25M"},
		{input: "2.5M", expected: "2.5M"},
	}

	for _, tc := range tests {
		got, err := normalizeSpeedLimitInput(tc.input)
		if err != nil {
			t.Fatalf("normalizeSpeedLimitInput(%q) returned error: %v", tc.input, err)
		}
		if got != tc.expected {
			t.Fatalf("normalizeSpeedLimitInput(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestNormalizeSpeedLimitInputError(t *testing.T) {
	t.Parallel()

	if _, err := normalizeSpeedLimitInput("500"); err == nil {
		t.Fatal("expected error for missing unit, got nil")
	}
	if _, err := normalizeSpeedLimitInput("-5%"); err == nil {
		t.Fatal("expected error for negative percent, got nil")
	}
}

func TestFormatFromMbps(t *testing.T) {
	t.Parallel()

	if got := formatFromMbps(10); got != "1.25M" {
		t.Fatalf("formatFromMbps(10) = %q, want 1.25M", got)
	}
	if got := formatFromMbps(0.5); got != "62.5K" {
		t.Fatalf("formatFromMbps(0.5) = %q, want 62.5K", got)
	}
}
