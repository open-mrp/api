package service

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// Parity coverage for the purchase order a supplier receives, against the dashboard's
// PurchaseOrderSubmissionEmail and PurchaseOrderPdf that these replaced.
//
// The purchase order is the document with the least in common with the others — a supplier number
// rather than a customer number, a requested delivery date rather than order terms, and unit prices
// at four decimals rather than two — so each of those is asserted rather than assumed to follow from
// the shared renderer.

// poPtr is a local generic pointer helper; the package's existing `ptr` is string-only.
func poPtr[T any](v T) *T { return &v }

// poFixture is a purchase order with every optional field populated, so a test can assert what a
// fully-specified document renders and then null fields out to assert the degraded forms.
func poFixture() (*domain.PurchaseOrder, []*domain.PurchaseOrderLine) {
	// The dates are constructed in the local zone because the renderers format in it, as the
	// dashboard's date-fns does; a UTC fixture would render a day early west of Greenwich.
	order := &domain.PurchaseOrder{
		Number:            "417",
		SupplierName:      "Fastener Supply Co",
		SupplierNumber:    "88",
		CreatedAt:         time.Date(2026, 3, 4, 15, 30, 0, 0, time.Local),
		PromisedAt:        poPtr(time.Date(2026, 4, 18, 0, 0, 0, 0, time.Local)),
		BillToName:        poPtr("Augno Manufacturing"),
		BillToStreetLine1: poPtr("100 Foundry Rd"),
		BillToStreetLine2: poPtr("Dock 4"),
		BillToLocality:    poPtr("Akron"),
		BillToState:       poPtr("OH"),
		BillToPostalCode:  poPtr("44301"),
		ShipToName:        poPtr("Augno Receiving"),
		ShipToStreetLine1: poPtr("240 Mill St"),
		ShipToLocality:    poPtr("Akron"),
		ShipToState:       poPtr("OH"),
		ShipToPostalCode:  poPtr("44302"),
	}
	lines := []*domain.PurchaseOrderLine{
		// Deliberately out of order: the document must sort by line item number.
		{
			LineItemNumber:               2,
			ProductSKU:                   "PRD-2",
			ItemSKU:                      poPtr("BOLT-M6"),
			ProductDescription:           poPtr("Hex bolt M6x40, zinc"),
			QuantityValue:                "5000",
			QuantityUnitName:             "each",
			QuantityUnitAbbreviation:     "ea",
			UnitPriceValue:               "0.0125",
			UnitPriceDenominatorUnitAbbr: "ea",
		},
		{
			LineItemNumber:               1,
			ProductSKU:                   "PRD-1",
			ItemSKU:                      poPtr("WSHR-M6"),
			ProductDescription:           poPtr("Flat washer M6"),
			QuantityValue:                "1200",
			QuantityUnitName:             "pair",
			QuantityUnitAbbreviation:     "pr",
			UnitPriceValue:               "8.5",
			UnitPriceDenominatorUnitAbbr: "pr",
		},
	}
	return order, lines
}

func TestPurchaseOrderDocMatchesLegacyFields(t *testing.T) {
	t.Parallel()

	order, lines := poFixture()
	account := &domain.Account{Name: "Augno Manufacturing"}
	doc := buildPurchaseOrderDoc(order, lines, account, nil, []string{"Buyer@Augno.com"})

	t.Run("document identity names the purchase order", func(t *testing.T) {
		if doc.Header.DocumentTitle != "PURCHASE ORDER" {
			t.Errorf("DocumentTitle = %q", doc.Header.DocumentTitle)
		}
		if doc.Header.NumberLabel != "Purchase Order Number" {
			t.Errorf("NumberLabel = %q", doc.Header.NumberLabel)
		}
		if doc.Header.OrderNumber != "000417" {
			t.Errorf("OrderNumber = %q, want the zero-padded record number", doc.Header.OrderNumber)
		}
	})

	t.Run("identity rows are the supplier set, not the customer set", func(t *testing.T) {
		want := []ackIdentityField{
			{Label: "Supplier Number", Value: "00088"},
			{Label: "Date", Value: "03/04/2026"},
			{Label: "Requested Delivery Date", Value: "04/18/2026"},
		}
		if len(doc.Header.IdentityRows) != len(want) {
			t.Fatalf("IdentityRows = %+v, want %+v", doc.Header.IdentityRows, want)
		}
		for i, row := range want {
			if doc.Header.IdentityRows[i] != row {
				t.Errorf("IdentityRows[%d] = %+v, want %+v", i, doc.Header.IdentityRows[i], row)
			}
		}
	})

	t.Run("lines sort by line item number", func(t *testing.T) {
		if len(doc.Header.Lines) != 2 {
			t.Fatalf("got %d lines", len(doc.Header.Lines))
		}
		if doc.Header.Lines[0].LineItem != "001" || doc.Header.Lines[1].LineItem != "002" {
			t.Errorf("lines out of order: %q then %q", doc.Header.Lines[0].LineItem, doc.Header.Lines[1].LineItem)
		}
		// The legacy document reads item.sku, falling back to the line's own product SKU.
		if doc.Header.Lines[0].SKU != "WSHR-M6" {
			t.Errorf("SKU = %q, want the linked item's SKU", doc.Header.Lines[0].SKU)
		}
	})

	t.Run("unit prices carry four decimals", func(t *testing.T) {
		if got := doc.Header.Lines[1].Price; got != "$0.0125 / ea" {
			t.Errorf("Price = %q, want the sub-cent price preserved at four decimals", got)
		}
		if got := doc.Header.Lines[0].Price; got != "$8.5000 / pr" {
			t.Errorf("Price = %q, want four decimals even when trailing zeros", got)
		}
	})

	t.Run("quantities round to whole units and name the unit in full", func(t *testing.T) {
		if got := doc.Header.Lines[0].Qty; got != "1,200 pair" {
			t.Errorf("Qty = %q, want the unit's full name", got)
		}
	})

	t.Run("line and order totals are money at two decimals", func(t *testing.T) {
		// 1200 * 8.50 = 10,200.00 and 5000 * 0.0125 = 62.50.
		if got := doc.Header.Lines[0].Total; got != "$10,200.00" {
			t.Errorf("line total = %q", got)
		}
		if got := doc.Header.Lines[1].Total; got != "$62.50" {
			t.Errorf("line total = %q", got)
		}
		if doc.Header.OrderTotal != "$10,262.50" {
			t.Errorf("OrderTotal = %q", doc.Header.OrderTotal)
		}
	})

	t.Run("contact emails are lowercased under bill to", func(t *testing.T) {
		if len(doc.Header.ContactEmails) != 1 || doc.Header.ContactEmails[0] != "buyer@augno.com" {
			t.Errorf("ContactEmails = %v, want the address lowercased", doc.Header.ContactEmails)
		}
	})

	t.Run("ship to is present and separate from bill to", func(t *testing.T) {
		if !doc.Header.HasShipTo {
			t.Fatal("HasShipTo = false")
		}
		if doc.Header.ShipTo.Name != "Augno Receiving" || doc.Header.BillTo.Name != "Augno Manufacturing" {
			t.Errorf("addresses crossed: ship=%q bill=%q", doc.Header.ShipTo.Name, doc.Header.BillTo.Name)
		}
		if doc.Header.ShipTo.CityStateZip != "Akron, OH 44302" {
			t.Errorf("ShipTo city/state/zip = %q", doc.Header.ShipTo.CityStateZip)
		}
	})
}

// An order with no promised date drops the row rather than printing an empty one, as the legacy
// template's conditional did.
func TestPurchaseOrderDocWithoutRequestedDeliveryDate(t *testing.T) {
	t.Parallel()

	order, lines := poFixture()
	order.PromisedAt = nil
	doc := buildPurchaseOrderDoc(order, lines, nil, nil, nil)

	if doc.RequestedDeliveryDate != "" {
		t.Errorf("RequestedDeliveryDate = %q, want empty", doc.RequestedDeliveryDate)
	}
	if got := doc.emailParams()["requested_delivery_date"]; got != "" {
		t.Errorf("template param = %q, want empty so the row is dropped", got)
	}
}

func TestPurchaseOrderEmailParamsCoverTheTemplate(t *testing.T) {
	t.Parallel()

	order, lines := poFixture()
	account := &domain.Account{
		Name: "Augno Manufacturing",
		Branding: &domain.AccountBranding{
			SupportEmail:    poPtr("orders@augno.com"),
			WebsiteURL:      poPtr("https://augno.com"),
			InstagramHandle: poPtr("augno"),
		},
	}
	params := buildPurchaseOrderDoc(order, lines, account, nil, nil).emailParams()

	// Every key the template dereferences. A rename on either side silently blanks a section, which
	// is exactly the failure this guards.
	for _, key := range []string{
		"account_name", "logo_url", "order_number", "requested_delivery_date", "submitted_on",
		"order_total", "has_ship_to", "ship_to_name", "ship_to_line1", "ship_to_line2",
		"ship_to_csz", "lines", "account_email", "account_website", "year", "email_subject",
		"instagram_handle", "twitter_handle", "facebook_handle", "linkedin_handle", "marketing_blurb",
	} {
		if _, ok := params[key]; !ok {
			t.Errorf("email params missing %q", key)
		}
	}

	if params["submitted_on"] != "03/04/2026" {
		t.Errorf("submitted_on = %v", params["submitted_on"])
	}
	if params["email_subject"] != "Purchase Order 000417 Submission" {
		t.Errorf("email_subject = %v, want the legacy mailto subject", params["email_subject"])
	}

	rows, ok := params["lines"].([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("lines = %#v", params["lines"])
	}
	// The email's four columns: product, quantity, unit price, total.
	for _, key := range []string{"sku", "description", "qty", "unit_price", "total"} {
		if _, ok := rows[0][key]; !ok {
			t.Errorf("line row missing %q", key)
		}
	}
	if rows[0]["qty"] != "1,200 pair" {
		t.Errorf("email qty = %v, want the unit's full name", rows[0]["qty"])
	}
	if rows[0]["unit_price"] != "$8.5000 / pr" {
		t.Errorf("unit_price = %v, want four decimals in the email too", rows[0]["unit_price"])
	}
}

func TestPurchaseOrderPDFRendersLegacyLayout(t *testing.T) {
	t.Parallel()

	order, lines := poFixture()
	account := &domain.Account{Name: "Augno Manufacturing"}
	doc := buildPurchaseOrderDoc(order, lines, account, nil, []string{"buyer@augno.com"})

	pdfBytes, err := buildPurchaseOrderPDF(doc.Header)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}
	runs := pdfText(t, pdfBytes)

	t.Run("header identifies the purchase order and its supplier", func(t *testing.T) {
		for _, want := range []string{
			"PURCHASE ORDER", "Purchase Order Number", "000417",
			"Supplier Number", "00088",
			"Requested Delivery Date", "04/18/2026",
			"Augno Manufacturing",
		} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})

	t.Run("addresses appear under both headings", func(t *testing.T) {
		for _, want := range []string{"BILL TO", "SHIP TO", "100 Foundry Rd", "240 Mill St", "buyer@augno.com"} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})

	t.Run("order summary carries the legacy columns and four-decimal prices", func(t *testing.T) {
		for _, want := range []string{
			"Order Summary", "Line Item", "SKU", "Description", "Price", "Qty", "Total",
			"001", "WSHR-M6", "$8.5000 / pr", "1,200 pair", "$10,200.00",
			"002", "BOLT-M6", "$0.0125 / ea", "5,000 each", "$62.50",
			"Total Due:", "$10,262.50",
		} {
			if !pdfContains(runs, want) {
				t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
			}
		}
	})

	t.Run("no order terms band, which a purchase order has no use for", func(t *testing.T) {
		for _, unwanted := range []string{"Carrier", "Payment Terms", "Sales Rep", "Priority"} {
			if pdfContains(runs, unwanted) {
				t.Errorf("PDF should not carry %q", unwanted)
			}
		}
	})
}

// A bare order must still produce a document rather than panicking: the supplier gets a thin
// purchase order, which beats the send failing.
func TestPurchaseOrderPDFDegradesWithoutOptionalData(t *testing.T) {
	t.Parallel()

	order := &domain.PurchaseOrder{Number: "1", CreatedAt: time.Now().UTC()}
	doc := buildPurchaseOrderDoc(order, nil, nil, nil, nil)

	pdfBytes, err := buildPurchaseOrderPDF(doc.Header)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}
	runs := pdfText(t, pdfBytes)
	if !pdfContains(runs, "PURCHASE ORDER") {
		t.Errorf("PDF lost its title\n%s", pdfJoined(runs))
	}
	if doc.Header.OrderTotal != "$0.00" {
		t.Errorf("OrderTotal = %q, want a zero total rather than a blank", doc.Header.OrderTotal)
	}
}

func TestPurchaseOrderAttachmentFilename(t *testing.T) {
	t.Parallel()

	if got := purchaseOrderAttachmentFilename("000417"); got != "purchase-order-000417.pdf" {
		t.Errorf("filename = %q", got)
	}
	// A number carrying separators must not open a path or break the MIME header.
	if got := purchaseOrderAttachmentFilename("PO/417 draft"); strings.ContainsAny(got, "/ ") {
		t.Errorf("filename = %q, want path- and space-free", got)
	}
}
