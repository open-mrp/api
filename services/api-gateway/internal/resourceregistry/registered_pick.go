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
		ObjectType: constants.ObjectTypePick,
		Load:       resourceloaders.LoadPicks,
		Subs: []resourcekit.SubField{
			{Key: "sales_order", Populate: populateSalesOrderOnPick},
			{
				Key:         "lines",
				Target:      constants.ObjectTypePickLine,
				ExtractRefs: extractLineRefsFromPick,
				Populate:    populateLinesOnPick,
			},
			{Key: "departments", Populate: populateDepartmentsOnPick},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePickLine,
		Load:       resourceloaders.LoadPickLines,
		Subs: []resourcekit.SubField{
			{Key: "sales_order_line", Populate: populateSalesOrderLineOnPickLine},
		},
	})
}

func populateSalesOrderOnPick(ctx context.Context, parent any, _ map[string]any) {
	meta := resourcekit.GetLoadMeta(ctx)
	switch p := parent.(type) {
	case *apiresource.PickDetail:
		v, ok := meta.Get(constants.ObjectTypePick, p.ID, "sales_order")
		if !ok {
			return
		}
		p.SalesOrder = v.(*apiresource.SalesOrderDetail)
	case *apiresource.PickSummary:
		v, ok := meta.Get(constants.ObjectTypePick, p.ID, "sales_order")
		if !ok {
			return
		}
		p.SalesOrder = v.(*apiresource.SalesOrderDetail)
	}
}

func extractLineRefsFromPick(_ context.Context, parent any) []any {
	p := parent.(*apiresource.PickDetail)
	if p.Lines == nil {
		return nil
	}
	refs := make([]any, len(p.Lines.Data))
	for i := range p.Lines.Data {
		refs[i] = &p.Lines.Data[i]
	}
	return refs
}

func populateLinesOnPick(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.PickDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypePick, p.ID, "lines")
	if !ok {
		return
	}
	p.Lines = v.(*apiresource.List[apiresource.PickLineDetail])
}

func populateDepartmentsOnPick(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.PickDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypePick, p.ID, "departments")
	if !ok {
		return
	}
	p.Departments = v.(*apiresource.List[apiresource.Department])
}

func populateSalesOrderLineOnPickLine(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.PickLineDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypePickLine, p.ID, "sales_order_line")
	if !ok {
		return
	}
	p.SalesOrderLine = v.(*apiresource.SalesOrderLineDetail)
}
