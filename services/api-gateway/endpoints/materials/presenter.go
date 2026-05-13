package materialep

import (
	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func materialQuantityInfoPresenter(q *pb.QuantityInfo) *apiresource.QuantityInfo {
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
	return &apiresource.QuantityInfo{
		Value: normalized,
		Unit: &apiresource.Unit{
			ID:     q.UnitId,
			Object: constants.ObjectTypeUnit,
		},
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
		OrderPoint: materialQuantityInfoPresenter(m.OrderPoint),
		LeadTime:   materialQuantityInfoPresenter(m.LeadTime),
		CreatedAt:  grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(m.UpdatedAt),
	}
}

func MaterialListPresenter(resp *pb.ListMaterialsResponse) *apiresource.List[apiresource.Material] {
	if resp == nil {
		return apiresource.NewList[apiresource.Material](nil, apiresource.PageInfo{})
	}
	materials := make([]apiresource.Material, len(resp.Materials))
	for i, m := range resp.Materials {
		materials[i] = MaterialPresenter(m)
	}
	return apiresource.NewList(materials, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func SupplierMaterialPresenter(sm *pb.SupplierMaterialInfo) apiresource.SupplierMaterial {
	if sm == nil {
		return apiresource.SupplierMaterial{}
	}
	var material *apiresource.Material
	itemID := ""
	if sm.Material != nil {
		m := MaterialPresenter(sm.Material)
		material = &m
		itemID = m.ID
	}
	return apiresource.SupplierMaterial{
		ID:                  itemID,
		Object:              constants.ObjectTypeSupplierMaterial,
		Material:            material,
		SupplierPartNumber:  sm.SupplierPartNumber,
		SupplierDescription: sm.SupplierDescription,
		Status: func() constants.SupplierMaterialStatus {
			if sm.IsActive {
				return constants.SupplierMaterialStatusActive
			}
			return constants.SupplierMaterialStatusInactive
		}(),
		CreatedAt: grpcutil.TimestampToTime(sm.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sm.UpdatedAt),
	}
}

func SupplierMaterialListPresenter(resp *pb.ListSupplierMaterialsResponse) *apiresource.List[apiresource.SupplierMaterial] {
	if resp == nil {
		return apiresource.NewList[apiresource.SupplierMaterial](nil, apiresource.PageInfo{})
	}
	items := make([]apiresource.SupplierMaterial, len(resp.SupplierMaterials))
	for i, sm := range resp.SupplierMaterials {
		items[i] = SupplierMaterialPresenter(sm)
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
