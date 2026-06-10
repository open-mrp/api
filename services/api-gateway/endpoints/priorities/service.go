package priorityep

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

type PrioritySvc interface {
	ListPriorities(ctx context.Context, req *ListPrioritiesRequest) (*apiresource.List[apiresource.Priority], *apierror.APIError)
	GetPriority(ctx context.Context, req *RetrievePriorityRequest) (*apiresource.Priority, *apierror.APIError)
}

type PrioritySvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
	ids := make([]string, len(resp.Priorities))
	for i, p := range resp.Priorities {
		ids[i] = p.Id
	}
	loaded, apiErr := resourceloaders.LoadPriorities(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	priorities := make([]apiresource.Priority, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			priorities = append(priorities, *(v.(*apiresource.Priority)))
		}
	}
	return apiresource.NewList(priorities, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

// GetPriority resolves a Priority by id-or-code. The legacy GetPriority gRPC
// accepts either form so we call it for the identifier resolution, then pipe
// the resolved ID through LoadPriorities so the V2 resolver gets a clean
// resource with LoadMeta — even though Priority has no FK info today, this
// keeps every endpoint going through the same loader path.
func (m *prioritySvcImpl) GetPriority(ctx context.Context, req *RetrievePriorityRequest) (*apiresource.Priority, *apierror.APIError) {
	pbReq := &pb.GetPriorityRequest{Identifier: req.PriorityID}
	resp, apiErr := grpcutil.CallRPC(ctx, prioritySvcTracer, "service.priorities.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPriorityResponse, error) {
			return m.coreClient.GetPriority(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	loaded, apiErr := resourceloaders.LoadPriorities(ctx, []string{resp.Priority.Id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[resp.Priority.Id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Priority not found.")
	}
	return v.(*apiresource.Priority), nil
}
