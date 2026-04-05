package partep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PartSvc interface {
	ListParts(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError)
	GetPart(ctx context.Context, req *GetPartRequest) (*apiresource.Part, *apierror.APIError)
	CreatePart(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError)
	UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError)
	DeletePart(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError)
}

type PartSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type partSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var partSvcTracer = tracing.GetTracer("api-gateway.endpoints.parts.service")

func (c *PartSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("part endpoint service: core client is required")
	}
	return nil
}

func NewPartSvc(config *PartSvcConfig) PartSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &partSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *partSvcImpl) ListParts(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError) {
	pbReq := &pb.ListPartsRequest{
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

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPartsResponse, error) {
			return m.coreClient.ListParts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return PartListPresenter(resp), nil
}

func (m *partSvcImpl) GetPart(ctx context.Context, req *GetPartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.GetPartRequest{
		ItemId: req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPartResponse, error) {
			return m.coreClient.GetPart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}

func (m *partSvcImpl) CreatePart(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.CreatePartRequest{
		Sku:         req.SKU,
		Description: req.Description,
		CategoryId:  req.CategoryID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePartResponse, error) {
			return m.coreClient.CreatePart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}

func (m *partSvcImpl) UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.UpdatePartRequest{
		ItemId: req.ItemID,
		Sku:    req.SKU,
	}

	if req.Description != nil {
		pbReq.UpdateDescription = true
		pbReq.Description = req.Description
	}

	if req.Notes != nil {
		pbReq.UpdateNotes = true
		pbReq.Notes = req.Notes
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePartResponse, error) {
			return m.coreClient.UpdatePart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}

func (m *partSvcImpl) DeletePart(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.DeletePartRequest{
		ItemId: req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeletePartResponse, error) {
			return m.coreClient.DeletePart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}
