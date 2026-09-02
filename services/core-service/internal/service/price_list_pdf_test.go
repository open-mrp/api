package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-pdf/fpdf"
)

func testRenderer() *priceListRenderer {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(plPageLeft, 15, 15)
	pdf.AddPage()
	return &priceListRenderer{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}
}

func testSection(descriptions ...string) (priceListLine, priceListSection) {
	rows := make([]priceListRow, 0, len(descriptions))
	for i, description := range descriptions {
		rows = append(rows, priceListRow{
			SKU:         "6801" + string(rune('0'+i%10)),
			Description: description,
			Values:      []string{"Black", "Regular"},
			Packing:     "10 Pairs Per Carton",
			Prices:      []string{"$18.00"},
		})
	}
	line := priceListLine{ProductLineName: "Couture", BaseUnitName: "Pair"}
	return line, priceListSection{
		Heading: "15-20 mmHg",
		Columns: []string{"Color", "Length"},
		Tiers:   []priceListTier{{Label: "1+", Quantity: "1"}},
		Rows:    rows,
	}
}

func testDocument(rowCount int) priceListDocument {
	rows := make([]priceListRow, 0, rowCount)
	for i := range rowCount {
		color := "Black"
		if i%2 == 1 {
			color = "Khaki"
		}
		rows = append(rows, priceListRow{
			SKU:         "6801" + string(rune('0'+i%10)),
			Description: "Below Knee, Closed Toe",
			Values:      []string{color, "Regular"},
			Packing:     "10 Pairs Per Carton",
			Prices:      []string{"$18.00"},
		})
	}

	return priceListDocument{
		MerchantName: "Seller Co",
		CustomerName: "Healthcare and Co",
		DateLong:     "July 27, 2026",
		PaymentTerm:  "Advance",
		ShippingTerm: "F.O.B. Springfield, IL 62701",
		Lines: []priceListLine{{
			ProductLineID:   "pl_1",
			ProductLineName: "Couture",
			BaseUnitName:    "Pair",
			Sections: []priceListSection{{
				Heading: "15-20 mmHg · Dress Socks",
				Columns: []string{"Color", "Length"},
				Tiers:   []priceListTier{{Label: "1+", Quantity: "1"}},
				Rows:    rows,
			}},
		}},
	}
}

func TestBuildPriceListPDF_ProducesAPDF(t *testing.T) {
	body, err := buildPriceListPDF(testDocument(6))
	if err != nil {
		t.Fatalf("buildPriceListPDF: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Errorf("output is not a PDF, starts with %q", string(body[:min(8, len(body))]))
	}
	if len(body) < 1000 {
		t.Errorf("PDF is suspiciously small: %d bytes", len(body))
	}
}

// A table longer than one page must keep rendering rather than overflowing off the bottom, so the row count has to survive a page break.
func TestBuildPriceListPDF_PaginatesLongSections(t *testing.T) {
	short, err := buildPriceListPDF(testDocument(10))
	if err != nil {
		t.Fatalf("short: %v", err)
	}
	long, err := buildPriceListPDF(testDocument(400))
	if err != nil {
		t.Fatalf("long: %v", err)
	}
	if len(long) <= len(short) {
		t.Errorf("400 rows (%d bytes) did not produce more output than 10 rows (%d bytes)", len(long), len(short))
	}
}

// The table always spans the page: columns are sized to their contents and whatever is left over belongs to the description.
func TestBuildTable_ColumnsSpanTheContentWidth(t *testing.T) {
	r := testRenderer()
	table := r.buildTable(testSection("Below Knee, Closed Toe", "Below Knee, Open Toe"))

	total := 0.0
	for _, column := range table.Columns {
		if column.Width <= 0 {
			t.Fatalf("column %q has width %v", column.Header, column.Width)
		}
		total += column.Width
	}
	if diff := total - plContentWidth; diff > 0.01 || diff < -0.01 {
		t.Errorf("columns total %v, want %v", total, plContentWidth)
	}
}

// A description too long for its column has to wrap and take its row with it; truncating it is what hid the product from the reader.
func TestBuildTable_LongDescriptionWrapsAndGrowsItsRow(t *testing.T) {
	long := "Essence 15-20 mmHg Closed Toe Thigh Length with Silicone Top Band, Silky Nude, Size A, Regular Length"
	r := testRenderer()
	line, section := testSection("Short", long)
	table := r.buildTable(line, section)

	info := plInfoColumn(section)
	if lines := len(table.Cells[1][info].Lines); lines < 2 {
		t.Fatalf("long description wrapped to %d lines, want at least 2", lines)
	}
	if !strings.HasPrefix(table.Cells[1][info].Lines[0], "Essence") {
		t.Errorf("first line = %q, want the description from its start", table.Cells[1][info].Lines[0])
	}
	if table.Heights[1] <= table.Heights[0] {
		t.Errorf("wrapped row height %v is not taller than the single-line row %v", table.Heights[1], table.Heights[0])
	}
}

// A price list quotes a bare number, so the column it sits under is the only thing that says what the number buys.
func TestCostHeader_NamesTheUnitThePriceIsPer(t *testing.T) {
	pair := priceListTier{Label: "1+ pr", Quantity: "1", UnitName: "Pair", UnitAbbreviation: "pr"}
	carton := priceListTier{Label: "50+ ctn", Quantity: "50", UnitName: "Carton", UnitAbbreviation: "ctn"}

	cases := []struct {
		name  string
		line  priceListLine
		tiers []priceListTier
		tier  priceListTier
		want  string
	}{
		{
			name:  "one price column is headed with its unit",
			line:  priceListLine{BaseUnitName: "Pair"},
			tiers: []priceListTier{pair},
			tier:  pair,
			want:  "Cost Per Pair",
		},
		{
			name:  "a volume column carries its break above the unit",
			line:  priceListLine{BaseUnitName: "Pair"},
			tiers: []priceListTier{pair, carton},
			tier:  carton,
			want:  "50+ ctn\nCost Per Carton",
		},
		{
			// The price under a tier is per the tier's own unit, so a carton break must not be headed with the line's pair.
			name:  "the tier's unit wins over the product line's",
			line:  priceListLine{BaseUnitName: "Pair"},
			tiers: []priceListTier{pair, carton},
			tier:  priceListTier{Label: "50+ ctn", UnitName: "Carton"},
			want:  "50+ ctn\nCost Per Carton",
		},
		{
			// A tier with no unit was priced against each product's own base unit, which within a product line is the line's.
			name:  "a tier with no unit falls back to the product line's",
			line:  priceListLine{BaseUnitName: "Pair"},
			tiers: []priceListTier{{Label: "1+"}},
			tier:  priceListTier{Label: "1+"},
			want:  "Cost Per Pair",
		},
		{
			name:  "an unnamed unit says no more than it knows",
			line:  priceListLine{},
			tiers: []priceListTier{{Label: "1+"}},
			tier:  priceListTier{Label: "1+"},
			want:  "Cost",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := plCostHeader(tc.line, priceListSection{Tiers: tc.tiers}, tc.tier)
			if got != tc.want {
				t.Errorf("header = %q, want %q", got, tc.want)
			}
		})
	}
}

// A header that names its own lines has to be measured and drawn a line at a time, or the unit it was widened to carry is the first thing cut off it.
func TestBuildTable_MultiLineHeaderIsDrawnInFull(t *testing.T) {
	r := testRenderer()
	line, section := testSection("Below Knee, Closed Toe")
	section.Tiers = []priceListTier{
		{Label: "1+ pr", Quantity: "1", UnitName: "Pair"},
		{Label: "50+ ctn", Quantity: "50", UnitName: "Carton"},
	}
	section.Rows[0].Prices = []string{"$18.00", "$160.00"}
	table := r.buildTable(line, section)

	cost := len(table.Columns) - 1
	header := strings.Join(table.Header[cost].Lines, " ")
	if !strings.Contains(header, "50+ CTN") || !strings.Contains(header, "COST PER CARTON") {
		t.Errorf("header lines = %q, want the break and the unit", table.Header[cost].Lines)
	}
	if fits := plLinesThatFit(table.HeaderHeight); fits < len(table.Header[cost].Lines) {
		t.Errorf("header is %v tall, which fits %d of its %d lines", table.HeaderHeight, fits, len(table.Header[cost].Lines))
	}
}

// The height a cell is given and the lines it is then allowed to draw are computed from the same line count, so they have to agree exactly — off by one line and text that was measured to fit is truncated anyway.
func TestLinesThatFit_AgreesWithTheHeightItInverts(t *testing.T) {
	for lines := 1; lines <= 12; lines++ {
		if got := plLinesThatFit(plCellHeight(lines)); got != lines {
			t.Errorf("a cell sized for %d lines fits %d", lines, got)
		}
	}
}

// A merged run interrupted by a page break has to close at the bottom of the page and reopen at the top of the next, or the value it carries is lost for every row after the break.
func TestTableSplitAt_ReopensMergedRuns(t *testing.T) {
	r := testRenderer()
	line, section := testSection("Same", "Same", "Same", "Same")
	table := r.buildTable(line, section)

	pack := len(section.Columns) + 2
	if table.Cells[0][pack].Span != 4 {
		t.Fatalf("pack span = %d, want the whole section merged", table.Cells[0][pack].Span)
	}

	table.splitAt(2)
	if table.Cells[0][pack].Span != 2 {
		t.Errorf("span above the break = %d, want 2", table.Cells[0][pack].Span)
	}
	if table.Cells[2][pack].Span != 2 {
		t.Errorf("span below the break = %d, want 2", table.Cells[2][pack].Span)
	}
	if len(table.Cells[2][pack].Lines) == 0 {
		t.Error("the reopened cell lost its text")
	}
}

func TestBuildPriceListPDF_EmptyCatalogStillRenders(t *testing.T) {
	body, err := buildPriceListPDF(priceListDocument{
		MerchantName: "Seller Co",
		CustomerName: "Healthcare and Co",
		DateLong:     "July 27, 2026",
		Notes:        []string{"No products are assigned to this customer."},
	})
	if err != nil {
		t.Fatalf("buildPriceListPDF: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
}
