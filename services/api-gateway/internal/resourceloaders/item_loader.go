package resourceloaders

import (
	"context"
	"github.com/open-mrp/apikit/include"
	"strings"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
)

var itemLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.item")

// ItemRecordIncludes are the stitches BatchGetItemsByIDs always applies; core GetItem needs them named to return the same record.
var ItemRecordIncludes = []string{"unit_value", "unit_cost", "burn_rate", "attributes"}

func LoadItems(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	withCategories, withAttributeProperties := ItemEmbedsRequested(ctx)
	resp, apiErr := grpcutil.CallRPC(ctx, itemLoaderTracer, "loader.items.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetItemsByIDsResponse, error) {
			return coreClient.BatchGetItemsByIDs(ctx, &pb.BatchGetItemsByIDsRequest{
				Ids:                     ids,
				WithCategories:          withCategories,
				WithAttributeProperties: withAttributeProperties,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return ItemsFromProto(ctx, resp.Items, resp.Categories, resp.AttributeProperties), nil
}

// ItemsFromProto builds items keyed by id and stashes what their expandable fields resolve from.
//
// categories and properties were read with the items, so whoever may read an item sees them without their own permissions.
func ItemsFromProto(ctx context.Context, items []*pb.ItemInfo, categories []*pb.ItemCategoryInfo, properties []*pb.PropertyInfo) map[string]any {
	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(items))

	categoryMap := ItemCategoriesFromProto(ctx, categories)
	propertyMap := make(map[string]*apiresource.Property, len(properties))
	for _, p := range properties {
		propertyMap[p.Id] = PropertyFromProto(p)
	}

	for _, item := range items {
		out[item.Id] = ItemFromProto(item, IsPreview9(ctx))

		if item.Category != nil {
			if c, ok := categoryMap[item.Category.Id]; ok {
				meta.Set(constants.ObjectTypeItem, item.Id, "category", c)
			}
		}

		if item.Sales != nil && item.Sales.ProductLineId != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, ItemSalesProductLineMetaKey, *item.Sales.ProductLineId)
		}
		meta.Set(constants.ObjectTypeItem, item.Id, ItemStockingUnitMetaKey, item.StockingUnitId)
		meta.Set(constants.ObjectTypeItem, item.Id, ItemDefaultOrderUnitMetaKey, item.DefaultOrderUnitId)
		meta.Set(constants.ObjectTypeItem, item.Id, ItemOrderUnitsMetaKey, ItemOrderUnitsFromProto(item.OrderUnits))

		attrs := make([]apiresource.Attribute, 0, len(item.Attributes))
		for _, a := range item.Attributes {
			if a == nil {
				continue
			}
			attr := apiresource.Attribute{
				ID:        a.Id,
				Object:    constants.ObjectTypeAttribute,
				Value:     a.Value,
				SortOrder: a.SortOrder,
				Status:    constants.ArchiveStatus(a.Status),
				CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
				UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
			}
			if a.ColorCode != nil {
				attr.ColorCode = constants.Color(*a.ColorCode)
			}
			StashAttributeProperty(ctx, a.Id, a.PropertyId, propertyMap[a.PropertyId])
			attrs = append(attrs, attr)
		}
		meta.Set(constants.ObjectTypeItem, item.Id, "attributes_list",
			apiresource.NewList(attrs, apiresource.PageInfo{}))
	}
	return out
}

// ItemEmbedsRequested reports whether some requested include reaches an item's category or its attributes' properties, at any depth.
func ItemEmbedsRequested(ctx context.Context) (categories, attributeProperties bool) {
	for _, include := range include.RequestedIncludes(ctx) {
		if include == "category" || strings.HasSuffix(include, ".category") {
			categories = true
		}
		if include == "attributes.property" || strings.HasSuffix(include, ".attributes.property") {
			attributeProperties = true
		}
	}
	return categories, attributeProperties
}

// ItemSalesProductLineMetaKey stashes the product line an item's sales section names, for `sales.product_line`.
const ItemSalesProductLineMetaKey = "sales.product_line_id"

// Meta keys for an item's units: the stocking and default order unit ids, and the order units list.
const (
	ItemStockingUnitMetaKey     = "stocking_unit_id"
	ItemDefaultOrderUnitMetaKey = "default_order_unit_id"
	ItemOrderUnitsMetaKey       = "order_units_list"
)

// ItemOrderUnitsFromProto presents an item's order units. Each unit loads before the response is written.
func ItemOrderUnitsFromProto(in []*pb.ItemOrderUnitInfo) *apiresource.List[apiresource.ItemOrderUnit] {
	out := make([]apiresource.ItemOrderUnit, 0, len(in))
	for _, ou := range in {
		if ou == nil {
			continue
		}
		out = append(out, apiresource.ItemOrderUnit{
			Unit:             apiresource.UnitStub(ou.UnitId),
			PortalVisibility: constants.CustomerPortalVisibility(ou.PortalVisibility),
			Discount:         DiscountFromProto(ou.Discount),
			Quantity:         ou.Quantity,
		})
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}

// IsPreview9 reports whether the request asked for 1.0.forge-preview.9.
func IsPreview9(ctx context.Context) bool {
	v, ok := appctx.GetAPIVersionFromContext(ctx)
	return ok && v.Equal(apiversion.V1_0_Forge_Preview9)
}

// ItemFromProto presents an item. For preview9 an item without a sales section still carries a non-zero stored
// selling price in one, as preview.9 reads it as `unit_value`; the downgrade then drops the section.
func ItemFromProto(i *pb.ItemInfo, preview9 bool) *apiresource.Item {
	item := &apiresource.Item{
		ID:          i.Id,
		Object:      constants.ObjectTypeItem,
		Type:        constants.ItemTypeCode(i.ItemTypeCode),
		Status:      constants.ArchiveStatus(i.Status),
		SKU:         i.Sku,
		Description: i.Description,
		UnitCost:    UnitPriceFromProto(i.UnitCost),
		BurnRate:    RateFromProto(i.BurnRate),
		CreatedAt:   grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(i.UpdatedAt),
	}
	if item.Status == "" {
		item.Status = constants.ArchiveStatusActive
	}
	if i.Supply != nil {
		supply := constants.Supply(*i.Supply)
		item.Supply = &supply
	}
	if s := i.Sales; s != nil {
		item.Sales = &apiresource.ItemSales{
			UnitPrice:        UnitPriceFromProto(s.UnitPrice),
			PortalVisibility: constants.CustomerPortalVisibility(s.PortalVisibility),
			LeadTime:         quantityValueFromProto(s.LeadTime),
		}
	}
	if item.Sales == nil && preview9 && nonZeroPrice(i.UnitValue) {
		item.Sales = &apiresource.ItemSales{
			UnitPrice:        UnitPriceFromProto(i.UnitValue),
			PortalVisibility: constants.CustomerPortalVisibilityHidden,
		}
	}
	if m := i.Material; m != nil {
		item.Material = &apiresource.ItemMaterial{
			OrderPoint: quantityValueFromProto(m.OrderPoint),
			LeadTime:   quantityValueFromProto(m.LeadTime),
		}
	}
	return item
}

// quantityValueFromProto presents a section's quantity, or nil when absent. Its unit loads before the response is written.
func quantityValueFromProto(q *pb.QuantityValue) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	return apiresource.NewQuantity(apiresource.NormalizeQuantityValue(q.Value, q.UnitType), q.UnitId)
}

func nonZeroPrice(p *pb.UnitPrice) bool {
	if p == nil {
		return false
	}
	amount, err := decimal.NewFromString(p.Amount)
	return err == nil && !amount.IsZero()
}
