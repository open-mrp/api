package materialep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/export"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func rateInputToProto(r *apirequest.RateInput) *pb.CreateRateInput {
	if r == nil {
		return nil
	}
	return &pb.CreateRateInput{
		Value:             r.Value,
		NumeratorUnitId:   r.NumeratorUnitID,
		DenominatorUnitId: r.DenominatorUnitID,
	}
}

type MaterialSvc interface {
	ListMaterials(ctx context.Context, req *ListMaterialsRequest) (*apiresource.List[apiresource.Material], *apierror.APIError)
	GetMaterial(ctx context.Context, req *RetrieveMaterialRequest) (*apiresource.Material, *apierror.APIError)
	CreateMaterial(ctx context.Context, req *CreateMaterialRequest) (*apiresource.Material, *apierror.APIError)
	UpdateMaterial(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError)
	DeleteMaterial(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError)
	ExportMaterials(ctx context.Context, req *ExportMaterialsRequest) (*httptransport.FileDownload, *apierror.APIError)
}

type MaterialSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type materialSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var materialSvcTracer = tracing.GetTracer("api-gateway.endpoints.materials.service")

func (c *MaterialSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("material endpoint service: core client is required")
	}
	return nil
}

func NewMaterialSvc(config *MaterialSvcConfig) MaterialSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &materialSvcImpl{coreClient: config.CoreClient}
}

func (m *materialSvcImpl) ListMaterials(ctx context.Context, req *ListMaterialsRequest) (*apiresource.List[apiresource.Material], *apierror.APIError) {
	pbReq := &pb.ListMaterialsRequest{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		CategoryIds:  req.CategoryIDs,
		AttributeIds: req.AttributeIDs,
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMaterialsResponse, error) {
			return m.coreClient.ListMaterials(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Materials))
	for i, mat := range resp.Materials {
		ids[i] = mat.Id
	}
	loaded, apiErr := resourceloaders.LoadMaterials(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	materials := make([]apiresource.Material, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			materials = append(materials, *(v.(*apiresource.Material)))
		}
	}
	return apiresource.NewList(materials, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *materialSvcImpl) GetMaterial(ctx context.Context, req *RetrieveMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	return loadMaterialByID(ctx, req.ItemID)
}

func (m *materialSvcImpl) CreateMaterial(ctx context.Context, req *CreateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	pbReq := &pb.CreateMaterialRequest{
		Sku:          req.SKU,
		Description:  req.Description.Ptr(),
		Notes:        req.Notes.Ptr(),
		CategoryId:   req.CategoryID,
		UnitPrice:    rateInputToProto(req.UnitPrice.Ptr()),
		UnitCost:     rateInputToProto(req.UnitCost.Ptr()),
		AttributeIds: req.AttributeIDs,
	}
	if q, ok := req.OrderPoint.Value(); ok {
		pbReq.OrderPoint = &pb.QuantityInput{Value: q.Value, UnitId: q.UnitID}
	}
	if q, ok := req.LeadTime.Value(); ok {
		pbReq.LeadTime = &pb.QuantityInput{Value: q.Value, UnitId: q.UnitID}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateMaterialResponse, error) {
			return m.coreClient.CreateMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadMaterialByID(ctx, resp.Material.Id)
}

func (m *materialSvcImpl) UpdateMaterial(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	pbReq := &pb.UpdateMaterialRequest{
		Id:                req.ItemID,
		Sku:               req.SKU.Ptr(),
		Description:       req.Description.Ptr(),
		UpdateDescription: req.Description.IsSet(),
		Notes:             req.Notes.Ptr(),
		UpdateNotes:       req.Notes.IsSet(),
		UnitCost:          rateInputToProto(req.UnitCost.Ptr()),
	}
	if q, ok := req.OrderPoint.Value(); ok {
		pbReq.OrderPoint = &pb.QuantityInput{Value: q.Value, UnitId: q.UnitID}
	}
	if q, ok := req.LeadTime.Value(); ok {
		pbReq.LeadTime = &pb.QuantityInput{Value: q.Value, UnitId: q.UnitID}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateMaterialResponse, error) {
			return m.coreClient.UpdateMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadMaterialByID(ctx, resp.Material.Id)
}

func (m *materialSvcImpl) DeleteMaterial(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	pbReq := &pb.DeleteMaterialRequest{Id: req.ItemID}
	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteMaterialResponse, error) {
			return m.coreClient.DeleteMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := MaterialPresenter(resp.Material)
	return &result, nil
}

func (m *materialSvcImpl) ExportMaterials(ctx context.Context, req *ExportMaterialsRequest) (*httptransport.FileDownload, *apierror.APIError) {
	pbReq := &pb.ExportMaterialsRequest{
		Query:        req.Query,
		CategoryIds:  req.CategoryIDs,
		AttributeIds: req.AttributeIDs,
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportMaterialsResponse, error) {
			return m.coreClient.ExportMaterials(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	materials := make([]apiresource.Material, len(resp.Materials))
	for i, mat := range resp.Materials {
		materials[i] = MaterialPresenter(mat)
	}

	body, err := export.MaterialsToExcel(materials)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    "materials.xlsx",
		Body:        body,
	}, nil
}

func loadMaterialByID(ctx context.Context, id string) (*apiresource.Material, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadMaterials(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Material not found.")
	}
	return v.(*apiresource.Material), nil
}
