package consumptionep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func quantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	norm := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
		Unit:         nil, // Populated via include expansion
	}
}

func ConsumptionPresenter(c *pb.ConsumptionInfo) apiresource.Consumption {
	if c == nil {
		return apiresource.Consumption{}
	}

	var consumedItem *apiresource.ConsumptionItem
	if c.ItemId != "" {
		consumedItem = &apiresource.ConsumptionItem{
			ID:           c.ItemId,
			Object:       constants.ObjectTypeItem,
			SKU:          c.ItemSku,
			Description:  c.ItemDescription,
			ItemTypeCode: constants.ItemTypeCode(c.ItemTypeCode),
		}
	}

	return apiresource.Consumption{
		ID:            c.Id,
		Object:        constants.ObjectTypeConsumption,
		Quantity:      quantityPresenter(c.Quantity),
		WasteQuantity: quantityPresenter(c.WasteQuantity),
		ConsumedItem:  consumedItem,
		Instructions:  c.Instructions,
		CreatedAt:     grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTime(c.UpdatedAt),
	}
}
