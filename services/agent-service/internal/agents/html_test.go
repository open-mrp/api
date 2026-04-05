package agents

import (
	"strings"
	"testing"
)

func TestHtmlToMarkdown_BasicConversion(t *testing.T) {
	t.Parallel()
	input := `<html><body><h1>Hello World</h1><p>This is a paragraph.</p></body></html>`
	result := htmlToMarkdown(input)

	if !strings.Contains(result, "Hello World") {
		t.Error("expected heading text to be preserved")
	}
	if !strings.Contains(result, "This is a paragraph") {
		t.Error("expected paragraph text to be preserved")
	}
	if strings.Contains(result, "<h1>") {
		t.Error("expected HTML tags to be stripped")
	}
}

func TestHtmlToMarkdown_StripsNoisyElements(t *testing.T) {
	t.Parallel()
	input := `<html>
		<head><style>body { color: red; }</style></head>
		<body>
			<nav><a href="/">Home</a></nav>
			<header><h1>Site Header</h1></header>
			<main><p>Useful content here.</p></main>
			<footer>Copyright 2024</footer>
			<script>alert('xss')</script>
		</body>
	</html>`
	result := htmlToMarkdown(input)

	if !strings.Contains(result, "Useful content") {
		t.Error("expected main content to be preserved")
	}
	if strings.Contains(result, "alert") {
		t.Error("expected script content to be stripped")
	}
	if strings.Contains(result, "color: red") {
		t.Error("expected style content to be stripped")
	}
	if strings.Contains(result, "Copyright") {
		t.Error("expected footer content to be stripped")
	}
}

func TestHtmlToMarkdown_CollapsesBlankLines(t *testing.T) {
	t.Parallel()
	input := `<html><body><p>Line 1</p><br/><br/><br/><br/><br/><p>Line 2</p></body></html>`
	result := htmlToMarkdown(input)

	if strings.Contains(result, "\n\n\n") {
		t.Error("expected no more than 2 consecutive newlines")
	}
}

func TestHtmlToMarkdown_PlainText(t *testing.T) {
	t.Parallel()
	input := "Just plain text, no HTML."
	result := htmlToMarkdown(input)
	if !strings.Contains(result, "Just plain text") {
		t.Error("expected plain text to pass through")
	}
}

func TestIsHTMLContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ct   string
		want bool
	}{
		{"text/html; charset=utf-8", true},
		{"text/html", true},
		{"TEXT/HTML", true},
		{"application/xhtml+xml", true},
		{"application/json", false},
		{"text/plain", false},
		{"text/markdown", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isHTMLContent(tt.ct)
		if got != tt.want {
			t.Errorf("isHTMLContent(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}
