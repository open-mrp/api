package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSupplier,
		Load:       resourceloaders.LoadSuppliers,
		Subs: []include.SubField{
			supplierRef("defaults.carrier", constants.ObjectTypeCarrier, resourceloaders.SupplierMetaCarrierID, func(d *apiresource.SupplierDefaults, v any) {
				d.Carrier = v.(*apiresource.Carrier)
			}),
			supplierRef("defaults.service_level", constants.ObjectTypeServiceLevel, resourceloaders.SupplierMetaServiceLevelID, func(d *apiresource.SupplierDefaults, v any) {
				d.ServiceLevel = v.(*apiresource.ServiceLevel)
			}),
			supplierRef("defaults.payment_term", constants.ObjectTypePaymentTerm, resourceloaders.SupplierMetaPaymentTermID, func(d *apiresource.SupplierDefaults, v any) {
				d.PaymentTerm = v.(*apiresource.PaymentTerm)
			}),
			supplierRef("defaults.shipping_term", constants.ObjectTypeShippingTerm, resourceloaders.SupplierMetaShippingTermID, func(d *apiresource.SupplierDefaults, v any) {
				d.ShippingTerm = v.(*apiresource.ShippingTerm)
			}),
			supplierAddress("defaults.bill_to_address", resourceloaders.SupplierMetaBillToAddress, func(d *apiresource.SupplierDefaults, a *apiresource.Address) {
				d.BillToAddress = a
			}),
			supplierAddress("defaults.ship_to_address", resourceloaders.SupplierMetaShipToAddress, func(d *apiresource.SupplierDefaults, a *apiresource.Address) {
				d.ShipToAddress = a
			}),
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSupplierItem,
		Load: func(context.Context, []string) (map[string]any, *apierror.APIError) {
			return nil, apierror.NewInvariantViolationError("Supplier items are not loaded by ID; nothing expands one.")
		},
		Subs: []include.SubField{
			supplierItemRef("supplier", constants.ObjectTypeSupplier, func(s *apiresource.SupplierItem, v any) {
				s.Supplier = v.(*apiresource.Supplier)
			}),
			supplierItemRef("item", constants.ObjectTypeItem, func(s *apiresource.SupplierItem, v any) {
				s.Item = v.(*apiresource.Item)
			}),
		},
	})
}

// supplierRef declares an include that loads one record a supplier's defaults name by the ID its presenter stashed
// under metaKey.
func supplierRef(key string, target constants.ObjectType, metaKey string, set func(d *apiresource.SupplierDefaults, v any)) include.SubField {
	idOf := func(ctx context.Context, parent any) string {
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeSupplier, parent.(*apiresource.Supplier).ID, metaKey)
		return id
	}
	return include.SubField{
		Key:         key,
		Target:      target,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if id := idOf(ctx, parent); id != "" {
				return []string{id}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			s := parent.(*apiresource.Supplier)
			if s.Defaults == nil {
				s.Defaults = &apiresource.SupplierDefaults{}
			}
			if v, ok := loaded[idOf(ctx, parent)]; ok && v != nil {
				set(s.Defaults, v)
			}
		},
	}
}

// supplierAddress declares a default-address include. A supplier's addresses belong to the supplier's own account,
// which the account-scoped address loader can't read, so core-service joins them and the presenter stashes them.
func supplierAddress(key, metaKey string, set func(d *apiresource.SupplierDefaults, a *apiresource.Address)) include.SubField {
	return include.SubField{
		Key: key,
		Populate: func(ctx context.Context, parent any, _ map[string]any) {
			s := parent.(*apiresource.Supplier)
			if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSupplier, s.ID, metaKey); ok && s.Defaults != nil {
				set(s.Defaults, v.(*apiresource.Address))
			}
		},
	}
}

// supplierItemRef declares an include that loads the record a supplier item names by the ID its presenter stashed.
func supplierItemRef(key string, target constants.ObjectType, set func(s *apiresource.SupplierItem, v any)) include.SubField {
	idOf := func(ctx context.Context, parent any) string {
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeSupplierItem, parent.(*apiresource.SupplierItem).ID, key+"_id")
		return id
	}
	return include.SubField{
		Key:         key,
		Target:      target,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if id := idOf(ctx, parent); id != "" {
				return []string{id}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			if v, ok := loaded[idOf(ctx, parent)]; ok && v != nil {
				set(parent.(*apiresource.SupplierItem), v)
			}
		},
	}
}
