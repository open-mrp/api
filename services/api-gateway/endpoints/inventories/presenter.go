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

// ListInventoriesUnitIDs names the units the on-hand figures are counted in, so the caller can resolve them before presenting.
func ListInventoriesUnitIDs(resp *pb.ListInventoriesResponse) []string {
	if resp == nil {
		return nil
	}
	ids := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		ids = append(ids, item.OnHandUnitId)
	}
	return ids
}

func ListInventoriesPresenter(ctx context.Context, resp *pb.ListInventoriesResponse, units map[string]*apiresource.Unit) *apiresource.List[apiresource.InventoryItem] {
	if resp == nil {
		return apiresource.NewList[apiresource.InventoryItem](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.InventoryItem, len(resp.Items))
	for i, item := range resp.Items {
		valueStr := strconv.FormatFloat(item.OnHandQuantity, 'f', -1, 64)
		presented := itemep.ItemPresenter(item.Item)

		items[i] = apiresource.InventoryItem{
			Object: constants.ObjectTypeInventoryItem,
			Item:   &presented,
			// On-hand is computed per request, so it is not a stored quantity and carries no id.
			Quantity: &apiresource.ComputedQuantity{
				Object:       constants.ObjectTypeComputedQuantity,
				Value:        apiresource.NormalizeQuantityValue(valueStr, item.OnHandUnitType),
				DisplayValue: apiresource.FormatDisplayValue(valueStr, item.OnHandUnitAbbreviation, item.OnHandUnitType),
				Unit:         units[item.OnHandUnitId],
			},
		}
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
