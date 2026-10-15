package resourceloaders

import (
	"context"
	"net/url"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var itemCategoryLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.item_category")

func LoadItemCategories(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, itemCategoryLoaderTracer, "loader.item_categories.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetItemCategoriesByIDsResponse, error) {
			return coreClient.BatchGetItemCategoriesByIDs(ctx, &pb.BatchGetItemCategoriesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return ItemCategoriesFromProto(ctx, resp.ItemCategories), nil
}

// ItemCategoriesFromProto builds categories keyed by id and stashes what their expandable fields resolve from.
func ItemCategoriesFromProto(ctx context.Context, categories []*pb.ItemCategoryInfo) map[string]any {
	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(categories))
	for _, ic := range categories {
		out[ic.Id] = itemCategoryFromProto(ic)

		var accountID string
		if ic.AccountId != nil {
			accountID = *ic.AccountId
		}
		meta.Set(constants.ObjectTypeItemCategory, ic.Id, "owner_account_id", accountID)

		props := make([]apiresource.Property, len(ic.Properties))
		for i, p := range ic.Properties {
			props[i] = apiresource.Property{
				ID:        p.Id,
				Object:    constants.ObjectTypeProperty,
				Name:      p.Name,
				CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
				UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
			}
		}
		meta.Set(constants.ObjectTypeItemCategory, ic.Id, "properties_list",
			apiresource.NewEmbeddedList(props, false, CategoryPropertiesListPath(ic.Id), nil))

		if ic.UnitGroup != nil {
			ug := buildUnitGroupFromProto(ic.UnitGroup)
			meta.Set(constants.ObjectTypeItemCategory, ic.Id, "unit_group", ug)
			stashUnitGroupMeta(meta, ug)
		}
	}
	return out
}

func itemCategoryFromProto(ic *pb.ItemCategoryInfo) *apiresource.ItemCategory {
	out := &apiresource.ItemCategory{
		ID:        ic.Id,
		Object:    constants.ObjectTypeItemCategory,
		Name:      ic.Name,
		Type:      constants.ItemCategoryTypeFromCode(ic.ItemCategoryTypeCode),
		CreatedAt: grpcutil.TimestampToTime(ic.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ic.UpdatedAt),
	}
	if ic.DefaultLotSizeValue != nil && ic.GetDefaultLotSizeUnitId() != "" {
		out.DefaultLotSize = apiresource.NewQuantity(apiresource.NormalizeQuantityValue(*ic.DefaultLotSizeValue, ""), *ic.DefaultLotSizeUnitId)
	}
	return out
}

// CategoryPropertiesListPath is the properties list narrowed to one category, where a category's embedded
// properties continue.
func CategoryPropertiesListPath(itemCategoryID string) string {
	return "/v1/catalog/properties?" + url.Values{"item_category_ids[]": {itemCategoryID}}.Encode()
}
