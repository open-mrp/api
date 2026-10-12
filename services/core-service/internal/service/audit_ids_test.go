package service

import (
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/assert"
)

func TestSupplierMaterialAuditEvent_KeysByThePublicMaterialIDUnderTheSupplier(t *testing.T) {
	sm := &domain.SupplierMaterial{ID: "suml_link", MaterialID: "ml_public", SupplierAccountID: "ac_supplier"}
	event := supplierMaterialAuditEvent(constants.AuditActionUpdate, sm, nil)
	assert.Equal(t, constants.ObjectTypeSupplierMaterial, event.ResourceType)
	assert.Equal(t, "ml_public", event.ResourceID)
	assert.Equal(t, constants.ObjectTypeSupplier, event.RootResourceType)
	assert.Equal(t, "ac_supplier", event.RootResourceID)
	assert.Equal(t, constants.AuditActionUpdate, event.Action)
}

func TestPurchaseOrderLineAuditEvent_RootsAtThePurchaseOrder(t *testing.T) {
	line := &domain.PurchaseOrderLine{ID: "orli_1", SalesOrderID: "or_po"}
	event := purchaseOrderLineAuditEvent(constants.AuditActionCreate, line, nil)
	assert.Equal(t, constants.ObjectTypePurchaseOrderLine, event.ResourceType)
	assert.Equal(t, "orli_1", event.ResourceID)
	assert.Equal(t, constants.ObjectTypePurchaseOrder, event.RootResourceType)
	assert.Equal(t, "or_po", event.RootResourceID)
}

func TestRateDiffers(t *testing.T) {
	current := &domain.Rate{Value: "0.750000", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"}
	for name, tc := range map[string]struct {
		next domain.CreateRateParams
		want bool
	}{
		"same value at another scale": {domain.CreateRateParams{Value: "0.75", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"}, false},
		"new value":                   {domain.CreateRateParams{Value: "1.25", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"}, true},
		"new denominator":             {domain.CreateRateParams{Value: "0.75", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_box"}, true},
		"new numerator":               {domain.CreateRateParams{Value: "0.75", NumeratorUnitID: "un_eur", DenominatorUnitID: "un_each"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, rateDiffers(current, &tc.next))
		})
	}
}
