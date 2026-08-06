package materialep

import (
	"context"
	"fmt"

	jobep "github.com/augno/api/services/api-gateway/endpoints/jobs"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
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
	BulkUpsertMaterials(ctx context.Context, req *BulkUpsertMaterialsRequest) (*apiresource.Job, *apierror.APIError)
	UpdateMaterial(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError)
	DeleteMaterial(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError)
	ExportMaterials(ctx context.Context, req *ExportMaterialsRequest) (*apiresource.Job, *apierror.APIError)
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

func (m *materialSvcImpl) BulkUpsertMaterials(ctx context.Context, req *BulkUpsertMaterialsRequest) (*apiresource.Job, *apierror.APIError) {
	pbMaterials := make([]*pb.UpsertMaterialInput, len(req.Materials))
	for i, m := range req.Materials {
		pbProps := make([]*pb.UpsertItemPropertyInput, len(m.Properties))
		for j, pr := range m.Properties {
			pbProps[j] = &pb.UpsertItemPropertyInput{Name: pr.Name, Value: pr.Value}
		}

		pbInput := &pb.UpsertMaterialInput{
			Sku:         m.SKU,
			Description: m.Description.Ptr(),
			Notes:       m.Notes.Ptr(),
			Category:    apirequest.ObjectIdentifierToProto(m.Category),
			UnitPrice:   rateInputToProto(m.UnitPrice.Ptr()),
			UnitCost:    rateInputToProto(m.UnitCost.Ptr()),
			Properties:  pbProps,
		}
		if q, ok := m.OrderPoint.Value(); ok {
			pbInput.OrderPoint = &pb.QuantityInput{Value: q.Value, UnitId: q.UnitID}
		}
		if q, ok := m.LeadTime.Value(); ok {
			pbInput.LeadTime = &pb.QuantityInput{Value: q.Value, UnitId: q.UnitID}
		}
		pbMaterials[i] = pbInput
	}

	pbReq := &pb.BulkUpsertMaterialsRequest{Materials: pbMaterials}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertMaterialsResponse, error) {
			return m.coreClient.BulkUpsertMaterials(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
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
	// Gated build: the expandable item stays nil and populates only when the caller requests it via ?include=item. MaterialPresenter would embed it unconditionally and is reserved for the Excel export path.
	result := resourceloaders.MaterialFromProto(resp.Material)
	resourceloaders.StashMaterialMeta(ctx, resp.Material)
	return result, nil
}

func (m *materialSvcImpl) ExportMaterials(ctx context.Context, req *ExportMaterialsRequest) (*apiresource.Job, *apierror.APIError) {
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

	return jobep.JobFromProto(resp.GetJob()), nil
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
