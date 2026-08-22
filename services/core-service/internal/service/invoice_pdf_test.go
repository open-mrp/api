package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invoiceDocFixture() ([]*domain.InvoiceLine, *domain.Invoice, *domain.SalesOrder) {
	sku1, desc1 := "SKU-1", "Widget, 6061-T6"
	sku2 := "SKU-2"
	num1, num2 := int32(1), int32(2)

	lines := []*domain.InvoiceLine{
		{
			QuantityValue: "6", QuantityUnitAbbr: "pr", QuantityUnitName: "pair",
			UnitPriceValue: "8.50", OrderLineQtyOrdered: "10",
			OrderLineItemNumber: &num1, OrderLineItemSKU: &sku1, OrderLineDescription: &desc1,
		},
		{
			QuantityValue: "1200", QuantityUnitAbbr: "ea", QuantityUnitName: "each",
			UnitPriceValue: "0.25", OrderLineQtyOrdered: "1200",
			OrderLineItemNumber: &num2, OrderLineItemSKU: &sku2,
		},
	}

	shipmentID := "sh_1"
	invoice := &domain.Invoice{
		Number:         "INV-9",
		CustomerNumber: "C-INVOICE",
		OrderID:        "or_1",
		ShipmentID:     &shipmentID,
		CreatedAt:      time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC),
	}

	po, carrier, term, rep := "PO-4242", "UPS", "Net 30", "Dana Rep"
	billName, billLine1, billCity, billState, billZip := "Acme Bill-To", "1 Main St", "Springfield", "IL", "62701"
	shipName := "Acme Dock 4"
	order := &domain.SalesOrder{
		Number:            "ORD-1",
		CustomerName:      "Acme Corp",
		CustomerNumber:    "C-ORDER",
		CustomerPONumber:  &po,
		PriorityName:      "Normal",
		CarrierName:       &carrier,
		PaymentTermName:   &term,
		SalesRepName:      &rep,
		BillToName:        &billName,
		BillToStreetLine1: &billLine1,
		BillToLocality:    &billCity,
		BillToState:       &billState,
		BillToPostalCode:  &billZip,
		ShipToName:        &shipName,
		CreatedAt:         time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	return lines, invoice, order
}

// Pins the invoice document's identity block. The customer number is the customer's account number,
// not the order number — legacy labels this row "Customer Number" and fills it from the customer.
func TestBuildInvoiceDoc_IdentityComesFromTheInvoiceAndCustomer(t *testing.T) {
	t.Parallel()

	lines, invoice, order := invoiceDocFixture()
	doc := buildInvoiceDoc(invoice, lines, order, &domain.Account{Name: "Seller Co"}, nil, nil, nil)

	assert.Equal(t, "INVOICE", doc.Header.DocumentTitle)
	assert.Equal(t, "Invoice Number", doc.Header.NumberLabel)
	assert.Equal(t, "INV-9", doc.Header.OrderNumber, "the number shown is the invoice's, not the order's")
	assert.Equal(t, "C-ORDER", doc.Header.CustomerNumber, "customer number comes from the order's customer")
	assert.NotEqual(t, "ORD-1", doc.Header.CustomerNumber, "the order number must never stand in for it")

	// The date carries a time, and renders in the server's zone as the dashboard's does — so the
	// expectation is derived rather than hard-coded, or it would only hold in UTC.
	assert.Equal(t, invoice.CreatedAt.Local().Format("01/02/2006 03:04 PM"), doc.Header.OrderDateLong)
	assert.Regexp(t, `^\d{2}/\d{2}/\d{4} \d{2}:\d{2} [AP]M$`, doc.Header.OrderDateLong)

	// The order's terms and both addresses ride along.
	assert.Equal(t, "PO-4242", doc.Header.CustomerPO)
	assert.Equal(t, "UPS", doc.Header.Carrier)
	assert.Equal(t, "Net 30", doc.Header.PaymentTerms)
	assert.Equal(t, "Dana Rep", doc.Header.SalesRep)
	assert.Equal(t, "Acme Bill-To", doc.Header.BillTo.Name)
	assert.True(t, doc.Header.HasShipTo, "the ship-to block is part of the invoice")
}

// Ordered and Invoiced are distinct columns: a partial shipment bills less than was ordered.
func TestBuildInvoiceDoc_LinesSplitOrderedFromInvoiced(t *testing.T) {
	t.Parallel()

	lines, invoice, order := invoiceDocFixture()
	doc := buildInvoiceDoc(invoice, lines, order, nil, nil, nil, nil)

	require.Len(t, doc.Lines, 2)

	first := doc.Lines[0]
	assert.Equal(t, "001", first.LineItem, "the line item number comes from the order line")
	assert.Equal(t, "SKU-1", first.SKU)
	assert.Equal(t, "$8.50 / pr", first.Price)
	assert.Equal(t, "10", first.Ordered)
	assert.Equal(t, "6", first.Invoiced)
	assert.Equal(t, "pair", first.Unit, "the Unit column carries the full unit name")
	assert.Equal(t, "$51.00", first.Total, "the total bills what was invoiced, not what was ordered")

	// Thousands separators, matching legacy's numeral '0,0' formatting.
	assert.Equal(t, "1,200", doc.Lines[1].Ordered)
	assert.Equal(t, "1,200", doc.Lines[1].Invoiced)

	assert.Equal(t, "$351.00", doc.OrderTotal, "51.00 + 300.00")
}

func TestBuildInvoiceDoc_CasesTable(t *testing.T) {
	t.Parallel()

	lines, invoice, order := invoiceDocFixture()
	tracking := "1Z-CASE-1"
	cases := []*domain.ShippingCase{
		{Number: "INV-9-1", TrackingNumber: &tracking, FreightWeightValue: "12.5", FreightWeightUnitAbbreviation: "lb"},
		{Number: "INV-9-2"},
	}

	doc := buildInvoiceDoc(invoice, lines, order, nil, nil, cases, nil)

	require.Len(t, doc.Cases, 2)
	assert.Equal(t, "INV-9-1", doc.Cases[0].Number)
	assert.Equal(t, "12.5 lb", doc.Cases[0].Weight)
	assert.Equal(t, "1Z-CASE-1", doc.Cases[0].Tracking)
	// A case with no weight or tracking still lists, so the customer sees every case.
	assert.Equal(t, "INV-9-2", doc.Cases[1].Number)
	assert.Empty(t, doc.Cases[1].Tracking)
}

func TestInvoiceDoc_EmailParams(t *testing.T) {
	t.Parallel()

	lines, invoice, order := invoiceDocFixture()
	doc := buildInvoiceDoc(invoice, lines, order, &domain.Account{Name: "Seller Co"}, nil, nil, nil)

	params := doc.emailParams("https://track.example/1Z")

	assert.Equal(t, "INV-9", params["invoice_number"])
	assert.Equal(t, "$351.00", params["invoice_total"])
	assert.Equal(t, "https://track.example/1Z", params["master_tracking_url"])
	assert.Equal(t, true, params["has_bill_to"])
	assert.Equal(t, "Acme Bill-To", params["bill_to_name"])
	assert.Equal(t, "2026", params["year"])

	emailLines, ok := params["lines"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, emailLines, 2)
	// The email's quantity cell carries its unit; the PDF splits the unit into its own column.
	assert.Equal(t, "6 pr", emailLines[0]["qty"])
	assert.Equal(t, "6", doc.Lines[0].Invoiced, "the PDF column stays bare")
	assert.Equal(t, "$51.00", emailLines[0]["total"])

	// The portal CTA quotes the number the customer types, unpadded.
	assert.Equal(t, "C-ORDER", params["customer_number"])
}

// The customer number pads to five as an account number, not six as a record number.
func TestBuildInvoiceDoc_CustomerNumberUsesAccountPadding(t *testing.T) {
	t.Parallel()

	lines, invoice, order := invoiceDocFixture()
	order.CustomerNumber = "8841"
	doc := buildInvoiceDoc(invoice, lines, order, nil, nil, nil, nil)

	assert.Equal(t, "08841", doc.Header.CustomerNumber)
	assert.Equal(t, "8841", doc.Header.CustomerNumberRaw)
}

// Drops the Track Shipment button rather than linking nowhere when tracking or carrier is missing.
func TestShipmentMasterTrackingURL_MissingPartsYieldNoLink(t *testing.T) {
	t.Parallel()

	tracking, code := "1Z999", "ups"
	assert.NotEmpty(t, shipmentMasterTrackingURL(&domain.Shipment{MasterTrackingNumber: &tracking, CarrierCode: &code}))
	assert.Empty(t, shipmentMasterTrackingURL(&domain.Shipment{CarrierCode: &code}))
	assert.Empty(t, shipmentMasterTrackingURL(&domain.Shipment{MasterTrackingNumber: &tracking}))
	assert.Empty(t, shipmentMasterTrackingURL(nil))
}

func TestBuildInvoicePDF_RendersAndDegrades(t *testing.T) {
	t.Parallel()

	lines, invoice, order := invoiceDocFixture()
	pdfBytes, err := buildInvoicePDF(buildInvoiceDoc(invoice, lines, order, &domain.Account{Name: "Seller Co"}, nil, nil, nil))
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF")))

	// A bare invoice with no order, lines or cases must still produce a document.
	bare, err := buildInvoicePDF(buildInvoiceDoc(&domain.Invoice{Number: "1", CreatedAt: time.Now().UTC()}, nil, nil, nil, nil, nil, nil))
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(bare, []byte("%PDF")))
}
