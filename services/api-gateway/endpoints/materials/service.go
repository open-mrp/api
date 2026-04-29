package materialep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
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
	GetMaterial(ctx context.Context, req *GetMaterialRequest) (*apiresource.Material, *apierror.APIError)
	CreateMaterial(ctx context.Context, req *CreateMaterialRequest) (*apiresource.Material, *apierror.APIError)
	UpdateMaterial(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError)
	DeleteMaterial(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError)
}

type MaterialSvcConfig struct {
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
	return MaterialListPresenter(resp), nil
}

func (m *materialSvcImpl) GetMaterial(ctx context.Context, req *GetMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	pbReq := &pb.GetMaterialRequest{Id: req.ItemID}
	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetMaterialResponse, error) {
			return m.coreClient.GetMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := MaterialPresenter(resp.Material)
	return &result, nil
}

func (m *materialSvcImpl) CreateMaterial(ctx context.Context, req *CreateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	pbReq := &pb.CreateMaterialRequest{
		Sku:          req.SKU,
		Description:  req.Description,
		Notes:        req.Notes,
		CategoryId:   req.CategoryID,
		UnitPrice:    rateInputToProto(req.UnitPrice),
		UnitCost:     rateInputToProto(req.UnitCost),
		BurnRate:     rateInputToProto(req.BurnRate),
		AttributeIds: req.AttributeIDs,
	}
	if req.OrderPoint != nil {
		pbReq.OrderPoint = &pb.QuantityInput{Value: req.OrderPoint.Value, UnitId: req.OrderPoint.UnitID}
	}
	if req.LeadTime != nil {
		pbReq.LeadTime = &pb.QuantityInput{Value: req.LeadTime.Value, UnitId: req.LeadTime.UnitID}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateMaterialResponse, error) {
			return m.coreClient.CreateMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := MaterialPresenter(resp.Material)
	return &result, nil
}

func (m *materialSvcImpl) UpdateMaterial(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
	pbReq := &pb.UpdateMaterialRequest{
		Id:                req.ItemID,
		Sku:               req.SKU,
		Description:       req.Description,
		UpdateDescription: req.Description != nil,
		Notes:             req.Notes,
		UpdateNotes:       req.Notes != nil,
	}
	if req.OrderPoint != nil {
		pbReq.OrderPoint = &pb.QuantityInput{Value: req.OrderPoint.Value, UnitId: req.OrderPoint.UnitID}
	}
	if req.LeadTime != nil {
		pbReq.LeadTime = &pb.QuantityInput{Value: req.LeadTime.Value, UnitId: req.LeadTime.UnitID}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, materialSvcTracer, "service.materials.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateMaterialResponse, error) {
			return m.coreClient.UpdateMaterial(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	result := MaterialPresenter(resp.Material)
	return &result, nil
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
