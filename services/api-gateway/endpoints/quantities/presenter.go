package quantityep

import (
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func QuantityPresenter(q *pb.QuantityInfo) apiresource.Quantity {
	if q == nil {
		return apiresource.Quantity{}
	}

	norm := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	return apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
		Unit: &apiresource.Unit{
			ID:     q.UnitId,
			Object: constants.ObjectTypeUnit,
		},
	}
}
