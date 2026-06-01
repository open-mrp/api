package edirunep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
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
	return &ediRunSvcImpl{coreClient: config.CoreClient}
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
	ids := make([]string, len(resp.EdiRuns))
	for i, r := range resp.EdiRuns {
		ids[i] = r.Id
	}
	loaded, apiErr := resourceloaders.LoadEDIRuns(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.EDIRun, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.EDIRun)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *ediRunSvcImpl) GetEDIRun(ctx context.Context, req *RetrieveEDIRunRequest) (*apiresource.EDIRun, *apierror.APIError) {
	return loadEDIRunByID(ctx, req.EDIRunID)
}

func loadEDIRunByID(ctx context.Context, id string) (*apiresource.EDIRun, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadEDIRuns(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("EDI run not found.")
	}
	return v.(*apiresource.EDIRun), nil
}
