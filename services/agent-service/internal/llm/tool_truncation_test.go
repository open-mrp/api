package llm

import (
	"strings"
	"testing"
)

func TestTruncateToolOutput_UnderLimit(t *testing.T) {
	t.Parallel()
	content := "short output"
	result := TruncateToolOutput(content, "some_tool")
	if result != content {
		t.Errorf("expected unchanged output, got %q", result)
	}
}

func TestTruncateToolOutput_DefaultLimit(t *testing.T) {
	t.Parallel()
	content := strings.Repeat("x", 60_000) // exceeds 50KB default
	result := TruncateToolOutput(content, "some_tool")

	if len(result) >= 60_000 {
		t.Errorf("expected truncated output, got len=%d", len(result))
	}
	if !strings.Contains(result, "truncated") {
		t.Error("expected truncation notice")
	}
}

func TestTruncateToolOutput_FetchURLLimit(t *testing.T) {
	t.Parallel()
	content := strings.Repeat("x", 35_000) // exceeds 30KB fetch_url limit
	result := TruncateToolOutput(content, "fetch_url")

	if len(result) >= 35_000 {
		t.Errorf("expected truncated output for fetch_url, got len=%d", len(result))
	}
	if !strings.Contains(result, "truncated") {
		t.Error("expected truncation notice")
	}
}

func TestTruncateToolOutput_ReadDocLimit(t *testing.T) {
	t.Parallel()
	content := strings.Repeat("x", 35_000)
	result := TruncateToolOutput(content, "read_doc")

	if len(result) >= 35_000 {
		t.Errorf("expected truncated output for read_doc, got len=%d", len(result))
	}
}

func TestTruncateToolOutput_PreservesTail(t *testing.T) {
	t.Parallel(
	// Create content where the tail is distinguishable.
	)

	head := strings.Repeat("H", 55_000)
	tail := strings.Repeat("T", 200)
	content := head + tail
	result := TruncateToolOutput(content, "some_tool")

	if !strings.HasSuffix(result, tail) {
		t.Error("expected tail to be preserved in truncated output")
	}
}
