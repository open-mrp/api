package materialep

import (
	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	quantityep "github.com/augno/api/services/api-gateway/endpoints/quantities"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

// TODO: this kind of work around where we send back various forms of quantities and do a unit from a quantity should be migrated to use a standard object no matter what. If the user does not request a sub-object, we should not send it. There should be exactly one quantity protobuf that is similar to our apiresource for quantity and this should be true for all objects. That way we only have to define them once and each has one way of being turned into a presenter and the presenters can be reused and drilled down e.g. the unit presenter would be used here with the quantity.
func materialQuantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	// Normalize first so we compare against the canonical form ("0") rather
	// than the raw DB decimal string (e.g. "0.000000000000000000000000000000").
	// A normalized value of "0" means the field was never set by the caller —
	// the service unconditionally inserts a zero-value quantity row as a default.
	normalized := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	if normalized == "0" {
		return nil
	}
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        normalized,
		DisplayValue: apiresource.FormatDisplayValue(normalized, q.UnitAbbreviation, q.UnitType),
		Unit:         quantityep.UnitFromQuantityInfo(q),
	}
}

func MaterialPresenter(m *pb.MaterialInfo) apiresource.Material {
	if m == nil {
		return apiresource.Material{}
	}
	var item *apiresource.Item
	if m.Item != nil {
		i := itemep.ItemPresenter(m.Item)
		item = &i
	}
	return apiresource.Material{
		ID:         m.Id,
		Object:     constants.ObjectTypeMaterial,
		Item:       item,
		OrderPoint: materialQuantityPresenter(m.OrderPoint),
		LeadTime:   materialQuantityPresenter(m.LeadTime),
		CreatedAt:  grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(m.UpdatedAt),
	}
}
