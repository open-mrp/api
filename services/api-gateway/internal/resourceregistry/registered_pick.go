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
			{
				Key:         "sales_order",
				Target:      constants.ObjectTypeSalesOrder,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractSalesOrderIDFromPick,
				Populate:    populateSalesOrderOnPick,
			},
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromPick,
				Populate:    populateCustomerOnPick,
			},
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

func extractSalesOrderIDFromPick(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Pick)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePick, p.ID, "sales_order_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateSalesOrderOnPick(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Pick)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePick, p.ID, "sales_order_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.SalesOrder = v.(*apiresource.SalesOrder)
	}
}

func extractCustomerIDFromPick(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Pick)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePick, p.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnPick(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Pick)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePick, p.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.Customer = v.(*apiresource.Customer)
	}
}

func extractLineRefsFromPick(_ context.Context, parent any) []any {
	p := parent.(*apiresource.Pick)
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
	p := parent.(*apiresource.Pick)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypePick, p.ID, "lines")
	if !ok {
		return
	}
	p.Lines = v.(*apiresource.List[apiresource.PickLine])
}

func populateDepartmentsOnPick(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.Pick)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypePick, p.ID, "departments")
	if !ok {
		return
	}
	p.Departments = v.(*apiresource.List[apiresource.Department])
}

func populateSalesOrderLineOnPickLine(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.PickLine)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypePickLine, p.ID, "sales_order_line")
	if !ok {
		return
	}
	p.SalesOrderLine = v.(*apiresource.SalesOrderLine)
}
