package service

import (
	_ "embed"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Text setup and fitting for the record PDFs (invoice, order acknowledgement, purchase order).
//
// The documents are set in IBM Plex Sans, the dashboard's typeface, so the emailed PDF reads as the
// same document the merchant previews. It is embedded as a UTF-8 font: fpdf's core fonts are cp1252,
// which turned the truncation ellipsis into "â€¦" and would do the same to any accented name.
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

var (
	//go:embed fonts/IBMPlexSans-Regular.ttf
	plexSansRegular []byte
	//go:embed fonts/IBMPlexSans-SemiBold.ttf
	plexSansSemiBold []byte
	//go:embed fonts/IBMPlexSans-Bold.ttf
	plexSansBold []byte
)

const (
	// ellipsis marks a value the column could not hold even at its smallest size.
	ellipsis = "…"

	pdfFontFamily = "IBMPlexSans"
	// fpdf knows only regular and bold per family, so the dashboard's font-semibold is its own family.
	pdfFontFamilySemiBold = "IBMPlexSansSemiBold"

	// Font styles accepted wherever these documents take a style: "" regular, "B" bold (700) and
	// "S" semibold (600), the three weights the dashboard's templates use.
	pdfStyleRegular  = ""
	pdfStyleBold     = "B"
	pdfStyleSemiBold = "S"
)

// newRecordPDF starts a record document on the dashboard's letter stock, with the fonts registered
// and fpdf's own cell margin removed so every inset on the page is one these renderers chose.
func newRecordPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.AddUTF8FontFromBytes(pdfFontFamily, "", plexSansRegular)
	pdf.AddUTF8FontFromBytes(pdfFontFamily, "B", plexSansBold)
	pdf.AddUTF8FontFromBytes(pdfFontFamilySemiBold, "", plexSansSemiBold)
	pdf.SetMargins(docPageMargin, docPageMargin, docPageMargin)
	pdf.SetCellMargin(0)
	// Tables break pages themselves so they can repeat their header row.
	pdf.SetAutoPageBreak(false, docPageMargin)
	pdf.AddPage()
	return pdf
}

// pdfSetFont selects the document typeface at one of the pdfStyle weights.
func pdfSetFont(pdf *fpdf.Fpdf, style string, size float64) {
	if style == pdfStyleSemiBold {
		pdf.SetFont(pdfFontFamilySemiBold, "", size)
		return
	}
	pdf.SetFont(pdfFontFamily, style, size)
}

// pdfFitSize returns the largest font size not exceeding maxSize at which text fits within maxW,
// down to minSize. It leaves the font set to the returned size.
//
// Sizes step down in half points: a finer step buys nothing at print resolution and costs a
// measurement per step.
func pdfFitSize(pdf *fpdf.Fpdf, text, style string, maxSize, minSize, maxW float64) float64 {
	size := maxSize
	for size > minSize {
		pdfSetFont(pdf, style, size)
		if pdf.GetStringWidth(text) <= maxW {
			return size
		}
		size -= 0.5
	}
	pdfSetFont(pdf, style, minSize)
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
		// marker alone rather than painting over the neighbor.
		return ellipsis
	}
	return strings.TrimRight(string(runes[:lo]), " ") + ellipsis
}

// pdfTrackedWidth is the width text occupies with letter-spacing applied after every character, as
// CSS tracking does.
func pdfTrackedWidth(pdf *fpdf.Fpdf, text string, tracking float64) float64 {
	return pdf.GetStringWidth(text) + tracking*float64(len([]rune(text)))
}

// pdfDrawTracked draws one line of letter-spaced text with its box's left edge at x. fpdf has no
// character-spacing setting, so the PDF Tc operator is set around the run and cleared after it.
func pdfDrawTracked(pdf *fpdf.Fpdf, x, y, h float64, text string, tracking float64) {
	pdf.SetXY(x, y)
	if tracking == 0 {
		pdf.CellFormat(pdf.GetStringWidth(text), h, text, "", 0, "L", false, 0, "")
		return
	}
	// Tc is in text-space units, which at these font matrices are points.
	pdf.RawWriteStr(strconv.FormatFloat(tracking*pdf.GetConversionRatio(), 'f', 3, 64) + " Tc")
	pdf.CellFormat(pdfTrackedWidth(pdf, text, tracking), h, text, "", 0, "L", false, 0, "")
	pdf.RawWriteStr("0 Tc")
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
	// Padding is the space kept clear inside the cell so adjacent columns never touch.
	Padding float64
}

// draw renders the cell, fitting its text to the available width first.
func (c pdfCellText) draw(pdf *fpdf.Fpdf) {
	avail := c.W - 2*c.Padding
	if avail <= 0 {
		avail = c.W
	}

	minSize := c.MinSize
	if minSize == 0 || minSize > c.Size {
		minSize = c.Size
	}

	pdfFitSize(pdf, c.Text, c.Style, c.Size, minSize, avail)
	text := pdfTruncateToWidth(pdf, c.Text, avail)

	// The padding is applied by insetting the text, so borders and fills still span the whole cell.
	pdf.SetCellMargin(c.Padding)
	pdf.CellFormat(c.W, c.H, text, c.Border, c.Ln, c.Align, c.Fill, 0, "")
	pdf.SetCellMargin(0)
}
