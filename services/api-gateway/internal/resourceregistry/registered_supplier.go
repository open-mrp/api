package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSupplier,
		Load:       resourceloaders.LoadSuppliers,
		Subs: []resourcekit.SubField{
			{Key: "bill_to_address", Populate: populateBillToAddressOnSupplier},
			{Key: "ship_to_address", Populate: populateShipToAddressOnSupplier},
		},
	})
}

func populateBillToAddressOnSupplier(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.SupplierDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSupplier, s.ID, "bill_to_address")
	if !ok {
		return
	}
	s.BillToAddress = v.(*apiresource.Address)
}

func populateShipToAddressOnSupplier(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.SupplierDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSupplier, s.ID, "ship_to_address")
	if !ok {
		return
	}
	s.ShipToAddress = v.(*apiresource.Address)
}
