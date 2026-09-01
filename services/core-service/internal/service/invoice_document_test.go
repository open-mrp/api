package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// Parity coverage for the invoice a customer receives, against the dashboard's InvoiceEmail and
// InvoicePdf that these replaced.
//
// The invoice is the document with the most moving parts: it bills what shipped against what was
// ordered, so its table carries both quantities; it labels a price with the rate's own pricing unit,
// which need not be the line's stocking unit; and it carries the shipment's cases, which are the one
// column that prints a measure verbatim.

// invoiceFixture is an invoice with every optional section populated.
func invoiceFixture() (*domain.Invoice, []*domain.InvoiceLine, *domain.SalesOrder) {
	// Constructed in the local zone because the renderers format in it, as the dashboard's date-fns
	// does; a UTC fixture would render a day early west of Greenwich.
	created := time.Date(2026, 7, 14, 14, 30, 0, 0, time.Local)

	invoice := &domain.Invoice{
		Number:         "5821",
		OrderID:        "so_1",
		CustomerName:   "Northwind Traders",
		CustomerNumber: "42",
		CreatedAt:      created,
	}
	lines := []*domain.InvoiceLine{
		{
			OrderLineItemNumber:  poPtr(int32(1)),
			OrderLineItemSKU:     poPtr("SOCK-CREW-BLK"),
			OrderLineDescription: poPtr("Crew sock, black, size L"),
			QuantityValue:        "1200",
			QuantityUnitAbbr:     "pr",
			QuantityUnitName:     "pair",
			OrderLineQtyOrdered:  "1500",
			UnitPriceValue:       "8.5",
			// Priced by the dozen while stocked in pairs: the price label must follow the rate.
			UnitPriceDenUnitAbbr: "dz",
		},
		{
			OrderLineItemNumber:  poPtr(int32(2)),
			OrderLineItemSKU:     poPtr("SOCK-ANK-WHT"),
			QuantityValue:        "300",
			QuantityUnitAbbr:     "pr",
			QuantityUnitName:     "pair",
			OrderLineQtyOrdered:  "300",
			UnitPriceValue:       "4.25",
			UnitPriceDenUnitAbbr: "pr",
		},
	}
	order := &domain.SalesOrder{
		Number:            "9001",
		CustomerName:      "Northwind Traders",
		CustomerNumber:    "42",
		CustomerPONumber:  poPtr("PO-77321"),
		PriorityName:      "Standard",
		CarrierName:       poPtr("UPS"),
		PaymentTermName:   poPtr("Net 30"),
		SalesRepName:      poPtr("Dana Reed"),
		BillToName:        poPtr("Northwind Traders"),
		BillToStreetLine1: poPtr("55 Harbour Way"),
		BillToLocality:    poPtr("Seattle"),
		BillToState:       poPtr("WA"),
		BillToPostalCode:  poPtr("98101"),
		ShipToName:        poPtr("Northwind DC"),
		ShipToStreetLine1: poPtr("9 Dock Rd"),
		ShipToLocality:    poPtr("Tacoma"),
		ShipToState:       poPtr("WA"),
		ShipToPostalCode:  poPtr("98402"),
		CreatedAt:         created,
	}
	return invoice, lines, order
}

func TestInvoiceDocMatchesLegacyFields(t *testing.T) {
	t.Parallel()

	invoice, lines, order := invoiceFixture()
	account := &domain.Account{Name: "Carolon Co"}
	cases := []*domain.ShippingCase{
		{Number: "CASE-1", FreightWeightValue: "1200", FreightWeightUnitAbbreviation: "lb", TrackingNumber: poPtr("1Z999")},
	}
	doc := buildInvoiceDoc(invoice, lines, order, account, nil, cases, []string{"ap@northwind.com"})

	t.Run("document identity names the invoice, not the order behind it", func(t *testing.T) {
		if doc.Header.DocumentTitle != "INVOICE" {
			t.Errorf("DocumentTitle = %q", doc.Header.DocumentTitle)
		}
		if doc.Header.NumberLabel != "Invoice Number" {
			t.Errorf("NumberLabel = %q", doc.Header.NumberLabel)
		}
		if doc.Header.OrderNumber != "005821" {
			t.Errorf("OrderNumber = %q, want the invoice's own zero-padded number", doc.Header.OrderNumber)
		}
		// The legacy PDF carries the time on an invoice, which is stamped at the moment of shipping.
		if doc.Header.OrderDateLong != "07/14/2026 02:30 PM" {
			t.Errorf("OrderDateLong = %q", doc.Header.OrderDateLong)
		}
		if doc.Header.OrderDateShort != "7/14/2026" {
			t.Errorf("OrderDateShort = %q, want the email's unpadded form", doc.Header.OrderDateShort)
		}
	})

	t.Run("the price is labelled with the rate's pricing unit", func(t *testing.T) {
		// Legacy reads RateUtils.abbreviate(unitPrice), whose denominator is the rate's own unit.
		if got := doc.Lines[0].Price; got != "$8.50 / dz" {
			t.Errorf("Price = %q, want the rate's denominator, not the line's stocking unit", got)
		}
	})

	t.Run("ordered and invoiced are separate whole-unit counts", func(t *testing.T) {
		if doc.Lines[0].Ordered != "1,500" || doc.Lines[0].Invoiced != "1,200" {
			t.Errorf("Ordered/Invoiced = %q/%q", doc.Lines[0].Ordered, doc.Lines[0].Invoiced)
		}
		// The email shows one quantity, naming the unit in full.
		if doc.Lines[0].InvoicedWithUnit != "1,200 pair" {
			t.Errorf("InvoicedWithUnit = %q", doc.Lines[0].InvoicedWithUnit)
		}
		if doc.Lines[0].Unit != "pair" {
			t.Errorf("Unit = %q, want the unit's full name in the PDF's Unit column", doc.Lines[0].Unit)
		}
	})

	t.Run("totals bill what was invoiced, not what was ordered", func(t *testing.T) {
		// 1200 * 8.50 = 10,200.00 and 300 * 4.25 = 1,275.00.
		if doc.Lines[0].Total != "$10,200.00" || doc.Lines[1].Total != "$1,275.00" {
			t.Errorf("line totals = %q, %q", doc.Lines[0].Total, doc.Lines[1].Total)
		}
		if doc.OrderTotal != "$11,475.00" {
			t.Errorf("OrderTotal = %q", doc.OrderTotal)
		}
	})

	t.Run("case weights print verbatim", func(t *testing.T) {
		if len(doc.Cases) != 1 {
			t.Fatalf("got %d cases", len(doc.Cases))
		}
		if doc.Cases[0].Weight != "1200 lb" {
			t.Errorf("Weight = %q, want the raw measure the legacy table interpolated", doc.Cases[0].Weight)
		}
		if doc.Cases[0].Tracking != "1Z999" {
			t.Errorf("Tracking = %q", doc.Cases[0].Tracking)
		}
	})

	t.Run("the order behind the invoice supplies the terms and addresses", func(t *testing.T) {
		if doc.Header.CustomerPO != "PO-77321" || doc.Header.PaymentTerms != "Net 30" || doc.Header.SalesRep != "Dana Reed" {
			t.Errorf("terms = %+v", doc.Header)
		}
		if doc.Header.BillTo.CityStateZip != "Seattle, WA 98101" || doc.Header.ShipTo.CityStateZip != "Tacoma, WA 98402" {
			t.Errorf("addresses = %q / %q", doc.Header.BillTo.CityStateZip, doc.Header.ShipTo.CityStateZip)
		}
	})
}

func TestInvoiceEmailParamsCoverTheTemplate(t *testing.T) {
	t.Parallel()

	invoice, lines, order := invoiceFixture()
	account := &domain.Account{
		Name: "Carolon Co",
		Branding: &domain.AccountBranding{
			SupportEmail: poPtr("service@carolon.com"),
			WebsiteURL:   poPtr("https://carolon.com"),
		},
	}
	doc := buildInvoiceDoc(invoice, lines, order, account, nil, nil, nil)
	params := doc.emailParams("https://track.example/1Z999")

	for _, key := range []string{
		"account_name", "logo_url", "invoice_number", "invoice_date", "invoice_total",
		"master_tracking_url", "has_bill_to", "bill_to_name", "bill_to_line1", "bill_to_line2",
		"bill_to_csz", "lines", "account_email", "account_website", "year", "customer_number",
		"order_online_link", "email_subject", "instagram_handle", "twitter_handle",
		"facebook_handle", "linkedin_handle", "marketing_blurb",
	} {
		if _, ok := params[key]; !ok {
			t.Errorf("email params missing %q", key)
		}
	}

	if params["invoice_number"] != "005821" || params["invoice_date"] != "7/14/2026" {
		t.Errorf("identity = %v / %v", params["invoice_number"], params["invoice_date"])
	}
	if params["email_subject"] != "Invoice 005821" {
		t.Errorf("email_subject = %v, want the legacy mailto subject", params["email_subject"])
	}
	// The CTA quotes the unpadded number, which is what the customer types to link their account.
	if params["customer_number"] != "42" {
		t.Errorf("customer_number = %v, want the unpadded number", params["customer_number"])
	}

	rows, ok := params["lines"].([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("lines = %#v", params["lines"])
	}
	// The email's three columns: product, quantity, and the line total under a "Price" heading.
	if rows[0]["qty"] != "1,200 pair" || rows[0]["total"] != "$10,200.00" {
		t.Errorf("line row = %#v", rows[0])
	}
}

// Without a tracking number the Track Shipment button has no destination, so the parameter is empty
// and the template drops the button rather than linking nowhere.
func TestInvoiceEmailWithoutTracking(t *testing.T) {
	t.Parallel()

	invoice, lines, order := invoiceFixture()
	params := buildInvoiceDoc(invoice, lines, order, nil, nil, nil, nil).emailParams("")
	if params["master_tracking_url"] != "" {
		t.Errorf("master_tracking_url = %v, want empty", params["master_tracking_url"])
	}
	if got := shipmentMasterTrackingURL(nil); got != "" {
		t.Errorf("shipmentMasterTrackingURL(nil) = %q", got)
	}
}

func TestInvoicePDFRendersLegacyLayout(t *testing.T) {
	t.Parallel()

	invoice, lines, order := invoiceFixture()
	account := &domain.Account{Name: "Carolon Co"}
	cases := []*domain.ShippingCase{
		{Number: "CASE-1", FreightWeightValue: "1200", FreightWeightUnitAbbreviation: "lb", TrackingNumber: poPtr("1Z999")},
	}
	doc := buildInvoiceDoc(invoice, lines, order, account, nil, cases, []string{"ap@northwind.com"})

	pdfBytes, err := buildInvoicePDF(doc)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}
	runs := pdfText(t, pdfBytes)

	t.Run("header identifies the invoice, its PO and its customer", func(t *testing.T) {
		for _, want := range []string{
			"INVOICE", "Invoice Number", "005821",
			"PO Number", "PO-77321",
			"Customer Number", "00042",
			"Date", "07/14/2026 02:30 PM",
			"Carolon Co",
		} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})

	t.Run("addresses and order terms both appear", func(t *testing.T) {
		for _, want := range []string{
			"BILL TO", "SHIP TO", "55 Harbour Way", "9 Dock Rd", "ap@northwind.com",
			"UPS", "Net 30", "Dana Reed",
		} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})

	t.Run("the cases table lists what the shipment travelled in", func(t *testing.T) {
		for _, want := range []string{"Cases", "Case Number", "Weight", "Tracking Number", "CASE-1", "1200 lb", "1Z999"} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})

	t.Run("invoice summary carries both quantity columns", func(t *testing.T) {
		for _, want := range []string{
			"Invoice Summary", "Line Item", "SKU", "Description", "Price", "Ordered", "Invoiced", "Unit", "Total",
			"001", "$8.50 / dz", "1,500", "1,200", "pair", "$10,200.00",
			"Total Due:", "$11,475.00",
		} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})
}

// A shipment with no cases renders no Cases table at all, rather than an empty one — matching the
// legacy component, which returned nothing for an empty shipment.
func TestInvoicePDFOmitsEmptyCasesTable(t *testing.T) {
	t.Parallel()

	invoice, lines, order := invoiceFixture()
	doc := buildInvoiceDoc(invoice, lines, order, nil, nil, nil, nil)

	pdfBytes, err := buildInvoicePDF(doc)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}
	runs := pdfText(t, pdfBytes)
	for _, unwanted := range []string{"Cases", "Case Number", "Tracking Number"} {
		if pdfContains(runs, unwanted) {
			t.Errorf("PDF should not carry %q with no cases\n%s", unwanted, pdfJoined(runs))
		}
	}
	if !pdfContains(runs, "Invoice Summary") {
		t.Error("PDF lost its line table")
	}
}

// An invoice whose order could not be loaded still bills correctly off its own billing address; the
// legacy renderer degraded the same way rather than withholding the document.
func TestInvoiceDocWithoutOrder(t *testing.T) {
	t.Parallel()

	invoice, lines, _ := invoiceFixture()
	invoice.BillingAddressName = poPtr("Northwind Traders")
	invoice.BillingAddressLine1 = poPtr("55 Harbour Way")
	invoice.BillingAddressCity = poPtr("Seattle")
	invoice.BillingAddressState = poPtr("WA")
	invoice.BillingAddressZip = poPtr("98101")

	doc := buildInvoiceDoc(invoice, lines, nil, nil, nil, nil, nil)

	if doc.Header.BillTo.CityStateZip != "Seattle, WA 98101" {
		t.Errorf("BillTo = %q, want the invoice's own billing address", doc.Header.BillTo.CityStateZip)
	}
	if doc.Header.CustomerNumber != "00042" {
		t.Errorf("CustomerNumber = %q", doc.Header.CustomerNumber)
	}
	if doc.OrderTotal != "$11,475.00" {
		t.Errorf("OrderTotal = %q, want the lines to still total", doc.OrderTotal)
	}
	if _, err := buildInvoicePDF(doc); err != nil {
		t.Fatalf("build pdf: %v", err)
	}
}
