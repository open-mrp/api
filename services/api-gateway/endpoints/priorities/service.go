package priorityep

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

type PrioritySvc interface {
	ListPriorities(ctx context.Context, req *ListPrioritiesRequest) (*apiresource.List[apiresource.Priority], *apierror.APIError)
	GetPriority(ctx context.Context, req *GetPriorityRequest) (*apiresource.Priority, *apierror.APIError)
}

type PrioritySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type prioritySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var prioritySvcTracer = tracing.GetTracer("api-gateway.endpoints.priorities.service")

func (c *PrioritySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("priority endpoint service: core client is required")
	}
	return nil
}

func NewPrioritySvc(config *PrioritySvcConfig) PrioritySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &prioritySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *prioritySvcImpl) ListPriorities(ctx context.Context, req *ListPrioritiesRequest) (*apiresource.List[apiresource.Priority], *apierror.APIError) {
	pbReq := &pb.ListPrioritiesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, prioritySvcTracer, "service.priorities.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPrioritiesResponse, error) {
			return m.coreClient.ListPriorities(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return PriorityListPresenter(resp), nil
}

func (m *prioritySvcImpl) GetPriority(ctx context.Context, req *GetPriorityRequest) (*apiresource.Priority, *apierror.APIError) {
	pbReq := &pb.GetPriorityRequest{
		Identifier: req.PriorityID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, prioritySvcTracer, "service.priorities.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPriorityResponse, error) {
			return m.coreClient.GetPriority(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PriorityPresenter(resp.Priority)
	return &result, nil
}
