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
	return &apiresource.QuantityInfo{
		Value: apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
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
	// The public identifier for a Material is its item_id: every URL handler
	// (Get/Update/Delete) maps the `{id}` path parameter to the item id when
	// calling the core service. Returning the item_id here keeps list and
	// single-GET responses consistent and routable.
	return apiresource.Material{
		ID:         m.ItemId,
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
		// MaterialPresenter already normalizes Material.ID to item_id, which is
		// the identifier the supplier-material GET handler expects as `{id}`.
		itemID = m.ID
	}
	return apiresource.SupplierMaterial{
		ID:                  itemID,
		Object:              constants.ObjectTypeSupplierMaterial,
		Material:            material,
		SupplierPartNumber:  sm.SupplierPartNumber,
		SupplierDescription: sm.SupplierDescription,
		IsActive:            sm.IsActive,
		CreatedAt:           grpcutil.TimestampToTime(sm.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(sm.UpdatedAt),
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
