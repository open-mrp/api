package suppliermaterialep

import (
	"context"
	"fmt"

	materialep "github.com/augno/api/services/api-gateway/endpoints/materials"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type SupplierMaterialSvc interface {
	ListSupplierMaterials(ctx context.Context, req *ListSupplierMaterialsRequest) (*apiresource.List[apiresource.SupplierMaterial], *apierror.APIError)
	GetSupplierMaterial(ctx context.Context, req *RetrieveSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError)
	CreateSupplierMaterial(ctx context.Context, req *CreateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError)
	UpdateSupplierMaterial(ctx context.Context, req *UpdateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError)
	DeleteSupplierMaterial(ctx context.Context, req *DeleteSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError)
}

type SupplierMaterialSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type supplierMaterialSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var supplierMaterialSvcTracer = tracing.GetTracer("api-gateway.endpoints.supplier-materials.service")

func (c *SupplierMaterialSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("supplier material endpoint service: core client is required")
	}
	return nil
}

func NewSupplierMaterialSvc(config *SupplierMaterialSvcConfig) SupplierMaterialSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &supplierMaterialSvcImpl{coreClient: config.CoreClient}
}

func (s *supplierMaterialSvcImpl) ListSupplierMaterials(ctx context.Context, req *ListSupplierMaterialsRequest) (*apiresource.List[apiresource.SupplierMaterial], *apierror.APIError) {
	pbReq := &pb.ListSupplierMaterialsRequest{
		SupplierAccountId: req.SupplierID,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierMaterialSvcTracer, "service.supplier_materials.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSupplierMaterialsResponse, error) {
			return s.coreClient.ListSupplierMaterials(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return supplierMaterialListFromProto(ctx, resp), nil
}

func (s *supplierMaterialSvcImpl) GetSupplierMaterial(ctx context.Context, req *RetrieveSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
	pbReq := &pb.GetSupplierMaterialRequest{
		SupplierAccountId: req.SupplierID,
		MaterialId:        req.MaterialID,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierMaterialSvcTracer, "service.supplier_materials.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSupplierMaterialResponse, error) {
			return s.coreClient.GetSupplierMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := supplierMaterialFromProto(resp.SupplierMaterial)
	stashSupplierMaterialMeta(ctx, resp.SupplierMaterial, &result)
	return &result, nil
}

func (s *supplierMaterialSvcImpl) CreateSupplierMaterial(ctx context.Context, req *CreateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
	isActive := true
	if v, ok := req.IsActive.Value(); ok {
		isActive = v
	}

	pbReq := &pb.CreateSupplierMaterialRequest{
		SupplierAccountId:   req.SupplierID,
		MaterialId:          req.MaterialID,
		SupplierPartNumber:  req.SupplierPartNumber,
		SupplierDescription: req.SupplierDescription.Ptr(),
		IsActive:            isActive,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierMaterialSvcTracer, "service.supplier_materials.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSupplierMaterialResponse, error) {
			return s.coreClient.CreateSupplierMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := supplierMaterialFromProto(resp.SupplierMaterial)
	stashSupplierMaterialMeta(ctx, resp.SupplierMaterial, &result)
	return &result, nil
}

func (s *supplierMaterialSvcImpl) UpdateSupplierMaterial(ctx context.Context, req *UpdateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
	pbReq := &pb.UpdateSupplierMaterialRequest{
		SupplierAccountId:   req.SupplierID,
		MaterialId:          req.MaterialID,
		SupplierPartNumber:  req.SupplierPartNumber.Ptr(),
		SupplierDescription: req.SupplierDescription.Ptr(),
		UpdateDescription:   req.SupplierDescription.Ptr() != nil,
		IsActive:            req.IsActive.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierMaterialSvcTracer, "service.supplier_materials.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSupplierMaterialResponse, error) {
			return s.coreClient.UpdateSupplierMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := supplierMaterialFromProto(resp.SupplierMaterial)
	stashSupplierMaterialMeta(ctx, resp.SupplierMaterial, &result)
	return &result, nil
}

func (s *supplierMaterialSvcImpl) DeleteSupplierMaterial(ctx context.Context, req *DeleteSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
	pbReq := &pb.DeleteSupplierMaterialRequest{
		SupplierAccountId: req.SupplierID,
		MaterialId:        req.MaterialID,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierMaterialSvcTracer, "service.supplier_materials.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteSupplierMaterialResponse, error) {
			return s.coreClient.DeleteSupplierMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := supplierMaterialFromProto(resp.SupplierMaterial)
	stashSupplierMaterialMeta(ctx, resp.SupplierMaterial, &result)
	return &result, nil
}

func supplierMaterialFromProto(sm *pb.SupplierMaterialInfo) apiresource.SupplierMaterial {
	if sm == nil {
		return apiresource.SupplierMaterial{}
	}
	itemID := ""
	if sm.Material != nil {
		itemID = sm.Material.Id
	}
	return apiresource.SupplierMaterial{
		ID:                  itemID,
		Object:              constants.ObjectTypeSupplierMaterial,
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

func stashSupplierMaterialMeta(ctx context.Context, sm *pb.SupplierMaterialInfo, d *apiresource.SupplierMaterial) {
	if sm == nil || sm.Material == nil {
		return
	}
	m := materialep.MaterialPresenter(sm.Material)
	m.Item = nil
	meta := resourcekit.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeSupplierMaterial, d.ID, "material", &m)
	if sm.Material.ItemId != "" {
		meta.Set(constants.ObjectTypeMaterial, m.ID, "item_id", sm.Material.ItemId)
	}
}

func supplierMaterialListFromProto(ctx context.Context, resp *pb.ListSupplierMaterialsResponse) *apiresource.List[apiresource.SupplierMaterial] {
	if resp == nil {
		return apiresource.NewList[apiresource.SupplierMaterial](nil, apiresource.PageInfo{})
	}
	items := make([]apiresource.SupplierMaterial, len(resp.SupplierMaterials))
	for i, sm := range resp.SupplierMaterials {
		items[i] = supplierMaterialFromProto(sm)
		stashSupplierMaterialMeta(ctx, sm, &items[i])
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
