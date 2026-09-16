package service

import (
	"strings"

	"github.com/go-pdf/fpdf"
)

// The record PDFs' tables, drawn the way the dashboard's PdfTable lays them out in the browser: a
// semibold title, a gray header row, rows ruled underneath, every column left-aligned except the
// last, and columns sized to their content by the browser's automatic table layout.

// pdfColumn is one table column.
type pdfColumn struct {
	Title string
	// Wrap marks a column whose text may wrap when the table is short of width, as PdfTable's
	// overflow columns do. The others keep their text on one line for as long as the wrapping
	// columns can give way.
	Wrap bool
	// MaxWidth caps how wide the column may grow before its text wraps. Zero leaves it uncapped.
	MaxWidth float64
}

type pdfTable struct {
	Title   string
	Columns []pdfColumn
	Rows    [][]string
	// Footer fills the table's last len(Footer) columns in bold, the way PdfTable right-packs its
	// footer cells.
	Footer []string
}

// draw renders the title, then the table beneath it, and leaves the cursor below the table.
func (t pdfTable) draw(pdf *fpdf.Fpdf) {
	x := docPageMargin
	pdf.SetXY(x, pdf.GetY())
	pdfSetFont(pdf, pdfStyleSemiBold, docFontXL)
	pdf.CellFormat(docContentWidth, docLineXL, t.Title, "", 1, "L", false, 0, "")
	pdf.SetY(pdf.GetY() + docSectionGap)

	widths := t.columnWidths(pdf)

	t.drawHeader(pdf, widths)
	for _, row := range t.Rows {
		lines, h := t.layoutRow(pdf, row, widths, pdfStyleRegular)
		if pdf.GetY()+h > docPageBottom() {
			pdf.AddPage()
			t.drawHeader(pdf, widths)
		}
		t.drawRow(pdf, lines, widths, h, pdfStyleRegular, true)
	}

	if len(t.Footer) > 0 {
		cells := make([]string, len(t.Columns))
		copy(cells[len(cells)-len(t.Footer):], t.Footer)
		lines, h := t.layoutRow(pdf, cells, widths, pdfStyleBold)
		if pdf.GetY()+h > docPageBottom() {
			pdf.AddPage()
		}
		t.drawRow(pdf, lines, widths, h, pdfStyleBold, false)
	}
}

func (t pdfTable) drawHeader(pdf *fpdf.Fpdf, widths []float64) {
	titles := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		titles[i] = c.Title
	}
	lines, h := t.layoutRow(pdf, titles, widths, pdfStyleBold)

	y := pdf.GetY()
	pdfUseGray(pdf)
	pdf.Rect(docPageMargin, y, sum(widths), h, "F")
	t.drawRow(pdf, lines, widths, h, pdfStyleBold, false)
}

// layoutRow wraps each cell to its column and returns the lines with the row's height.
func (t pdfTable) layoutRow(pdf *fpdf.Fpdf, cells []string, widths []float64, style string) ([][]string, float64) {
	pdfSetFont(pdf, style, docFontXS)
	lines := make([][]string, len(widths))
	most := 1
	for i := range widths {
		text := ""
		if i < len(cells) {
			text = cells[i]
		}
		lines[i] = pdfWrap(pdf, text, widths[i]-2*docCellPadding)
		most = max(most, len(lines[i]))
	}
	return lines, 2*docCellPadding + float64(most)*docLineXS
}

// drawRow paints one row. Cells are vertically centered, as table cells are by default, so a single
// line sits level with the middle of a wrapped description beside it.
func (t pdfTable) drawRow(pdf *fpdf.Fpdf, lines [][]string, widths []float64, h float64, style string, rule bool) {
	y := pdf.GetY()
	x := docPageMargin
	last := len(widths) - 1
	for i, w := range widths {
		align := "L"
		if i == last {
			align = "R"
		}
		textY := y + (h-float64(len(lines[i]))*docLineXS)/2
		for j, line := range lines[i] {
			pdf.SetXY(x, textY+float64(j)*docLineXS)
			pdfCellText{W: w, H: docLineXS, Text: line, Align: align, Style: style, Size: docFontXS, MinSize: 6.5, Padding: docCellPadding}.draw(pdf)
		}
		x += w
	}
	if rule {
		pdfUseGray(pdf)
		pdf.Line(docPageMargin, y+h, docPageMargin+sum(widths), y+h)
	}
	pdf.SetXY(docPageMargin, y+h)
}

// columnWidths approximates the browser's automatic table layout, which is what sizes PdfTable's
// columns: each column wants its widest content (its max-content width) and can go no narrower than
// its longest word (its min-content width). When every column fits at its widest, the spare width is
// shared out in proportion to those widths. When they do not, the wrapping columns give way first,
// in proportion to how much they would still like, so a long description wraps before a price or a
// "Line Item" header does. Only when that is not enough does every column give way, which is what the
// browser does from the start.
func (t pdfTable) columnWidths(pdf *fpdf.Fpdf) []float64 {
	n := len(t.Columns)
	minW := make([]float64, n)
	maxW := make([]float64, n)

	measure := func(i int, text, style string) {
		pdfSetFont(pdf, style, docFontXS)
		maxW[i] = max(maxW[i], pdf.GetStringWidth(text))
		for _, word := range strings.Fields(text) {
			minW[i] = max(minW[i], pdf.GetStringWidth(word))
		}
	}
	for i, c := range t.Columns {
		measure(i, c.Title, pdfStyleBold)
		for _, row := range t.Rows {
			if i < len(row) {
				measure(i, row[i], pdfStyleRegular)
			}
		}
	}
	for j, text := range t.Footer {
		measure(n-len(t.Footer)+j, text, pdfStyleBold)
	}

	growable := make([]bool, n)
	for i, c := range t.Columns {
		// A hair over the measured width, so text measured to exactly fill its column still fits
		// when the same sum is recomputed at draw time.
		minW[i] += 2*docCellPadding + 0.01
		maxW[i] += 2*docCellPadding + 0.01
		if c.MaxWidth > 0 {
			maxW[i] = max(minW[i], min(maxW[i], c.MaxWidth))
		} else {
			growable[i] = true
		}
	}

	// The narrowest each column may go while only the wrapping columns wrap.
	softMin := make([]float64, n)
	for i, c := range t.Columns {
		softMin[i] = maxW[i]
		if c.Wrap {
			softMin[i] = minW[i]
		}
	}

	widths := make([]float64, n)
	sumMax := sum(maxW)
	switch {
	case sumMax <= docContentWidth:
		share := 0.0
		for i := range maxW {
			if growable[i] {
				share += maxW[i]
			}
		}
		spare := docContentWidth - sumMax
		for i := range maxW {
			widths[i] = maxW[i]
			if growable[i] && share > 0 {
				widths[i] += spare * maxW[i] / share
			}
		}
	case sum(softMin) < docContentWidth:
		giveWay(widths, softMin, maxW)
	case sum(minW) < docContentWidth:
		giveWay(widths, minW, maxW)
	default:
		// Not even the longest words fit; scale everything down and let the cells shrink and
		// truncate their text.
		for i := range minW {
			widths[i] = minW[i] * docContentWidth / sum(minW)
		}
	}
	return widths
}

// giveWay fills the content width starting from each column's floor, sharing what is left in
// proportion to how far each column is from its preferred width.
func giveWay(widths, floor, preferred []float64) {
	spare := docContentWidth - sum(floor)
	wanted := sum(preferred) - sum(floor)
	for i := range widths {
		widths[i] = floor[i]
		if wanted > 0 {
			widths[i] += spare * (preferred[i] - floor[i]) / wanted
		}
	}
}

// pdfWrap breaks text into the lines it takes at width w in the current font. A single word wider
// than w stays one line; the cell drawing it shrinks or truncates it.
func pdfWrap(pdf *fpdf.Fpdf, text string, w float64) []string {
	if strings.TrimSpace(text) == "" {
		return []string{""}
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if pdf.GetStringWidth(line+" "+word) <= w {
				line += " " + word
				continue
			}
			out = append(out, line)
			line = word
		}
		out = append(out, line)
	}
	return out
}

func sum(vals []float64) float64 {
	total := 0.0
	for _, v := range vals {
		total += v
	}
	return total
}
