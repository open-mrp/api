package service

import (
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
	tests := []struct {
		name  string
		event audit.ObservedEvent
		want  domain.SalesFactScope
		ok    bool
	}{
		{"invoice", audit.ObservedEvent{ResourceType: constants.ObjectTypeInvoice}, domain.SalesFactScopeInvoice, true},
		{"sales order", audit.ObservedEvent{ResourceType: constants.ObjectTypeSalesOrder}, domain.SalesFactScopeSalesOrder, true},
		{"order line", audit.ObservedEvent{ResourceType: constants.ObjectTypeSalesOrderLine}, domain.SalesFactScopeSalesOrderLine, true},
		{"product line moved", audit.ObservedEvent{ResourceType: constants.ObjectTypeProduct, ChangedFields: []string{"name", "product_line_id"}}, domain.SalesFactScopeProduct, true},
		{"product renamed", audit.ObservedEvent{ResourceType: constants.ObjectTypeProduct, ChangedFields: []string{"name"}}, domain.SalesFactScopeProduct, false},
		{"unrelated", audit.ObservedEvent{ResourceType: constants.ObjectTypeShipment}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := salesFactScopeFor(tt.event)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Fatalf("salesFactScopeFor = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}
