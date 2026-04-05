package salestargetep

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type SalesTargetSvc interface {
	ListSalesTargets(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError)
	CreateSalesTarget(ctx context.Context, req *CreateSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError)
	UpsertSalesTarget(ctx context.Context, req *UpsertSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError)
}

type SalesTargetSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type salesTargetSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var salesTargetSvcTracer = tracing.GetTracer("api-gateway.endpoints.sales_targets.service")

func (c *SalesTargetSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales target endpoint service: core client is required")
	}
	return nil
}

func NewSalesTargetSvc(config *SalesTargetSvcConfig) SalesTargetSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &salesTargetSvcImpl{coreClient: config.CoreClient}
}

func (m *salesTargetSvcImpl) ListSalesTargets(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError) {
	if req.Cursor != nil && *req.Cursor != "" {
		return nil, apierror.NewValidationError("Invalid pagination cursor.")
	}

	pbReq := &pb.ListSalesTargetsRequest{
		SalesRepId: req.SalesRepID,
		Limit:      req.Limit,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesTargetSvcTracer, "service.sales_targets.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesTargetsResponse, error) {
			return m.coreClient.ListSalesTargets(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return SalesTargetListPresenter(resp), nil
}

func (m *salesTargetSvcImpl) CreateSalesTarget(ctx context.Context, req *CreateSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
	pbReq := &pb.CreateSalesTargetRequest{
		SalesRepId:   req.SalesRepID,
		StartDate:    req.StartDate.Format(time.RFC3339),
		EndDate:      req.EndDate.Format(time.RFC3339),
		AmountValue:  req.AmountValue,
		AmountUnitId: req.AmountUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesTargetSvcTracer, "service.sales_targets.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesTargetResponse, error) {
			return m.coreClient.CreateSalesTarget(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesTargetPresenter(resp.SalesTarget)
	return &result, nil
}

func (m *salesTargetSvcImpl) UpsertSalesTarget(ctx context.Context, req *UpsertSalesTargetRequest) (*apiresource.SalesTarget, *apierror.APIError) {
	pbReq := &pb.UpsertSalesTargetRequest{
		TargetId:     req.TargetID,
		SalesRepId:   req.SalesRepID,
		StartDate:    req.StartDate.Format(time.RFC3339),
		EndDate:      req.EndDate.Format(time.RFC3339),
		AmountValue:  req.AmountValue,
		AmountUnitId: req.AmountUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesTargetSvcTracer, "service.sales_targets.upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertSalesTargetResponse, error) {
			return m.coreClient.UpsertSalesTarget(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesTargetPresenter(resp.SalesTarget)
	return &result, nil
}
