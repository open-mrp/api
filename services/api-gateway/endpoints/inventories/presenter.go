package inventoryep

import (
	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ListInventoriesPresenter(resp *pb.ListInventoriesResponse) *apiresource.ListInventoriesResponse {
	if resp == nil {
		return &apiresource.ListInventoriesResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.InventoryItem{},
			Count:  0,
		}
	}

	items := make([]apiresource.InventoryItem, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = apiresource.InventoryItem{
			Item: itemep.ItemPresenter(item.Item),
			Quantity: apiresource.BaseQuantity{
				Measure: item.OnHandQuantity,
				Unit: apiresource.BaseQuantityUnit{
					Name:         item.OnHandUnitAbbreviation,
					Abbreviation: item.OnHandUnitAbbreviation,
					Type:         item.OnHandUnitType,
				},
			},
		}
	}

	return &apiresource.ListInventoriesResponse{
		Object:   constants.ObjectTypeList,
		PageInfo: grpcutil.MapProtoPageInfo(resp.PageInfo),
		Data:     items,
		Count:    resp.Count,
	}
}
