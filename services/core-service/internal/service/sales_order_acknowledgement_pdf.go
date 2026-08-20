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
// document-title block (right: "ORDER ACKNOWLEDGEMENT" + order/customer identity).
func ackHeader(pdf *fpdf.Fpdf, data ackData) {
	startY := pdf.GetY()

	// --- Left: account letterhead (logo above the name, when available) ---
	nameY := startY
	if h := ackDrawLogo(pdf, data, startY); h > 0 {
		nameY = startY + h + 2
	}
	pdf.SetXY(ackPageLeft, nameY)
	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(98, 8, data.AccountName, "", 2, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10.5)
	// Address, then the merchant's support email and phone (from account branding).
	for _, line := range nonEmpty(data.AccountAddress.Line1, data.AccountAddress.Line2, data.AccountAddress.CityStateZip, data.AccountPhone, data.AccountEmail) {
		pdf.CellFormat(98, 5.6, line, "", 2, "L", false, 0, "")
	}
	leftEndY := pdf.GetY()

	// --- Right: document title + identity block, all left-aligned within the block ---
	title := data.DocumentTitle
	if title == "" {
		title = "ORDER ACKNOWLEDGEMENT"
	}
	numberLabel := data.NumberLabel
	if numberLabel == "" {
		numberLabel = "Sales Order Number"
	}
	titleX := 118.0
	pdf.SetXY(titleX, startY)
	pdf.SetFont("Helvetica", "", 18)
	pdf.CellFormat(ackPageRight-titleX, 9, title, "", 2, "R", false, 0, "")
	pdf.Ln(3)

	ackIdentityRow(pdf, titleX, numberLabel, data.OrderNumber, true)
	if data.CustomerPO != "" {
		ackIdentityRow(pdf, titleX, "PO Number", data.CustomerPO, false)
	}
	if data.CustomerNumber != "" {
		ackIdentityRow(pdf, titleX, "Customer Number", data.CustomerNumber, false)
	}
	ackIdentityRow(pdf, titleX, "Date", data.OrderDateLong, false)
	rightEndY := pdf.GetY()

	pdf.SetXY(ackPageLeft, maxF(leftEndY, rightEndY))
}

// ackIdentityRow renders a left-aligned "Label   Value" pair in the header block:
// the label in a fixed-width column and the value left-aligned beside it.
func ackIdentityRow(pdf *fpdf.Fpdf, leftX float64, label, value string, main bool) {
	y := pdf.GetY()
	const labelW = 40.0
	valueX := leftX + labelW
	if main {
		pdf.SetFont("Helvetica", "B", 11.5)
	} else {
		pdf.SetFont("Helvetica", "", 10.5)
	}
	pdf.SetXY(leftX, y)
	pdf.CellFormat(labelW, 6.4, label, "", 0, "L", false, 0, "")
	if main {
		pdf.SetFont("Helvetica", "B", 11.5)
	} else {
		pdf.SetFont("Helvetica", "", 10.5)
	}
	pdf.SetXY(valueX, y)
	pdf.CellFormat(ackPageRight-valueX, 6.4, value, "", 0, "L", false, 0, "")
	pdf.SetXY(leftX, y+6.4)
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
	// Price ("$8.50 / pr") and Qty ("1,200 pair") are wider to fit their units.
	wLine, wSKU, wDesc, wPrice, wQty, wTotal := 20.0, 26.0, 51.0, 30.0, 27.0, 26.0

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
		pdf.CellFormat(wLine, rowH, line.LineItem, "B", 0, "L", false, 0, "")
		pdf.CellFormat(wSKU, rowH, truncate(line.SKU, 20), "B", 0, "L", false, 0, "")
		// Description can wrap to multiple lines; draw it as a multi-line block.
		dx := x + wLine + wSKU
		pdf.SetXY(dx, y)
		pdf.MultiCell(wDesc, 7.5, strings.Join(desc, "\n"), "B", "L", false)
		pdf.SetXY(dx+wDesc, y)
		pdf.CellFormat(wPrice, rowH, line.Price, "B", 0, "R", false, 0, "")
		pdf.CellFormat(wQty, rowH, line.Qty, "B", 0, "R", false, 0, "")
		pdf.CellFormat(wTotal, rowH, line.Total, "B", 1, "R", false, 0, "")
	}

	// Total Due footer (right-aligned into the last two columns).
	pdf.SetFont("Helvetica", "B", 10.5)
	pdf.CellFormat(wLine+wSKU+wDesc+wPrice, 10, "", "", 0, "R", false, 0, "")
	pdf.CellFormat(wQty, 10, "Total Due:", "", 0, "R", false, 0, "")
	pdf.CellFormat(wTotal, 10, data.OrderTotal, "", 1, "R", false, 0, "")
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
