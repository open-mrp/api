package service

import (
	"strings"

	"github.com/go-pdf/fpdf"
)

// Text fitting for the record PDFs (invoice, order acknowledgement, purchase order).
//
// Every column in these documents is a fixed width in millimetres, but the text going into it is
// account- and item-supplied and set in a proportional font, so its width is not knowable from its
// length. Drawing it with a plain CellFormat overflows silently: fpdf neither wraps nor clips, it
// just keeps painting, and the run collides with whatever is drawn next. That is how "Purchase Order
// Number" came to sit on top of its own value, and how "ORDER ACKNOWLEDGEMENT" ran off the page.
//
// So nothing in these documents draws unmeasured text. A cell shrinks its font to fit, and only
// truncates once it has hit the floor — the point being that the reader sees the whole value
// wherever that is possible at all, and an unmistakable ellipsis where it is not.

const (
	// ellipsis marks a value the column could not hold even at its smallest size.
	ellipsis = "…"
)

// pdfFitSize returns the largest font size not exceeding maxSize at which text fits within maxW,
// down to minSize. It leaves the font set to the returned size.
//
// Sizes step down in half points: a finer step buys nothing at print resolution and costs a
// measurement per step.
func pdfFitSize(pdf *fpdf.Fpdf, text, style string, maxSize, minSize, maxW float64) float64 {
	size := maxSize
	for size > minSize {
		pdf.SetFont("Helvetica", style, size)
		if pdf.GetStringWidth(text) <= maxW {
			return size
		}
		size -= 0.5
	}
	pdf.SetFont("Helvetica", style, minSize)
	return minSize
}

// pdfTruncateToWidth shortens text until it fits maxW at the current font, appending an ellipsis.
// Returns text unchanged when it already fits.
//
// It measures rather than counting characters: a 14-character run of "W" is more than twice the
// width of one of "l", so a character budget either wraps early or overflows anyway.
func pdfTruncateToWidth(pdf *fpdf.Fpdf, text string, maxW float64) string {
	if pdf.GetStringWidth(text) <= maxW {
		return text
	}

	runes := []rune(text)
	// Binary search the longest prefix that fits with the ellipsis appended.
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if pdf.GetStringWidth(string(runes[:mid])+ellipsis) <= maxW {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		// Not even one character plus the ellipsis fits; the column is unusably narrow, so give the
		// marker alone rather than painting over the neighbour.
		return ellipsis
	}
	return strings.TrimRight(string(runes[:lo]), " ") + ellipsis
}

// pdfCellText is one measured cell: it shrinks to fit, then truncates, then draws.
//
// style/size/minSize describe the type; the font is left set to the size actually used, so callers
// that draw a run of cells at one size should pass the same values each time rather than assuming
// the font survives the call.
type pdfCellText struct {
	W, H    float64
	Text    string
	Border  string
	Ln      int
	Align   string
	Fill    bool
	Style   string
	Size    float64
	MinSize float64
	// Padding is the space kept clear inside the cell so adjacent columns never touch. Zero uses a
	// default of one millimetre either side.
	Padding float64
}

// draw renders the cell, fitting its text to the available width first.
func (c pdfCellText) draw(pdf *fpdf.Fpdf) {
	padding := c.Padding
	if padding == 0 {
		padding = 1
	}
	avail := c.W - 2*padding
	if avail <= 0 {
		avail = c.W
	}

	minSize := c.MinSize
	if minSize == 0 || minSize > c.Size {
		minSize = c.Size
	}

	pdfFitSize(pdf, c.Text, c.Style, c.Size, minSize, avail)
	text := pdfTruncateToWidth(pdf, c.Text, avail)
	pdf.CellFormat(c.W, c.H, text, c.Border, c.Ln, c.Align, c.Fill, 0, "")
}
