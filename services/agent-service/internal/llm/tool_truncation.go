package llm

import "fmt"

const (
	// DefaultToolOutputMaxBytes is the default cap for tool output content.
	DefaultToolOutputMaxBytes = 50_000

	// truncationTailChars is how many chars to preserve from the end when truncating.
	truncationTailChars = 200
)

// ToolOutputLimits maps tool names to their specific output byte caps.
// Tools not listed here use DefaultToolOutputMaxBytes.
var ToolOutputLimits = map[string]int{
	"fetch_url": 30_000,
	"read_doc":  30_000,
}

// TruncationResult holds the outcome of truncating a tool output.
type TruncationResult struct {
	Content      string
	WasTruncated bool
	FullLength   int
}

// TruncateToolOutput caps the content of a tool result at the given byte limit.
// When truncated, it preserves the first (limit - tail) bytes and the last tail
// bytes, inserting a truncation notice in between.
func TruncateToolOutput(content string, toolName string) string {
	return TruncateToolOutputResult(content, toolName).Content
}

// TruncateToolOutputResult is like TruncateToolOutput but returns a TruncationResult
// indicating whether truncation occurred and the original content length.
func TruncateToolOutputResult(content string, toolName string) TruncationResult {
	maxBytes := DefaultToolOutputMaxBytes
	if limit, ok := ToolOutputLimits[toolName]; ok {
		maxBytes = limit
	}

	if len(content) <= maxBytes {
		return TruncationResult{Content: content, WasTruncated: false, FullLength: len(content)}
	}

	removed := len(content) - maxBytes
	headLen := maxBytes - truncationTailChars
	if headLen < 0 {
		headLen = maxBytes
	}

	head := content[:headLen]
	tail := ""
	if headLen < maxBytes && len(content) >= truncationTailChars {
		tail = content[len(content)-truncationTailChars:]
	}

	truncated := fmt.Sprintf("%s\n...[truncated: %d bytes removed. If you need more detail, make a more specific query.]\n%s",
		head, removed, tail)
	return TruncationResult{Content: truncated, WasTruncated: true, FullLength: len(content)}
}
