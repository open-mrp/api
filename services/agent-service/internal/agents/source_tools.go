package agents

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/shared/buildinfo"
)

const (
	sourceRepo = "open-mrp/api"

	maxSourceArchiveBytes = 300 << 20
	maxSourceFileBytes    = 2 << 20
	maxSourceTreeBytes    = 200 << 20

	defaultSourceResults = 10
	maxSourceResults     = 20
	defaultSourceLines   = 200
	maxSourceLines       = 400
	maxSourceLineChars   = 240

	// sourceRetryAfter keeps a failed download from being retried on every call.
	sourceRetryAfter = 5 * time.Minute
)

// sourceStore downloads the public repository at the deployed ref once and serves searches and reads from a
// local copy. Only hand-written behavior is kept: Go, SQL, proto, and Markdown, minus tests and generated code.
type sourceStore struct {
	client     *http.Client
	ref        func() string
	archiveURL func(ref string) string

	mu       sync.Mutex
	tree     *sourceTree
	failed   error
	failedAt time.Time
}

type sourceTree struct {
	ref   string
	dir   string
	files []string
	index map[string]bool
}

var defaultSourceStore = &sourceStore{
	client: &http.Client{Timeout: 2 * time.Minute},
	ref:    buildinfo.SourceRef,
	archiveURL: func(ref string) string {
		return "https://codeload.github.com/" + sourceRepo + "/tar.gz/" + ref
	},
}

func (s *sourceStore) load(ctx context.Context) (*sourceTree, error) {
	ref := s.ref()
	if ref == "" {
		return nil, errors.New("source tools are unavailable: this build is not pinned to a released version of the source")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tree != nil && s.tree.ref == ref {
		return s.tree, nil
	}
	if s.failed != nil && time.Since(s.failedAt) < sourceRetryAfter {
		return nil, s.failed
	}
	// One run's cancellation shouldn't fail the shared download for everyone; the client timeout bounds it.
	tree, err := s.download(context.WithoutCancel(ctx), ref)
	if err != nil {
		s.failed, s.failedAt = fmt.Errorf("source tools are unavailable: could not download %s at %s: %w", sourceRepo, ref, err), time.Now()
		return nil, s.failed
	}
	s.tree, s.failed = tree, nil
	return tree, nil
}

func (s *sourceStore) download(ctx context.Context, ref string) (*sourceTree, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.archiveURL(ref), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req) // #nosec G704 -- fixed archive host; ref comes from the build, not the agent
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive returned HTTP %d", resp.StatusCode)
	}
	gz, err := gzip.NewReader(io.LimitReader(resp.Body, maxSourceArchiveBytes))
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "agent-source-")
	if err != nil {
		return nil, err
	}
	tree, err := extractSourceArchive(tar.NewReader(gz), dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	tree.ref = ref
	return tree, nil
}

// extractSourceArchive writes the kept files of a GitHub archive (whose entries share one top-level directory) under dir.
func extractSourceArchive(tr *tar.Reader, dir string) (*sourceTree, error) {
	tree := &sourceTree{dir: dir, index: map[string]bool{}}
	var total int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		_, rel, ok := strings.Cut(hdr.Name, "/")
		rel = path.Clean(rel)
		if !ok || rel == "." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) || !keepSourceFile(rel) || hdr.Size > maxSourceFileBytes {
			continue
		}
		if total += hdr.Size; total > maxSourceTreeBytes {
			return nil, errors.New("source tree exceeds the size limit")
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return nil, err
		}
		if err := writeSourceFile(dst, tr, hdr.Size); err != nil {
			return nil, err
		}
		tree.files = append(tree.files, rel)
		tree.index[rel] = true
	}
	slices.Sort(tree.files)
	return tree, nil
}

func writeSourceFile(dst string, r io.Reader, size int64) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G304 -- path cleaned and rooted in our temp dir
	if err != nil {
		return err
	}
	if _, err := io.CopyN(f, r, size); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// keepSourceFile keeps hand-written behavior: tests, mocks, and generated code would mostly crowd search results.
func keepSourceFile(rel string) bool {
	switch path.Ext(rel) {
	case ".go", ".sql", ".proto", ".md":
	default:
		return false
	}
	base := path.Base(rel)
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".pb.go") || strings.HasSuffix(base, ".pb.gw.go") || strings.HasSuffix(base, "_gen.go") {
		return false
	}
	for _, dir := range []string{"mock", "mocks", "sqlc", "testdata", "node_modules"} {
		if strings.Contains("/"+rel, "/"+dir+"/") {
			return false
		}
	}
	return !strings.HasPrefix(rel, "tests/")
}

// globRegexp compiles a path glob where * matches within a path segment and ** across segments.
func globRegexp(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch c := glob[i]; c {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				i++
				if i+1 < len(glob) && glob[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func HandleSearchSource(ctx context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	return defaultSourceStore.search(ctx, input)
}

func HandleReadSource(ctx context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	return defaultSourceStore.read(ctx, input)
}

func (s *sourceStore) search(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Query      string `json:"query"`
		PathGlob   string `json:"path_glob"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid search_source input: %w", err)
	}
	if strings.TrimSpace(params.Query) == "" {
		return "", errors.New("search_source: query is required")
	}
	pattern := params.Query
	// Smart case: an all-lowercase query matches case-insensitively.
	if strings.ToLower(pattern) == pattern {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("search_source: invalid regular expression: %w", err)
	}
	var glob *regexp.Regexp
	if params.PathGlob != "" {
		if glob, err = globRegexp(params.PathGlob); err != nil {
			return "", fmt.Errorf("search_source: invalid path_glob: %w", err)
		}
	}
	limit := params.MaxResults
	if limit <= 0 {
		limit = defaultSourceResults
	}
	limit = min(limit, maxSourceResults)

	tree, err := s.load(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	matches := 0
	for _, rel := range tree.files {
		if glob != nil && !glob.MatchString(rel) {
			continue
		}
		if err := scanSourceFile(filepath.Join(tree.dir, filepath.FromSlash(rel)), func(n int, line string) bool {
			if !re.MatchString(line) {
				return true
			}
			fmt.Fprintf(&b, "%s:%d: %s\n", rel, n, truncateRunes(strings.TrimSpace(line), maxSourceLineChars))
			matches++
			return matches < limit && b.Len() < maxFetchOutputBytes
		}); err != nil {
			return "", fmt.Errorf("search_source: %w", err)
		}
		if matches >= limit || b.Len() >= maxFetchOutputBytes {
			fmt.Fprintf(&b, "[stopped at %d matches; narrow the query or path_glob for more]\n", matches)
			break
		}
	}
	if matches == 0 {
		return fmt.Sprintf("No matches for %q in %s at %s.", params.Query, sourceRepo, tree.ref), nil
	}
	return fmt.Sprintf("Matches in %s at %s (read a file with read_source):\n%s", sourceRepo, tree.ref, b.String()), nil
}

// scanSourceFile calls fn with each 1-based line number and line until fn returns false.
func scanSourceFile(file string, fn func(n int, line string) bool) error {
	f, err := os.Open(file) // #nosec G304 -- file comes from the extracted tree's index
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), maxSourceFileBytes)
	for n := 1; sc.Scan(); n++ {
		if !fn(n, sc.Text()) {
			return nil
		}
	}
	return sc.Err()
}

func (s *sourceStore) read(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid read_source input: %w", err)
	}
	rel := path.Clean(strings.TrimPrefix(strings.TrimSpace(params.Path), "/"))
	start := max(params.Offset, 1)
	limit := params.Limit
	if limit <= 0 {
		limit = defaultSourceLines
	}
	limit = min(limit, maxSourceLines)

	tree, err := s.load(ctx)
	if err != nil {
		return "", err
	}
	if !tree.index[rel] {
		return "", fmt.Errorf("read_source: %q is not a searchable file in %s at %s; use search_source to find paths", params.Path, sourceRepo, tree.ref)
	}
	var b strings.Builder
	last, total := 0, 0
	err = scanSourceFile(filepath.Join(tree.dir, filepath.FromSlash(rel)), func(n int, line string) bool {
		total = n
		if n >= start && n < start+limit && b.Len() < maxFetchOutputBytes {
			fmt.Fprintf(&b, "%6d\t%s\n", n, line)
			last = n
		}
		return true
	})
	if err != nil {
		return "", fmt.Errorf("read_source: %w", err)
	}
	if last == 0 {
		return fmt.Sprintf("%s has %d lines; offset %d is past the end.", rel, total, start), nil
	}
	return fmt.Sprintf("%s at %s, lines %d-%d of %d:\n%s", rel, tree.ref, start, last, total, b.String()), nil
}
