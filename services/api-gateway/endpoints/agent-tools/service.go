package agenttoolep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"

	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentToolSvc interface {
	ListTools(ctx context.Context, req *ListToolsRequest) (*apiresource.List[apiresource.AvailableTool], *apierror.APIError)
	ListToolGroups(ctx context.Context, req *ListToolGroupsRequest) (*apiresource.List[apiresource.ToolGroup], *apierror.APIError)
}

type AgentToolSvcConfig struct {
	AgentClient pb.AgentServiceClient
}

type agentToolSvcImpl struct {
	agentClient pb.AgentServiceClient
}

var toolSvcTracer = tracing.GetTracer("api-gateway.endpoints.agent_tools.service")

func (c *AgentToolSvcConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent tool endpoint service: agent client is required")
	}
	return nil
}

func NewAgentToolSvc(config *AgentToolSvcConfig) AgentToolSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentToolSvcImpl{
		agentClient: config.AgentClient,
	}
}

func (m *agentToolSvcImpl) ListTools(ctx context.Context, req *ListToolsRequest) (*apiresource.List[apiresource.AvailableTool], *apierror.APIError) {
	pbReq := &pb.ListAvailableToolsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, toolSvcTracer, "service.agent_tools.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAvailableToolsResponse, error) {
			return m.agentClient.ListAvailableTools(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	return AvailableToolListPresenter(resp), nil
}

func (m *agentToolSvcImpl) ListToolGroups(ctx context.Context, req *ListToolGroupsRequest) (*apiresource.List[apiresource.ToolGroup], *apierror.APIError) {
	pbReq := &pb.ListAvailableToolsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, toolSvcTracer, "service.agent_tools.list_groups", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAvailableToolsResponse, error) {
			return m.agentClient.ListAvailableTools(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	includes := appctx.GetRequestedIncludeKeys(ctx)

	return ToolGroupListPresenter(resp, includes), nil
}
