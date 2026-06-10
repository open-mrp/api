package agenttoolep

import (
	"context"
	"fmt"

	"encoding/json"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
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
	// AgentClient (required) is the agent-service gRPC client.
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

	tools := make([]apiresource.AvailableTool, len(resp.Tools))
	for i, t := range resp.Tools {
		tools[i] = availableToolFromProto(t)
	}

	return apiresource.NewList(tools, apiresource.PageInfo{}), nil
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

	meta := resourcekit.GetLoadMeta(ctx)

	toolsByGroup := make(map[string][]apiresource.AvailableTool, len(resp.Groups))
	for _, t := range resp.Tools {
		toolsByGroup[t.GroupId] = append(toolsByGroup[t.GroupId], availableToolFromProto(t))
	}

	groups := make([]apiresource.ToolGroup, len(resp.Groups))
	for i, g := range resp.Groups {
		groups[i] = toolGroupFromProto(g)
		tools := toolsByGroup[g.Id]
		if tools == nil {
			tools = []apiresource.AvailableTool{}
		}
		meta.Set(constants.ObjectTypeToolGroup, g.Id, "tools", apiresource.NewList(tools, apiresource.PageInfo{}))
	}

	return apiresource.NewList(groups, apiresource.PageInfo{}), nil
}

func availableToolFromProto(t *pb.AvailableToolInfo) apiresource.AvailableTool {
	if t == nil {
		return apiresource.AvailableTool{}
	}

	perms := t.RequiredPermissions
	if perms == nil {
		perms = []string{}
	}

	return apiresource.AvailableTool{
		ID:                  t.Id,
		Object:              constants.ObjectTypeAvailableTool,
		Name:                t.DisplayName,
		Description:         &t.Description,
		ConfigSchema:        json.RawMessage(t.ConfigSchemaJson),
		Category:            t.Category,
		RequiredPermissions: perms,
	}
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toolGroupFromProto(g *pb.ToolGroupInfo) apiresource.ToolGroup {
	if g == nil {
		return apiresource.ToolGroup{}
	}

	return apiresource.ToolGroup{
		ID:          g.Id,
		Object:      constants.ObjectTypeToolGroup,
		Name:        g.Name,
		Description: stringPtrOrNil(g.Description),
		Slug:        g.Slug,
		Icon:        g.Icon,
		SortOrder:   g.SortOrder,
	}
}
