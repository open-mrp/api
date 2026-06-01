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
		ObjectType: constants.ObjectTypeItemInventory,
		Load:       resourceloaders.LoadItemInventories,
		Subs: []resourcekit.SubField{
			{Key: "on_hand", Populate: populateOnHandOnItemInventory},
			{Key: "reserved", Populate: populateReservedOnItemInventory},
			{Key: "available_to_promise", Populate: populateATPOnItemInventory},
			{Key: "short", Populate: populateShortOnItemInventory},
		},
	})
}

func populateOnHandOnItemInventory(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.ItemInventory)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeItemInventory, "singleton", "on_hand")
	if !ok {
		return
	}
	inv.OnHand = v.(*apiresource.Quantity)
}

func populateReservedOnItemInventory(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.ItemInventory)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeItemInventory, "singleton", "reserved")
	if !ok {
		return
	}
	inv.Reserved = v.(*apiresource.Quantity)
}

func populateATPOnItemInventory(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.ItemInventory)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeItemInventory, "singleton", "available_to_promise")
	if !ok {
		return
	}
	inv.AvailableToPromise = v.(*apiresource.Quantity)
}

func populateShortOnItemInventory(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.ItemInventory)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeItemInventory, "singleton", "short")
	if !ok {
		return
	}
	inv.Short = v.(*apiresource.Quantity)
}
