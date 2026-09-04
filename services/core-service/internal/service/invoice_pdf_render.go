package service

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Renders the invoice PDF customers receive on ship, mirroring the legacy InvoicePdf layout: the
// shared letterhead, addresses and terms, then the shipment's cases and the invoice summary.
func buildInvoicePDF(doc invoiceDoc) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(ackPageLeft, ackPageTop, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	ackHeader(pdf, doc.Header)
	ackHR(pdf)
	ackCustomerAddresses(pdf, doc.Header)
	ackHR(pdf)
	ackOrderTerms(pdf, doc.Header)
	ackHR(pdf)
	invoiceCaseTable(pdf, doc)
	invoiceSummary(pdf, doc)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Renders the Cases table listing what the shipment traveled in. Skipped when the shipment has no
// cases, matching legacy, which renders nothing rather than an empty table.
func invoiceCaseTable(pdf *fpdf.Fpdf, doc invoiceDoc) {
	if len(doc.Cases) == 0 {
		return
	}

	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 11, "Cases", "", 1, "L", false, 0, "")

	// Columns: Case Number | Weight | Tracking Number (sum = 180mm).
	wNumber, wWeight, wTracking := 55.0, 35.0, 90.0

	pdf.SetFont("Helvetica", "B", 9.5)
	pdf.SetFillColor(243, 244, 246)
	pdf.SetDrawColor(229, 231, 235)
	pdf.CellFormat(wNumber, 9, "Case Number", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wWeight, 9, "Weight", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wTracking, 9, "Tracking Number", "B", 1, "L", true, 0, "")

	for _, c := range doc.Cases {
		for _, cell := range []struct {
			w    float64
			text string
			ln   int
		}{{wNumber, c.Number, 0}, {wWeight, c.Weight, 0}, {wTracking, c.Tracking, 1}} {
			pdfCellText{W: cell.w, H: 7.5, Text: cell.text, Border: "B", Ln: cell.ln, Align: "L", Size: 9.5, MinSize: 6.5, Padding: ackCellPadding}.draw(pdf)
		}
	}

	pdf.Ln(4)
}

// Renders the invoice line table and the Total Due footer. Ordered and Invoiced sit side by side so
// the customer can see what was billed against what they asked for.
func invoiceSummary(pdf *fpdf.Fpdf, doc invoiceDoc) {
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 11, "Invoice Summary", "", 1, "L", false, 0, "")

	// Columns: Line Item | SKU | Description | Price | Ordered | Invoiced | Unit | Total (sum = 180mm).
	//
	// Eight columns in 180mm is the tightest table here, so each is sized to its header plus padding
	// and the widest ordinary value it carries; Description takes what is left and wraps.
	wLine, wSKU, wDesc, wPrice := 18.0, 32.0, 30.0, 25.0
	wOrdered, wInvoiced, wUnit, wTotal := 18.0, 18.0, 17.0, 22.0

	pdf.SetFont("Helvetica", "B", 9.5)
	pdf.SetFillColor(243, 244, 246)
	pdf.SetDrawColor(229, 231, 235)
	pdf.CellFormat(wLine, 9, "Line Item", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wSKU, 9, "SKU", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wDesc, 9, "Description", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wPrice, 9, "Price", "B", 0, "R", true, 0, "")
	pdf.CellFormat(wOrdered, 9, "Ordered", "B", 0, "R", true, 0, "")
	pdf.CellFormat(wInvoiced, 9, "Invoiced", "B", 0, "R", true, 0, "")
	pdf.CellFormat(wUnit, 9, "Unit", "B", 0, "L", true, 0, "")
	pdf.CellFormat(wTotal, 9, "Total", "B", 1, "R", true, 0, "")

	pdf.SetFont("Helvetica", "", 9.5)
	pdf.SetDrawColor(229, 231, 235)
	for _, line := range doc.Lines {
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
		// Ordered and Invoiced are narrow columns holding numbers that can reach seven figures, and
		// Unit holds a spelled-out unit name; all three overflowed before they were fitted.
		cell(wPrice, line.Price, "R")
		cell(wOrdered, line.Ordered, "R")
		cell(wInvoiced, line.Invoiced, "R")
		cell(wUnit, line.Unit, "L")
		cell(wTotal, line.Total, "R")
		pdf.Ln(-1)
	}

	// Total Due footer (right-aligned into the last columns).
	pdf.SetFont("Helvetica", "B", 10.5)
	pdf.CellFormat(wLine+wSKU+wDesc+wPrice+wOrdered, 10, "", "", 0, "R", false, 0, "")
	pdf.CellFormat(wInvoiced+wUnit, 10, "Total Due:", "", 0, "R", false, 0, "")
	pdfCellText{W: wTotal, H: 10, Text: doc.OrderTotal, Ln: 1, Align: "R", Style: "B", Size: 10.5, MinSize: 7}.draw(pdf)
}
