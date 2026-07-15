package service

import (
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/field"
	"github.com/stretchr/testify/assert"
)

// salesOrderShippingChanged decides whether the async shipment-cascade event fires.
// It must read the raw request (an omitted field is never a change) and handle the
// clearable service level correctly.
func TestSalesOrderShippingChanged(t *testing.T) {
	t.Parallel()

	existing := &domain.SalesOrder{
		CarrierID:         ptrStr("cr_1"),
		ServiceLevelID:    ptrStr("sl_1"),
		ShippingAddressID: "addr_1",
	}

	cases := []struct {
		name   string
		params domain.UpdateSalesOrderParams
		want   bool
	}{
		{"nothing provided", domain.UpdateSalesOrderParams{}, false},
		{"same carrier", domain.UpdateSalesOrderParams{CarrierID: ptrStr("cr_1")}, false},
		{"new carrier", domain.UpdateSalesOrderParams{CarrierID: ptrStr("cr_2")}, true},
		{"same ship-to", domain.UpdateSalesOrderParams{ShippingAddressID: ptrStr("addr_1")}, false},
		{"new ship-to", domain.UpdateSalesOrderParams{ShippingAddressID: ptrStr("addr_2")}, true},
		{"service level unset", domain.UpdateSalesOrderParams{ServiceLevelID: field.Unset[string]()}, false},
		{"service level same", domain.UpdateSalesOrderParams{ServiceLevelID: field.Set("sl_1")}, false},
		{"service level changed", domain.UpdateSalesOrderParams{ServiceLevelID: field.Set("sl_2")}, true},
		{"service level cleared", domain.UpdateSalesOrderParams{ServiceLevelID: field.Clear[string]()}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, salesOrderShippingChanged(existing, tc.params))
		})
	}
}

func ptrStr(s string) *string { return &s }
