package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

var docsHTTPClient = &http.Client{Timeout: 15 * time.Second}

// docsURLPrefixes are the documentation origins read_doc will fetch from.
var docsURLPrefixes = []string{"https://docs.openmrp.ai/", "https://docs.openmrp.ai/"}

func HandleReadDoc(ctx context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid read_doc input: %w", err)
	}

	// docs.openmrp.ai is the live docs host since the DNS cutover; docs.openmrp.ai is still
	// accepted because it keeps serving, and an agent may be working from a URL it read
	// before the move or from a page that still links the old host.
	if !slices.ContainsFunc(docsURLPrefixes, func(prefix string) bool {
		return strings.HasPrefix(params.URL, prefix)
	}) {
		return "", fmt.Errorf("read_doc: URL must be from docs.openmrp.ai or docs.openmrp.ai")
	}

	content, err := fetchDoc(ctx, docsHTTPClient, params.URL)
	if err != nil {
		return "", err
	}
	if len(content) == 0 {
		return "The page returned empty content.", nil
	}

	// Cap output to avoid bloating context.
	if len(content) > maxFetchOutputBytes {
		content = content[:maxFetchOutputBytes] + "\n...[content truncated]"
		if strings.HasSuffix(params.URL, "/llms.txt") {
			content += "\nThis index is longer than one read. Use search_docs to find pages in it."
		}
	}
	return content, nil
}

// fetchDoc reads a docs page as text, converting HTML to markdown. Index links point at a page's .md form, which not
// every page serves, so a 404 on a .md URL retries the page URL without the extension.
func fetchDoc(ctx context.Context, client *http.Client, url string) (string, error) {
	status, body, contentType, err := getDoc(ctx, client, url)
	if err == nil && status == http.StatusNotFound && strings.HasSuffix(url, ".md") {
		status, body, contentType, err = getDoc(ctx, client, strings.TrimSuffix(url, ".md"))
	}
	if err != nil {
		return "", fmt.Errorf("read_doc: failed to fetch page: %w", err)
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("read_doc: page returned status %d", status)
	}
	if isHTMLContent(contentType) {
		return htmlToMarkdown(body), nil
	}
	return body, nil
}

func getDoc(ctx context.Context, client *http.Client, url string) (status int, body, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", "", err
	}
	resp, err := client.Do(req) // #nosec G704 -- URL restricted to the docs origins by the caller
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxDocBytes))
	if err != nil {
		return 0, "", "", err
	}
	return resp.StatusCode, string(b), resp.Header.Get("Content-Type"), nil
}

const (
	// maxDocBytes bounds a single docs download; the llms.txt index is well under it.
	maxDocBytes = 2_000_000

	docsIndexTTL      = time.Hour
	docsSearchResults = 8
)

// docEntry is one page of the docs index: an llms.txt link line, plus its full text when llms-full.txt has it.
type docEntry struct {
	Section     string
	Title       string
	URL         string
	Description string
	Content     string
}

// docsIndex caches the parsed docs index; search_docs exists because a single read_doc of llms.txt is truncated
// long before the API reference sections.
type docsIndex struct {
	client   *http.Client
	indexURL string
	fullURL  string

	mu        sync.Mutex
	entries   []docEntry
	fetchedAt time.Time
}

var defaultDocsIndex = &docsIndex{
	client:   docsHTTPClient,
	indexURL: "https://docs.openmrp.ai/llms.txt",
	fullURL:  "https://docs.openmrp.ai/llms-full.txt",
}

func (x *docsIndex) load(ctx context.Context) ([]docEntry, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.entries != nil && time.Since(x.fetchedAt) < docsIndexTTL {
		return x.entries, nil
	}
	status, body, _, err := getDoc(ctx, x.client, x.indexURL)
	if err != nil {
		return nil, fmt.Errorf("search_docs: failed to fetch the docs index: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("search_docs: the docs index returned status %d", status)
	}
	entries := parseLLMsIndex(body)
	// llms-full.txt is optional; when present it lets search match page bodies, not just titles.
	if status, full, _, err := getDoc(ctx, x.client, x.fullURL); err == nil && status == http.StatusOK {
		attachFullText(entries, full)
	}
	x.entries, x.fetchedAt = entries, time.Now()
	return entries, nil
}

var llmsLinkLine = regexp.MustCompile(`^\s*-\s*\[([^\]]+)\]\(([^)\s]+)\)\s*:?\s*(.*)$`)

// parseLLMsIndex reads the llms.txt format: "## Section" / "### Subsection" headings over "- [Title](url): description" lines.
func parseLLMsIndex(text string) []docEntry {
	var entries []docEntry
	var section, subsection string
	for line := range strings.SplitSeq(text, "\n") {
		switch {
		case strings.HasPrefix(line, "### "):
			subsection = strings.TrimSpace(line[4:])
		case strings.HasPrefix(line, "## "):
			section, subsection = strings.TrimSpace(line[3:]), ""
		default:
			m := llmsLinkLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			s := section
			if subsection != "" {
				s += " > " + subsection
			}
			entries = append(entries, docEntry{Section: s, Title: m[1], URL: m[2], Description: strings.TrimSpace(m[3])})
		}
	}
	return entries
}

// attachFullText splits llms-full.txt into pages on "Source: <url>" lines and attaches each page's text to its entry.
func attachFullText(entries []docEntry, full string) {
	byURL := make(map[string]int, len(entries))
	for i, e := range entries {
		byURL[strings.TrimSuffix(e.URL, ".md")] = i
	}
	var current = -1
	var buf strings.Builder
	flush := func() {
		if current >= 0 {
			entries[current].Content = buf.String()
		}
		buf.Reset()
	}
	for line := range strings.SplitSeq(full, "\n") {
		if src, ok := strings.CutPrefix(strings.TrimSpace(line), "Source: "); ok {
			flush()
			current = -1
			if i, found := byURL[strings.TrimSuffix(strings.TrimSpace(src), ".md")]; found {
				current = i
			}
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
}

// searchDocEntries ranks entries by where the query's terms appear: title, then section, description, and body.
func searchDocEntries(entries []docEntry, query string, limit int) []docEntry {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	type scored struct {
		e     docEntry
		score int
	}
	var hits []scored
	for _, e := range entries {
		title, section := strings.ToLower(e.Title), strings.ToLower(e.Section)
		desc, content := strings.ToLower(e.Description), strings.ToLower(e.Content)
		score := 0
		for _, t := range terms {
			switch {
			case strings.Contains(title, t):
				score += 5
			case strings.Contains(section, t):
				score += 3
			case strings.Contains(desc, t):
				score += 2
			case strings.Contains(content, t):
				score++
			}
		}
		if score > 0 {
			hits = append(hits, scored{e, score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	out := make([]docEntry, 0, min(limit, len(hits)))
	for _, h := range hits[:min(limit, len(hits))] {
		out = append(out, h.e)
	}
	return out
}

// docSnippet returns the first body line mentioning a query term, for context beyond the title.
func docSnippet(e docEntry, query string) string {
	terms := tokenize(query)
	for line := range strings.SplitSeq(e.Content, "\n") {
		l := strings.ToLower(line)
		if slices.ContainsFunc(terms, func(t string) bool { return strings.Contains(l, t) }) {
			return truncateRunes(strings.TrimSpace(line), 240)
		}
	}
	return ""
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func HandleSearchDocs(ctx context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	return defaultDocsIndex.search(ctx, input)
}

func (x *docsIndex) search(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid search_docs input: %w", err)
	}
	entries, err := x.load(ctx)
	if err != nil {
		return "", err
	}
	hits := searchDocEntries(entries, params.Query, docsSearchResults)
	if len(hits) == 0 {
		return fmt.Sprintf("No documentation pages match %q.", params.Query), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Documentation pages matching %q (read one with read_doc):\n", params.Query)
	for _, h := range hits {
		fmt.Fprintf(&b, "- [%s] %s — %s\n  %s\n", h.Section, h.Title, h.URL, h.Description)
		if s := docSnippet(h, params.Query); s != "" {
			fmt.Fprintf(&b, "  > %s\n", s)
		}
		if b.Len() > maxFetchOutputBytes {
			break
		}
	}
	return b.String(), nil
}
