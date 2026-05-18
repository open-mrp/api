package edirunep

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
)

type EDIRunSvc interface {
	ListEDIRuns(ctx context.Context, req *ListEDIRunsRequest) (*apiresource.List[apiresource.EDIRun], *apierror.APIError)
	GetEDIRun(ctx context.Context, req *RetrieveEDIRunRequest) (*apiresource.EDIRun, *apierror.APIError)
}

type EDIRunSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type ediRunSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var ediRunSvcTracer = tracing.GetTracer("api-gateway.endpoints.edi-runs.service")

func (c *EDIRunSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("edi run endpoint service: core client is required")
	}
	return nil
}

func NewEDIRunSvc(config *EDIRunSvcConfig) EDIRunSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &ediRunSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *ediRunSvcImpl) ListEDIRuns(ctx context.Context, req *ListEDIRunsRequest) (*apiresource.List[apiresource.EDIRun], *apierror.APIError) {
	pbReq := &pb.ListEDIRunsRequest{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		HasSucceeded: req.HasSucceeded,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}

	resp, apiErr := grpcutil.CallRPC(ctx, ediRunSvcTracer, "service.edi-runs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListEDIRunsResponse, error) {
			return m.coreClient.ListEDIRuns(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return EDIRunListPresenter(ctx, resp), nil
}

func (m *ediRunSvcImpl) GetEDIRun(ctx context.Context, req *RetrieveEDIRunRequest) (*apiresource.EDIRun, *apierror.APIError) {
	pbReq := &pb.GetEDIRunRequest{
		Id: req.EDIRunID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, ediRunSvcTracer, "service.edi-runs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetEDIRunResponse, error) {
			return m.coreClient.GetEDIRun(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := EDIRunPresenter(resp.EdiRun)
	return &result, nil
}
