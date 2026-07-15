package recordsep

import (
	"testing"

	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func strp(s string) *string { return &s }

func sampleShipment() *pb.ShipmentInfo {
	return &pb.ShipmentInfo{
		Id:           "shp_1",
		Number:       "SH-001",
		SalesOrderId: "so_1",
		AccountId:    "acc_1",
		CarrierName:  "UPS",
		ShippedAt:    timestamppb.Now(),
		Lines: []*pb.ShipmentLineInfo{
			// Deliberately out of line-number order to exercise sorting.
			{Id: "shln_2", SalesOrderLineId: "sol_2", OrderLineSku: "RED", OrderLineDescription: strp("Red Widget"), QuantityValue: "5", QuantityUnitName: "each"},
			{Id: "shln_1", SalesOrderLineId: "sol_1", OrderLineSku: "BLUE", OrderLineDescription: strp("Blue Widget"), QuantityValue: "10", QuantityUnitName: "each"},
		},
		ShippingCases: []*pb.ShippingCaseDetailInfo{
			{Id: "case_b", Number: "CASE-002", FreightWeightValue: "8", FreightWeightUnitAbbreviation: "lb", TrackingNumber: strp("1ZB"), CarrierName: "UPS"},
			{Id: "case_a", Number: "CASE-001", FreightWeightValue: "12", FreightWeightUnitAbbreviation: "lb", CarrierName: "UPS"},
		},
	}
}

func sampleOrder() *pb.SalesOrderInfo {
	return &pb.SalesOrderInfo{
		Id:               "so_1",
		Number:           "000123",
		CustomerPoNumber: strp("PO-9"),
		BillToName:       strp("Acme"),
		BillToPhone:      strp("555-0100"),
		PriorityName:     "Standard",
		PaymentTermName:  strp("Net 30"),
		SalesRepName:     strp("Jordan"),
		InvoiceEmails:    []string{"AR@acme.com"},
		Lines: []*pb.SalesOrderLineInfo{
			// Sale line, fully packed → not back-ordered.
			{Id: "sol_1", LineItemNumber: 1, ProductSku: "BLUE", ProductDescription: strp("Blue Widget"), QuantityValue: "10", QuantityPackedValue: strp("10"), QuantityUnitName: "each", ProductTypeCode: strp("sale")},
			// Sale line, partially packed → back-ordered by 5.
			{Id: "sol_2", LineItemNumber: 2, ProductSku: "RED", ProductDescription: strp("Red Widget"), QuantityValue: "20", QuantityPackedValue: strp("15"), QuantityUnitName: "each", ProductTypeCode: strp("sale")},
			// Non-sale (shipping) line, unpacked → must be excluded from back-orders.
			{Id: "sol_3", LineItemNumber: 3, ProductSku: "FREIGHT", QuantityValue: "1", QuantityPackedValue: strp("0"), QuantityUnitName: "each", ProductTypeCode: strp("shipping")},
		},
	}
}

func TestAssemblePackList_LineItemNumbersJoinedAndSorted(t *testing.T) {
	pl := assemblePackList(sampleShipment(), sampleOrder(), "Acme", strp("https://logo"))

	items := pl.LineItems.Data
	if len(items) != 2 {
		t.Fatalf("expected 2 line items, got %d", len(items))
	}
	// Sorted by the joined line number: BLUE (1) before RED (2).
	if items[0].SKU != "BLUE" || items[0].LineItemNumber == nil || *items[0].LineItemNumber != 1 {
		t.Errorf("first line = %+v; want BLUE with line number 1", items[0])
	}
	if items[1].SKU != "RED" || items[1].LineItemNumber == nil || *items[1].LineItemNumber != 2 {
		t.Errorf("second line = %+v; want RED with line number 2", items[1])
	}
}

func TestAssemblePackList_BackOrdersOnlySaleAndUnderpacked(t *testing.T) {
	pl := assemblePackList(sampleShipment(), sampleOrder(), "Acme", nil)

	bo := pl.BackOrders.Data
	if len(bo) != 1 {
		t.Fatalf("expected 1 back-order (only the under-packed sale line), got %d: %+v", len(bo), bo)
	}
	row := bo[0]
	if row.SKU != "RED" {
		t.Errorf("back-order SKU = %q; want RED", row.SKU)
	}
	if row.QuantityOrdered != "20" || row.QuantityShipped != "15" || row.QuantityBackOrdered != "5" {
		t.Errorf("back-order qty = ordered %q shipped %q back %q; want 20/15/5", row.QuantityOrdered, row.QuantityShipped, row.QuantityBackOrdered)
	}
}

func TestAssemblePackList_ContactInfoAndCases(t *testing.T) {
	pl := assemblePackList(sampleShipment(), sampleOrder(), "Acme", nil)

	// Emails lowercased, then the bill-to phone.
	want := []string{"ar@acme.com", "555-0100"}
	if len(pl.ContactInformation) != len(want) {
		t.Fatalf("contact info = %v; want %v", pl.ContactInformation, want)
	}
	for i := range want {
		if pl.ContactInformation[i] != want[i] {
			t.Errorf("contact[%d] = %q; want %q", i, pl.ContactInformation[i], want[i])
		}
	}

	// Cases sorted by number; a missing tracking number maps to nil.
	cases := pl.ShippingCases.Data
	if len(cases) != 2 || cases[0].Number != "CASE-001" || cases[1].Number != "CASE-002" {
		t.Fatalf("cases not sorted by number: %+v", cases)
	}
	if cases[0].TrackingNumber != nil {
		t.Errorf("CASE-001 tracking = %v; want nil", *cases[0].TrackingNumber)
	}
	if cases[1].TrackingNumber == nil || *cases[1].TrackingNumber != "1ZB" {
		t.Errorf("CASE-002 tracking = %v; want 1ZB", cases[1].TrackingNumber)
	}
}
