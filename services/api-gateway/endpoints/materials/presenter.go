package materialep

import (
	itemep "github.com/open-mrp/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func materialQuantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	// A zero value means the field was never set: the service inserts a zero quantity row as a default.
	normalized := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	if normalized == "0" {
		return nil
	}
	return apiresource.NewQuantity(normalized, q.UnitId)
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
