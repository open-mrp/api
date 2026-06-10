package agentep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	corepb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentSvc interface {
	CreateAgent(ctx context.Context, req *CreateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	ListAgents(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError)
	GetAgent(ctx context.Context, req *RetrieveAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdateAgentStatus(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	ListUsage(ctx context.Context, req *ListUsageRequest) (*apiresource.List[apiresource.AgentTokenUsage], *apierror.APIError)
}

type AgentSvcConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient pb.AgentServiceClient

	// CoreClient (required) is the core-service gRPC client.
	CoreClient corepb.CoreServiceClient
}

type agentSvcImpl struct {
	agentClient pb.AgentServiceClient
	coreClient  corepb.CoreServiceClient
}

var agentSvcTracer = tracing.GetTracer("api-gateway.endpoints.agents.service")
var agentIncludes = []string{"config", "tools", "role"}

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

func (m *agentSvcImpl) resolveRole(ctx context.Context, roleID string) *ResolvedRole {
	if roleID == "" {
		return nil
	}
	resp, err := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.resolve_role", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.GetRoleInfoResponse, error) {
			return m.coreClient.GetRoleInfo(ctx, &corepb.GetRoleInfoRequest{RoleId: roleID}, opts...)
		})
	if err != nil {
		return nil
	}
	resolved := &ResolvedRole{
		Name:     resp.Name,
		RoleType: resp.RoleTypeCode,
	}
	permResp, permErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.resolve_role_permissions", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.GetRolePermissionsResponse, error) {
			return m.coreClient.GetRolePermissions(ctx, &corepb.GetRolePermissionsRequest{RoleId: roleID}, opts...)
		})
	if permErr == nil {
		resolved.Permissions = permResp.Permissions
	}
	return resolved
}

type ResolvedRole struct {
	Name        string
	RoleType    string
	Permissions map[string]bool
}

func marshalConfig(cfg ConfigInput) (string, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal agent config: %w", err)
	}
	return string(b), nil
}

// toolConfigFromInput maps a ToolInput to its proto representation; unset
// optional fields map to proto zero values.
func toolConfigFromInput(t ToolInput) *pb.AgentToolConfig {
	cfg := &pb.AgentToolConfig{ToolId: t.ToolID}
	if v, ok := t.ConfigJSON.Value(); ok {
		cfg.ConfigJson = v
	}
	if v, ok := t.SortOrder.Value(); ok {
		cfg.SortOrder = v
	}
	if v, ok := t.RequireReview.Value(); ok {
		cfg.RequireReview = v
	}
	return cfg
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
		tools[i] = toolConfigFromInput(t)
	}

	pbReq := &pb.CreateCustomAgentRequest{
		Name:         req.Name,
		Slug:         req.Slug,
		CategoryCode: req.CategoryCode,
		TriggerType:  string(req.TriggerType),
		ConfigJson:   configJSON,
		Tools:        tools,
		Includes:     resourcekit.FilterIncludes(ctx, agentIncludes...),
	}
	if v, ok := req.Description.Value(); ok {
		pbReq.Description = v
	}
	if v, ok := req.RoleID.Value(); ok {
		pbReq.RoleId = v
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomAgentResponse, error) {
			return m.agentClient.CreateCustomAgent(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := AgentDefinitionPresenter(resp.Agent)
	StashAgentDefinitionMeta(meta, resp.Agent, m.resolveRole(ctx, resp.Agent.GetRoleId()))
	return &result, nil
}

func (m *agentSvcImpl) ListAgents(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError) {
	statuses := make([]string, len(req.Status))
	for i, s := range req.Status {
		statuses[i] = string(s)
	}

	definitionTypes := make([]string, len(req.DefinitionType))
	for i, d := range req.DefinitionType {
		definitionTypes[i] = string(d)
	}

	triggerTypes := make([]string, len(req.TriggerType))
	for i, t := range req.TriggerType {
		triggerTypes[i] = string(t)
	}

	pbReq := &pb.ListAgentDefinitionsRequest{
		Includes:        resourcekit.FilterIncludes(ctx, agentIncludes...),
		Statuses:        statuses,
		DefinitionTypes: definitionTypes,
		TriggerTypes:    triggerTypes,
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

	return AgentDefinitionListPresenter(ctx, resp, func(roleID string) *ResolvedRole {
		return m.resolveRole(ctx, roleID)
	}), nil
}

func (m *agentSvcImpl) GetAgent(ctx context.Context, req *RetrieveAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
	pbReq := &pb.GetAgentDefinitionRequest{
		AgentDefinitionId: req.AgentDefinitionID,
		Includes:          resourcekit.FilterIncludes(ctx, agentIncludes...),
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAgentDefinitionResponse, error) {
			return m.agentClient.GetAgentDefinition(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := AgentDefinitionPresenter(resp.Agent)
	StashAgentDefinitionMeta(meta, resp.Agent, m.resolveRole(ctx, resp.Agent.GetRoleId()))
	return &result, nil
}

func (m *agentSvcImpl) UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
	var triggerTypePtr *string
	if tt, ok := req.TriggerType.Value(); ok {
		s := string(tt)
		triggerTypePtr = &s
	}

	pbReq := &pb.UpdateCustomAgentRequest{
		AgentDefinitionId: req.AgentDefinitionID,
		Name:              req.Name.Ptr(),
		Slug:              req.Slug.Ptr(),
		Description:       req.Description.Ptr(),
		CategoryCode:      req.CategoryCode.Ptr(),
		TriggerType:       triggerTypePtr,
		RoleId:            req.RoleID.Ptr(),
		Includes:          resourcekit.FilterIncludes(ctx, agentIncludes...),
	}

	if cfg, ok := req.Config.Value(); ok {
		if tt, ttOK := req.TriggerType.Value(); ttOK {
			if err := cfg.Validate(tt); err != nil {
				return nil, apierror.NewValidationError(err.Error())
			}
		}
		configJSON, err := marshalConfig(cfg)
		if err != nil {
			return nil, apierror.NewInternalError(err, "failed to marshal agent config")
		}
		pbReq.ConfigJson = &configJSON
	}

	if toolInputs, ok := req.Tools.Value(); ok {
		pbReq.ToolsProvided = true
		tools := make([]*pb.AgentToolConfig, len(toolInputs))
		for i, t := range toolInputs {
			tools[i] = toolConfigFromInput(t)
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

	meta := resourcekit.GetLoadMeta(ctx)
	result := AgentDefinitionPresenter(resp.Agent)
	StashAgentDefinitionMeta(meta, resp.Agent, m.resolveRole(ctx, resp.Agent.GetRoleId()))
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
		StatusCode:        req.Status,
	}

	if _, rpcErr := grpcutil.CallRPC(ctx, agentSvcTracer, "service.agents.update_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAgentAccountStatusResponse, error) {
			return m.agentClient.UpdateAgentAccountStatus(ctx, pbReq, opts...)
		}); rpcErr != nil {
		return nil, rpcErr
	}

	return m.GetAgent(ctx, &RetrieveAgentRequest{AgentDefinitionID: req.AgentDefinitionID})
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

	return AgentTokenUsageListPresenter(ctx, resp), nil
}
