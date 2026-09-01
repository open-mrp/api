package service

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strings"
	"testing"
)

// Reading text back out of a generated PDF, so the document tests can assert what a supplier or
// customer actually sees rather than that some bytes were produced.
//
// The generator compresses its content streams, so asserting on the raw output would only ever
// confirm the "%PDF" header — which is what the previous tests did, and why a wrong column label or
// a price rounded to the wrong number of decimals could ship without failing anything.

var (
	pdfStreamRe = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	// Text-showing operators: "(...) Tj" and the array form "[(...) -250 (...)] TJ".
	pdfShowRe = regexp.MustCompile(`\((?:[^()\\]|\\.)*\)`)
)

// pdfText extracts the visible text of a generated PDF as one string per drawn run, in page order.
// Runs are returned separately because adjacent cells become adjacent runs, so a test can assert a
// label and its value independently.
func pdfText(t *testing.T, pdfBytes []byte) []string {
	t.Helper()

	var out []string
	for _, match := range pdfStreamRe.FindAllSubmatch(pdfBytes, -1) {
		content := match[1]
		// Streams are Flate-compressed unless the generator was told otherwise; an undecodable
		// stream is a font or image, not text, so it is skipped rather than failing the test.
		if inflated, err := inflate(content); err == nil {
			content = inflated
		}
		if !bytes.Contains(content, []byte("Tj")) && !bytes.Contains(content, []byte("TJ")) {
			continue
		}
		for _, lit := range pdfShowRe.FindAll(content, -1) {
			s := unescapePDFString(string(lit[1 : len(lit)-1]))
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// pdfContains reports whether any extracted run equals want after trimming. Exact rather than
// substring, so a test asserting "$8.5000 / pr" cannot be satisfied by "$8.50 / pr".
func pdfContains(runs []string, want string) bool {
	for _, r := range runs {
		if strings.TrimSpace(r) == want {
			return true
		}
	}
	return false
}

// pdfJoined flattens the runs for assertions about text that the generator may have split across
// cells (a wrapped description, say).
func pdfJoined(runs []string) string {
	return strings.Join(runs, "\n")
}

func inflate(b []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

// unescapePDFString resolves the escapes a PDF string literal can carry. Only the ones the generator
// emits matter: escaped parens and backslashes, plus the octal form for non-ASCII.
func unescapePDFString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch c := s[i]; c {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '(', ')', '\\':
			b.WriteByte(c)
		default:
			if c >= '0' && c <= '7' && i+2 < len(s) {
				v := (int(c-'0') << 6) | (int(s[i+1]-'0') << 3) | int(s[i+2]-'0')
				b.WriteByte(byte(v))
				i += 2
				continue
			}
			b.WriteByte(c)
		}
	}
	return b.String()
}

// The extractor is the thing every document assertion rests on, so it is itself tested: a helper
// that silently returned nothing would make every parity test below pass vacuously.
func TestPDFTextExtractorReadsGeneratedText(t *testing.T) {
	t.Parallel()

	data := ackData{
		DocumentTitle: "INVOICE",
		NumberLabel:   "Invoice Number",
		AccountName:   "Extractor Probe Co",
		OrderNumber:   "000123",
		Lines:         []ackLine{{LineItem: "001", SKU: "PROBE-SKU", Price: "$1.00", Qty: "2 ea", Total: "$2.00"}},
		OrderTotal:    "$2.00",
	}
	pdfBytes, err := buildPurchaseOrderPDF(data)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}

	runs := pdfText(t, pdfBytes)
	if len(runs) == 0 {
		t.Fatal("extractor found no text; every parity assertion built on it would pass vacuously")
	}
	for _, want := range []string{"Extractor Probe Co", "PROBE-SKU", "$2.00"} {
		if !pdfContains(runs, want) {
			t.Errorf("extractor did not find %q in %q", want, pdfJoined(runs))
		}
	}
}
