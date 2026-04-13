package itemcategoryep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ItemCategoryPresenter(ic *pb.ItemCategoryInfo, ownerAccount *apiresource.Account) apiresource.ItemCategory {
	if ic == nil {
		return apiresource.ItemCategory{}
	}

	result := apiresource.ItemCategory{
		ID:        ic.Id,
		Object:    constants.ObjectTypeItemCategory,
		Name:      ic.Name,
		Notes:     ic.Notes,
		Type:      constants.ItemCategoryType(ic.ItemCategoryTypeCode),
		Owner:     apiresource.NewOwnerWithAccount(ic.AccountId, ownerAccount),
		CreatedAt: grpcutil.TimestampToTime(ic.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ic.UpdatedAt),
	}

	if len(ic.Properties) > 0 {
		props := make([]apiresource.Property, len(ic.Properties))
		for i, p := range ic.Properties {
			props[i] = apiresource.Property{
				ID:     p.Id,
				Object: constants.ObjectTypeProperty,
				Name:   p.Name,
			}
		}
		result.Properties = apiresource.NewList(props, apiresource.PageInfo{})
	}

	if ic.UnitGroup != nil {
		result.UnitGroup = &apiresource.UnitGroup{
			ID:     ic.UnitGroup.Id,
			Object: constants.ObjectTypeUnitGroup,
			Name:   ic.UnitGroup.Name,
			Type:   constants.UnitType(ic.UnitGroup.Type),
		}
	}

	return result
}

func ItemCategoryListPresenter(resp *pb.ListItemCategoriesResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.ItemCategory] {
	if resp == nil {
		return apiresource.NewList[apiresource.ItemCategory](nil, apiresource.PageInfo{})
	}

	categories := make([]apiresource.ItemCategory, len(resp.ItemCategories))
	for i, ic := range resp.ItemCategories {
		categories[i] = ItemCategoryPresenter(ic, ownerAccount)
	}

	return apiresource.NewList(categories, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
