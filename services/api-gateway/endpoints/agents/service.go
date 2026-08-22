package agentep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/agent"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentSvc interface {
	CreateAgent(ctx context.Context, req *CreateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	ListAgents(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError)
	GetAgent(ctx context.Context, req *RetrieveAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError)
	DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdateAgentStatus(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError)
}

type AgentSvcConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient pb.AgentServiceClient
}

type agentSvcImpl struct {
	agentClient pb.AgentServiceClient
}

var agentSvcTracer = tracing.GetTracer("api-gateway.endpoints.agents.service")
var agentIncludes = []string{"config", "tools", "role"}

func (c *AgentSvcConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent endpoint service: agent client is required")
	}
	return nil
}

func NewAgentSvc(config *AgentSvcConfig) AgentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentSvcImpl{
		agentClient: config.AgentClient,
	}
}

// wireTriggerConfig is the persisted shape of an agent's trigger configuration.
type wireTriggerConfig struct {
	CronSchedule *string  `json:"cron_schedule,omitempty"`
	Timezone     *string  `json:"timezone,omitempty"`
	EventFilters []string `json:"event_filters,omitempty"`
}

// wireConfig is the persisted shape of an agent's config. We translate the
// public request fields into this shape before persisting. The LLM provider is
// derived by agent-service from the model and is therefore not persisted.
type wireConfig struct {
	SystemPrompt       *string            `json:"system_prompt,omitempty"`
	Model              *string            `json:"model,omitempty"`
	Tier               *string            `json:"tier,omitempty"`
	Temperature        *float64           `json:"temperature,omitempty"`
	TriggerConfig      *wireTriggerConfig `json:"trigger_config,omitempty"`
	EndpointToolSlugs  []string           `json:"endpoint_tool_slugs,omitempty"`
	EndpointToolReview map[string]bool    `json:"endpoint_tool_review,omitempty"`
}

func marshalConfig(cfg ConfigInput) (string, error) {
	wire := wireConfig{
		SystemPrompt:      cfg.SystemPrompt.Ptr(),
		Temperature:       cfg.Temperature.Ptr(),
		EndpointToolSlugs: cfg.EndpointToolSlugs,
	}
	if review, ok := cfg.EndpointToolReview.Value(); ok {
		wire.EndpointToolReview = review
	}
	if tier, ok := cfg.Tier.Value(); ok {
		t := string(tier)
		wire.Tier = &t
	}
	if tc, ok := cfg.TriggerConfig.Value(); ok {
		wire.TriggerConfig = &wireTriggerConfig{
			CronSchedule: tc.CronSchedule.Ptr(),
			Timezone:     tc.Timezone.Ptr(),
			EventFilters: tc.EventFilters,
		}
	}

	b, err := json.Marshal(wire)
	if err != nil {
		return "", fmt.Errorf("marshal agent config: %w", err)
	}
	return string(b), nil
}

// toolConfigFromInput maps a ToolInput to its proto representation; unset
// optional fields map to proto zero values.
func toolConfigFromInput(t ToolInput) *pb.AgentToolConfig {
	cfg := &pb.AgentToolConfig{ToolSlug: string(t.Tool)}
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
	StashAgentDefinitionMeta(meta, resp.Agent)
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

	return AgentDefinitionListPresenter(ctx, resp), nil
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
	StashAgentDefinitionMeta(meta, resp.Agent)
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
		Description:       req.Description.ValuePtr(),
		ClearDescription:  req.Description.IsClear(),
		CategoryCode:      req.CategoryCode.Ptr(),
		TriggerType:       triggerTypePtr,
		RoleId:            req.RoleID.ValuePtr(),
		ClearRoleId:       req.RoleID.IsClear(),
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
	StashAgentDefinitionMeta(meta, resp.Agent)
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
