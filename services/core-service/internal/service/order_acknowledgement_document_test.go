package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// Parity coverage for the order acknowledgement, against the dashboard's OrderConfirmationEmail and
// OrderAcknowledgementPdf.
//
// The acknowledgement is where the quantity bug was found: the legacy PDF interpolated the unit's
// full name while its email abbreviated, and the port rendered both from one unrounded string. Both
// documents now name the unit in full, so the mail and the PDF a customer files against their PO
// read the same.

func ackFixture() (*domain.SalesOrder, []*domain.SalesOrderLine) {
	// Constructed in the local zone because the renderers format in it, as date-fns does.
	created := time.Date(2026, 5, 10, 9, 5, 0, 0, time.Local)

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
	lines := []*domain.SalesOrderLine{
		{
			LineItemNumber:               1,
			ProductSKU:                   "SOCK-CREW-BLK",
			ProductDescription:           poPtr("Crew sock, black, size L"),
			QuantityValue:                "1199.5",
			QuantityUnitName:             "pair",
			QuantityUnitAbbreviation:     "pr",
			UnitPriceValue:               "8.5",
			UnitPriceDenominatorUnitAbbr: "dz",
		},
		{
			LineItemNumber:               2,
			ProductSKU:                   "SOCK-ANK-WHT",
			QuantityValue:                "300",
			QuantityUnitName:             "pair",
			QuantityUnitAbbreviation:     "pr",
			UnitPriceValue:               "4.25",
			UnitPriceDenominatorUnitAbbr: "pr",
		},
	}
	return order, lines
}

func TestOrderAcknowledgementQuantityNamesTheUnit(t *testing.T) {
	t.Parallel()

	order, lines := ackFixture()
	data := buildOrderAcknowledgementData(order, lines, &domain.Account{Name: "Carolon Co"}, nil)

	first := data.Lines[0]

	t.Run("the unit is spelled out on both the PDF and the email", func(t *testing.T) {
		// A deliberate divergence from legacy, which abbreviated in the email and spelled the unit
		// out in the PDF: a customer comparing the mail against its attachment had to match "pr" to
		// "pair". Both now read the same.
		if first.Qty != "1,200 pair" {
			t.Errorf("Qty = %q, want the rounded measure with the unit's full name", first.Qty)
		}
		if got := data.emailParams()["lines"].([]map[string]any)[0]["qty"]; got != "1,200 pair" {
			t.Errorf("email param qty = %v, want the unit's full name", got)
		}
	})

	t.Run("the measure rounds rather than exposing the stored fraction", func(t *testing.T) {
		// 1199.5 pairs is a stored measure, not something a customer should read on their order.
		if first.Qty == "1,199.5 pair" {
			t.Errorf("quantity %q leaked the unrounded measure", first.Qty)
		}
	})
}

func TestOrderAcknowledgementDocMatchesLegacyFields(t *testing.T) {
	t.Parallel()

	order, lines := ackFixture()
	data := buildOrderAcknowledgementData(order, lines, &domain.Account{Name: "Carolon Co"}, nil)

	t.Run("identity and terms come from the order", func(t *testing.T) {
		if data.OrderNumber != "009001" {
			t.Errorf("OrderNumber = %q", data.OrderNumber)
		}
		if data.CustomerPO != "PO-77321" || data.CustomerNumber != "00042" {
			t.Errorf("customer identity = %q / %q", data.CustomerPO, data.CustomerNumber)
		}
		if data.OrderDateLong != "05/10/2026 09:05 AM" {
			t.Errorf("OrderDateLong = %q", data.OrderDateLong)
		}
		if data.Carrier != "UPS" || data.PaymentTerms != "Net 30" || data.SalesRep != "Dana Reed" {
			t.Errorf("terms = %q / %q / %q", data.Carrier, data.PaymentTerms, data.SalesRep)
		}
	})

	t.Run("the price is labeled with the rate's pricing unit", func(t *testing.T) {
		if data.Lines[0].Price != "$8.50 / dz" {
			t.Errorf("Price = %q, want the rate's denominator at two decimals", data.Lines[0].Price)
		}
	})

	t.Run("totals bill the rounded-down stored measure, not the display value", func(t *testing.T) {
		// The display rounds to 1,200 but the money must follow the stored 1199.5 * 8.50.
		if data.Lines[0].Total != "$10,195.75" {
			t.Errorf("line total = %q, want the exact measure priced", data.Lines[0].Total)
		}
		if data.OrderTotal != "$11,470.75" {
			t.Errorf("OrderTotal = %q", data.OrderTotal)
		}
	})
}

func TestOrderAcknowledgementPDFRendersLegacyLayout(t *testing.T) {
	t.Parallel()

	order, lines := ackFixture()
	data := buildOrderAcknowledgementData(order, lines, &domain.Account{Name: "Carolon Co"}, nil)
	data.ContactEmails = []string{"ap@northwind.com"}

	pdfBytes, err := buildOrderAcknowledgementPDF(data)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}
	runs := pdfText(t, pdfBytes)

	for _, want := range []string{
		"ORDER ACKNOWLEDGEMENT", "Sales Order Number", "009001",
		"PO Number", "PO-77321", "Customer Number", "00042",
		"BILL TO", "SHIP TO", "55 Harbour Way", "9 Dock Rd", "ap@northwind.com",
		"UPS", "Net 30", "Dana Reed",
		"Order Summary", "Line Item", "SKU", "Description", "Price", "Qty", "Total",
		"001", "SOCK-CREW-BLK", "$8.50 / dz", "1,200 pair", "$10,195.75",
		"Total Due:", "$11,470.75",
	} {
		if !pdfContains(runs, want) {
			t.Errorf("PDF missing %q\n%s", want, pdfJoined(runs))
		}
	}

	// Nothing abbreviates the unit any more.
	if pdfContains(runs, "1,200 pr") {
		t.Error("PDF abbreviated the unit instead of naming it")
	}
}
