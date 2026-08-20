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
				Key:         "related.sales_order",
				Target:      constants.ObjectTypeSalesOrder,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractSalesOrderIDFromPick,
				Populate:    populateSalesOrderOnPickRelated,
			},
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromPick,
				Populate:    populateCustomerOnPick,
			},
			{
				Key:         "related.shipments",
				Target:      constants.ObjectTypeShipment,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractShipmentIDsFromPickRelated,
				Populate:    populateShipmentsOnPickRelated,
			},
			{
				Key:         "lines",
				Target:      constants.ObjectTypePickLine,
				ExtractRefs: extractLineRefsFromPick,
				Populate:    populateLinesOnPick,
			},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePickLine,
		Load:       resourceloaders.LoadPickLines,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromPickLine,
				Populate:    populateItemOnPickLine,
			},
			// ExtractRefs rather than a loader: the pick presenter already stashed the line, so this
			// exists only so the resolver descends for lines.sales_order_line.product.
			{
				Key:         "sales_order_line",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractSalesOrderLineRefFromPickLine,
				Populate:    populateSalesOrderLineOnPickLine,
			},
			// The quantities are already attached by the presenter; Target + ExtractRefs exist so
			// the resolver can recurse into them and hydrate their unit via LoadUnits (the unit FK
			// is stashed on each Quantity in stashPickLineDetailMeta).
			{
				Key:         "quantity",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractQuantityRefFromPickLine,
			},
			{
				Key:         "ordered_quantity",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractOrderedQuantityRefFromPickLine,
			},
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

// Lazily creates the related group on first expanded member, so it serializes to null when no
// related include was requested.
func ensurePickRelated(p *apiresource.Pick) *apiresource.PickRelated {
	if p.Related == nil {
		p.Related = &apiresource.PickRelated{Object: constants.ObjectTypePickRelated}
	}
	return p.Related
}

func populateSalesOrderOnPickRelated(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Pick)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePick, p.ID, "sales_order_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	so := v.(*apiresource.SalesOrder)
	rec := apiresource.NewRecord(id, constants.RecordTypeSalesOrder)
	rec.Number = &so.Number
	status := string(so.Status)
	rec.Status = &status
	ensurePickRelated(p).SalesOrder = rec
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

func extractQuantityRefFromPickLine(_ context.Context, parent any) []any {
	p := parent.(*apiresource.PickLine)
	if p.Quantity == nil {
		return nil
	}
	return []any{p.Quantity}
}

func extractOrderedQuantityRefFromPickLine(_ context.Context, parent any) []any {
	p := parent.(*apiresource.PickLine)
	if p.OrderedQuantity == nil {
		return nil
	}
	return []any{p.OrderedQuantity}
}

// Runs after Populate has attached the line, so the resolver has something to descend into.
func extractSalesOrderLineRefFromPickLine(_ context.Context, parent any) []any {
	p := parent.(*apiresource.PickLine)
	if p.SalesOrderLine == nil {
		return nil
	}
	return []any{p.SalesOrderLine}
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

func extractShipmentIDsFromPickRelated(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Pick)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypePick, p.ID, "related_shipment_ids")
	return ids
}

func populateShipmentsOnPickRelated(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Pick)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypePick, p.ID, "related_shipment_ids")
	if len(ids) == 0 {
		return
	}
	records := make([]apiresource.Record, 0, len(ids))
	for _, id := range ids {
		v, ok := loaded[id]
		if !ok {
			continue
		}
		s := v.(*apiresource.Shipment)
		rec := apiresource.NewRecord(id, constants.RecordTypeShipment)
		rec.Number = &s.Number
		status := string(s.Status)
		rec.Status = &status
		// The shipment loader stashes tracking/carrier/ship-date, so a pick can preview each
		// linked shipment without expanding the full shipment resource.
		if m, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, id, "record_metadata"); ok {
			if meta, ok := m.(map[string]string); ok && len(meta) > 0 {
				rec.Metadata = meta
			}
		}
		records = append(records, *rec)
	}
	if len(records) == 0 {
		return
	}
	ensurePickRelated(p).Shipments = apiresource.NewList(records, apiresource.PageInfo{})
}

func extractItemIDFromPickLine(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.PickLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePickLine, p.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnPickLine(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.PickLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePickLine, p.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.Item = v.(*apiresource.Item)
	}
}
