package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
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

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ItemCategories))
	for _, ic := range resp.ItemCategories {
		out[ic.Id] = itemCategoryFromProto(ic)

		var accountID string
		if ic.AccountId != nil {
			accountID = *ic.AccountId
		}
		meta.Set(constants.ObjectTypeItemCategory, ic.Id, "owner_account_id", accountID)

		if len(ic.Properties) > 0 {
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
				apiresource.NewList(props, apiresource.PageInfo{}))
		}

		if ic.UnitGroup != nil {
			ug := buildUnitGroupFromProto(ic.UnitGroup)
			meta.Set(constants.ObjectTypeItemCategory, ic.Id, "unit_group", ug)
			stashUnitGroupMeta(meta, ug)
		}
	}
	return out, nil
}

func itemCategoryFromProto(ic *pb.ItemCategoryInfo) *apiresource.ItemCategory {
	return &apiresource.ItemCategory{
		ID:        ic.Id,
		Object:    constants.ObjectTypeItemCategory,
		Name:      ic.Name,
		Notes:     ic.Notes,
		Type:      constants.ItemCategoryType(ic.ItemCategoryTypeCode),
		CreatedAt: grpcutil.TimestampToTime(ic.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ic.UpdatedAt),
	}
}
