package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/augno/api/services/agent-service/internal/domain"
	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	"github.com/jackc/pgx/v5/pgtype"
)

var agentDefSvcTracer = tracing.GetTracer("agent-service.agent_definition_service")

// auditIncludes lists the includes needed to fully populate audited fields
// (e.g. Config). These are always loaded when building old/new snapshots for
// audit diffs so that the comparison is accurate.
var auditIncludes = []string{"config"}

// mergeIncludes returns a combined list of includes with duplicates removed.
func mergeIncludes(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

type PlanGate interface {
	CanUseAgents(ctx context.Context, accountID string) (bool, error)
}

type agentDefSvcImpl struct {
	repos           domain.RepoFactory
	txManager       TransactionManager
	mediatorFactory domain.MediatorFactory
	planGate        PlanGate
}

type AgentDefinitionSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
	PlanGate        PlanGate
}

func NewAgentDefinitionSvc(config *AgentDefinitionSvcConfig) domain.AgentDefinitionSvc {
	if config.Repos == nil {
		panic(fmt.Errorf("agent definition service: repos is required"))
	}
	if config.MediatorFactory == nil {
		panic(fmt.Errorf("agent definition service: mediator factory is required"))
	}
	if config.TxManager == nil {
		panic(fmt.Errorf("agent definition service: tx manager is required"))
	}

	return &agentDefSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		planGate:        config.PlanGate,
	}
}

func (s *agentDefSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *agentDefSvcImpl) withTx(ctx context.Context, fn func(context.Context, *agentDefSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &agentDefSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			planGate:        s.planGate,
		}
		return fn(txCtx, txSvc)
	})
}

// CreateCustomAgent creates a new custom agent definition with its associated tool links, with idempotency support.
//
// 1. Extract the caller's identity and upsert an idempotency key; if already finished, return the cached response.
// 2. Validate that all referenced tool definitions exist.
// 3. Generate a unique agent definition ID.
// 4. Insert the agent definition record with config, category, trigger type, and role.
// 5. Create agent-definition-tool links for each tool in the params.
// 6. Build and cache the result, then return the created agent definition.
func (s *agentDefSvcImpl) CreateCustomAgent(ctx context.Context, params domain.CreateCustomAgentParams) (*domain.AgentDefinitionInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AgentDefinitionInfo](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		toolDefRepo := s.repos.NewToolDefinitionRepo()

		for _, t := range params.Tools {
			if _, apiErr := toolDefRepo.GetByID(ctx, t.ToolID); apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationErrorWithParam("Tool not found: "+t.ToolID, "tools"))
			}
		}

		defID, genErr := id.GenID(id.AgentDefinitionIDPrefix, nil)
		if genErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, genErr)
		}

		configJSON := json.RawMessage(`{}`)
		if params.ConfigJSON != "" {
			configJSON = json.RawMessage(params.ConfigJSON)
		}

		var result *domain.AgentDefinitionInfo
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			defRepo := txSvc.repos.NewAgentDefinitionRepo()
			adtRepo := txSvc.repos.NewAgentDefinitionToolRepo()
			statusRepo := txSvc.repos.NewAgentAccountStatusRepo()

			if apiErr := defRepo.Insert(txCtx, sqlc.InsertAgentDefinitionParams{
				ID:             defID,
				AccountID:      agentdb.PgText(accountID),
				Name:           params.Name,
				Slug:           params.Slug,
				Description:    agentdb.PgText(params.Description),
				DefinitionType: domain.DefinitionTypeCustom,
				CategoryCode:   params.CategoryCode,
				TriggerType:    params.TriggerType,
				IsActive:       true,
				Config:         configJSON,
				RoleID:         agentdb.PgText(params.RoleID),
			}); apiErr != nil {
				return apiErr
			}

			for _, t := range params.Tools {
				linkID, linkGenErr := id.GenID(id.AgentDefinitionToolIDPrefix, nil)
				if linkGenErr != nil {
					return linkGenErr
				}
				toolConfig := json.RawMessage(`{}`)
				if t.ConfigJSON != "" {
					toolConfig = json.RawMessage(t.ConfigJSON)
				}
				if apiErr := adtRepo.Insert(txCtx, sqlc.InsertAgentDefinitionToolParams{
					ID:                linkID,
					AgentDefinitionID: defID,
					ToolDefinitionID:  t.ToolID,
					Config:            toolConfig,
					SortOrder:         t.SortOrder,
					RequireReview:     t.RequireReview,
				}); apiErr != nil {
					return apiErr
				}
			}

			statusID, statusGenErr := id.GenID(id.AgentAccountStatusIDPrefix, nil)
			if statusGenErr != nil {
				return statusGenErr
			}
			if apiErr := statusRepo.Upsert(txCtx, sqlc.UpsertAgentAccountStatusParams{
				ID:                statusID,
				AccountID:         accountID,
				AgentDefinitionID: defID,
				StatusCode:        string(constants.AgentAccountStatusActive),
			}); apiErr != nil {
				return apiErr
			}

			built, apiErr := txSvc.buildResultForAccount(txCtx, defID, accountID, params.Includes)
			if apiErr != nil {
				return apiErr
			}
			result = built

			changes := audit.ComputeChanges(nil, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAgentDefinition,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// UpdateCustomAgent modifies an existing custom agent definition and replaces its tool links, with idempotency support.
//
// 1. Extract the caller's identity and upsert an idempotency key; if already finished, return the cached response.
// 2. Verify the agent definition exists, is custom (not system), and belongs to the caller's account.
// 3. Validate that all referenced tool definitions exist.
// 4. Update the agent definition's fields (name, slug, description, category, trigger, config, role).
// 5. Delete all existing tool links and re-create them from the params.
// 6. Build and cache the result, then return the updated agent definition.
func (s *agentDefSvcImpl) UpdateCustomAgent(ctx context.Context, params domain.UpdateCustomAgentParams) (*domain.AgentDefinitionInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AgentDefinitionInfo](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		defRepo := s.repos.NewAgentDefinitionRepo()
		toolDefRepo := s.repos.NewToolDefinitionRepo()

		def, apiErr := defRepo.GetByID(ctx, params.AgentDefinitionID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent definition not found: "+params.AgentDefinitionID))
		}
		if def.DefinitionType != domain.DefinitionTypeCustom {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewAuthorizationError("Cannot edit system agent definitions."))
		}
		if !def.AccountID.Valid || def.AccountID.String != accountID {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewAuthorizationError("Agent definition does not belong to this account."))
		}

		if params.ToolsProvided {
			for _, t := range params.Tools {
				if _, toolErr := toolDefRepo.GetByID(ctx, t.ToolID); toolErr != nil {
					return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
						apierror.NewValidationErrorWithParam("Tool not found: "+t.ToolID, "tools"))
				}
			}
		}

		updateConfig := params.ConfigJSON != nil
		var configBytes []byte
		if updateConfig {
			raw := *params.ConfigJSON
			if raw == "" {
				raw = "{}"
			}
			configBytes = []byte(raw)
		}

		var result *domain.AgentDefinitionInfo
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txDefRepo := txSvc.repos.NewAgentDefinitionRepo()
			txAdtRepo := txSvc.repos.NewAgentDefinitionToolRepo()

			old, apiErr := txSvc.buildResult(txCtx, params.AgentDefinitionID, auditIncludes)
			if apiErr != nil {
				return apiErr
			}

			if updateErr := txDefRepo.Update(txCtx, sqlc.UpdateAgentDefinitionParams{
				Name:         agentdb.PgTextPtr(params.Name),
				Slug:         agentdb.PgTextPtr(params.Slug),
				Description:  agentdb.PgTextPtr(params.Description),
				CategoryCode: agentdb.PgTextPtr(params.CategoryCode),
				TriggerType:  agentdb.PgTextPtr(params.TriggerType),
				UpdateConfig: updateConfig,
				NewConfig:    configBytes,
				RoleID:       agentdb.PgTextPtr(params.RoleID),
				ID:           params.AgentDefinitionID,
				AccountID:    agentdb.PgText(accountID),
			}); updateErr != nil {
				return updateErr
			}

			if params.ToolsProvided {
				if deleteErr := txAdtRepo.DeleteByAgentID(txCtx, params.AgentDefinitionID); deleteErr != nil {
					return deleteErr
				}

				for _, t := range params.Tools {
					linkID, linkGenErr := id.GenID(id.AgentDefinitionToolIDPrefix, nil)
					if linkGenErr != nil {
						return linkGenErr
					}
					toolConfig := json.RawMessage(`{}`)
					if t.ConfigJSON != "" {
						toolConfig = json.RawMessage(t.ConfigJSON)
					}
					if insertErr := txAdtRepo.Insert(txCtx, sqlc.InsertAgentDefinitionToolParams{
						ID:                linkID,
						AgentDefinitionID: params.AgentDefinitionID,
						ToolDefinitionID:  t.ToolID,
						Config:            toolConfig,
						SortOrder:         t.SortOrder,
						RequireReview:     t.RequireReview,
					}); insertErr != nil {
						return insertErr
					}
				}
			}

			built, apiErr := txSvc.buildResult(txCtx, params.AgentDefinitionID, mergeIncludes(params.Includes, auditIncludes))
			if apiErr != nil {
				return apiErr
			}
			result = built

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAgentDefinition,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// DeleteCustomAgent soft-deletes a custom agent definition.
//
// 1. Extract the caller's identity and verify permissions.
// 2. Verify the agent definition exists, is custom (not system), and belongs to the caller's account.
// 3. Soft-delete the agent definition record.
func (s *agentDefSvcImpl) DeleteCustomAgent(ctx context.Context, params domain.DeleteCustomAgentParams) *apierror.APIError {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	defRepo := s.repos.NewAgentDefinitionRepo()

	def, apiErr := defRepo.GetByID(ctx, params.AgentDefinitionID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeCustomAgent, params.AgentDefinitionID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This custom agent has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Agent definition not found: "+params.AgentDefinitionID))
	}
	if def.DefinitionType != domain.DefinitionTypeCustom {
		return tracing.Trace(span, apierror.NewAuthorizationError("Cannot delete system agent definitions."))
	}
	if !def.AccountID.Valid || def.AccountID.String != accountID {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Agent definition not found: "+params.AgentDefinitionID))
	}

	oldInfo, buildErr := s.buildResultForAccount(ctx, params.AgentDefinitionID, accountID, auditIncludes)
	if buildErr != nil {
		return tracing.Trace(span, buildErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
		txDefRepo := txSvc.repos.NewAgentDefinitionRepo()
		statusRepo := txSvc.repos.NewAgentAccountStatusRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeCustomAgent, def.ID, def); apiErr != nil {
			return apiErr
		}

		if deleteErr := txDefRepo.SoftDelete(txCtx, params.AgentDefinitionID, accountID); deleteErr != nil {
			return deleteErr
		}
		if deleteErr := statusRepo.DeleteByAccountAndDefinition(txCtx, accountID, params.AgentDefinitionID); deleteErr != nil {
			return deleteErr
		}

		changes := audit.ComputeChanges(oldInfo, (*domain.AgentDefinitionInfo)(nil))

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAgentDefinition,
			ResourceID:   oldInfo.ID,
			Changes:      changes,
		})
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// GetAgentDefinition retrieves a single agent definition by ID, verifying account ownership for custom definitions.
//
// 1. Fetch the agent definition by ID from the repository.
// 2. If the definition is account-scoped and belongs to a different account, return not-found.
// 3. Build and return the result with optional includes (tools, config).
func (s *agentDefSvcImpl) GetAgentDefinition(ctx context.Context, agentDefinitionID string, includes []string) (*domain.AgentDefinitionInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	defRepo := s.repos.NewAgentDefinitionRepo()

	def, apiErr := defRepo.GetByID(ctx, agentDefinitionID)
	if apiErr != nil {
		return nil, apierror.NewResourceNotFoundError("Agent definition not found: " + agentDefinitionID)
	}

	if def.AccountID.Valid && def.AccountID.String != accountID {
		return nil, apierror.NewResourceNotFoundError("Agent definition not found: " + agentDefinitionID)
	}

	return s.buildResultForAccount(ctx, def.ID, accountID, includes)
}

// ListAgentDefinitions returns all agent definitions visible to the given account (system + custom).
//
// 1. Query the repository for all definitions accessible by the account with cursor pagination.
// 2. For each definition, optionally fetch associated tools if "tools" is in the includes list.
// 3. Convert and return the results as domain objects with page info.
func (s *agentDefSvcImpl) ListAgentDefinitions(ctx context.Context, params domain.ListAgentDefinitionsParams) (*domain.ListAgentDefinitionsResult, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	filterQuery := params.Query != nil && *params.Query != ""
	search := pgtype.Text{}
	if filterQuery {
		search = pgtype.Text{String: *params.Query, Valid: true}
	}

	hasCursor := params.Cursor != nil && *params.Cursor != ""
	cursorID := ""
	if hasCursor {
		cursorID = *params.Cursor
	}

	defRepo := s.repos.NewAgentDefinitionRepo()

	sqlParams := sqlc.ListAgentDefinitionsByAccountCursorParams{
		AccountID:            accountID,
		FilterDefinitionType: len(params.DefinitionTypes) > 0,
		DefinitionTypes:      params.DefinitionTypes,
		FilterTriggerType:    len(params.TriggerTypes) > 0,
		TriggerTypes:         params.TriggerTypes,
		FilterStatus:         len(params.Statuses) > 0,
		StatusCodes:          params.Statuses,
		FilterQuery:          filterQuery,
		Search:               search,
		HasCursor:            hasCursor,
		CursorID:             cursorID,
		Lim:                  limit + 1,
	}

	defs, apiErr := defRepo.ListByAccountCursor(ctx, sqlParams)
	if apiErr != nil {
		return nil, apiErr
	}

	// Detect next page via limit+1 pattern.
	hasNextPage := len(defs) > int(limit)
	if hasNextPage {
		defs = defs[:limit]
	}

	// Pre-fetch account statuses for the response.
	var statusMap map[string]*sqlc.AgentAccountStatus
	statusRepo := s.repos.NewAgentAccountStatusRepo()
	allStatuses, statusErr := statusRepo.ListByAccount(ctx, accountID)
	if statusErr == nil {
		statusMap = make(map[string]*sqlc.AgentAccountStatus, len(allStatuses))
		for i := range allStatuses {
			statusMap[allStatuses[i].AgentDefinitionID] = &allStatuses[i]
		}
	}

	results := make([]domain.AgentDefinitionInfo, 0, len(defs))
	for _, def := range defs {
		var tools []sqlc.ListToolsByAgentDefinitionIDRow
		if includesContains(params.Includes, "tools") {
			adtRepo := s.repos.NewAgentDefinitionToolRepo()
			var toolErr *apierror.APIError
			tools, toolErr = adtRepo.ListByAgentDefinitionID(ctx, def.ID)
			if toolErr != nil {
				return nil, toolErr
			}
		}
		var accountStatus *sqlc.AgentAccountStatus
		if statusMap != nil {
			accountStatus = statusMap[def.ID]
		}
		results = append(results, *sqlToAgentDefinitionInfo(&def, tools, accountStatus, params.Includes))
	}

	// Build page info.
	var nextCursor *string
	if hasNextPage && len(results) > 0 {
		lastID := results[len(results)-1].ID
		nextCursor = &lastID
	}
	var prevCursor *string
	if hasCursor && len(results) > 0 {
		firstID := results[0].ID
		prevCursor = &firstID
	}

	return &domain.ListAgentDefinitionsResult{
		Items: results,
		PageInfo: domain.PageInfo{
			NextCursor:  nextCursor,
			PrevCursor:  prevCursor,
			HasNextPage: hasNextPage,
			HasPrevPage: hasCursor,
		},
	}, nil
}

// ListAvailableTools returns all tool definitions that can be attached to agent definitions.
//
// 1. Fetch all tool definitions from the repository.
// 2. Map each tool to a domain AvailableToolInfo with display name, description, config schema, and category.
func (s *agentDefSvcImpl) ListAvailableTools(ctx context.Context, params domain.ListAvailableToolsParams) ([]domain.AvailableToolInfo, []domain.ToolGroupInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.list_available_tools")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionRead); apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	toolDefRepo := s.repos.NewToolDefinitionRepo()
	tools, apiErr := toolDefRepo.ListAll(ctx)
	if apiErr != nil {
		return nil, nil, apiErr
	}

	results := make([]domain.AvailableToolInfo, 0, len(tools))
	for _, t := range tools {
		desc := ""
		if t.Description.Valid {
			desc = t.Description.String
		}
		var groupID string
		if t.ToolGroupID.Valid {
			groupID = t.ToolGroupID.String
		}
		var groupName string
		if t.GroupName.Valid {
			groupName = t.GroupName.String
		}
		results = append(results, domain.AvailableToolInfo{
			ID:                  t.ID,
			DisplayName:         t.DisplayName,
			Description:         desc,
			ConfigSchema:        t.ConfigSchema,
			Category:            t.Category,
			GroupID:             groupID,
			GroupName:           groupName,
			RequiredPermissions: unmarshalPermissions(t.RequiredPermissions),
		})
	}

	groups, groupErr := toolDefRepo.ListToolGroups(ctx)
	if groupErr != nil {
		return nil, nil, groupErr
	}
	domainGroups := make([]domain.ToolGroupInfo, 0, len(groups))
	for _, g := range groups {
		desc := ""
		if g.Description.Valid {
			desc = g.Description.String
		}
		icon := ""
		if g.Icon.Valid {
			icon = g.Icon.String
		}
		domainGroups = append(domainGroups, domain.ToolGroupInfo{
			ID:          g.ID,
			Name:        g.Name,
			Description: desc,
			Slug:        g.Slug,
			Icon:        icon,
			SortOrder:   g.SortOrder,
		})
	}

	// Apply query filter to tools and groups.
	if params.Query != nil && *params.Query != "" {
		q := strings.ToLower(*params.Query)
		filtered := make([]domain.AvailableToolInfo, 0, len(results))
		for _, t := range results {
			if strings.Contains(strings.ToLower(t.DisplayName), q) || strings.Contains(strings.ToLower(t.GroupName), q) {
				filtered = append(filtered, t)
			}
		}
		results = filtered

		filteredGroups := make([]domain.ToolGroupInfo, 0, len(domainGroups))
		for _, g := range domainGroups {
			if strings.Contains(strings.ToLower(g.Name), q) {
				filteredGroups = append(filteredGroups, g)
			}
		}
		domainGroups = filteredGroups
	}

	// Apply cursor-based pagination to tools.
	if params.Cursor != nil && *params.Cursor != "" {
		cursor := *params.Cursor
		idx := -1
		for i, t := range results {
			if t.ID == cursor {
				idx = i
				break
			}
		}

		gIdx := -1
		for i, g := range domainGroups {
			if g.ID == cursor {
				gIdx = i
				break
			}
		}

		if idx == -1 && gIdx == -1 {
			return nil, nil, apierror.NewValidationError("Invalid pagination cursor.")
		}

		if idx >= 0 && idx+1 < len(results) {
			results = results[idx+1:]
		} else {
			results = nil
		}

		if gIdx >= 0 && gIdx+1 < len(domainGroups) {
			domainGroups = domainGroups[gIdx+1:]
		} else if gIdx >= 0 {
			domainGroups = nil
		}
	}

	// Apply limit to tools and groups.
	if params.Limit > 0 {
		if int(params.Limit) < len(results) {
			results = results[:params.Limit]
		}
		if int(params.Limit) < len(domainGroups) {
			domainGroups = domainGroups[:params.Limit]
		}
	}

	return results, domainGroups, nil
}

func (s *agentDefSvcImpl) UpdateAgentAccountStatus(ctx context.Context, params domain.UpdateAgentAccountStatusParams) (*domain.AgentAccountStatusInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.update_account_status")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AgentAccountStatusInfo](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		defRepo := s.repos.NewAgentDefinitionRepo()
		if _, apiErr := defRepo.GetByID(ctx, params.AgentDefinitionID); apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent definition not found: "+params.AgentDefinitionID))
		}

		statusID, genErr := id.GenID(id.AgentAccountStatusIDPrefix, nil)
		if genErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, genErr)
		}

		// Fetch old status before upsert for audit diff (may not exist yet).
		var oldStatus *domain.AgentAccountStatusInfo
		statusRepo := s.repos.NewAgentAccountStatusRepo()
		if oldRow, oldErr := statusRepo.GetByAccountAndDefinition(ctx, accountID, params.AgentDefinitionID); oldErr == nil {
			oldStatus = sqlToAgentAccountStatusInfo(oldRow)
		}

		var result *domain.AgentAccountStatusInfo
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txStatusRepo := txSvc.repos.NewAgentAccountStatusRepo()

			if apiErr := txStatusRepo.Upsert(txCtx, sqlc.UpsertAgentAccountStatusParams{
				ID:                statusID,
				AccountID:         accountID,
				AgentDefinitionID: params.AgentDefinitionID,
				StatusCode:        params.StatusCode,
			}); apiErr != nil {
				return apiErr
			}

			row, apiErr := txStatusRepo.GetByAccountAndDefinition(txCtx, accountID, params.AgentDefinitionID)
			if apiErr != nil {
				return apiErr
			}
			result = sqlToAgentAccountStatusInfo(row)

			auditAction := constants.AuditActionUpdate
			if oldStatus == nil {
				auditAction = constants.AuditActionCreate
			}

			changes := audit.ComputeChanges(oldStatus, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       auditAction,
				ResourceType: constants.ObjectTypeAgentAccountStatus,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// TriggerRun creates an agent run and publishes an outbox message to execute it, with idempotency support.
//
// 1. Look up the agent definition by slug.
// 2. Find or create a config for the account + definition.
// 3. Insert the agent run record and outbox message within a transaction.
// 4. Cache the success response for idempotent replay.
func (s *agentDefSvcImpl) TriggerRun(ctx context.Context, params domain.TriggerRunParams) (string, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.trigger_run")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return "", tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return "", apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[string](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return "", tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Error != nil {
			return "", cached.Error
		}
		if cached.Data != nil {
			return *cached.Data, nil
		}
		return "", nil

	case domain.RecoveryPointStarted:
		if s.planGate != nil {
			allowed, gateErr := s.planGate.CanUseAgents(ctx, accountID)
			if gateErr != nil {
				return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewInternalError(gateErr, "Failed to check plan eligibility."))
			}
			if !allowed {
				return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationError("Agent usage is not available on your current plan. Please upgrade to use agents."))
			}
		}

		defRepo := s.repos.NewAgentDefinitionRepo()
		configRepo := s.repos.NewAgentConfigRepo()

		def, defErr := defRepo.GetBySlug(ctx, params.AgentDefinitionCode)
		if defErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent definition not found: "+params.AgentDefinitionCode))
		}

		config, cfgErr := configRepo.GetByAccountAndDefinition(ctx, accountID, def.ID)
		if cfgErr != nil {
			configID, genErr := id.GenID(id.AgentConfigIDPrefix, nil)
			if genErr != nil {
				return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, genErr)
			}
			if insertErr := configRepo.Insert(ctx, sqlc.InsertAgentConfigParams{
				ID:                configID,
				AccountID:         accountID,
				AgentDefinitionID: def.ID,
				IsEnabled:         true,
				Config:            json.RawMessage(`{}`),
			}); insertErr != nil {
				return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, insertErr)
			}
			config, cfgErr = configRepo.GetByID(ctx, configID)
			if cfgErr != nil {
				return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, cfgErr)
			}
		}

		runID, genErr := id.GenID(id.AgentRunIDPrefix, nil)
		if genErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, genErr)
		}

		runInput := json.RawMessage(`{}`)
		if params.Input != "" {
			inputJSON, _ := json.Marshal(map[string]string{"message": params.Input})
			runInput = inputJSON
		}

		length := id.IDLength22
		msgID, msgGenErr := id.GenID(id.MessageIDPrefix, &length)
		if msgGenErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, msgGenErr)
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txRunRepo := txSvc.repos.NewAgentRunRepo()
			txOutbox := txSvc.repos.NewOutboxRepo()

			if insertErr := txRunRepo.Insert(txCtx, sqlc.InsertAgentRunParams{
				ID:                      runID,
				AccountID:               accountID,
				AgentDefinitionID:       def.ID,
				AgentConfigID:           agentdb.PgText(config.ID),
				StatusCode:              domain.RunStatusPending,
				TriggerType:             domain.TriggerManual,
				Input:                   runInput,
				Output:                  json.RawMessage(`{}`),
				TriggeredByActorID:      agentdb.PgText(identity.Actor.ID),
				TriggeredByIdentityType: agentdb.PgText(string(identity.Type)),
				TriggeredByActorName:    agentdb.PgTextPtr(identity.Actor.Name),
			}); insertErr != nil {
				return insertErr
			}

			data := messaging.AgentExecuteRunData{
				AgentRunID:    runID,
				AgentConfigID: config.ID,
				AccountID:     accountID,
				TriggerType:   domain.TriggerManual,
			}
			dataBytes, _ := json.Marshal(data)

			if _, outboxErr := txOutbox.Create(txCtx, messaging.OutboxMessageInput{
				MessageID:   msgID,
				ServiceName: domain.ServiceName,
				MessageType: string(contracts.AgentCmdExecuteRun),
				Destination: messaging.ApplicationExchange,
				RoutingKey:  string(contracts.AgentCmdExecuteRun),
				Payload: contracts.AmqpMessage{
					Data:      dataBytes,
					MessageID: msgID,
				},
				MaxAttempts: 3,
			}); outboxErr != nil {
				return apierror.NewInternalError(outboxErr, "Failed to write outbox message.")
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, runID)
		})

		if apiErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return runID, nil

	default:
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// CancelRun cancels a pending or running agent run, with idempotency support.
func (s *agentDefSvcImpl) CancelRun(ctx context.Context, params domain.CancelRunParams) *apierror.APIError {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.cancel_run")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Error != nil {
			return cached.Error
		}
		return nil

	case domain.RecoveryPointStarted:
		runRepo := s.repos.NewAgentRunRepo()
		run, runErr := runRepo.GetByID(ctx, params.AgentRunID)
		if runErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent run not found."))
		}
		if run.AccountID != accountID {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent run not found."))
		}
		if run.StatusCode != domain.RunStatusPending && run.StatusCode != domain.RunStatusRunning {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError("Run cannot be cancelled (status: "+run.StatusCode+")."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txRunRepo := txSvc.repos.NewAgentRunRepo()
			if updateErr := txRunRepo.UpdateStatus(txCtx, params.AgentRunID, domain.RunStatusCancelled); updateErr != nil {
				return updateErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// ContinueRun continues an agent run that is awaiting input, with idempotency support.
//
// 1. Validate the run exists, belongs to the account, and is awaiting input.
// 2. Update status to running and create an outbox message atomically.
// 3. Cache the success response for idempotent replay.
func (s *agentDefSvcImpl) ContinueRun(ctx context.Context, params domain.ContinueRunParams) (string, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.continue_run")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return "", tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return "", apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[string](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return "", tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Error != nil {
			return "", cached.Error
		}
		if cached.Data != nil {
			return *cached.Data, nil
		}
		return "", nil

	case domain.RecoveryPointStarted:
		runRepo := s.repos.NewAgentRunRepo()
		run, runErr := runRepo.GetByID(ctx, params.AgentRunID)
		if runErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent run not found."))
		}
		if run.AccountID != accountID {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent run not found."))
		}
		if run.StatusCode != domain.RunStatusAwaitingInput && run.StatusCode != domain.RunStatusAwaitingApproval {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError(fmt.Sprintf("Run is not awaiting input or approval (status: %s).", run.StatusCode)))
		}

		length := id.IDLength22
		msgID, msgGenErr := id.GenID(id.MessageIDPrefix, &length)
		if msgGenErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, msgGenErr)
		}

		// Extract actor info to propagate through the message queue
		var actorID, actorType, actorName string
		if identity.Actor != nil {
			actorID = identity.Actor.ID
			actorType = string(identity.Type)
			if identity.Actor.Name != nil {
				actorName = *identity.Actor.Name
			}
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txRunRepo := txSvc.repos.NewAgentRunRepo()
			txOutbox := txSvc.repos.NewOutboxRepo()

			if updateErr := txRunRepo.UpdateStatus(txCtx, params.AgentRunID, domain.RunStatusRunning); updateErr != nil {
				return updateErr
			}

			data := messaging.AgentContinueRunData{
				AgentRunID:        params.AgentRunID,
				AccountID:         accountID,
				Message:           params.Message,
				ApprovedToolSlugs: params.ApprovedToolSlugs,
				AllowedToolSlugs:  params.AllowedToolSlugs,
				ActorID:           actorID,
				ActorType:         actorType,
				ActorName:         actorName,
			}
			dataBytes, _ := json.Marshal(data)

			if _, outboxErr := txOutbox.Create(txCtx, messaging.OutboxMessageInput{
				MessageID:   msgID,
				ServiceName: domain.ServiceName,
				MessageType: string(contracts.AgentCmdContinueRun),
				Destination: messaging.ApplicationExchange,
				RoutingKey:  string(contracts.AgentCmdContinueRun),
				Payload: contracts.AmqpMessage{
					Data:      dataBytes,
					MessageID: msgID,
				},
				MaxAttempts: 3,
			}); outboxErr != nil {
				return apierror.NewInternalError(outboxErr, "Failed to write outbox message.")
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, params.AgentRunID)
		})

		if apiErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return params.AgentRunID, nil

	default:
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// CreateAgentMemory creates a new agent memory record, with idempotency support.
func (s *agentDefSvcImpl) CreateAgentMemory(ctx context.Context, params domain.CreateAgentMemoryParams) (*domain.AgentMemoryInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.create_agent_memory")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AgentMemoryInfo](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		memoryID, genErr := id.GenID(id.AgentMemoryIDPrefix, nil)
		if genErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, genErr)
		}

		var metadata []byte
		if params.MetadataJSON != "" {
			metadata = []byte(params.MetadataJSON)
		} else {
			metadata = []byte("{}")
		}

		var expiresAt pgtype.Timestamptz
		if params.ExpiresAt != "" {
			t, parseErr := parseTimestamp(params.ExpiresAt)
			if parseErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationErrorWithParam("Invalid expires_at: "+parseErr.Error(), "expires_at"))
			}
			expiresAt = pgtype.Timestamptz{Time: t, Valid: true}
		}

		var result *domain.AgentMemoryInfo
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			memoryRepo := txSvc.repos.NewAgentMemoryRepo()

			if insertErr := memoryRepo.Insert(txCtx, sqlc.InsertAgentMemoryParams{
				ID:         memoryID,
				AccountID:  accountID,
				Category:   params.Category,
				Content:    params.Content,
				Metadata:   metadata,
				EntityType: pgtype.Text{String: params.EntityType, Valid: params.EntityType != ""},
				EntityID:   pgtype.Text{String: params.EntityID, Valid: params.EntityID != ""},
				Importance: params.Importance,
				ExpiresAt:  expiresAt,
			}); insertErr != nil {
				return insertErr
			}

			memory, getErr := memoryRepo.GetByID(txCtx, memoryID)
			if getErr != nil {
				return getErr
			}
			result = sqlcMemoryToDomain(memory)

			changes := audit.ComputeChanges(nil, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAgentMemory,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// AcknowledgeAgentAlert acknowledges an agent alert, with idempotency support.
func (s *agentDefSvcImpl) AcknowledgeAgentAlert(ctx context.Context, params domain.AcknowledgeAgentAlertParams) (*domain.AgentAlertInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.acknowledge_agent_alert")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AgentAlertInfo](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		alertRepo := s.repos.NewAgentAlertRepo()
		alert, alertErr := alertRepo.GetByID(ctx, params.AlertID)
		if alertErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent alert not found."))
		}
		if alert.AccountID != accountID {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent alert not found."))
		}

		acknowledgedBy := ""
		acknowledgedByActorType := ""
		acknowledgedByActorName := ""
		if identity.Actor != nil {
			acknowledgedBy = identity.Actor.ID
			acknowledgedByActorType = string(identity.Type)
			if identity.Actor.Name != nil {
				acknowledgedByActorName = *identity.Actor.Name
			}
		}

		var result *domain.AgentAlertInfo
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txAlertRepo := txSvc.repos.NewAgentAlertRepo()

			if ackErr := txAlertRepo.Acknowledge(txCtx, sqlc.AcknowledgeAgentAlertParams{
				AcknowledgedByActorID:   pgtype.Text{String: acknowledgedBy, Valid: acknowledgedBy != ""},
				AcknowledgedByActorType: pgtype.Text{String: acknowledgedByActorType, Valid: acknowledgedByActorType != ""},
				AcknowledgedByActorName: pgtype.Text{String: acknowledgedByActorName, Valid: acknowledgedByActorName != ""},
				ID:                      params.AlertID,
				AccountID:               accountID,
			}); ackErr != nil {
				return ackErr
			}

			updated, getErr := txAlertRepo.GetByID(txCtx, params.AlertID)
			if getErr != nil {
				return getErr
			}
			result = sqlcAlertToDomain(updated)

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func sqlcMemoryToDomain(m *sqlc.AgentMemory) *domain.AgentMemoryInfo {
	var metadataStr string
	if m.Metadata != nil {
		metadataStr = string(m.Metadata)
	}
	var entityType, entityID string
	if m.EntityType.Valid {
		entityType = m.EntityType.String
	}
	if m.EntityID.Valid {
		entityID = m.EntityID.String
	}
	var expiresAt string
	if m.ExpiresAt.Valid {
		expiresAt = m.ExpiresAt.Time.Format("2006-01-02T15:04:05.000Z")
	}
	var createdAt string
	if m.CreatedAt.Valid {
		createdAt = m.CreatedAt.Time.Format("2006-01-02T15:04:05.000Z")
	}
	var updatedAt string
	if m.UpdatedAt.Valid {
		updatedAt = m.UpdatedAt.Time.Format("2006-01-02T15:04:05.000Z")
	}
	return &domain.AgentMemoryInfo{
		ID:         m.ID,
		AccountID:  m.AccountID,
		Category:   m.Category,
		Content:    m.Content,
		Metadata:   metadataStr,
		EntityType: entityType,
		EntityID:   entityID,
		Importance: m.Importance,
		ExpiresAt:  expiresAt,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

func sqlcAlertToDomain(a *sqlc.AgentAlert) *domain.AgentAlertInfo {
	var metadataStr string
	if a.Metadata != nil {
		metadataStr = string(a.Metadata)
	}
	pgText := func(t pgtype.Text) string {
		if t.Valid {
			return t.String
		}
		return ""
	}
	pgTs := func(t pgtype.Timestamptz) string {
		if t.Valid {
			return t.Time.Format("2006-01-02T15:04:05.000Z")
		}
		return ""
	}
	return &domain.AgentAlertInfo{
		ID:                      a.ID,
		AccountID:               a.AccountID,
		AgentRunID:              pgText(a.AgentRunID),
		AgentActionID:           pgText(a.AgentActionID),
		SeverityCode:            a.SeverityCode,
		StatusCode:              a.StatusCode,
		Title:                   a.Title,
		Message:                 pgText(a.Message),
		Metadata:                metadataStr,
		AcknowledgedAt:          pgTs(a.AcknowledgedAt),
		AcknowledgedBy:          pgText(a.AcknowledgedByActorID),
		AcknowledgedByActorType: pgText(a.AcknowledgedByActorType),
		AcknowledgedByActorName: pgText(a.AcknowledgedByActorName),
		CreatedAt:               pgTs(a.CreatedAt),
		UpdatedAt:               pgTs(a.UpdatedAt),
	}
}

func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format: %s", s)
}

func sqlToAgentAccountStatusInfo(row *sqlc.AgentAccountStatus) *domain.AgentAccountStatusInfo {
	return &domain.AgentAccountStatusInfo{
		ID:                row.ID,
		AccountID:         row.AccountID,
		AgentDefinitionID: row.AgentDefinitionID,
		StatusCode:        row.StatusCode,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func (s *agentDefSvcImpl) buildResult(ctx context.Context, defID string, includes []string) (*domain.AgentDefinitionInfo, *apierror.APIError) {
	return s.buildResultForAccount(ctx, defID, "", includes)
}

func (s *agentDefSvcImpl) buildResultForAccount(ctx context.Context, defID, accountID string, includes []string) (*domain.AgentDefinitionInfo, *apierror.APIError) {
	defRepo := s.repos.NewAgentDefinitionRepo()

	def, apiErr := defRepo.GetByID(ctx, defID)
	if apiErr != nil {
		return nil, apiErr
	}

	var tools []sqlc.ListToolsByAgentDefinitionIDRow
	if includesContains(includes, "tools") {
		adtRepo := s.repos.NewAgentDefinitionToolRepo()
		var toolErr *apierror.APIError
		tools, toolErr = adtRepo.ListByAgentDefinitionID(ctx, defID)
		if toolErr != nil {
			return nil, toolErr
		}
	}

	var accountStatus *sqlc.AgentAccountStatus
	if accountID != "" {
		statusRepo := s.repos.NewAgentAccountStatusRepo()
		row, statusErr := statusRepo.GetByAccountAndDefinition(ctx, accountID, defID)
		if statusErr == nil {
			accountStatus = row
		}
	}

	return sqlToAgentDefinitionInfo(def, tools, accountStatus, includes), nil
}

// includesContains returns true when key is present in the includes slice.
// A nil or empty includes slice means "include nothing".
func includesContains(includes []string, key string) bool {
	return slices.Contains(includes, key)
}

func unmarshalPermissions(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	var perms []string
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil
	}
	return perms
}

// sqlToAgentDefinitionInfo converts sqlc rows into a domain AgentDefinitionInfo.
// When includes is non-nil, fields not listed are set to nil.
func sqlToAgentDefinitionInfo(def *sqlc.AgentDefinition, tools []sqlc.ListToolsByAgentDefinitionIDRow, accountStatus *sqlc.AgentAccountStatus, includes []string) *domain.AgentDefinitionInfo {
	desc := ""
	if def.Description.Valid {
		desc = def.Description.String
	}

	var roleID string
	if def.RoleID.Valid {
		roleID = def.RoleID.String
	}

	var domainTools []domain.AgentDefinitionToolInfo
	if includesContains(includes, "tools") {
		domainTools = make([]domain.AgentDefinitionToolInfo, 0, len(tools))
		for _, t := range tools {
			toolDesc := ""
			if t.ToolDescription.Valid {
				toolDesc = t.ToolDescription.String
			}
			var groupID string
			if t.ToolGroupID.Valid {
				groupID = t.ToolGroupID.String
			}
			var groupName string
			if t.ToolGroupName.Valid {
				groupName = t.ToolGroupName.String
			}
			domainTools = append(domainTools, domain.AgentDefinitionToolInfo{
				ID:                  t.ID,
				ToolID:              t.ToolDefinitionID,
				DisplayName:         t.ToolDisplayName,
				Description:         toolDesc,
				ConfigSchema:        t.ToolConfigSchema,
				Category:            t.ToolCategory,
				Config:              t.Config,
				SortOrder:           t.SortOrder,
				RequireReview:       t.RequireReview,
				GroupID:             groupID,
				GroupName:           groupName,
				RequiredPermissions: unmarshalPermissions(t.RequiredPermissions),
			})
		}
	}

	var config json.RawMessage
	if includesContains(includes, "config") {
		config = def.Config
	}

	var domainAccountStatus *domain.AgentAccountStatusInfo
	if accountStatus != nil {
		domainAccountStatus = &domain.AgentAccountStatusInfo{
			ID:                accountStatus.ID,
			AccountID:         accountStatus.AccountID,
			AgentDefinitionID: accountStatus.AgentDefinitionID,
			StatusCode:        accountStatus.StatusCode,
			CreatedAt:         accountStatus.CreatedAt.Time,
			UpdatedAt:         accountStatus.UpdatedAt.Time,
		}
	}

	return &domain.AgentDefinitionInfo{
		ID:             def.ID,
		Name:           def.Name,
		Slug:           def.Slug,
		Description:    desc,
		DefinitionType: def.DefinitionType,
		CategoryCode:   def.CategoryCode,
		TriggerType:    def.TriggerType,
		IsEditable:     def.DefinitionType == domain.DefinitionTypeCustom,
		Config:         config,
		RoleID:         roleID,
		Tools:          domainTools,
		AccountStatus:  domainAccountStatus,
		CreatedAt:      def.CreatedAt.Time,
		UpdatedAt:      def.UpdatedAt.Time,
	}
}
