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
		ObjectType: constants.ObjectTypeInvoice,
		Load:       resourceloaders.LoadInvoices,
		Subs: []resourcekit.SubField{
			{Key: "lines", Populate: populateLinesOnInvoice},
			{Key: "allocations", Populate: populateAllocationsOnInvoice},
		},
	})
}

func populateLinesOnInvoice(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.Invoice)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoice, inv.ID, "lines")
	if !ok {
		return
	}
	inv.Lines = v.(*apiresource.List[apiresource.InvoiceLine])
}

func populateAllocationsOnInvoice(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.Invoice)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoice, inv.ID, "allocations")
	if !ok {
		return
	}
	inv.Allocations = v.(*apiresource.List[apiresource.InvoiceAllocation])
}
