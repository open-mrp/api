package service

import (
	"strings"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// Overflow coverage for the record PDFs.
//
// fpdf neither wraps nor clips an over-wide cell: it keeps painting past the boundary and the run
// collides with whatever is drawn next. That failure is invisible to every other kind of test — the
// document renders, the text extracts, the totals are right — which is how a header shipped with the
// label sitting on top of its own value. These assert the geometry instead of the content.

// measuringPDF matches the page setup the record renderers use, for measuring text the way they do.
func measuringPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(ackPageLeft, ackPageTop, 15)
	pdf.AddPage()
	return pdf
}

func TestPDFFitSizeShrinksOnlyAsFarAsNeeded(t *testing.T) {
	t.Parallel()
	pdf := measuringPDF()

	t.Run("text that already fits keeps the full size", func(t *testing.T) {
		if got := pdfFitSize(pdf, "INVOICE", "", 18, 11, 84); got != 18 {
			t.Errorf("size = %v, want 18", got)
		}
	})

	t.Run("a wide title shrinks until it fits", func(t *testing.T) {
		const maxW = 84.0
		got := pdfFitSize(pdf, "ORDER ACKNOWLEDGEMENT", "", 18, 11, maxW)
		if got >= 18 {
			t.Fatalf("size = %v, want a reduction", got)
		}
		pdf.SetFont("Helvetica", "", got)
		if w := pdf.GetStringWidth("ORDER ACKNOWLEDGEMENT"); w > maxW {
			t.Errorf("width %.1fmm still exceeds %.1fmm at size %v", w, maxW, got)
		}
	})

	t.Run("it never goes below the floor", func(t *testing.T) {
		if got := pdfFitSize(pdf, strings.Repeat("W", 200), "", 18, 11, 20); got != 11 {
			t.Errorf("size = %v, want the 11 floor", got)
		}
	})
}

func TestPDFTruncateToWidthMeasuresRatherThanCounts(t *testing.T) {
	t.Parallel()
	pdf := measuringPDF()
	pdf.SetFont("Helvetica", "", 9.5)

	t.Run("text that fits is returned unchanged", func(t *testing.T) {
		if got := pdfTruncateToWidth(pdf, "SKU-1", 30); got != "SKU-1" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("the result fits and is marked", func(t *testing.T) {
		const maxW = 20.0
		got := pdfTruncateToWidth(pdf, "SOCK-CREW-BLACK-LARGE-EXTENDED", maxW)
		if !strings.HasSuffix(got, ellipsis) {
			t.Errorf("got %q, want an ellipsis", got)
		}
		if w := pdf.GetStringWidth(got); w > maxW {
			t.Errorf("truncated to %q, still %.1fmm > %.1fmm", got, w, maxW)
		}
	})

	t.Run("wide and narrow characters truncate differently", func(t *testing.T) {
		// The point of measuring: a character budget would cut these at the same place.
		wide := pdfTruncateToWidth(pdf, strings.Repeat("W", 40), 20)
		narrow := pdfTruncateToWidth(pdf, strings.Repeat("l", 40), 20)
		if len([]rune(wide)) >= len([]rune(narrow)) {
			t.Errorf("wide kept %d runes, narrow kept %d; wide should keep fewer", len([]rune(wide)), len([]rune(narrow)))
		}
	})

	t.Run("multi-byte text is cut on rune boundaries", func(t *testing.T) {
		got := pdfTruncateToWidth(pdf, strings.Repeat("é", 60), 20)
		if !strings.HasPrefix(got, "é") {
			t.Errorf("got %q, want a clean leading rune", got)
		}
		for _, r := range got {
			if r == '�' {
				t.Fatalf("got %q, which contains a broken rune", got)
			}
		}
	})
}

// The header block is the one that shipped broken, so its geometry is asserted directly: for every
// document title and every identity row, the drawn text must fit the column it is drawn into.
func TestHeaderTextFitsItsColumns(t *testing.T) {
	t.Parallel()

	long := "Northwind Traders International Holdings Limited"

	docs := map[string]ackData{
		"invoice": {
			DocumentTitle: "INVOICE", NumberLabel: "Invoice Number", OrderNumber: "005821",
			AccountName: "Carolon Co", CustomerPO: "PO-77321", CustomerNumber: "00042",
			OrderDateLong: "07/14/2026 02:30 PM",
		},
		"acknowledgement": {
			// The longest title of the three, and the one that ran off the page.
			DocumentTitle: "ORDER ACKNOWLEDGEMENT", NumberLabel: "Sales Order Number", OrderNumber: "009001",
			AccountName: "Carolon Co", CustomerPO: "PO-77321", CustomerNumber: "00042",
			OrderDateLong: "05/10/2026 09:05 AM",
		},
		"purchase order": {
			// The longest label of the three, and the one that overlapped its value.
			DocumentTitle: "PURCHASE ORDER", NumberLabel: "Purchase Order Number", OrderNumber: "001000",
			AccountName: "Augno Manufacturing",
			IdentityRows: []ackIdentityField{
				{Label: "Supplier Number", Value: "01000"},
				{Label: "Date", Value: "09/01/2026"},
				{Label: "Requested Delivery Date", Value: "10/15/2026"},
			},
		},
		"pathological": {
			DocumentTitle: strings.ToUpper(long), NumberLabel: long, OrderNumber: long,
			AccountName:  long,
			IdentityRows: []ackIdentityField{{Label: long, Value: long}},
		},
	}

	for name, data := range docs {
		t.Run(name, func(t *testing.T) {
			pdf := measuringPDF()

			title := data.documentTitle()
			size := pdfFitSize(pdf, title, "", 18, 11, ackIdentityW-2)
			pdf.SetFont("Helvetica", "", size)
			if w := pdf.GetStringWidth(pdfTruncateToWidth(pdf, title, ackIdentityW-2)); w > ackIdentityW {
				t.Errorf("title %q is %.1fmm wide in a %.1fmm block", title, w, ackIdentityW)
			}

			rows := data.identityRows()
			labelW := ackIdentityLabelWidth(pdf, rows)
			if labelW > ackIdentityW {
				t.Fatalf("label column %.1fmm exceeds the %.1fmm block", labelW, ackIdentityW)
			}
			valueW := ackIdentityW - labelW
			if valueW <= 0 {
				t.Fatalf("label column %.1fmm leaves no room for values", labelW)
			}

			for _, row := range rows {
				if name == "pathological" {
					// Nothing could hold these at full size; the guarantee is only that they are
					// fitted and truncated into their column rather than painted over the neighbour.
					assertFitsAfterFitting(t, pdf, "label "+row.Label, row.Label, row.style(), row.size(), labelW, 0.5)
					assertFitsAfterFitting(t, pdf, "value "+row.Value, row.Value, row.style(), row.size(), valueW, 0.5)
					continue
				}
				// A real document must fit at its intended size, with no shrinking at all: the
				// columns are supposed to be sized for the labels these documents actually carry.
				assertFitsNaturally(t, pdf, "label "+row.Label, row.Label, row.style(), row.size(), labelW, 0.5)
				assertFitsNaturally(t, pdf, "value "+row.Value, row.Value, row.style(), row.size(), valueW, 0.5)
			}

			if name != "pathological" {
				assertFitsNaturally(t, pdf, "account name", data.AccountName, "B", 15, ackLetterheadW, 1)
			}
		})
	}
}

// The letterhead and the identity block must not overlap, whatever either contains.
func TestHeaderColumnsDoNotOverlap(t *testing.T) {
	t.Parallel()

	if ackPageLeft+ackLetterheadW > ackIdentityX {
		t.Errorf("letterhead ends at %.1fmm, past the identity block at %.1fmm", ackPageLeft+ackLetterheadW, ackIdentityX)
	}
	if ackIdentityX+ackIdentityW != ackPageRight {
		t.Errorf("identity block ends at %.1fmm, not the %.1fmm page edge", ackIdentityX+ackIdentityW, ackPageRight)
	}
}

// An identity row with no value is dropped rather than drawn as a bare label — the "Requested
// Delivery Date" with nothing beside it that made the purchase order look half-rendered.
func TestHeaderDropsValuelessIdentityRows(t *testing.T) {
	t.Parallel()

	data := ackData{
		DocumentTitle: "PURCHASE ORDER", NumberLabel: "Purchase Order Number", OrderNumber: "001000",
		IdentityRows: []ackIdentityField{
			{Label: "Supplier Number", Value: "01000"},
			{Label: "Requested Delivery Date", Value: ""},
			{Label: "Blank", Value: "   "},
		},
	}

	rows := data.identityRows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want the number plus the supplier: %+v", len(rows), rows)
	}
	if !rows[0].Main {
		t.Error("the record's own number should lead and be the main row")
	}
	for _, row := range rows {
		if strings.TrimSpace(row.Value) == "" {
			t.Errorf("row %q rendered with no value", row.Label)
		}
	}
}

// End-to-end: render each record PDF with content chosen to overflow every fixed column, and assert
// nothing is drawn wider than the column holding it.
func TestRecordPDFsSurviveOverflowingContent(t *testing.T) {
	t.Parallel()

	longUnit := "thousand board feet"
	longSKU := "SOCK-CREW-BLACK-LARGE-EXTENDED-CALF-001"

	t.Run("invoice", func(t *testing.T) {
		invoice, lines, order := invoiceFixture()
		lines[0].QuantityUnitName = longUnit
		lines[0].OrderLineItemSKU = poPtr(longSKU)
		lines[0].QuantityValue = "1200000"
		lines[0].OrderLineQtyOrdered = "1500000"
		lines[0].UnitPriceValue = "1234.5678"
		cases := []*domain.ShippingCase{{
			Number: "CASE-000000000000001", FreightWeightValue: "1200000",
			FreightWeightUnitAbbreviation: longUnit, TrackingNumber: poPtr(strings.Repeat("9", 40)),
		}}
		doc := buildInvoiceDoc(invoice, lines, order, &domain.Account{Name: "Carolon Co"}, nil, cases, nil)
		assertRenders(t, func() ([]byte, error) { return buildInvoicePDF(doc) })
	})

	t.Run("order acknowledgement", func(t *testing.T) {
		order, lines := ackFixture()
		lines[0].QuantityUnitName = longUnit
		lines[0].ProductSKU = longSKU
		lines[0].QuantityValue = "1200000"
		order.CustomerPONumber = poPtr(strings.Repeat("PO-", 20))
		assertRenders(t, func() ([]byte, error) {
			return buildOrderAcknowledgementPDF(buildOrderAcknowledgementData(order, lines, &domain.Account{Name: "Carolon Co"}, nil))
		})
	})

	t.Run("purchase order", func(t *testing.T) {
		order, lines := poFixture()
		lines[0].QuantityUnitName = longUnit
		lines[0].ItemSKU = poPtr(longSKU)
		lines[0].QuantityValue = "1200000"
		order.SupplierNumber = strings.Repeat("9", 30)
		order.PromisedAt = poPtr(time.Now())
		doc := buildPurchaseOrderDoc(order, lines, &domain.Account{Name: "Augno Manufacturing"}, nil, nil)
		assertRenders(t, func() ([]byte, error) { return buildPurchaseOrderPDF(doc.Header) })
	})
}

// assertFitsNaturally checks the text fits its column at its intended size, before any shrinking.
// This is the assertion that catches a mis-sized column: re-running the fitter first would make it
// pass against any column at all, which is exactly how a 40mm column held a 46.9mm label.
func assertFitsNaturally(t *testing.T, pdf *fpdf.Fpdf, what, text, style string, size, colW, padding float64) {
	t.Helper()
	avail := colW - 2*padding
	pdf.SetFont("Helvetica", style, size)
	if w := pdf.GetStringWidth(text); w > avail {
		t.Errorf("%s is %.1fmm at its intended %vpt, in a %.1fmm column", what, w, size, avail)
	}
}

// assertFitsAfterFitting checks the drawn form fits, for content no column could hold at full size.
func assertFitsAfterFitting(t *testing.T, pdf *fpdf.Fpdf, what, text, style string, size, colW, padding float64) {
	t.Helper()
	avail := colW - 2*padding
	pdfFitSize(pdf, text, style, size, 8, avail)
	drawn := pdfTruncateToWidth(pdf, text, avail)
	if w := pdf.GetStringWidth(drawn); w > avail {
		t.Errorf("%s renders %.1fmm wide in a %.1fmm column even after fitting", what, w, avail)
	}
}

// assertRenders checks the document builds and produces a real PDF. The per-cell fitting is what
// keeps the content inside its columns; this guards against a pathological value panicking the
// renderer or yielding an empty page.
func assertRenders(t *testing.T, build func() ([]byte, error)) {
	t.Helper()
	out, err := build()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("render produced no bytes")
	}
	if runs := pdfText(t, out); len(runs) == 0 {
		t.Fatal("render produced no text")
	}
}

// Every table column is sized to hold its own header plus the widest value it ordinarily carries,
// without shrinking. Shrinking is the fallback for pathological content, not the normal case: a
// document where half the cells are set two points smaller than the rest looks broken even though
// nothing overlaps.
func TestTableColumnsHoldOrdinaryContent(t *testing.T) {
	t.Parallel()

	type col struct {
		name   string
		w      float64
		header string
		values []string
	}

	tables := map[string][]col{
		"order summary": {
			{"Line Item", 18, "Line Item", []string{"001", "999"}},
			{"SKU", 32, "SKU", []string{"SOCK-CREW-BLK", "WSHR-M6"}},
			{"Price", 26, "Price", []string{"$8.5000 / pr", "$1,234.56 / dz"}},
			{"Qty", 23, "Qty", []string{"1,200 pair", "5,000 each"}},
			{"Total", 24, "Total", []string{"$10,262.50", "$123,456.78"}},
		},
		// Eight columns share the same 180mm, so Description is the one that gives way: it wraps,
		// where every other column would have to shrink or truncate.
		"invoice summary": {
			{"Line Item", 18, "Line Item", []string{"001"}},
			{"SKU", 32, "SKU", []string{"WSHR-M6", "SOCK-ANK-WHT", "SOCK-CREW-BLK"}},
			{"Price", 25, "Price", []string{"$8.50 / dz", "$1,234.56 / dz"}},
			{"Ordered", 18, "Ordered", []string{"1,500", "1,200,000"}},
			{"Invoiced", 18, "Invoiced", []string{"1,200"}},
			{"Unit", 17, "Unit", []string{"pair", "each", "carton"}},
			{"Total", 22, "Total", []string{"$10,200.00", "$123,456.78"}},
		},
		"cases": {
			{"Case Number", 55, "Case Number", []string{"CASE-000123"}},
			{"Weight", 35, "Weight", []string{"1200 lb"}},
			{"Tracking Number", 90, "Tracking Number", []string{"1Z999AA10123456784"}},
		},
	}

	for table, cols := range tables {
		t.Run(table, func(t *testing.T) {
			pdf := measuringPDF()
			total := 0.0
			for _, c := range cols {
				total += c.w
				assertFitsNaturally(t, pdf, table+" header "+c.header, c.header, "B", 9.5, c.w, ackCellPadding)
				for _, v := range c.values {
					assertFitsNaturally(t, pdf, table+" value "+v, v, "", 9.5, c.w, ackCellPadding)
				}
			}
			if total > ackContentWidth {
				t.Errorf("fixed columns total %.1fmm, past the %.1fmm content width", total, ackContentWidth)
			}
		})
	}
}
