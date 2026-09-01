package service

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Page geometry (A4, 15mm margins => 180mm content width).
const (
	ackPageLeft = 15.0
	// The dashboard prints onto letter stock whose padding starts the content lower than a bare
	// print margin would, so match where its letterhead lands.
	ackPageTop      = 20.0
	ackPageRight    = 195.0
	ackContentWidth = ackPageRight - ackPageLeft

	// The header splits into a letterhead on the left and the document-title block on the right. The
	// split is fixed so every record PDF has the same shape; the text inside each half is fitted to
	// it rather than the halves being sized to the text, which would make one document's header sit
	// somewhere different from the next.
	ackLetterheadW = 90.0
	ackIdentityX   = 111.0
	ackIdentityW   = ackPageRight - ackIdentityX
	// ackIdentityGap separates a label from its value. Without it the two runs abut and read as one
	// word, which is what "Purchase Order Number001000" was.
	ackIdentityGap = 4.0
	// ackIdentityLabelMaxW caps the label column so a long label cannot squeeze its value to nothing;
	// past this the label itself shrinks instead.
	ackIdentityLabelMaxW = ackIdentityW * 0.62

	// ackCellPadding is the space kept clear inside every table cell. Without it a value that exactly
	// fills its column touches the one beside it and the two read as one string.
	ackCellPadding = 1.5
)

// buildOrderAcknowledgementPDF renders the order-acknowledgement PDF, mirroring the
// legacy OrderAcknowledgementPdf layout: an account letterhead with a right-aligned
// document title block, bill-to / ship-to addresses, order terms, a line-item table
// (Line Item / SKU / Description / Price / Qty / Total), and a Total Due footer.
func buildOrderAcknowledgementPDF(data ackData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(ackPageLeft, ackPageTop, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

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

// ackHeader renders the letterhead (left: account name + address + contact) and the
// document-title block (right: the document title + the record's identity rows).
//
// Both halves fit their text to a fixed column rather than assuming it fits: the account name, the
// title and every label and value here are variable-length, and the widest of them overflowed.
func ackHeader(pdf *fpdf.Fpdf, data ackData) {
	startY := pdf.GetY()

	// --- Left: account letterhead (logo above the name, when available) ---
	nameY := startY
	if h := ackDrawLogo(pdf, data, startY); h > 0 {
		nameY = startY + h + 2
	}
	pdf.SetXY(ackPageLeft, nameY)
	// A long trading name shrinks rather than running under the title block.
	pdfCellText{W: ackLetterheadW, H: 8, Text: data.AccountName, Ln: 2, Align: "L", Style: "B", Size: 15, MinSize: 10}.draw(pdf)
	// Address, then the merchant's support email and phone (from account branding).
	for _, line := range nonEmpty(data.AccountAddress.Line1, data.AccountAddress.Line2, data.AccountAddress.CityStateZip, data.AccountPhone, data.AccountEmail) {
		pdfCellText{W: ackLetterheadW, H: 5.6, Text: line, Ln: 2, Align: "L", Size: 10.5, MinSize: 8}.draw(pdf)
	}
	leftEndY := pdf.GetY()

	// --- Right: document title + identity rows ---
	rows := data.identityRows()

	pdf.SetXY(ackIdentityX, startY)
	// "ORDER ACKNOWLEDGEMENT" is half again as wide as "INVOICE" at the same size, so the title is
	// fitted to the block instead of every title being set at whatever size the shortest one allows.
	pdfCellText{W: ackIdentityW, H: 9, Text: data.documentTitle(), Ln: 2, Align: "R", Size: 18, MinSize: 11}.draw(pdf)
	pdf.Ln(3)

	labelW := ackIdentityLabelWidth(pdf, rows)
	for _, row := range rows {
		ackIdentityRow(pdf, labelW, row)
	}
	rightEndY := pdf.GetY()

	pdf.SetXY(ackPageLeft, maxF(leftEndY, rightEndY))
}

// ackIdentityLabelWidth sizes the label column to the widest label actually present, so the values
// line up with each other and never overlap the labels. Capped, past which the labels shrink.
func ackIdentityLabelWidth(pdf *fpdf.Fpdf, rows []ackIdentityField) float64 {
	widest := 0.0
	for _, row := range rows {
		pdf.SetFont("Helvetica", row.style(), row.size())
		widest = maxF(widest, pdf.GetStringWidth(row.Label))
	}
	return minF(widest+ackIdentityGap, ackIdentityLabelMaxW)
}

// ackIdentityRow renders one "Label   Value" pair in the header block. Both halves are fitted to
// their column, so a label that outgrows the column shrinks instead of running into its value.
func ackIdentityRow(pdf *fpdf.Fpdf, labelW float64, row ackIdentityField) {
	y := pdf.GetY()

	pdf.SetXY(ackIdentityX, y)
	pdfCellText{W: labelW, H: 6.4, Text: row.Label, Align: "L", Style: row.style(), Size: row.size(), MinSize: 8, Padding: 0.5}.draw(pdf)

	pdf.SetXY(ackIdentityX+labelW, y)
	pdfCellText{W: ackIdentityW - labelW, H: 6.4, Text: row.Value, Align: "L", Style: row.style(), Size: row.size(), MinSize: 8, Padding: 0.5}.draw(pdf)

	pdf.SetXY(ackIdentityX, y+6.4)
}

// ackCustomerAddresses renders the two-column Bill To / Ship To block.
func ackCustomerAddresses(pdf *fpdf.Fpdf, data ackData) {
	right := data.ShipTo
	rightTitle := "Ship To"
	if !data.HasShipTo {
		right = ackAddress{}
		rightTitle = ""
	}
	ackTwoColumnBlock(pdf, "Bill To", ackBillToBody(data), rightTitle, ackAddressLines(right))
}

// ackBillToBody renders the Bill To block: the billing address followed by the
// order's acknowledgement contact emails and the billing phone (mirrors legacy).
func ackBillToBody(data ackData) string {
	lines := nonEmpty(data.BillTo.Name, data.BillTo.Line1, data.BillTo.Line2, data.BillTo.CityStateZip)
	for _, e := range data.ContactEmails {
		if strings.TrimSpace(e) != "" {
			lines = append(lines, e)
		}
	}
	if data.BillTo.Phone != "" {
		lines = append(lines, data.BillTo.Phone)
	}
	if len(lines) == 0 {
		return "—"
	}
	return strings.Join(lines, "\n")
}

// ackOrderTerms renders the four order-term columns (Ship Via / Priority / Terms /
// Representative), skipping any with no value.
func ackOrderTerms(pdf *fpdf.Fpdf, data ackData) {
	type term struct{ title, value string }
	terms := []term{
		{"Ship Via", data.Carrier},
		{"Priority", data.Priority},
		{"Terms", data.PaymentTerms},
		{"Representative", data.SalesRep},
	}
	present := make([]term, 0, len(terms))
	for _, t := range terms {
		if strings.TrimSpace(t.value) != "" {
			present = append(present, t)
		}
	}
	if len(present) == 0 {
		return
	}

	colW := ackContentWidth / float64(len(present))
	startY := pdf.GetY()
	for i, t := range present {
		x := ackPageLeft + float64(i)*colW
		pdf.SetXY(x, startY)
		ackOverline(pdf, colW, t.title)
		pdf.SetXY(x, pdf.GetY())
		pdf.SetFont("Helvetica", "", 10.5)
		pdf.MultiCell(colW, 6.4, t.value, "", "L", false)
	}
}

// ackOrderSummary renders the line-item table and the Total Due footer.
func ackOrderSummary(pdf *fpdf.Fpdf, data ackData) {
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 11, "Order Summary", "", 1, "L", false, 0, "")

	// Columns: Line Item | SKU | Description | Price | Qty | Total (sum = 180mm).
	//
	// Sized from the measured width of each column's header and its widest realistic value, plus the
	// cell padding, so ordinary content is never shrunk: a SKU like "SOCK-CREW-BLK" is 28.3mm, which
	// is why SKU is 32 and not the 26 that squeezed it. Description takes the remainder and wraps.
	wLine, wSKU, wDesc, wPrice, wQty, wTotal := 18.0, 32.0, 57.0, 26.0, 23.0, 24.0

	pdf.SetFont("Helvetica", "B", 9.5)
	pdf.SetFillColor(243, 244, 246)
	pdf.SetDrawColor(229, 231, 235)
	pdf.CellFormat(wLine, 9, "Line Item", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wSKU, 9, "SKU", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wDesc, 9, "Description", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wPrice, 9, "Price", "B", 0, "R", true, 0, "")
	pdf.CellFormat(wQty, 9, "Qty", "B", 0, "R", true, 0, "")
	pdf.CellFormat(wTotal, 9, "Total", "B", 1, "R", true, 0, "")

	pdf.SetFont("Helvetica", "", 9.5)
	pdf.SetDrawColor(229, 231, 235)
	for _, line := range data.Lines {
		desc := ackWrap(pdf, line.Description, wDesc-2)
		rowH := 7.5 * float64(len(desc))

		x, y := pdf.GetX(), pdf.GetY()
		cell := func(w float64, text, align string) {
			pdfCellText{W: w, H: rowH, Text: text, Border: "B", Align: align, Size: 9.5, MinSize: 6.5, Padding: ackCellPadding}.draw(pdf)
		}
		cell(wLine, line.LineItem, "L")
		cell(wSKU, line.SKU, "L")
		// Description can wrap to multiple lines; draw it as a multi-line block.
		dx := x + wLine + wSKU
		pdf.SetXY(dx, y)
		pdf.SetFont("Helvetica", "", 9.5)
		pdf.MultiCell(wDesc, 7.5, strings.Join(desc, "\n"), "B", "L", false)
		pdf.SetXY(dx+wDesc, y)
		// A price carrying its pricing unit, or a quantity carrying a spelled-out unit name, is the
		// widest thing in these rows and the first to collide with the column beside it.
		cell(wPrice, line.Price, "R")
		cell(wQty, line.Qty, "R")
		cell(wTotal, line.Total, "R")
		pdf.Ln(-1)
	}

	// Total Due footer (right-aligned into the last two columns).
	pdf.SetFont("Helvetica", "B", 10.5)
	pdf.CellFormat(wLine+wSKU+wDesc+wPrice, 10, "", "", 0, "R", false, 0, "")
	pdf.CellFormat(wQty, 10, "Total Due:", "", 0, "R", false, 0, "")
	pdfCellText{W: wTotal, H: 10, Text: data.OrderTotal, Ln: 1, Align: "R", Style: "B", Size: 10.5, MinSize: 7}.draw(pdf)
}

// --- helpers ---

// ackDrawLogo embeds the account logo at the top-left, scaled to a 0.5in height
// (object-contain within a max width), and returns its drawn height in mm. Returns
// 0 when there is no logo or the image can't be decoded, leaving a text-only
// letterhead. A decode failure is cleared so it doesn't fail PDF output.
func ackDrawLogo(pdf *fpdf.Fpdf, data ackData, y float64) float64 {
	if len(data.LogoImage) == 0 || data.LogoImageType == "" {
		return 0
	}
	opts := fpdf.ImageOptions{ImageType: data.LogoImageType, ReadDpi: true}
	info := pdf.RegisterImageOptionsReader("ack_logo", opts, bytes.NewReader(data.LogoImage))
	if pdf.Err() || info == nil || info.Width() <= 0 || info.Height() <= 0 {
		pdf.ClearError()
		return 0
	}
	const targetH, maxW = 12.7, 50.0 // 0.5in tall, capped width
	ratio := info.Width() / info.Height()
	w, h := targetH*ratio, targetH
	if w > maxW {
		w, h = maxW, maxW/ratio
	}
	pdf.ImageOptions("ack_logo", ackPageLeft, y, w, h, false, opts, 0, "")
	return h
}

func ackHR(pdf *fpdf.Fpdf) {
	pdf.Ln(6)
	pdf.SetDrawColor(229, 231, 235)
	y := pdf.GetY()
	pdf.Line(ackPageLeft, y, ackPageRight, y)
	pdf.Ln(7.5)
}

func ackOverline(pdf *fpdf.Fpdf, w float64, title string) {
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(120, 128, 140)
	pdf.CellFormat(w, 6.5, strings.ToUpper(title), "", 2, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func ackTwoColumnBlock(pdf *fpdf.Fpdf, leftTitle, leftBody, rightTitle, rightBody string) {
	colW := ackContentWidth / 2
	startY := pdf.GetY()

	pdf.SetXY(ackPageLeft, startY)
	ackOverline(pdf, colW, leftTitle)
	pdf.SetXY(ackPageLeft, pdf.GetY())
	pdf.SetFont("Helvetica", "", 10.5)
	pdf.MultiCell(colW, 5.6, leftBody, "", "L", false)
	leftEndY := pdf.GetY()

	if rightTitle != "" {
		rightX := ackPageLeft + colW
		pdf.SetXY(rightX, startY)
		ackOverline(pdf, colW, rightTitle)
		pdf.SetXY(rightX, pdf.GetY())
		pdf.SetFont("Helvetica", "", 10.5)
		pdf.MultiCell(colW, 5.6, rightBody, "", "L", false)
	}

	pdf.SetXY(ackPageLeft, maxF(leftEndY, pdf.GetY()))
}

// ackAddressLines renders an address block as newline-joined lines (name bold is
// handled by the caller's font; here we keep it plain to match the compact block).
func ackAddressLines(a ackAddress) string {
	lines := nonEmpty(a.Name, a.Line1, a.Line2, a.CityStateZip, a.Phone, a.Email)
	if len(lines) == 0 {
		return "—"
	}
	return strings.Join(lines, "\n")
}

// ackWrap splits text into lines that fit within width w (mm) at the current font.
func ackWrap(pdf *fpdf.Fpdf, text string, w float64) []string {
	if strings.TrimSpace(text) == "" {
		return []string{""}
	}
	lines := pdf.SplitLines([]byte(text), w)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, string(l))
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
