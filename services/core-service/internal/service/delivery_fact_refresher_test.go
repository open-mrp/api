package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
)

func TestDeliveryFactScopeFor(t *testing.T) {
	update, create, del := constants.AuditActionUpdate, constants.AuditActionCreate, constants.AuditActionDelete
	cases := []struct {
		name  string
		event audit.ObservedEvent
		scope domain.DeliveryFactScope
		id    string
		ok    bool
	}{
		{"a line rooted in its order marks the order",
			audit.ObservedEvent{Action: create, ResourceType: constants.ObjectTypePickLine, ResourceID: "pkl_1", RootResourceType: constants.ObjectTypeSalesOrder, RootResourceID: "or_1"},
			domain.DeliveryFactScopeSalesOrder, "or_1", true},
		{"an order", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypeSalesOrder, ResourceID: "or_1"},
			domain.DeliveryFactScopeSalesOrder, "or_1", true},
		{"an unrooted pick line", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypePickLine, ResourceID: "pkl_1"},
			domain.DeliveryFactScopePickLine, "pkl_1", true},
		{"an unrooted pick", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypePick, ResourceID: "pk_1"},
			domain.DeliveryFactScopePick, "pk_1", true},
		{"a quantity edit", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypeQuantity, ResourceID: "qy_1"},
			domain.DeliveryFactScopeQuantity, "qy_1", true},
		{"a quantity create", audit.ObservedEvent{Action: create, ResourceType: constants.ObjectTypeQuantity, ResourceID: "qy_1"},
			domain.DeliveryFactScopeQuantity, "qy_1", false},
		{"a product's line", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypeProduct, ResourceID: "pd_1", ChangedFields: []string{"product_line_id"}},
			domain.DeliveryFactScopeProduct, "pd_1", true},
		{"a product's type", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypeProduct, ResourceID: "pd_1", ChangedFields: []string{"product_type_code"}},
			domain.DeliveryFactScopeProduct, "pd_1", true},
		{"a product's name", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypeProduct, ResourceID: "pd_1", ChangedFields: []string{"name"}},
			domain.DeliveryFactScopeProduct, "pd_1", false},
		{"a customer merged away", audit.ObservedEvent{Action: del, ResourceType: constants.ObjectTypeCustomer, ResourceID: "ac_1"},
			domain.DeliveryFactScopeReconcile, "all", true},
		{"a customer renamed", audit.ObservedEvent{Action: update, ResourceType: constants.ObjectTypeCustomer, ResourceID: "ac_1"},
			domain.DeliveryFactScopeReconcile, "all", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			scope, id, ok := deliveryFactScopeFor(c.event)
			require.Equal(t, c.ok, ok)
			if ok {
				require.Equal(t, c.scope, scope)
				require.Equal(t, c.id, id)
			}
		})
	}
}

func TestDiffDeliveryFacts(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC) }
	row := func(shipBy time.Time, line, ordered string) domain.DeliveryOrderFact {
		return domain.DeliveryOrderFact{OwnerAccountID: "ac", ShipByDate: shipBy, SalesOrderID: "or", ProductLineKey: line,
			SalesOrderNumber: "1", BuyerAccountID: "b", StatusCode: "issued", QuantityOrdered: ordered, QuantityPacked: "0"}
	}
	stored := []domain.DeliveryOrderFact{row(day(1), "a", "1"), row(day(1), "b", "2"), row(day(1), "", "3")}
	computed := []domain.DeliveryOrderFact{row(day(1), "a", "1"), row(day(1), "b", "5"), row(day(2), "", "3")}

	upserts, deletes := diffDeliveryFacts(computed, stored)
	require.Equal(t, []domain.DeliveryOrderFact{row(day(1), "b", "5"), row(day(2), "", "3")}, upserts, "changed and re-keyed rows are written")
	require.Equal(t, []domain.DeliveryOrderFact{row(day(1), "", "3")}, deletes, "a row whose key moved is deleted at the old key")

	upserts, deletes = diffDeliveryFacts(stored, stored)
	require.Empty(t, upserts)
	require.Empty(t, deletes)
}

func TestNextDeliveryBatch(t *testing.T) {
	require.Less(t, nextDeliveryBatch(20, 100*time.Millisecond), 20, "a slow recompute shrinks the batch")
	require.Equal(t, 1, nextDeliveryBatch(1, time.Second), "never below one order")
	require.Greater(t, nextDeliveryBatch(20, time.Millisecond), 20, "a fast recompute grows it")
	require.Equal(t, deliveryFactMaxBatch, nextDeliveryBatch(deliveryFactMaxBatch, time.Millisecond), "up to the cap")
	require.Equal(t, 20, nextDeliveryBatch(20, 20*time.Millisecond), "near the target it holds")
}
