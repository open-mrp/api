package service

import (
	"slices"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
)

func fact(line, total string) domain.SalesLineFact {
	return domain.SalesLineFact{
		AccountID:     "ac_1",
		InvoicedAt:    time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		InvoiceLineID: line,
		InvoiceID:     "iv_1",
		TotalInvoiced: &total,
	}
}

func TestDiffSalesFacts(t *testing.T) {
	unchanged, changedOld, changedNew, gone, added := fact("il_same", "10.5"), fact("il_changed", "3"), fact("il_changed", "4"), fact("il_gone", "7"), fact("il_new", "1")

	upserts, deletes := diffSalesFacts(
		[]domain.SalesLineFact{unchanged, changedNew, added},
		[]domain.SalesLineFact{unchanged, changedOld, gone},
	)

	if len(upserts) != 2 || upserts[0].InvoiceLineID != "il_changed" || upserts[1].InvoiceLineID != "il_new" {
		t.Fatalf("upserts = %+v, want il_changed then il_new", upserts)
	}
	if len(deletes) != 1 || deletes[0].InvoiceLineID != "il_gone" {
		t.Fatalf("deletes = %+v, want il_gone", deletes)
	}
}

func TestSalesFactsEqualComparesEveryField(t *testing.T) {
	base := fact("il_1", "10")
	rep := "acus_1"
	variants := map[string]func(f *domain.SalesLineFact){
		"invoiced_at": func(f *domain.SalesLineFact) { f.InvoicedAt = f.InvoicedAt.Add(time.Millisecond) },
		"buyer":       func(f *domain.SalesLineFact) { f.BuyerAccountID = "ac_other" },
		"sales rep":   func(f *domain.SalesLineFact) { f.SalesRepID = &rep },
		"type":        func(f *domain.SalesLineFact) { f.SalesOrderTypeCode = "purchase_order" },
		"total":       func(f *domain.SalesLineFact) { v := "10.000000000000000000000000000001"; f.TotalInvoiced = &v },
		"cost nil":    func(f *domain.SalesLineFact) { v := "0"; f.TotalCost = &v },
	}
	for name, mutate := range variants {
		changed := base
		mutate(&changed)
		if salesFactsEqual(base, changed) {
			t.Errorf("%s: a changed fact compared equal", name)
		}
	}
	if !salesFactsEqual(base, base) {
		t.Error("a fact did not equal itself")
	}
}

func TestSalesFactScopeFor(t *testing.T) {
	update := func(rt constants.ObjectType, fields ...string) audit.ObservedEvent {
		return audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: rt, ResourceID: "id_1", ChangedFields: fields}
	}
	tests := []struct {
		name    string
		event   audit.ObservedEvent
		want    domain.SalesFactScope
		wantID  string
		matches bool
	}{
		{"invoice", audit.ObservedEvent{ResourceType: constants.ObjectTypeInvoice, ResourceID: "iv_1"}, domain.SalesFactScopeInvoice, "iv_1", true},
		{"sales order", audit.ObservedEvent{ResourceType: constants.ObjectTypeSalesOrder, ResourceID: "or_1"}, domain.SalesFactScopeSalesOrder, "or_1", true},
		{"order line", audit.ObservedEvent{ResourceType: constants.ObjectTypeSalesOrderLine, ResourceID: "sol_1"}, domain.SalesFactScopeSalesOrderLine, "sol_1", true},
		{"product line moved", update(constants.ObjectTypeProduct, "name", "product_line_id"), domain.SalesFactScopeProduct, "id_1", true},
		{"product renamed", update(constants.ObjectTypeProduct, "name"), "", "", false},
		{"quantity edited", update(constants.ObjectTypeQuantity, "value"), domain.SalesFactScopeQuantity, "id_1", true},
		{"quantity created", audit.ObservedEvent{Action: constants.AuditActionCreate, ResourceType: constants.ObjectTypeQuantity, ResourceID: "id_1"}, "", "", false},
		{"rate edited", update(constants.ObjectTypeRate, "value"), domain.SalesFactScopeRate, "id_1", true},
		{"item recategorized", update(constants.ObjectTypeItem, "item_category_id", "category_name"), domain.SalesFactScopeItem, "id_1", true},
		{"item renamed", update(constants.ObjectTypeItem, "description"), "", "", false},
		{"customer merged away", audit.ObservedEvent{Action: constants.AuditActionDelete, ResourceType: constants.ObjectTypeCustomer, ResourceID: "ac_old"}, domain.SalesFactScopeBuyer, "ac_old", true},
		{"customer edited", update(constants.ObjectTypeCustomer, "name"), "", "", false},
		{"unit ratio changed", update(constants.ObjectTypeUnit, "ratio_numerator"), domain.SalesFactScopeReconcile, salesFactReconcileScopeID, true},
		{"unit renamed", update(constants.ObjectTypeUnit, "name"), "", "", false},
		{"unit group base unit changed", update(constants.ObjectTypeUnitGroup, "base_unit.id"), domain.SalesFactScopeReconcile, salesFactReconcileScopeID, true},
		{"category unit group changed", update(constants.ObjectTypeItemCategory, "unit_group_id"), domain.SalesFactScopeReconcile, salesFactReconcileScopeID, true},
		{"unrelated", audit.ObservedEvent{ResourceType: constants.ObjectTypeShipment, ResourceID: "sh_1"}, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, id, ok := salesFactScopeFor(tt.event)
			if ok != tt.matches || (ok && (got != tt.want || id != tt.wantID)) {
				t.Fatalf("salesFactScopeFor = (%q, %q, %v), want (%q, %q, %v)", got, id, ok, tt.want, tt.wantID, tt.matches)
			}
		})
	}
}

func TestTouchedRollupDaysIncludesTheDayALineLeft(t *testing.T) {
	moved := fact("il_moved", "5")
	before := moved
	before.InvoicedAt = moved.InvoicedAt.AddDate(0, 0, -3)
	gone := fact("il_gone", "1")
	gone.AccountID = "ac_2"

	days := touchedRollupDays([]domain.SalesLineFact{moved}, []domain.SalesLineFact{gone}, []domain.SalesLineFact{before, gone})

	want := []domain.SalesRollupDay{
		{AccountID: "ac_1", Day: utcDay(moved.InvoicedAt)},
		{AccountID: "ac_1", Day: utcDay(before.InvoicedAt)},
		{AccountID: "ac_2", Day: utcDay(gone.InvoicedAt)},
	}
	if !slices.Equal(days, want) {
		t.Fatalf("touchedRollupDays = %v, want %v", days, want)
	}
}
