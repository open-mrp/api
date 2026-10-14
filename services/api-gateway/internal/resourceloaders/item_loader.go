package resourceloaders

import (
	"context"
	"github.com/open-mrp/apikit/include"
	"strings"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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
		out[item.Id] = itemFromProto(item)

		if item.Category != nil {
			if c, ok := categoryMap[item.Category.Id]; ok {
				meta.Set(constants.ObjectTypeItem, item.Id, "category", c)
			}
		}

		if item.UnitValue != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "unit_value", UnitPriceFromProto(item.UnitValue))
		}
		if item.UnitCost != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "unit_cost", UnitPriceFromProto(item.UnitCost))
		}
		if item.BurnRate != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "burn_rate", RateFromProto(item.BurnRate))
		}

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

func itemFromProto(i *pb.ItemInfo) *apiresource.Item {
	return &apiresource.Item{
		ID:           i.Id,
		Object:       constants.ObjectTypeItem,
		SKU:          i.Sku,
		Description:  i.Description,
		Notes:        i.Notes,
		ItemTypeCode: constants.ItemTypeCode(i.ItemTypeCode),
		CreatedAt:    grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(i.UpdatedAt),
	}
}

