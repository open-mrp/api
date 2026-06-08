package inventoryep

import (
	"context"
	"strconv"

	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/core"
)

func ListInventoriesPresenter(ctx context.Context, resp *pb.ListInventoriesResponse) *apiresource.ListInventoriesResponse {
	if resp == nil {
		return &apiresource.ListInventoriesResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.InventoryItem{},
			Count:  0,
		}
	}

	meta := resourcekit.GetLoadMeta(ctx)
	items := make([]apiresource.InventoryItem, len(resp.Items))
	for i, item := range resp.Items {
		valueStr := strconv.FormatFloat(item.OnHandQuantity, 'f', -1, 64)
		qid, _ := id.GenID(id.QuantityIDPrefix, nil)

		// unit is an expandable reference on the quantity: stash the FK id so
		// LoadUnits fetches the real Unit on ?include=...unit. Never fabricate.
		if item.OnHandUnitId != "" {
			meta.Set(constants.ObjectTypeQuantity, qid, "unit_id", item.OnHandUnitId)
		}

		items[i] = apiresource.InventoryItem{
			Object: constants.ObjectTypeInventoryItem,
			Item:   itemep.ItemPresenter(item.Item),
			Quantity: &apiresource.Quantity{
				ID:           qid,
				Object:       constants.ObjectTypeQuantity,
				Value:        apiresource.NormalizeQuantityValue(valueStr, item.OnHandUnitType),
				DisplayValue: apiresource.FormatDisplayValue(valueStr, item.OnHandUnitAbbreviation, item.OnHandUnitType),
			},
		}
	}

	return &apiresource.ListInventoriesResponse{
		Object:   constants.ObjectTypeList,
		PageInfo: grpcutil.MapProtoPageInfo(ctx, resp.PageInfo),
		Data:     items,
		Count:    resp.Count,
	}
}
