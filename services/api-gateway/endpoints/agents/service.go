package agentep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	corepb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentSvc interface {
	CreateAgent(ctx context.Context, req *CreateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	ListAgents(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError)
	GetAgent(ctx context.Context, req *GetAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdateAgentStatus(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	ListUsage(ctx context.Context, req *ListUsageRequest) (*apiresource.List[apiresource.AgentTokenUsage], *apierror.APIError)
}

type AgentSvcConfig struct {
	AgentClient pb.AgentServiceClient
	CoreClient  corepb.CoreServiceClient
}

type agentSvcImpl struct {
	agentClient pb.AgentServiceClient
	coreClient  corepb.CoreServiceClient
}

var agentSvcTracer = tracing.GetTracer("api-gateway.endpoints.agents.service")

func (c *AgentSvcConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent endpoint service: agent client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("agent endpoint service: core client is required")
	}
	return nil
}

func NewAgentSvc(config *AgentSvcConfig) AgentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentSvcImpl{
		agentClient: config.AgentClient,
		coreClient:  config.CoreClient,
	}
}

func (m *agentSvcImpl) resolveRole(ctx context.Context, roleID string) *resolvedRole {
	if roleID == "" {
		return nil
	}
	resp, err := m.coreClient.GetRoleInfo(ctx, &corepb.GetRoleInfoRequest{RoleId: roleID})
	if err != nil {
		return nil
	}
	resolved := &resolvedRole{
		Name:         resp.Name,
		RoleTypeCode: resp.RoleTypeCode,
	}
	if appctx.IsIncludeRequested(ctx, "role.permissions") {
		permResp, permErr := m.coreClient.GetRolePermissions(ctx, &corepb.GetRolePermissionsRequest{RoleId: roleID})
		if permErr == nil {
			resolved.Permissions = permResp.Permissions
		}
	}
	return resolved
}

type resolvedRole struct {
	Name         string
	RoleTypeCode string
	Permissions  map[string]bool
}

func marshalConfig(cfg ConfigInput) (string, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal agent config: %w", err)
	}
	return string(b), nil
}

func (m *agentSvcImpl) CreateAgent(ctx context.Context, req *CreateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
	if err := req.Config.Validate(req.TriggerType); err != nil {
		return nil, apierror.NewValidationError(err.Error())
	}

	configJSON, err := marshalConfig(req.Config)
	if err != nil {
		return nil, apierror.NewInternalError(err, "failed to marshal agent config")
	}

	tools := make([]*pb.AgentToolConfig, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = &pb.AgentToolConfig{
			ToolId:        t.ToolID,
			ConfigJson:    t.ConfigJSON,
			SortOrder:     t.SortOrder,
			RequireReview: t.RequireReview,
		}
	}

	pbReq := &pb.CreateCustomAgentRequest{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		CategoryCode: req.CategoryCode,
		TriggerType:  string(req.TriggerType),
		ConfigJson:   configJSON,
		Tools:        tools,
		Includes:     appctx.GetRequestedIncludeKeys(ctx),
		RoleId:       req.RoleID,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomAgentResponse, error) {
			return m.agentClient.CreateCustomAgent(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	result := AgentDefinitionPresenter(resp.Agent, m.resolveRole(ctx, resp.Agent.GetRoleId()))
	return &result, nil
}

func (m *agentSvcImpl) ListAgents(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError) {
	statuses := make([]string, len(req.Status))
	for i, s := range req.Status {
		statuses[i] = string(s)
	}

	pbReq := &pb.ListAgentDefinitionsRequest{
		Includes:        appctx.GetRequestedIncludeKeys(ctx),
		Statuses:        statuses,
		DefinitionTypes: req.DefinitionType,
		TriggerTypes:    req.TriggerType,
		Cursor:          req.Cursor,
		Limit:           req.Limit,
		Query:           req.Query,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAgentDefinitionsResponse, error) {
			return m.agentClient.ListAgentDefinitions(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	return AgentDefinitionListPresenter(resp, func(roleID string) *resolvedRole {
		return m.resolveRole(ctx, roleID)
	}), nil
}

func (m *agentSvcImpl) GetAgent(ctx context.Context, req *GetAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
	pbReq := &pb.GetAgentDefinitionRequest{
		AgentDefinitionId: req.AgentDefinitionID,
		Includes:          appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAgentDefinitionResponse, error) {
			return m.agentClient.GetAgentDefinition(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	result := AgentDefinitionPresenter(resp.Agent, m.resolveRole(ctx, resp.Agent.GetRoleId()))
	return &result, nil
}

func (m *agentSvcImpl) UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
	pbReq := &pb.UpdateCustomAgentRequest{
		AgentDefinitionId: req.AgentDefinitionID,
		Name:              req.Name,
		Slug:              req.Slug,
		Description:       req.Description,
		CategoryCode:      req.CategoryCode,
		TriggerType:       req.TriggerType.StringPtr(),
		RoleId:            req.RoleID,
		Includes:          appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.Config != nil {
		if req.TriggerType != nil {
			if err := req.Config.Validate(*req.TriggerType); err != nil {
				return nil, apierror.NewValidationError(err.Error())
			}
		}
		configJSON, err := marshalConfig(*req.Config)
		if err != nil {
			return nil, apierror.NewInternalError(err, "failed to marshal agent config")
		}
		pbReq.ConfigJson = &configJSON
	}

	if req.Tools != nil {
		pbReq.ToolsProvided = true
		tools := make([]*pb.AgentToolConfig, len(*req.Tools))
		for i, t := range *req.Tools {
			tools[i] = &pb.AgentToolConfig{
				ToolId:        t.ToolID,
				ConfigJson:    t.ConfigJSON,
				SortOrder:     t.SortOrder,
				RequireReview: t.RequireReview,
			}
		}
		pbReq.Tools = tools
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCustomAgentResponse, error) {
			return m.agentClient.UpdateCustomAgent(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	result := AgentDefinitionPresenter(resp.Agent, m.resolveRole(ctx, resp.Agent.GetRoleId()))
	return &result, nil
}

func (m *agentSvcImpl) DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteCustomAgentRequest{
		AgentDefinitionId: req.AgentDefinitionID,
	}

	_, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteCustomAgentResponse, error) {
			return m.agentClient.DeleteCustomAgent(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *agentSvcImpl) UpdateAgentStatus(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
	pbReq := &pb.UpdateAgentAccountStatusRequest{
		AgentDefinitionId: req.AgentDefinitionID,
		StatusCode:        req.StatusCode,
	}

	if _, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.update_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAgentAccountStatusResponse, error) {
			return m.agentClient.UpdateAgentAccountStatus(ctx, pbReq, opts...)
		}); rpcErr != nil {
		return nil, rpcErr
	}

	return m.GetAgent(ctx, &GetAgentRequest{AgentDefinitionID: req.AgentDefinitionID})
}

func (m *agentSvcImpl) ListUsage(ctx context.Context, req *ListUsageRequest) (*apiresource.List[apiresource.AgentTokenUsage], *apierror.APIError) {
	pbReq := &pb.ListTokenUsageRequest{
		Days:   req.Days,
		Limit:  req.Limit,
		Cursor: req.Cursor,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.list_usage", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListTokenUsageResponse, error) {
			return m.agentClient.ListTokenUsage(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	return AgentTokenUsageListPresenter(resp), nil
}
