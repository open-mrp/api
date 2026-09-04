package resourceregistry

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSupplier,
		Load:       resourceloaders.LoadSuppliers,
		Subs: []resourcekit.SubField{
			{
				Key:         "bill_to_address",
				Target:      constants.ObjectTypeAddress,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractBillToAddressIDFromSupplier,
				Populate:    populateBillToAddressOnSupplier,
			},
			{
				Key:         "ship_to_address",
				Target:      constants.ObjectTypeAddress,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractShipToAddressIDFromSupplier,
				Populate:    populateShipToAddressOnSupplier,
			},
		},
	})
}

func extractBillToAddressIDFromSupplier(ctx context.Context, parent any) []string {
	return supplierAddressID(ctx, parent, "bill_to_address_id")
}

func extractShipToAddressIDFromSupplier(ctx context.Context, parent any) []string {
	return supplierAddressID(ctx, parent, "ship_to_address_id")
}

func populateBillToAddressOnSupplier(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Supplier)
	s.BillToAddress = supplierAddress(ctx, s.ID, "bill_to_address", loaded)
}

func populateShipToAddressOnSupplier(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Supplier)
	s.ShipToAddress = supplierAddress(ctx, s.ID, "ship_to_address", loaded)
}

func supplierAddressID(ctx context.Context, parent any, key string) []string {
	s := parent.(*apiresource.Supplier)
	meta := resourcekit.GetLoadMeta(ctx)
	// Nothing to fetch when the backend already expanded the record.
	if _, ok := meta.Get(constants.ObjectTypeSupplier, s.ID, strings.TrimSuffix(key, "_id")); ok {
		return nil
	}
	id, _ := meta.GetString(constants.ObjectTypeSupplier, s.ID, key)
	if id == "" {
		return nil
	}
	return []string{id}
}

// Every supplier endpoint asks the backend to expand the addresses, so the record is already
// stashed; the id path remains for a caller that reaches a supplier from a document, where only the
// id is known. Either way the caller sees the same address.
func supplierAddress(ctx context.Context, supplierID, key string, loaded map[string]any) *apiresource.Address {
	meta := resourcekit.GetLoadMeta(ctx)
	if v, ok := meta.Get(constants.ObjectTypeSupplier, supplierID, key); ok {
		return v.(*apiresource.Address)
	}
	id, _ := meta.GetString(constants.ObjectTypeSupplier, supplierID, key+"_id")
	if id == "" {
		return nil
	}
	if v, ok := loaded[id]; ok {
		return v.(*apiresource.Address)
	}
	return nil
}
