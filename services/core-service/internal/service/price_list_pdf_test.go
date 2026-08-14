package service

import (
	"bytes"
	"testing"
)

func testDocument(rowCount int) priceListDocument {
	rows := make([]priceListRow, 0, rowCount)
	for i := 0; i < rowCount; i++ {
		colour := "Black"
		if i%2 == 1 {
			colour = "Khaki"
		}
		rows = append(rows, priceListRow{
			SKU:         "6801" + string(rune('0'+i%10)),
			Description: "Below Knee, Closed Toe",
			Values:      []string{colour, "Regular"},
			Packing:     "10 Pairs Per Carton",
			Prices:      []string{"$18.00"},
		})
	}

	return priceListDocument{
		MerchantName: "Seller Co",
		CustomerName: "Healthcare and Co",
		DateLong:     "July 27, 2026",
		PaymentTerm:  "Advance",
		ShippingTerm: "F.O.B. Springfield, IL 62701",
		Lines: []priceListLine{{
			ProductLineID:   "pl_1",
			ProductLineName: "Couture",
			BaseUnitName:    "Pair",
			Sections: []priceListSection{{
				Heading: "15-20 mmHg · Dress Socks",
				Columns: []string{"Color", "Length"},
				Tiers:   []priceListTier{{Label: "1+", Quantity: "1"}},
				Rows:    rows,
			}},
		}},
	}
}

func TestBuildPriceListPDF_ProducesAPDF(t *testing.T) {
	body, err := buildPriceListPDF(testDocument(6))
	if err != nil {
		t.Fatalf("buildPriceListPDF: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Errorf("output is not a PDF, starts with %q", string(body[:min(8, len(body))]))
	}
	if len(body) < 1000 {
		t.Errorf("PDF is suspiciously small: %d bytes", len(body))
	}
}

// A table longer than one page must keep rendering rather than overflowing off the bottom, so the row count has to survive a page break.
func TestBuildPriceListPDF_PaginatesLongSections(t *testing.T) {
	short, err := buildPriceListPDF(testDocument(10))
	if err != nil {
		t.Fatalf("short: %v", err)
	}
	long, err := buildPriceListPDF(testDocument(400))
	if err != nil {
		t.Fatalf("long: %v", err)
	}
	if len(long) <= len(short) {
		t.Errorf("400 rows (%d bytes) did not produce more output than 10 rows (%d bytes)", len(long), len(short))
	}
}

func TestBuildPriceListPDF_EmptyCatalogStillRenders(t *testing.T) {
	body, err := buildPriceListPDF(priceListDocument{
		MerchantName: "Seller Co",
		CustomerName: "Healthcare and Co",
		DateLong:     "July 27, 2026",
		Notes:        []string{"No products are assigned to this customer."},
	})
	if err != nil {
		t.Fatalf("buildPriceListPDF: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
}
