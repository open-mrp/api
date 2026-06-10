package agentrunep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	corepb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentRunSvc interface {
	ListRuns(ctx context.Context, req *ListRunsRequest) (*apiresource.List[apiresource.AgentRun], *apierror.APIError)
	GetRun(ctx context.Context, req *RetrieveRunRequest) (*apiresource.AgentRun, *apierror.APIError)
	TriggerAgentRun(ctx context.Context, req *TriggerRunRequest) (*apiresource.AgentRun, *apierror.APIError)
	CancelAgentRun(ctx context.Context, req *CancelRunRequest) (*apiresource.AgentRun, *apierror.APIError)
	ContinueAgentRun(ctx context.Context, req *ContinueRunRequest) (*apiresource.AgentRun, *apierror.APIError)
}

type AgentRunSvcConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient pb.AgentServiceClient

	// CoreClient (optional; default: nil) is the core-service gRPC client used to
	// resolve role info at runtime. It may be nil in static-reflection contexts
	// (e.g. OpenAPI generation) where no RPCs are made.
	CoreClient corepb.CoreServiceClient
}

type resolvedRole struct {
	Name     string
	RoleType string
}

type agentRunSvcImpl struct {
	agentClient pb.AgentServiceClient
	coreClient  corepb.CoreServiceClient
}

var runSvcTracer = tracing.GetTracer("api-gateway.endpoints.agent_runs.service")

var agentRunIncludes = []string{"actions", "definition", "steps", "definition.config", "definition.tools", "definition.role"}

func (c *AgentRunSvcConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent run endpoint service: agent client is required")
	}
	return nil
}

func NewAgentRunSvc(config *AgentRunSvcConfig) AgentRunSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentRunSvcImpl{
		agentClient: config.AgentClient,
		coreClient:  config.CoreClient,
	}
}

func (m *agentRunSvcImpl) resolveRole(ctx context.Context, roleID string) *resolvedRole {
	if roleID == "" || m.coreClient == nil {
		return nil
	}
	resp, err := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.resolve_role", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.GetRoleInfoResponse, error) {
			return m.coreClient.GetRoleInfo(ctx, &corepb.GetRoleInfoRequest{RoleId: roleID}, opts...)
		})
	if err != nil {
		return nil
	}
	return &resolvedRole{
		Name:     resp.Name,
		RoleType: resp.RoleTypeCode,
	}
}

func (m *agentRunSvcImpl) ListRuns(ctx context.Context, req *ListRunsRequest) (*apiresource.List[apiresource.AgentRun], *apierror.APIError) {
	pbReq := &pb.ListRunsRequest{
		Limit:    req.Limit,
		Cursor:   req.Cursor,
		Includes: resourcekit.FilterIncludes(ctx, agentRunIncludes...),
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.StatusCode != nil {
		pbReq.StatusCode = *req.StatusCode
	}
	if req.AgentDefinitionID != nil {
		pbReq.AgentDefinitionId = *req.AgentDefinitionID
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRunsResponse, error) {
			return m.agentClient.ListRuns(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	runs := make([]apiresource.AgentRun, len(resp.Runs))
	for i, r := range resp.Runs {
		var roleInfo *resolvedRole
		if r.Definition != nil {
			roleInfo = m.resolveRole(ctx, r.Definition.RoleId)
		}
		runs[i] = AgentRunPresenterWithRole(r, roleInfo)
		stashAgentRunMeta(ctx, &runs[i], r, roleInfo)
	}
	return apiresource.NewList(runs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *agentRunSvcImpl) GetRun(ctx context.Context, req *RetrieveRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
	pbReq := &pb.GetRunRequest{
		AgentRunId: req.AgentRunID,
		Includes:   resourcekit.FilterIncludes(ctx, agentRunIncludes...),
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRunResponse, error) {
			return m.agentClient.GetRun(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	var roleInfo *resolvedRole
	if resp.Run != nil && resp.Run.Definition != nil {
		roleInfo = m.resolveRole(ctx, resp.Run.Definition.RoleId)
	}

	result := AgentRunPresenterWithRole(resp.Run, roleInfo)
	stashAgentRunMeta(ctx, &result, resp.Run, roleInfo)
	return &result, nil
}

func (m *agentRunSvcImpl) TriggerAgentRun(ctx context.Context, req *TriggerRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
	defResp, rpcErr := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.get_for_trigger", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAgentDefinitionResponse, error) {
			return m.agentClient.GetAgentDefinition(ctx, &pb.GetAgentDefinitionRequest{
				AgentDefinitionId: req.AgentDefinitionID,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	if defResp.Agent.AccountStatus == nil || defResp.Agent.AccountStatus.StatusCode != string(constants.AgentAccountStatusActive) {
		return nil, apierror.NewValidationError("agent definition is inactive and cannot be triggered")
	}

	pbReq := &pb.TriggerRunRequest{
		AgentDefinitionCode: defResp.Agent.Slug,
	}
	if v, ok := req.Input.Value(); ok {
		pbReq.Input = v
	}

	resp, triggerErr := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.trigger", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.TriggerRunResponse, error) {
			return m.agentClient.TriggerRun(ctx, pbReq, opts...)
		})
	if triggerErr != nil {
		return nil, triggerErr
	}

	return m.GetRun(ctx, &RetrieveRunRequest{AgentRunID: resp.AgentRunId})
}

func (m *agentRunSvcImpl) CancelAgentRun(ctx context.Context, req *CancelRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
	pbReq := &pb.CancelRunRequest{
		AgentRunId: req.AgentRunID,
	}

	if _, rpcErr := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.cancel", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CancelRunResponse, error) {
			return m.agentClient.CancelRun(ctx, pbReq, opts...)
		}); rpcErr != nil {
		return nil, rpcErr
	}

	return m.GetRun(ctx, &RetrieveRunRequest{AgentRunID: req.AgentRunID})
}

func (m *agentRunSvcImpl) ContinueAgentRun(ctx context.Context, req *ContinueRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
	pbReq := &pb.ContinueRunRequest{
		AgentRunId:        req.AgentRunID,
		Message:           req.Message,
		ApprovedToolSlugs: req.ApprovedToolSlugs,
		AllowedToolSlugs:  req.AllowedToolSlugs,
	}

	if _, rpcErr := grpcutil.CallRPC(ctx, runSvcTracer, "service.agent_runs.continue", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ContinueRunResponse, error) {
			return m.agentClient.ContinueRun(ctx, pbReq, opts...)
		}); rpcErr != nil {
		return nil, rpcErr
	}

	return m.GetRun(ctx, &RetrieveRunRequest{AgentRunID: req.AgentRunID})
}
