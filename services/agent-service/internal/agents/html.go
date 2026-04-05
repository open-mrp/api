package agents

import (
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"golang.org/x/net/html"
)

// tagsToStrip are HTML elements that add noise without useful content.
var tagsToStrip = map[string]bool{
	"script":   true,
	"style":    true,
	"nav":      true,
	"footer":   true,
	"header":   true,
	"noscript": true,
	"svg":      true,
}

// htmlToMarkdown converts raw HTML to clean markdown text.
// It strips noisy elements (script, style, nav, footer, header) before
// converting, producing compact output suitable for LLM consumption.
func htmlToMarkdown(rawHTML string) string {
	// Strip noisy tags before conversion.
	cleaned := stripTags(rawHTML)

	md, err := htmltomarkdown.ConvertString(cleaned)
	if err != nil {
		// Fallback: return the cleaned HTML rather than failing.
		return cleaned
	}

	// Collapse excessive blank lines (3+ newlines → 2).
	md = collapseBlankLines(md)

	return strings.TrimSpace(md)
}

// isHTMLContent returns true if the content-type header indicates HTML.
func isHTMLContent(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}

// stripTags removes elements matching tagsToStrip from the HTML.
func stripTags(rawHTML string) string {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return rawHTML
	}
	removeMatchingNodes(doc)
	var buf strings.Builder
	if err := html.Render(&buf, doc); err != nil {
		return rawHTML
	}
	return buf.String()
}

// removeMatchingNodes walks the tree and removes nodes whose tag is in tagsToStrip.
func removeMatchingNodes(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type == html.ElementNode && tagsToStrip[c.Data] {
			n.RemoveChild(c)
		} else {
			removeMatchingNodes(c)
		}
	}
}

// collapseBlankLines replaces runs of 3+ newlines with exactly 2.
func collapseBlankLines(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	newlines := 0
	for _, r := range s {
		if r == '\n' {
			newlines++
			if newlines <= 2 {
				b.WriteRune(r)
			}
		} else {
			newlines = 0
			b.WriteRune(r)
		}
	}
	return b.String()
}
