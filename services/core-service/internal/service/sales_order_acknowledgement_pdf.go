package service

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Layout shared by the record PDFs, taken from the dashboard templates they reproduce
// (PdfLetterStockBox, PdfHeader, PdfCustomerAddressSection, PdfOrderTerms, PdfTable). Those are sized
// in CSS pixels and Tailwind steps, so the values here are the same measurements converted at 96px to
// the inch.
//
// The dashboard prints the document as an 8.5in-wide box onto letter paper with half-inch margins,
// and the browser shrinks it to fit the 7.5in between them. docScale is that shrink, applied to every
// measurement and font size here, so the PDF is the page a merchant gets when they print.
const (
	docScale = 7.5 / 8.5
	docPx    = 25.4 / 96 * docScale

	docPageWidth    = 215.9
	docPageHeight   = 279.4
	docPageMargin   = 12.7
	docContentWidth = docPageWidth - 2*docPageMargin
	docPageRight    = docPageWidth - docPageMargin

	// gap-[0.25in] between the page's blocks, and the my-4 around each rule.
	docSectionGap = 24 * docPx
	docRuleMargin = 16 * docPx

	// Tailwind's text steps as font size and line height: xs 12/16px, sm 14/20px, base 16/24px,
	// lg 18/28px, xl 20/28px. A CSS pixel is three quarters of a point.
	docFontXS   = 12 * 0.75 * docScale
	docLineXS   = 16 * docPx
	docFontSM   = 14 * 0.75 * docScale
	docLineSM   = 20 * docPx
	docFontBase = 16 * 0.75 * docScale
	docLineBase = 24 * docPx
	docFontLG   = 18 * 0.75 * docScale
	docLineLG   = 28 * docPx
	docFontXL   = 20 * 0.75 * docScale
	docLineXL   = 28 * docPx

	// tracking-wider, in ems.
	docTracking = 0.05

	// p-2 inside every table cell.
	docCellPadding = 8 * docPx

	// The letterhead's logo box (h-[0.5in], max-w-[144px]) and the gap-6 beneath it.
	docLogoHeight   = 48 * docPx
	docLogoMaxWidth = 144 * docPx
	docLogoGap      = 24 * docPx

	// The identity block: gap-[0.1in] under the title, mr-2 between a label and its value, and the
	// widest the block may grow before its text is fitted instead. The cap keeps the letterhead from
	// being squeezed out by a pathological label.
	docTitleGap        = 9.6 * docPx
	docIdentityGap     = 8 * docPx
	docIdentityMaxW    = docContentWidth * 0.55
	docIdentityLabelMx = docIdentityMaxW * 0.62

	// pb-2 under an overline title.
	docOverlineGap = 8 * docPx

	// max-w-[2in] on the invoice's Description column.
	docDescriptionMaxW = 192 * docPx

	// gray-200, the dashboard's rule and table-header color, and #666 for overline titles.
	docGrayR, docGrayG, docGrayB = 229, 231, 235
	docMutedGray                 = 102
)

func docPageBottom() float64 { return docPageHeight - docPageMargin }

// docTrackingFor is tracking-wider in millimetres at a point size.
func docTrackingFor(size float64) float64 { return docTracking * size * 25.4 / 72 }

// pdfUseGray sets the fill and stroke to gray-200 with a one-pixel line.
func pdfUseGray(pdf *fpdf.Fpdf) {
	pdf.SetFillColor(docGrayR, docGrayG, docGrayB)
	pdf.SetDrawColor(docGrayR, docGrayG, docGrayB)
	pdf.SetLineWidth(docPx)
}

// buildOrderAcknowledgementPDF renders the order-acknowledgement PDF, mirroring the dashboard's
// OrderAcknowledgementPdf: the letterhead and identity block, bill-to / ship-to, order terms, and the
// order summary with its Total Due footer.
func buildOrderAcknowledgementPDF(data ackData) ([]byte, error) {
	pdf := newRecordPDF()

	ackHeader(pdf, data)
	ackHR(pdf)
	ackCustomerAddresses(pdf, data)
	ackHR(pdf)
	ackOrderTerms(pdf, data)
	ackHR(pdf)
	ackOrderSummary(pdf, data)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ackHeader renders the letterhead on the left and the document title with its identity rows on the
// right, both starting at the top of the page.
func ackHeader(pdf *fpdf.Fpdf, data ackData) {
	startY := pdf.GetY()

	// --- Right: document title + identity rows, flush to the right margin ---
	rows := data.identityRows()
	labelW, valueW := ackIdentityColumns(pdf, rows)
	tableW := labelW + valueW

	title := data.documentTitle()
	titleSize := docFontXL
	for ; titleSize > 10; titleSize -= 0.5 {
		pdfSetFont(pdf, pdfStyleRegular, titleSize)
		if pdfTrackedWidth(pdf, title, docTrackingFor(titleSize)) <= docIdentityMaxW {
			break
		}
	}
	pdfSetFont(pdf, pdfStyleRegular, titleSize)
	tracking := docTrackingFor(titleSize)
	title = pdfTruncateToWidth(pdf, title, docIdentityMaxW-tracking*float64(len([]rune(title))))
	titleW := pdfTrackedWidth(pdf, title, tracking)
	pdfDrawTracked(pdf, docPageRight-titleW, startY, docLineXL, title, tracking)

	y := startY + docLineXL + docTitleGap
	tableX := docPageRight - tableW
	for _, row := range rows {
		h := row.lineHeight()
		pdf.SetXY(tableX, y)
		pdfCellText{W: labelW - docIdentityGap, H: h, Text: row.Label, Align: "L", Style: row.style(), Size: row.labelSize(), MinSize: 8}.draw(pdf)
		pdf.SetXY(tableX+labelW, y)
		pdfCellText{W: valueW, H: h, Text: row.Value, Align: "L", Style: row.style(), Size: row.valueSize(), MinSize: 8}.draw(pdf)
		y += h
	}
	rightEndY := y

	// --- Left: account letterhead, in whatever the identity block leaves ---
	letterheadW := docContentWidth - max(tableW, titleW) - docSectionGap
	nameY := startY
	if ackDrawLogo(pdf, data, startY) {
		nameY = startY + docLogoHeight + docLogoGap
	}
	pdf.SetXY(docPageMargin, nameY)
	pdfCellText{W: letterheadW, H: docLineLG, Text: data.AccountName, Ln: 2, Align: "L", Style: pdfStyleSemiBold, Size: docFontLG, MinSize: 9}.draw(pdf)
	for _, line := range nonEmpty(data.AccountAddress.Line1, data.AccountAddress.Line2, data.AccountAddress.CityStateZip, data.AccountPhone, data.AccountEmail) {
		pdf.SetX(docPageMargin)
		pdfCellText{W: letterheadW, H: docLineSM, Text: line, Ln: 2, Align: "L", Size: docFontSM, MinSize: 8}.draw(pdf)
	}
	leftEndY := pdf.GetY()

	pdf.SetXY(docPageMargin, max(leftEndY, rightEndY))
}

// ackIdentityColumns sizes the identity table the way the browser sizes its two auto columns: the
// label column holds the widest label plus the gap before its value, and the value column the widest
// value. Past the block's cap the label column is held back and the values take what is left, with
// both fitted into their share.
func ackIdentityColumns(pdf *fpdf.Fpdf, rows []ackIdentityField) (labelW, valueW float64) {
	for _, row := range rows {
		pdfSetFont(pdf, row.style(), row.labelSize())
		labelW = max(labelW, pdf.GetStringWidth(row.Label))
		pdfSetFont(pdf, row.style(), row.valueSize())
		valueW = max(valueW, pdf.GetStringWidth(row.Value))
	}
	// A hair over the measured widths, so text measured to exactly fill its column still fits when
	// the column is recomputed at draw time.
	labelW += docIdentityGap + 0.01
	valueW += 0.01
	if labelW+valueW > docIdentityMaxW {
		labelW = min(labelW, docIdentityLabelMx)
		valueW = docIdentityMaxW - labelW
	}
	return labelW, valueW
}

// ackBlock is one titled column of the address or terms bands.
type ackBlock struct {
	Title string
	// Name leads the block in semibold, as an address's name does.
	Name  string
	Lines []string
}

func (b ackBlock) width(pdf *fpdf.Fpdf) float64 {
	pdfSetFont(pdf, pdfStyleRegular, docFontXS)
	w := pdfTrackedWidth(pdf, strings.ToUpper(b.Title), docTrackingFor(docFontXS))
	if b.Name != "" {
		pdfSetFont(pdf, pdfStyleSemiBold, docFontSM)
		w = max(w, pdf.GetStringWidth(b.Name))
	}
	pdfSetFont(pdf, pdfStyleRegular, docFontSM)
	for _, line := range b.Lines {
		w = max(w, pdf.GetStringWidth(line))
	}
	return w
}

// draw renders the block at x, wrapping its lines to w, and returns the y it ends at.
func (b ackBlock) draw(pdf *fpdf.Fpdf, x, y, w float64) float64 {
	pdf.SetTextColor(docMutedGray, docMutedGray, docMutedGray)
	pdfSetFont(pdf, pdfStyleRegular, docFontXS)
	title := strings.ToUpper(b.Title)
	tracking := docTrackingFor(docFontXS)
	pdfDrawTracked(pdf, x, y, docLineXS, pdfTruncateToWidth(pdf, title, w-tracking*float64(len([]rune(title)))), tracking)
	pdf.SetTextColor(0, 0, 0)
	y += docLineXS + docOverlineGap

	line := func(text, style string) {
		pdfSetFont(pdf, style, docFontSM)
		for _, part := range pdfWrap(pdf, text, w) {
			pdf.SetXY(x, y)
			pdfCellText{W: w, H: docLineSM, Text: part, Align: "L", Style: style, Size: docFontSM, MinSize: 8}.draw(pdf)
			y += docLineSM
		}
	}
	if b.Name != "" {
		line(b.Name, pdfStyleSemiBold)
	}
	for _, l := range b.Lines {
		line(l, pdfStyleRegular)
	}
	return y
}

// ackSpreadBlocks lays blocks across the content width like a justify-between flex row: the first
// flush left, the last flush right, the rest evenly between. With grow set, each block also takes an
// equal share of the spare width, as PdfCustomerAddressSection's flex-grow columns do, which starts
// the second block partway across rather than against the right margin. When the blocks cannot all
// fit side by side they split the width equally and wrap.
func ackSpreadBlocks(pdf *fpdf.Fpdf, blocks []ackBlock, grow bool) {
	if len(blocks) == 0 {
		return
	}

	widths := make([]float64, len(blocks))
	for i, b := range blocks {
		widths[i] = b.width(pdf)
	}
	spare := docContentWidth - sum(widths)
	if spare < 0 {
		for i := range widths {
			widths[i] = docContentWidth/float64(len(blocks)) - docSectionGap/2
		}
		spare = docContentWidth - sum(widths)
	}

	startY := pdf.GetY()
	endY := startY
	x := docPageMargin
	for i, b := range blocks {
		w := widths[i]
		switch {
		case grow:
			w += spare / float64(len(blocks))
		case i > 0:
			x += spare / float64(len(blocks)-1)
		}
		endY = max(endY, b.draw(pdf, x, startY, w))
		x += w
	}
	pdf.SetXY(docPageMargin, endY)
}

// ackCustomerAddresses renders the Bill To / Ship To band. Either side is left out when its address
// is empty, as the dashboard omits a missing address.
func ackCustomerAddresses(pdf *fpdf.Fpdf, data ackData) {
	var blocks []ackBlock
	if !data.BillTo.Empty() {
		lines := ackAddressLines(data.BillTo)
		for _, e := range data.ContactEmails {
			if strings.TrimSpace(e) != "" {
				lines = append(lines, lowerASCII(e))
			}
		}
		if data.ContactPhone != "" {
			lines = append(lines, data.ContactPhone)
		}
		blocks = append(blocks, ackBlock{Title: "Bill To", Name: data.BillTo.Name, Lines: lines})
	}
	if data.HasShipTo {
		blocks = append(blocks, ackBlock{Title: "Ship To", Name: data.ShipTo.Name, Lines: ackAddressLines(data.ShipTo)})
	}
	ackSpreadBlocks(pdf, blocks, true)
}

// ackOrderTerms renders the order-term band. Ship Via and Priority always show, Ship Via as a dash
// when the order names no carrier; Terms and Representative only when set.
func ackOrderTerms(pdf *fpdf.Fpdf, data ackData) {
	carrier := data.Carrier
	if strings.TrimSpace(carrier) == "" {
		carrier = "—"
	}
	blocks := []ackBlock{
		{Title: "Ship Via", Lines: []string{carrier}},
		{Title: "Priority", Lines: []string{data.Priority}},
	}
	if strings.TrimSpace(data.PaymentTerms) != "" {
		blocks = append(blocks, ackBlock{Title: "Terms", Lines: []string{data.PaymentTerms}})
	}
	if strings.TrimSpace(data.SalesRep) != "" {
		blocks = append(blocks, ackBlock{Title: "Representative", Lines: []string{data.SalesRep}})
	}
	ackSpreadBlocks(pdf, blocks, false)
}

// ackOrderSummary renders the line-item table and the Total Due footer.
func ackOrderSummary(pdf *fpdf.Fpdf, data ackData) {
	rows := make([][]string, len(data.Lines))
	for i, l := range data.Lines {
		rows[i] = []string{l.LineItem, l.SKU, l.Description, l.Price, l.Qty, l.Total}
	}
	pdfTable{
		Title: "Order Summary",
		Columns: []pdfColumn{
			{Title: "Line Item"}, {Title: "SKU", Wrap: true}, {Title: "Description", Wrap: true},
			{Title: "Price"}, {Title: "Qty"}, {Title: "Total"},
		},
		Rows:   rows,
		Footer: []string{"Total Due:", data.OrderTotal},
	}.draw(pdf)
}

// --- helpers ---

// ackDrawLogo embeds the account logo at the top-left the way the letterhead's img lays it out: a
// half-inch-tall box at most 144px wide, with the image contained and centered in it. Returns false
// when there is no logo or it cannot be decoded, leaving a text-only letterhead. A decode failure is
// cleared so it does not fail the PDF output.
func ackDrawLogo(pdf *fpdf.Fpdf, data ackData, y float64) bool {
	if len(data.LogoImage) == 0 || data.LogoImageType == "" {
		return false
	}
	opts := fpdf.ImageOptions{ImageType: data.LogoImageType, ReadDpi: true}
	info := pdf.RegisterImageOptionsReader("ack_logo", opts, bytes.NewReader(data.LogoImage))
	if pdf.Err() || info == nil || info.Width() <= 0 || info.Height() <= 0 {
		pdf.ClearError()
		return false
	}
	ratio := info.Width() / info.Height()
	w, h := docLogoHeight*ratio, docLogoHeight
	if w > docLogoMaxWidth {
		w, h = docLogoMaxWidth, docLogoMaxWidth/ratio
	}
	pdf.ImageOptions("ack_logo", docPageMargin, y+(docLogoHeight-h)/2, w, h, false, opts, 0, "")
	return true
}

// ackHR draws the gray rule between bands, with the band gap and the rule's own margin either side.
func ackHR(pdf *fpdf.Fpdf) {
	y := pdf.GetY() + docSectionGap + docRuleMargin
	pdfUseGray(pdf)
	pdf.Line(docPageMargin, y, docPageRight, y)
	pdf.SetXY(docPageMargin, y+docRuleMargin+docSectionGap)
}

// ackAddressLines is an address's street and city lines, without its name, which the block sets
// apart.
func ackAddressLines(a ackAddress) []string {
	return nonEmpty(a.Line1, a.Line2, a.CityStateZip)
}
