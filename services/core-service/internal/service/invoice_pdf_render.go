package service

import (
	"bytes"

	"github.com/go-pdf/fpdf"
)

// Renders the invoice PDF customers receive on ship, mirroring the dashboard's InvoicePdf: the
// shared letterhead, addresses and terms, then the shipment's cases and the invoice summary.
func buildInvoicePDF(doc invoiceDoc) ([]byte, error) {
	pdf := newRecordPDF()

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
// cases, rather than drawing a header over nothing.
func invoiceCaseTable(pdf *fpdf.Fpdf, doc invoiceDoc) {
	if len(doc.Cases) == 0 {
		return
	}
	rows := make([][]string, len(doc.Cases))
	for i, c := range doc.Cases {
		rows[i] = []string{c.Number, c.Weight, c.Tracking}
	}
	pdfTable{
		Title:   "Cases",
		Columns: []pdfColumn{{Title: "Case Number", Wrap: true}, {Title: "Weight", Wrap: true}, {Title: "Tracking Number", Wrap: true}},
		Rows:    rows,
	}.draw(pdf)
	pdf.SetY(pdf.GetY() + docSectionGap)
}

// Renders the invoice line table and the Total Due footer. Ordered and Invoiced sit side by side so
// the customer can see what was billed against what they asked for.
func invoiceSummary(pdf *fpdf.Fpdf, doc invoiceDoc) {
	rows := make([][]string, len(doc.Lines))
	for i, l := range doc.Lines {
		rows[i] = []string{l.LineItem, l.SKU, l.Description, l.Price, l.Ordered, l.Invoiced, l.Unit, l.Total}
	}
	pdfTable{
		Title: "Invoice Summary",
		Columns: []pdfColumn{
			{Title: "Line Item"},
			{Title: "SKU", Wrap: true},
			// max-w-[2in], which is what wraps a long description instead of widening the table.
			{Title: "Description", Wrap: true, MaxWidth: docDescriptionMaxW},
			{Title: "Price"},
			{Title: "Ordered"},
			{Title: "Invoiced"},
			{Title: "Unit"},
			{Title: "Total"},
		},
		Rows:   rows,
		Footer: []string{"Total Due:", doc.OrderTotal},
	}.draw(pdf)
}
