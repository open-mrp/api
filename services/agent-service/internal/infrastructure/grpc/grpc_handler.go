package grpc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"database/sql"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	agentpagination "github.com/open-mrp/api/services/agent-service/internal/pagination"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	pb "github.com/open-mrp/api/shared/proto/agent"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var handlerTracer = tracing.GetTracer("agent-service.grpc_handler")

type agentHandler struct {
	pb.UnimplementedAgentServiceServer
	repos       domain.RepoFactory
	agentDefSvc domain.AgentDefinitionSvc
}

func NewAgentHandler(server *grpc.Server, repos domain.RepoFactory, agentDefSvc domain.AgentDefinitionSvc) *agentHandler {
	handler := &agentHandler{
		repos:       repos,
		agentDefSvc: agentDefSvc,
	}
	pb.RegisterAgentServiceServer(server, handler)
	return handler
}

// requireTargetAccount rejects a call whose identity names no target account.
func requireTargetAccount(ctx context.Context) error {
	_, err := getAccountIDFromContext(ctx)
	return err
}

func getAccountIDFromContext(ctx context.Context) (string, error) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsTargetAccountSet() {
		return "", status.Error(codes.Unauthenticated, "identity with account ID is required")
	}
	return identity.Target.AccountID, nil
}

func (h *agentHandler) TriggerRun(ctx context.Context, req *pb.TriggerRunRequest) (*pb.TriggerRunResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.trigger_run")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AgentDefinitionCode == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_definition_code is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	runID, apiErr := h.agentDefSvc.TriggerRun(ctx, domain.TriggerRunParams{
		AgentDefinitionCode: req.AgentDefinitionCode,
		Input:               req.Input,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.TriggerRunResponse{
		AgentRunId: runID,
	}, nil
}

func (h *agentHandler) GetRunStatus(ctx context.Context, req *pb.GetRunStatusRequest) (*pb.GetRunStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.get_run_status")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AgentRunId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_run_id is required")
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	runRepo := h.repos.NewAgentRunRepo()
	run, runErr := runRepo.GetByID(ctx, req.AgentRunId)
	if runErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Agent run not found."))
	}

	if run.AccountID != accountID {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Agent run not found."))
	}

	var outputSummary string
	if run.Output != nil {
		var output map[string]string
		if jsonErr := json.Unmarshal(run.Output, &output); jsonErr == nil {
			if resp, ok := output["response"]; ok {
				outputSummary = resp
			}
		}
	}

	return &pb.GetRunStatusResponse{
		AgentRunId:    run.ID,
		StatusCode:    run.StatusCode,
		OutputSummary: outputSummary,
		TotalTokens:   safeconv.Int64ToInt32(run.TotalInputTokens + run.TotalOutputTokens),
		DurationMs:    run.DurationMs.Int32,
	}, nil
}

func (h *agentHandler) CancelRun(ctx context.Context, req *pb.CancelRunRequest) (*pb.CancelRunResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.cancel_run")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AgentRunId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_run_id is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.agentDefSvc.CancelRun(ctx, domain.CancelRunParams{
		AgentRunID: req.AgentRunId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CancelRunResponse{Success: true}, nil
}

func (h *agentHandler) ContinueRun(ctx context.Context, req *pb.ContinueRunRequest) (*pb.ContinueRunResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.continue_run")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AgentRunId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_run_id is required")
	}
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	runID, apiErr := h.agentDefSvc.ContinueRun(ctx, domain.ContinueRunParams{
		AgentRunID:          req.AgentRunId,
		Message:             req.Message,
		ApprovedToolSlugs:   req.ApprovedToolSlugs,
		RejectedToolSlugs:   req.RejectedToolSlugs,
		ApprovedToolCallIDs: req.ApprovedToolCallIds,
		RejectedToolCallIDs: req.RejectedToolCallIds,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ContinueRunResponse{
		AgentRunId: runID,
	}, nil
}

func (h *agentHandler) RetryRun(ctx context.Context, req *pb.RetryRunRequest) (*pb.RetryRunResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.retry_run")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AgentRunId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_run_id is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	runID, apiErr := h.agentDefSvc.RetryRun(ctx, domain.RetryRunParams{
		AgentRunID: req.AgentRunId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RetryRunResponse{
		AgentRunId: runID,
	}, nil
}

func (h *agentHandler) CreateCustomAgent(ctx context.Context, req *pb.CreateCustomAgentRequest) (*pb.CreateCustomAgentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.create_custom_agent")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Slug == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	if req.CategoryCode == "" {
		return nil, status.Error(codes.InvalidArgument, "category_code is required")
	}
	if req.TriggerType == "" {
		return nil, status.Error(codes.InvalidArgument, "trigger_type is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	tools := make([]domain.ToolLinkParams, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = domain.ToolLinkParams{
			ToolSlug:      t.ToolSlug,
			ConfigJSON:    t.ConfigJson,
			SortOrder:     t.SortOrder,
			RequireReview: t.RequireReview,
		}
	}

	result, apiErr := h.agentDefSvc.CreateCustomAgent(ctx, domain.CreateCustomAgentParams{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		CategoryCode: req.CategoryCode,
		TriggerType:  req.TriggerType,
		ConfigJSON:   req.ConfigJson,
		RoleID:       req.RoleId,
		Tools:        tools,
		Includes:     req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCustomAgentResponse{Agent: domainToProtoAgentDefinition(result)}, nil
}

func (h *agentHandler) UpdateCustomAgent(ctx context.Context, req *pb.UpdateCustomAgentRequest) (*pb.UpdateCustomAgentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.update_custom_agent")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req.AgentDefinitionId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_definition_id is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var tools []domain.ToolLinkParams
	if req.ToolsProvided {
		tools = make([]domain.ToolLinkParams, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = domain.ToolLinkParams{
				ToolSlug:      t.ToolSlug,
				ConfigJSON:    t.ConfigJson,
				SortOrder:     t.SortOrder,
				RequireReview: t.RequireReview,
			}
		}
	}

	result, apiErr := h.agentDefSvc.UpdateCustomAgent(ctx, domain.UpdateCustomAgentParams{
		AgentDefinitionID: req.AgentDefinitionId,
		Name:              req.Name,
		Slug:              req.Slug,
		Description:       req.Description,
		ClearDescription:  req.ClearDescription,
		CategoryCode:      req.CategoryCode,
		TriggerType:       req.TriggerType,
		ConfigJSON:        req.ConfigJson,
		RoleID:            req.RoleId,
		ClearRoleID:       req.ClearRoleId,
		Tools:             tools,
		ToolsProvided:     req.ToolsProvided,
		Includes:          req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateCustomAgentResponse{Agent: domainToProtoAgentDefinition(result)}, nil
}

func (h *agentHandler) DeleteCustomAgent(ctx context.Context, req *pb.DeleteCustomAgentRequest) (*pb.DeleteCustomAgentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.delete_custom_agent")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req.AgentDefinitionId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_definition_id is required")
	}

	apiErr := h.agentDefSvc.DeleteCustomAgent(ctx, domain.DeleteCustomAgentParams{
		AgentDefinitionID: req.AgentDefinitionId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteCustomAgentResponse{Success: true}, nil
}

func (h *agentHandler) GetAgentDefinition(ctx context.Context, req *pb.GetAgentDefinitionRequest) (*pb.GetAgentDefinitionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.get_agent_definition")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req.AgentDefinitionId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_definition_id is required")
	}

	result, apiErr := h.agentDefSvc.GetAgentDefinition(ctx, req.AgentDefinitionId, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAgentDefinitionResponse{Agent: domainToProtoAgentDefinition(result)}, nil
}

func (h *agentHandler) ListAgentDefinitions(ctx context.Context, req *pb.ListAgentDefinitionsRequest) (*pb.ListAgentDefinitionsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.list_agent_definitions")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	params := domain.ListAgentDefinitionsParams{
		Includes:        req.Includes,
		Statuses:        req.Statuses,
		DefinitionTypes: req.DefinitionTypes,
		TriggerTypes:    req.TriggerTypes,
		Cursor:          req.Cursor,
		Limit:           req.Limit,
		Query:           req.Query,
	}

	result, apiErr := h.agentDefSvc.ListAgentDefinitions(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	agents := make([]*pb.AgentDefinitionInfo, 0, len(result.Items))
	for i := range result.Items {
		agents = append(agents, domainToProtoAgentDefinition(&result.Items[i]))
	}

	return &pb.ListAgentDefinitionsResponse{
		Agents: agents,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *agentHandler) ListAvailableTools(ctx context.Context, req *pb.ListAvailableToolsRequest) (*pb.ListAvailableToolsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.list_available_tools")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	params := domain.ListAvailableToolsParams{}
	if req != nil {
		params.Limit = req.GetLimit()
		if req.Cursor != nil {
			c := req.GetCursor()
			params.Cursor = &c
		}
		if req.Query != nil {
			q := req.GetQuery()
			params.Query = &q
		}
		params.PaginateResource = req.GetPaginateResource()
	}

	results, groups, apiErr := h.agentDefSvc.ListAvailableTools(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbTools := make([]*pb.AvailableToolInfo, 0, len(results))
	for _, t := range results {
		pbTools = append(pbTools, &pb.AvailableToolInfo{
			Slug:                t.Slug,
			DisplayName:         t.DisplayName,
			Description:         t.Description,
			ConfigSchemaJson:    string(t.ConfigSchema),
			Category:            t.Category,
			GroupId:             t.GroupID,
			GroupName:           t.GroupName,
			RequiredPermissions: t.RequiredPermissions,
			RequiredRoleType:    t.RequiredRoleType,
			Mutating:            t.Mutating,
		})
	}

	pbGroups := make([]*pb.ToolGroupInfo, 0, len(groups))
	for _, g := range groups {
		pbGroups = append(pbGroups, &pb.ToolGroupInfo{
			Id:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			Slug:        g.Slug,
			Icon:        g.Icon,
			SortOrder:   g.SortOrder,
		})
	}

	return &pb.ListAvailableToolsResponse{Tools: pbTools, Groups: pbGroups}, nil
}

func (h *agentHandler) ListAvailableModels(ctx context.Context, _ *pb.ListAvailableModelsRequest) (*pb.ListAvailableModelsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.list_available_models")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	models := make([]*pb.AvailableModelInfo, 0, len(domain.AvailableModels))
	for _, m := range domain.AvailableModels {
		models = append(models, &pb.AvailableModelInfo{
			Code:     string(m.Code),
			Name:     m.Name,
			Provider: m.Provider,
		})
	}

	return &pb.ListAvailableModelsResponse{Models: models}, nil
}

func (h *agentHandler) UpdateAgentAccountStatus(ctx context.Context, req *pb.UpdateAgentAccountStatusRequest) (*pb.UpdateAgentAccountStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.update_agent_account_status")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req.AgentDefinitionId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_definition_id is required")
	}
	if req.StatusCode == "" {
		return nil, status.Error(codes.InvalidArgument, "status_code is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.agentDefSvc.UpdateAgentAccountStatus(ctx, domain.UpdateAgentAccountStatusParams{
		AgentDefinitionID: req.AgentDefinitionId,
		StatusCode:        req.StatusCode,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAgentAccountStatusResponse{
		Status: domainToProtoAccountStatus(result),
	}, nil
}

func formatNullTime(ts sql.NullTime) string {
	if !ts.Valid {
		return ""
	}
	return formatTime(ts.Time)
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000Z")
}

func formatNullString(t sql.NullString) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func sqlcRunToProto(run *sqlc.AgentRun) *pb.AgentRunInfo {
	var inputStr, outputStr string
	if run.Input != nil {
		inputStr = string(run.Input)
	}
	if run.Output != nil {
		outputStr = string(run.Output)
	}

	return &pb.AgentRunInfo{
		Id:                      run.ID,
		AccountId:               run.AccountID,
		AgentDefinitionId:       run.AgentDefinitionID,
		AgentConfigId:           formatNullString(run.AgentConfigID),
		StatusCode:              run.StatusCode,
		TriggerType:             run.TriggerType,
		Input:                   inputStr,
		Output:                  outputStr,
		ErrorMessage:            formatNullString(run.ErrorMessage),
		StartedAt:               formatNullTime(run.StartedAt),
		CompletedAt:             formatNullTime(run.CompletedAt),
		DurationMs:              run.DurationMs.Int32,
		TotalInputTokens:        run.TotalInputTokens,
		TotalOutputTokens:       run.TotalOutputTokens,
		CreatedAt:               formatTime(run.CreatedAt),
		UpdatedAt:               formatTime(run.UpdatedAt),
		TriggeredByActorId:      formatNullString(run.TriggeredByActorID),
		TriggeredByIdentityType: formatNullString(run.TriggeredByIdentityType),
		TriggeredByActorName:    formatNullString(run.TriggeredByActorName),
	}
}

var tokenCountKeys = []string{"input_tokens", "output_tokens", "input_tokens_before", "tokens_freed"}

func withoutTokenCounts(metadata []byte) []byte {
	if len(metadata) == 0 {
		return metadata
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(metadata, &fields); err != nil {
		return metadata
	}
	stripped := false
	for _, k := range tokenCountKeys {
		if _, ok := fields[k]; ok {
			delete(fields, k)
			stripped = true
		}
	}
	if !stripped {
		return metadata
	}
	out, err := json.Marshal(fields)
	if err != nil {
		return metadata
	}
	return out
}

func sqlcEventToProto(event *sqlc.AgentRunEvent) *pb.AgentRunStepInfo {
	return &pb.AgentRunStepInfo{
		Id:            event.ID,
		AgentRunId:    event.AgentRunID,
		StepType:      event.StepType,
		Title:         event.Title,
		Content:       formatNullString(event.Content),
		Sequence:      event.Sequence,
		DurationMs:    event.DurationMs.Int32,
		AgentActionId: formatNullString(event.AgentActionID),
		MetadataJson:  string(withoutTokenCounts(event.Metadata)),
		CreatedAt:     formatTime(event.CreatedAt),
		ActorId:       formatNullString(event.ActorID),
		ActorType:     formatNullString(event.ActorType),
		ActorName:     formatNullString(event.ActorName),
	}
}

func sqlcActionToProto(action *sqlc.AgentAction, runStatusCode, runTriggerType *string) *pb.AgentActionInfo {
	var inputStr, outputStr string
	if action.Input != nil {
		inputStr = string(action.Input)
	}
	if action.Output != nil {
		outputStr = string(action.Output)
	}

	return &pb.AgentActionInfo{
		Id:                  action.ID,
		AccountId:           action.AccountID,
		AgentRunId:          action.AgentRunID,
		ToolSlug:            action.ToolSlug,
		StatusCode:          action.StatusCode,
		Label:               formatNullString(action.Label),
		Description:         formatNullString(action.Description),
		Input:               inputStr,
		Output:              outputStr,
		ErrorMessage:        formatNullString(action.ErrorMessage),
		EntityType:          formatNullString(action.EntityType),
		EntityId:            formatNullString(action.EntityID),
		RequiresReview:      action.RequiresReview,
		ReviewedAt:          formatNullTime(action.ReviewedAt),
		ReviewedBy:          formatNullString(action.ReviewedBy),
		ReviewedByActorType: formatNullString(action.ReviewedByActorType),
		ReviewedByActorName: formatNullString(action.ReviewedByActorName),
		ExecutedAt:          formatNullTime(action.ExecutedAt),
		CreatedAt:           formatTime(action.CreatedAt),
		UpdatedAt:           formatTime(action.UpdatedAt),
		RunStatusCode:       runStatusCode,
		RunTriggerType:      runTriggerType,
	}
}

func (h *agentHandler) GetRun(ctx context.Context, req *pb.GetRunRequest) (*pb.GetRunResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.get_run")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AgentRunId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_run_id is required")
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActorForRead(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentRuns, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	runRepo := h.repos.NewAgentRunRepo()
	run, runErr := runRepo.GetByID(ctx, req.AgentRunId)
	if runErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Agent run not found."))
	}

	if run.AccountID != accountID {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Agent run not found."))
	}

	pbRun := sqlcRunToProto(run)

	includeSet := make(map[string]bool, len(req.Includes))
	for _, inc := range req.Includes {
		includeSet[inc] = true
	}

	if includeSet["actions"] {
		actionRepo := h.repos.NewAgentActionRepo()
		actions, actionsErr := actionRepo.ListByRun(ctx, run.ID)
		if actionsErr == nil {
			pbActions := make([]*pb.AgentActionInfo, len(actions))
			for i := range actions {
				pbActions[i] = sqlcActionToProto(&actions[i], &run.StatusCode, &run.TriggerType)
			}
			pbRun.Actions = pbActions
		}
	}

	if includeSet["definition"] {
		defResult, defErr := h.agentDefSvc.GetAgentDefinition(ctx, run.AgentDefinitionID, nestedIncludes(req.Includes, "definition"))
		if defErr == nil {
			pbRun.Definition = domainToProtoAgentDefinition(defResult)
		}
	}

	if includeSet["steps"] {
		eventRepo := h.repos.NewAgentRunEventRepo()
		events, eventsErr := eventRepo.ListByRunID(ctx, run.ID)
		if eventsErr == nil {
			pbSteps := make([]*pb.AgentRunStepInfo, len(events))
			for i := range events {
				pbSteps[i] = sqlcEventToProto(&events[i])
			}
			pbRun.Steps = pbSteps
		}
	}

	return &pb.GetRunResponse{Run: pbRun}, nil
}

func (h *agentHandler) ListRuns(ctx context.Context, req *pb.ListRunsRequest) (*pb.ListRunsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.list_runs")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentRuns, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	cursorID, cursorDir, apiErr := agentpagination.ParseOptionalStringCursor(req.Cursor)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	params := domain.AgentRunPageQuery{
		AccountID:         accountID,
		StatusCode:        req.StatusCode,
		AgentDefinitionID: req.AgentDefinitionId,
		Query:             req.GetQuery(),
		CursorID:          cursorID,
		Limit:             limit + 1,
	}

	runRepo := h.repos.NewAgentRunRepo()
	runs, runsErr := runRepo.ListByAccountFiltered(ctx, params)
	if runsErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(runsErr)
	}

	runs, pageInfo := pagination.BuildPageString(
		runs,
		limit,
		cursorDir,
		func(r sqlc.AgentRun) time.Time { return r.CreatedAt },
		func(r sqlc.AgentRun) string { return r.ID },
	)

	// Treat `foo.bar` requests as implying `foo` — the parent resource must be attached for the nested include to survive the api-gateway's collapse.
	includedRoot := func(key string) bool {
		prefix := key + "."
		for _, inc := range req.Includes {
			if inc == key || strings.HasPrefix(inc, prefix) {
				return true
			}
		}
		return false
	}

	pbRuns := make([]*pb.AgentRunInfo, len(runs))
	for i := range runs {
		pbRuns[i] = sqlcRunToProto(&runs[i])

		if includedRoot("actions") {
			actionRepo := h.repos.NewAgentActionRepo()
			actions, actionsErr := actionRepo.ListByRun(ctx, runs[i].ID)
			if actionsErr == nil {
				pbActions := make([]*pb.AgentActionInfo, len(actions))
				for j := range actions {
					pbActions[j] = sqlcActionToProto(&actions[j], &runs[i].StatusCode, &runs[i].TriggerType)
				}
				pbRuns[i].Actions = pbActions
			}
		}

		if includedRoot("definition") {
			// Pass the client's requested nested includes (e.g. "tools", "role", "config") so buildResultForAccount loads the matching sub-resources onto the returned definition.
			defIncludes := nestedIncludes(req.Includes, "definition")
			defResult, defErr := h.agentDefSvc.GetAgentDefinition(ctx, runs[i].AgentDefinitionID, defIncludes)
			if defErr == nil {
				pbRuns[i].Definition = domainToProtoAgentDefinition(defResult)
			}
		}

		if includedRoot("steps") {
			eventRepo := h.repos.NewAgentRunEventRepo()
			events, eventsErr := eventRepo.ListByRunID(ctx, runs[i].ID)
			if eventsErr == nil {
				pbSteps := make([]*pb.AgentRunStepInfo, len(events))
				for j := range events {
					pbSteps[j] = sqlcEventToProto(&events[j])
				}
				pbRuns[i].Steps = pbSteps
			}
		}
	}

	return &pb.ListRunsResponse{
		Runs:     pbRuns,
		PageInfo: agentpagination.ToProtoPageInfo(pageInfo),
	}, nil
}

func (h *agentHandler) ListTokenUsage(ctx context.Context, req *pb.ListTokenUsageRequest) (*pb.ListTokenUsageResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.list_token_usage")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	days := req.Days
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	sinceDate := time.Now().AddDate(0, 0, -int(days))

	cursorID, cursorDir, apiErr := agentpagination.ParseOptionalStringCursor(req.Cursor)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	params := domain.AgentTokenUsagePageQuery{
		AccountID: accountID,
		SinceDate: sinceDate,
		CursorID:  cursorID,
		Limit:     limit + 1,
	}

	tokenUsageRepo := h.repos.NewAgentTokenUsageRepo()
	rows, repoErr := tokenUsageRepo.ListByAccount(ctx, params)
	if repoErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(repoErr)
	}

	rows, pageInfo := pagination.BuildPageString(
		rows,
		limit,
		cursorDir,
		func(r sqlc.AgentTokenUsage) time.Time { return r.Date },
		func(r sqlc.AgentTokenUsage) string { return r.ID },
	)

	pbUsage := make([]*pb.AgentTokenUsageInfo, len(rows))
	for i := range rows {
		pbUsage[i] = &pb.AgentTokenUsageInfo{
			Id:           rows[i].ID,
			Date:         rows[i].Date.Format("2006-01-02"),
			InputTokens:  rows[i].InputTokens,
			OutputTokens: rows[i].OutputTokens,
			TotalCost:    rows[i].TotalCost,
			RunCount:     rows[i].RunCount,
			CreatedAt:    rows[i].CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    rows[i].UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return &pb.ListTokenUsageResponse{
		Usage:    pbUsage,
		PageInfo: agentpagination.ToProtoPageInfo(pageInfo),
	}, nil
}

func sqlcMemoryToProto(m *sqlc.AgentMemory) *pb.AgentMemoryInfo {
	var metadataStr string
	if m.Metadata != nil {
		metadataStr = string(m.Metadata)
	}
	return &pb.AgentMemoryInfo{
		Id:           m.ID,
		AccountId:    m.AccountID,
		Category:     m.Category,
		Content:      m.Content,
		MetadataJson: metadataStr,
		EntityType:   formatNullString(m.EntityType),
		EntityId:     formatNullString(m.EntityID),
		Importance:   m.Importance,
		ExpiresAt:    formatNullTime(m.ExpiresAt),
		CreatedAt:    formatTime(m.CreatedAt),
		UpdatedAt:    formatTime(m.UpdatedAt),
	}
}

func (h *agentHandler) ListAgentMemories(ctx context.Context, req *pb.ListAgentMemoriesRequest) (*pb.ListAgentMemoriesResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.list_agent_memories")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentMemories, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	memoryRepo := h.repos.NewAgentMemoryRepo()

	cursorID, cursorDir, apiErr := agentpagination.ParseOptionalStringCursor(req.Cursor)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	params := domain.AgentMemoryPageQuery{
		AccountID:  accountID,
		Category:   req.Category,
		EntityType: req.EntityType,
		Query:      req.GetQuery(),
		CursorID:   cursorID,
		Limit:      limit + 1,
	}

	rows, repoErr := memoryRepo.ListByAccountCursor(ctx, params)
	if repoErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(repoErr)
	}

	rows, pageInfo := pagination.BuildPageString(
		rows,
		limit,
		cursorDir,
		func(m sqlc.AgentMemory) time.Time { return m.CreatedAt },
		func(m sqlc.AgentMemory) string { return m.ID },
	)

	pbMemories := make([]*pb.AgentMemoryInfo, len(rows))
	for i := range rows {
		pbMemories[i] = sqlcMemoryToProto(&rows[i])
	}

	return &pb.ListAgentMemoriesResponse{
		Memories: pbMemories,
		PageInfo: agentpagination.ToProtoPageInfo(pageInfo),
	}, nil
}

func (h *agentHandler) GetAgentMemory(ctx context.Context, req *pb.GetAgentMemoryRequest) (*pb.GetAgentMemoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.get_agent_memory")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentMemories, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	memoryRepo := h.repos.NewAgentMemoryRepo()
	memory, memErr := memoryRepo.GetByID(ctx, req.Id)
	if memErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Agent memory not found."))
	}

	if memory.AccountID != accountID {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Agent memory not found."))
	}

	return &pb.GetAgentMemoryResponse{Memory: sqlcMemoryToProto(memory)}, nil
}

func (h *agentHandler) BatchGetAgentMemoriesByIDs(ctx context.Context, req *pb.BatchGetAgentMemoriesByIDsRequest) (*pb.BatchGetAgentMemoriesByIDsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.batch_get_agent_memories_by_ids")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentMemories, types.ActionRead); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	accountID, err := getAccountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	memoryRepo := h.repos.NewAgentMemoryRepo()
	rows, apiErr := memoryRepo.GetByIDs(ctx, accountID, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbMemories := make([]*pb.AgentMemoryInfo, len(rows))
	for i := range rows {
		pbMemories[i] = sqlcMemoryToProto(&rows[i])
	}
	return &pb.BatchGetAgentMemoriesByIDsResponse{Memories: pbMemories}, nil
}

func (h *agentHandler) CreateAgentMemory(ctx context.Context, req *pb.CreateAgentMemoryRequest) (*pb.CreateAgentMemoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.create_agent_memory")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.Category == "" {
		return nil, status.Error(codes.InvalidArgument, "category is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.agentDefSvc.CreateAgentMemory(ctx, domain.CreateAgentMemoryParams{
		Category:     req.Category,
		Content:      req.Content,
		MetadataJSON: req.MetadataJson,
		EntityType:   req.EntityType,
		EntityID:     req.EntityId,
		Importance:   req.Importance,
		ExpiresAt:    req.ExpiresAt,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAgentMemoryResponse{Memory: domainMemoryToProto(result)}, nil
}

func (h *agentHandler) UpdateAgentMemory(ctx context.Context, req *pb.UpdateAgentMemoryRequest) (*pb.UpdateAgentMemoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.update_agent_memory")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.agentDefSvc.UpdateAgentMemory(ctx, domain.UpdateAgentMemoryParams{
		MemoryID:       req.Id,
		Category:       req.Category,
		Content:        req.Content,
		MetadataJSON:   req.MetadataJson,
		EntityType:     req.EntityType,
		EntityID:       req.EntityId,
		Importance:     req.Importance,
		ExpiresAt:      req.ExpiresAt,
		ClearEntity:    req.ClearEntity,
		ClearExpiresAt: req.ClearExpiresAt,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAgentMemoryResponse{Memory: domainMemoryToProto(result)}, nil
}

func (h *agentHandler) DeleteAgentMemory(ctx context.Context, req *pb.DeleteAgentMemoryRequest) (*pb.DeleteAgentMemoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, handlerTracer, "grpc.delete_agent_memory")
	defer span.End()

	if err := requireTargetAccount(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	apiErr := h.agentDefSvc.DeleteAgentMemory(ctx, domain.DeleteAgentMemoryParams{MemoryID: req.Id})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteAgentMemoryResponse{Success: true}, nil
}

func domainMemoryToProto(m *domain.AgentMemoryInfo) *pb.AgentMemoryInfo {
	return &pb.AgentMemoryInfo{
		Id:           m.ID,
		AccountId:    m.AccountID,
		Category:     m.Category,
		Content:      m.Content,
		MetadataJson: m.Metadata,
		EntityType:   m.EntityType,
		EntityId:     m.EntityID,
		Importance:   m.Importance,
		ExpiresAt:    m.ExpiresAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// nestedIncludes filters an includes slice to the sub-paths under a parent key, stripped of the parent prefix. For includes=["definition.role","config"] and parent="definition" it returns ["role"]. Returns nil when nothing matches.
func nestedIncludes(includes []string, parent string) []string {
	prefix := parent + "."
	var out []string
	for _, inc := range includes {
		if after, ok := strings.CutPrefix(inc, prefix); ok {
			out = append(out, after)
		}
	}
	return out
}

func domainToProtoAccountStatus(info *domain.AgentAccountStatusInfo) *pb.AgentAccountStatusInfo {
	if info == nil {
		return nil
	}
	return &pb.AgentAccountStatusInfo{
		Id:                info.ID,
		AccountId:         info.AccountID,
		AgentDefinitionId: info.AgentDefinitionID,
		StatusCode:        info.StatusCode,
		CreatedAt:         info.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:         info.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
	}
}

func domainToProtoAgentDefinition(info *domain.AgentDefinitionInfo) *pb.AgentDefinitionInfo {
	pbTools := make([]*pb.AgentDefinitionToolInfo, 0, len(info.Tools))
	for _, t := range info.Tools {
		pbTools = append(pbTools, &pb.AgentDefinitionToolInfo{
			Id:                  t.ID,
			ToolSlug:            t.ToolSlug,
			DisplayName:         t.DisplayName,
			Description:         t.Description,
			ConfigSchemaJson:    string(t.ConfigSchema),
			Category:            t.Category,
			ConfigJson:          string(t.Config),
			SortOrder:           t.SortOrder,
			RequireReview:       t.RequireReview,
			GroupId:             t.GroupID,
			GroupName:           t.GroupName,
			RequiredPermissions: t.RequiredPermissions,
		})
	}

	return &pb.AgentDefinitionInfo{
		Id:             info.ID,
		Name:           info.Name,
		Slug:           info.Slug,
		Description:    info.Description,
		DefinitionType: info.DefinitionType,
		CategoryCode:   info.CategoryCode,
		TriggerType:    info.TriggerType,
		IsEditable:     info.IsEditable,
		ConfigJson:     string(info.Config),
		Tools:          pbTools,
		CreatedAt:      info.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:      info.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
		RoleId:         info.RoleID,
		AccountStatus:  domainToProtoAccountStatus(info.AccountStatus),
	}
}
