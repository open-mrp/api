package inventoryep

import (
	"context"
	"strconv"

	itemep "github.com/open-mrp/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func ListInventoriesPresenter(ctx context.Context, resp *pb.ListInventoriesResponse) *apiresource.List[apiresource.InventoryItem] {
	if resp == nil {
		return apiresource.NewList[apiresource.InventoryItem](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.InventoryItem, len(resp.Items))
	for i, item := range resp.Items {
		valueStr := strconv.FormatFloat(item.OnHandQuantity, 'f', -1, 64)

		items[i] = apiresource.InventoryItem{
			Object: constants.ObjectTypeInventoryItem,
			Item:   itemPtr(itemep.ItemPresenter(item.Item)),
			// On-hand is computed per request, so it is not a stored quantity and carries no id.
			Quantity: &apiresource.ComputedQuantity{
				Object:       constants.ObjectTypeComputedQuantity,
				Value:        apiresource.NormalizeQuantityValue(valueStr, item.OnHandUnitType),
				DisplayValue: apiresource.FormatDisplayValue(valueStr, item.OnHandUnitAbbreviation, item.OnHandUnitType),
			},
		}
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

// itemPtr adapts the shared item presenter, which returns a value, to the pointer the resource holds.
func itemPtr(item apiresource.Item) *apiresource.Item {
	return &item
}
