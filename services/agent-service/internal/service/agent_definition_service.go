package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/augno/api/services/agent-service/internal/agents"
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

// auditIncludes lists the includes needed to fully populate audited fields (e.g. Config). These are always loaded when building old/new snapshots for audit diffs so that the comparison is accurate.
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
	outboxNotifier  messaging.OutboxNotifier
}

type AgentDefinitionSvcConfig struct {
	// Repos (required) is the repository factory for agent persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used for access checks.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// PlanGate (optional; default: nil) checks whether an account's plan allows agents. When nil, plan gating is skipped and all accounts are allowed.
	PlanGate PlanGate

	// OutboxNotifier (optional; default: nil) wakes the outbox enqueuer the instant a chat run is enqueued, so the run starts (and its "thinking" indicator appears) without waiting out the enqueuer's idle poll backoff. When nil, the run is still picked up on the next poll.
	OutboxNotifier messaging.OutboxNotifier
}

func (c *AgentDefinitionSvcConfig) WithDefaults() *AgentDefinitionSvcConfig {
	if c == nil {
		c = &AgentDefinitionSvcConfig{}
	}
	return c
}

func (c *AgentDefinitionSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("agent definition service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("agent definition service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("agent definition service: tx manager is required")
	}
	return nil
}

func NewAgentDefinitionSvc(config *AgentDefinitionSvcConfig) domain.AgentDefinitionSvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &agentDefSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		planGate:        config.PlanGate,
		outboxNotifier:  config.OutboxNotifier,
	}
}

func (s *agentDefSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

// kickOutbox wakes the outbox enqueuer so a just-committed command (e.g. a chat-run execution) is published immediately rather than on the enqueuer's next idle poll, which can be up to MaxPollInterval away. No-op when no notifier was injected. Call only after the writing transaction has committed — kicking mid-transaction would race the poll against an as-yet-invisible row.
func (s *agentDefSvcImpl) kickOutbox() {
	if s.outboxNotifier != nil {
		s.outboxNotifier.Notify()
	}
}

func (s *agentDefSvcImpl) withTx(ctx context.Context, fn func(context.Context, *agentDefSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &agentDefSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			planGate:        s.planGate,
			outboxNotifier:  s.outboxNotifier,
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
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
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
		for _, t := range params.Tools {
			if _, ok := agents.LookupBuiltinTool(t.ToolSlug); !ok {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationErrorWithParam("Tool not found: "+t.ToolSlug, "tools"))
			}
		}
		if apiErr := validateEndpointToolSlugs(params.ConfigJSON); apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
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
				DefinitionType: string(constants.AgentDefinitionTypeCustom),
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
					ToolSlug:          t.ToolSlug,
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
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
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

		def, apiErr := defRepo.GetByID(ctx, params.AgentDefinitionID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent definition not found: "+params.AgentDefinitionID))
		}
		if def.DefinitionType != string(constants.AgentDefinitionTypeCustom) {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewAuthorizationError("Cannot edit system agent definitions."))
		}
		if !def.AccountID.Valid || def.AccountID.String != accountID {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewAuthorizationError("Agent definition does not belong to this account."))
		}

		if params.ToolsProvided {
			for _, t := range params.Tools {
				if _, ok := agents.LookupBuiltinTool(t.ToolSlug); !ok {
					return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
						apierror.NewValidationErrorWithParam("Tool not found: "+t.ToolSlug, "tools"))
				}
			}
		}

		if params.ConfigJSON != nil {
			if apiErr := validateEndpointToolSlugs(*params.ConfigJSON); apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
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

			// An empty description is stored as NULL, never "" (matches the create path's PgText behavior).
			descUpdate := agentdb.PgText("")
			if params.Description != nil {
				descUpdate = agentdb.PgText(*params.Description)
			}

			if updateErr := txDefRepo.Update(txCtx, sqlc.UpdateAgentDefinitionParams{
				Name:             agentdb.PgTextPtr(params.Name),
				Slug:             agentdb.PgTextPtr(params.Slug),
				Description:      descUpdate,
				ClearDescription: params.ClearDescription,
				CategoryCode:     agentdb.PgTextPtr(params.CategoryCode),
				TriggerType:      agentdb.PgTextPtr(params.TriggerType),
				UpdateConfig:     updateConfig,
				NewConfig:        configBytes,
				RoleID:           agentdb.PgTextPtr(params.RoleID),
				ClearRoleID:      params.ClearRoleID,
				ID:               params.AgentDefinitionID,
				AccountID:        agentdb.PgText(accountID),
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
						ToolSlug:          t.ToolSlug,
						Config:            toolConfig,
						SortOrder:         t.SortOrder,
						RequireReview:     t.RequireReview,
					}); insertErr != nil {
						return insertErr
					}
				}
			}

			built, apiErr := txSvc.buildResultForAccount(txCtx, params.AgentDefinitionID, accountID, mergeIncludes(params.Includes, auditIncludes))
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
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
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
	if def.DefinitionType != string(constants.AgentDefinitionTypeCustom) {
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
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
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
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
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
		return nil, nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgents, types.ActionRead); apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	// Built-in tools come from the code catalog (agents.BuiltinTools); they carry category "built_in" and a selection grants the tool via an agent_definition_tool link keyed by slug.
	results, domainGroups := builtinToolCatalogInfos()

	// Append the code-defined endpoint-tools (and their groups) so the catalog the UI shows includes every agent-grantable API operation. These carry category "api_endpoint"; selecting one grants it via the agent's endpoint_tool_slugs rather than a tool link.
	etTools, etGroups := endpointToolCatalogInfos()
	results = append(results, etTools...)
	domainGroups = append(domainGroups, etGroups...)

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

	// Each list route paginates exactly one resource type, so the cursor and limit must be scoped to that resource: a tools cursor only slices tools and a tool-groups cursor only slices groups. This prevents one resource's cursor from leaking into the other's slice and, critically, keeps a groups page from truncating the full tools set that feeds each group's ?include=tools.
	if params.PaginateResource == "tool_groups" {
		// Apply cursor-based pagination to groups only, validating the cursor against group ids.
		if params.Cursor != nil && *params.Cursor != "" {
			cursor := *params.Cursor
			gIdx := -1
			for i, g := range domainGroups {
				if g.ID == cursor {
					gIdx = i
					break
				}
			}

			if gIdx == -1 {
				return nil, nil, apierror.NewValidationError("Invalid pagination cursor.")
			}

			if gIdx+1 < len(domainGroups) {
				domainGroups = domainGroups[gIdx+1:]
			} else {
				domainGroups = nil
			}
		}

		// Apply limit to groups only; leave the full tools set intact so each returned group's ?include=tools stays complete.
		if params.Limit > 0 && int(params.Limit) < len(domainGroups) {
			domainGroups = domainGroups[:params.Limit]
		}
	} else {
		// Default ("tools"): apply cursor-based pagination to tools only, validating the cursor against tool slugs.
		if params.Cursor != nil && *params.Cursor != "" {
			cursor := *params.Cursor
			idx := -1
			for i, t := range results {
				if t.Slug == cursor {
					idx = i
					break
				}
			}

			if idx == -1 {
				return nil, nil, apierror.NewValidationError("Invalid pagination cursor.")
			}

			if idx+1 < len(results) {
				results = results[idx+1:]
			} else {
				results = nil
			}
		}

		// Apply limit to tools only; leave groups untouched.
		if params.Limit > 0 && int(params.Limit) < len(results) {
			results = results[:params.Limit]
		}
	}

	return results, domainGroups, nil
}

// builtinToolCatalogInfos turns the code-defined built-in tool catalog (agents.BuiltinTools) into AvailableToolInfo entries plus their groups, for the tool-selection UI. Tools carry category "built_in"; a selection is persisted as an agent_definition_tool link keyed by the tool slug.
func builtinToolCatalogInfos() ([]domain.AvailableToolInfo, []domain.ToolGroupInfo) {
	tools := make([]domain.AvailableToolInfo, 0, len(agents.BuiltinTools))
	for _, d := range agents.BuiltinTools {
		tools = append(tools, domain.AvailableToolInfo{
			Slug:                string(d.Slug),
			DisplayName:         d.DisplayName,
			Description:         d.Description,
			ConfigSchema:        json.RawMessage(`{}`),
			Category:            "built_in",
			GroupID:             d.Group.ID,
			GroupName:           d.Group.Name,
			RequiredPermissions: d.RequiredPermissions,
			Mutating:            d.Mutating,
		})
	}

	catalogGroups := agents.BuiltinToolGroups()
	groups := make([]domain.ToolGroupInfo, 0, len(catalogGroups))
	for _, g := range catalogGroups {
		groups = append(groups, domain.ToolGroupInfo{
			ID:        g.ID,
			Name:      g.Name,
			Slug:      g.Slug,
			Icon:      g.Icon,
			SortOrder: g.SortOrder,
		})
	}
	return tools, groups
}

// endpointToolCatalogInfos turns the generated endpoint-tool catalog into AvailableToolInfo entries plus their groups, for the tool-selection UI. Tools carry category "api_endpoint" so the frontend can route a selection into the agent's endpoint_tool_slugs grant.
func endpointToolCatalogInfos() ([]domain.AvailableToolInfo, []domain.ToolGroupInfo) {
	tools := make([]domain.AvailableToolInfo, 0, len(agents.EndpointTools))
	groupSeen := map[string]bool{}
	groupOrder := make([]string, 0)
	for _, d := range agents.EndpointTools {
		groupID, _ := endpointToolGroupID(d.Group)
		tools = append(tools, domain.AvailableToolInfo{
			Slug:                d.Slug,
			DisplayName:         d.DisplayName,
			Description:         d.Description,
			ConfigSchema:        json.RawMessage(`{}`),
			Category:            "api_endpoint",
			GroupID:             groupID,
			GroupName:           d.Group,
			RequiredPermissions: d.RequiredPermissions,
			RequiredRoleType:    d.RequiredRoleType,
			Mutating:            d.Mutating(),
		})
		if d.Group != "" && !groupSeen[d.Group] {
			groupSeen[d.Group] = true
			groupOrder = append(groupOrder, d.Group)
		}
	}

	slices.Sort(groupOrder)
	groups := make([]domain.ToolGroupInfo, 0, len(groupOrder))
	for i, name := range groupOrder {
		groupID, groupSlug := endpointToolGroupID(name)
		groups = append(groups, domain.ToolGroupInfo{
			ID:        groupID,
			Name:      name,
			Slug:      groupSlug,
			Icon:      "api",
			SortOrder: int32(100 + i), // after the built-in groups (sort_order 0-4)
		})
	}
	return tools, groups
}

// endpointToolGroupID derives a stable group id + slug for an endpoint-tool group name (e.g. "Sales Orders" -> tgrp_api_sales_orders / api_sales_orders), namespaced with an "api_" prefix so it never collides with the built-in tool group ids (tgrp_builtin_*).
func endpointToolGroupID(group string) (id, slug string) {
	slug = "api_" + strings.ToLower(strings.ReplaceAll(group, " ", "_"))
	return "tgrp_" + slug, slug
}

func (s *agentDefSvcImpl) UpdateAgentAccountStatus(ctx context.Context, params domain.UpdateAgentAccountStatusParams) (*domain.AgentAccountStatusInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.update_account_status")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
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
// Terminal markers a dead run's transcript ends with. They're stripped when an heir run inherits the transcript so it doesn't re-read the response that killed the original. These must match the step types written by emitFailureEvent / cancelledResult in runner.go.
const (
	stepTypeError     = "error"
	stepTypeCancelled = "cancelled"
)

// continueChatRun handles a reply to an agent's message (the user replied to the run that produced it).
// It either resumes that run, or — when the run died — forks an heir run that inherits its work. Returns false (so CreateChatRun starts a clean run seeded with conversation history) when the run is missing, owned by another account, diverged, or still in-flight/completed and so neither resumable nor a useful base to inherit from.
func (s *agentDefSvcImpl) continueChatRun(ctx context.Context, in domain.ChatRunInput) (bool, *apierror.APIError) {
	run, runErr := s.repos.NewAgentRunRepo().GetByID(ctx, in.ContinueRunID)
	if runErr != nil || run.AccountID != in.AccountID {
		return false, nil
	}
	// A run that took an off-conversation turn (free-text typed into the agent-run console) carries private fork context the conversation never saw. Whatever its status, never resume it or inherit its transcript here — fall through to a clean run seeded from the conversation's own history, which by construction excludes the fork.
	if run.DivergedFromConversation {
		return false, nil
	}

	switch run.StatusCode {
	case domain.RunStatusAwaitingInput, domain.RunStatusAwaitingApproval:
		// Still live and waiting on us — take the next turn on the same run.
		apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			if updErr := txSvc.repos.NewAgentRunRepo().UpdateStatus(txCtx, run.ID, domain.RunStatusRunning); updErr != nil {
				return updErr
			}
			return txSvc.enqueueChatContinue(txCtx, run.ID, in.AccountID, in.Message, in.TriggerMessageID)
		})
		if apiErr != nil {
			return false, apiErr
		}
		return true, nil
	case domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusCompleted:
		// The run reached a terminal state, but its work shouldn't be thrown away. Fork an heir run that inherits the transcript (minus any failure/cancel tail) and drive the reply through it as the next turn — so replying to a finished agent message continues that agent with full context instead of starting a blank run.
		return s.forkDeadChatRun(ctx, in, run)
	default:
		// running / pending → still in-flight, so neither resumable nor a clean base; start a fresh run.
		return false, nil
	}
}

// enqueueChatContinue writes the outbox command that drives runID's next turn from message and posts the result back into the conversation (ReplyToMessageID threads the reply under the trigger). Must run inside a withTx callback — it uses the transactional repos on s.
func (s *agentDefSvcImpl) enqueueChatContinue(ctx context.Context, runID, accountID, message, replyToMessageID string) *apierror.APIError {
	length := id.IDLength22
	msgID, genErr := id.GenID(id.MessageIDPrefix, &length)
	if genErr != nil {
		return apierror.NewInternalError(genErr, "Failed to generate continuation message id.")
	}
	dataBytes, _ := json.Marshal(messaging.AgentContinueRunData{
		AgentRunID:       runID,
		AccountID:        accountID,
		Message:          message,
		ReplyToMessageID: replyToMessageID,
	})
	if _, outboxErr := s.repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.AgentCmdContinueRun),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.AgentCmdContinueRun),
		Payload:     contracts.AmqpMessage{Data: dataBytes, MessageID: msgID},
		MaxAttempts: 3,
	}); outboxErr != nil {
		return apierror.NewInternalError(outboxErr, "Failed to enqueue chat continuation.")
	}
	return nil
}

// forkDeadChatRun handles a reply to a failed or cancelled chat run. Resuming it is impossible and starting from scratch throws away everything it did, so instead an heir run is created that inherits the dead run's transcript with the terminal failure/cancel markers stripped — "the failed responses".
// The reply is then driven through the heir as its next turn; the heir reconstructs the copied transcript via the normal continue path, so the agent resumes exactly where the dead run left off.
func (s *agentDefSvcImpl) forkDeadChatRun(ctx context.Context, in domain.ChatRunInput, dead *sqlc.AgentRun) (bool, *apierror.APIError) {
	// The dead run is terminal, so its event log is immutable — safe to read before the transaction.
	events, evErr := s.repos.NewAgentRunEventRepo().ListByRunID(ctx, dead.ID)
	if evErr != nil {
		return false, evErr
	}

	heirID, genErr := id.GenID(id.AgentRunIDPrefix, nil)
	if genErr != nil {
		return false, genErr
	}
	heirInput, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{Message: in.Message})

	var triggerMessageID pgtype.Text
	if in.TriggerMessageID != "" {
		triggerMessageID = agentdb.PgText(in.TriggerMessageID)
	}

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
		runRepo := txSvc.repos.NewAgentRunRepo()
		if insErr := runRepo.Insert(txCtx, sqlc.InsertAgentRunParams{
			ID:                heirID,
			AccountID:         dead.AccountID,
			AgentDefinitionID: dead.AgentDefinitionID,
			AgentConfigID:     dead.AgentConfigID,
			StatusCode:        domain.RunStatusPending,
			TriggerType:       string(constants.AgentTriggerTypeChat),
			Input:             heirInput,
			Output:            json.RawMessage(`{}`),
			ConversationID:    dead.ConversationID,
			TriggerMessageID:  triggerMessageID,
		}); insErr != nil {
			return insErr
		}
		if copyErr := txSvc.copyTranscript(txCtx, events, heirID, dead.AccountID); copyErr != nil {
			return copyErr
		}
		// The continue turn requires a running run (ContinueRun asserts it); also stamps started_at. The heir was just inserted as 'pending' in this same tx, so the guarded claim always wins (1 row); 0 would mean a concurrent claim and must abort.
		claimed, startErr := runRepo.UpdateStarted(txCtx, heirID)
		if startErr != nil {
			return startErr
		}
		if claimed == 0 {
			return apierror.NewInternalError(fmt.Errorf("heir run %s could not be claimed", heirID), "Failed to start heir run.")
		}
		return txSvc.enqueueChatContinue(txCtx, heirID, dead.AccountID, in.Message, in.TriggerMessageID)
	})
	if apiErr != nil {
		return false, apiErr
	}
	return true, nil
}

// copyTranscript clones a dead run's transcript events onto the heir run, re-sequenced from zero with fresh ids. The terminal failure markers (error / cancelled) are dropped so the heir doesn't inherit the response that killed the original. agent_action_id is cleared — the linked actions belong to the dead run — while tool-level results (including legitimate tool errors the agent already saw) are kept so it retains the context of what it already tried. Runs inside a withTx callback (transactional repo).
func (s *agentDefSvcImpl) copyTranscript(ctx context.Context, events []sqlc.AgentRunEvent, heirID, accountID string) *apierror.APIError {
	eventRepo := s.repos.NewAgentRunEventRepo()
	seq := int32(0)
	for _, e := range events {
		if e.StepType == stepTypeError || e.StepType == stepTypeCancelled {
			continue
		}
		evID, genErr := id.GenID(id.AgentRunEventIDPrefix, nil)
		if genErr != nil {
			return apierror.NewInternalError(genErr, "Failed to generate heir event id.")
		}
		metadata := e.Metadata
		if metadata == nil {
			metadata = json.RawMessage(`{}`)
		}
		if insErr := eventRepo.Insert(ctx, sqlc.InsertAgentRunEventParams{
			ID:         evID,
			AgentRunID: heirID,
			AccountID:  accountID,
			StepType:   e.StepType,
			Title:      e.Title,
			Content:    e.Content,
			Sequence:   seq,
			DurationMs: e.DurationMs,
			Metadata:   metadata,
			ActorID:    e.ActorID,
			ActorType:  e.ActorType,
			ActorName:  e.ActorName,
			// AgentActionID intentionally left null: those actions belong to the dead run.
		}); insErr != nil {
			return insErr
		}
		seq++
	}
	return nil
}

func (s *agentDefSvcImpl) CreateChatRun(ctx context.Context, in domain.ChatRunInput) *apierror.APIError {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.create_chat_run")
	defer span.End()

	if in.AccountID == "" || in.AgentDefinitionID == "" || in.ConversationID == "" {
		return nil
	}

	// A reply to an agent's message continues that run instead of starting a new one. If the run is gone or no longer continuable (still running a prior turn, or terminal), fall through to a fresh run.
	if in.ContinueRunID != "" {
		continued, apiErr := s.continueChatRun(ctx, in)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if continued {
			// The continuation command committed inside continueChatRun — kick the enqueuer so the next turn starts at once.
			s.kickOutbox()
			return nil
		}
	}

	defRepo := s.repos.NewAgentDefinitionRepo()
	configRepo := s.repos.NewAgentConfigRepo()

	def, defErr := defRepo.GetByID(ctx, in.AgentDefinitionID)
	if defErr != nil {
		return tracing.Trace(span, defErr)
	}

	// Ensure a per-account config exists (mirrors TriggerRun); the run needs a config id.
	config, cfgErr := configRepo.GetByAccountAndDefinition(ctx, in.AccountID, def.ID)
	if cfgErr != nil {
		configID, genErr := id.GenID(id.AgentConfigIDPrefix, nil)
		if genErr != nil {
			return tracing.Trace(span, genErr)
		}
		if insertErr := configRepo.Insert(ctx, sqlc.InsertAgentConfigParams{
			ID:                configID,
			AccountID:         in.AccountID,
			AgentDefinitionID: def.ID,
			IsEnabled:         true,
			Config:            json.RawMessage(`{}`),
		}); insertErr != nil {
			return tracing.Trace(span, insertErr)
		}
		config, cfgErr = configRepo.GetByAccountAndDefinition(ctx, in.AccountID, def.ID)
		if cfgErr != nil {
			return tracing.Trace(span, cfgErr)
		}
	}

	runID, genErr := id.GenID(id.AgentRunIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	// Fill in display names for history turns authored by *other* agents — notif-service carries their definition ids since it can't resolve agent names itself.
	s.resolveHistoryAgentNames(ctx, defRepo, in.History)
	runInput, _ := json.Marshal(struct {
		Message string                      `json:"message"`
		History []domain.ChatHistoryMessage `json:"history,omitempty"`
	}{Message: in.Message, History: in.History})
	length := id.IDLength22
	msgID, msgGenErr := id.GenID(id.MessageIDPrefix, &length)
	if msgGenErr != nil {
		return tracing.Trace(span, msgGenErr)
	}

	var triggerMessageID pgtype.Text
	if in.TriggerMessageID != "" {
		triggerMessageID = agentdb.PgText(in.TriggerMessageID)
	}

	if apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
		txRunRepo := txSvc.repos.NewAgentRunRepo()
		txOutbox := txSvc.repos.NewOutboxRepo()

		if insertErr := txRunRepo.Insert(txCtx, sqlc.InsertAgentRunParams{
			ID:                runID,
			AccountID:         in.AccountID,
			AgentDefinitionID: def.ID,
			AgentConfigID:     agentdb.PgText(config.ID),
			StatusCode:        domain.RunStatusPending,
			TriggerType:       string(constants.AgentTriggerTypeChat),
			Input:             runInput,
			Output:            json.RawMessage(`{}`),
			ConversationID:    agentdb.PgText(in.ConversationID),
			TriggerMessageID:  triggerMessageID,
		}); insertErr != nil {
			return insertErr
		}

		data := messaging.AgentExecuteRunData{
			AgentRunID:    runID,
			AgentConfigID: config.ID,
			AccountID:     in.AccountID,
			TriggerType:   string(constants.AgentTriggerTypeChat),
		}
		dataBytes, _ := json.Marshal(data)
		if _, outboxErr := txOutbox.Create(txCtx, messaging.OutboxMessageInput{
			MessageID:   msgID,
			ServiceName: domain.ServiceName,
			MessageType: string(contracts.AgentCmdExecuteRun),
			Destination: messaging.ApplicationExchange,
			RoutingKey:  string(contracts.AgentCmdExecuteRun),
			Payload:     contracts.AmqpMessage{Data: dataBytes, MessageID: msgID},
			MaxAttempts: 3,
		}); outboxErr != nil {
			return apierror.NewInternalError(outboxErr, "Failed to enqueue chat run execution.")
		}
		return nil
	}); apiErr != nil {
		return apiErr
	}

	// Run row + execute command are committed — wake the enqueuer so the run starts (and its live "thinking" indicator appears) right away instead of after an idle poll backoff.
	s.kickOutbox()
	return nil
}

// resolveHistoryAgentNames fills in the display Name for chat-history turns authored by other agents.
// notif-service carries those turns' agent-definition ids (it can't resolve agent names — definitions live here), so each is looked up by id (deduped; the window holds only a handful of distinct agents) and stamped with the agent's name, falling back to a generic label when unresolvable.
func (s *agentDefSvcImpl) resolveHistoryAgentNames(ctx context.Context, defRepo domain.AgentDefinitionRepo, history []domain.ChatHistoryMessage) {
	const fallbackAgentName = "another assistant"
	resolved := make(map[string]string)
	for i := range history {
		h := &history[i]
		if h.AgentConfigID == "" || h.Name != "" {
			continue
		}
		name, ok := resolved[h.AgentConfigID]
		if !ok {
			name = fallbackAgentName
			if def, err := defRepo.GetByID(ctx, h.AgentConfigID); err == nil && def != nil && def.Name != "" {
				name = def.Name
			}
			resolved[h.AgentConfigID] = name
		}
		h.Name = name
	}
}

func (s *agentDefSvcImpl) TriggerRun(ctx context.Context, params domain.TriggerRunParams) (string, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.trigger_run")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentRuns, types.ActionCreate); apiErr != nil {
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
				TriggerType:             string(constants.AgentTriggerTypeManual),
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
				TriggerType:   string(constants.AgentTriggerTypeManual),
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

			// Audit the newly-created run atomically with its insert, so the run's lifecycle start is recorded even though this is a Public:false endpoint. Changes are built explicitly (not via ComputeChanges) because a run has no hand-written domain struct to carry audit tags — its sqlc model is regenerated.
			if auditErr := audit.NewPublisher().Publish(txCtx, txOutbox, audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAgentRun,
				ResourceID:   runID,
				Changes: []audit.FieldChange{
					audit.NewFieldChange("status_code", nil, domain.RunStatusPending),
					audit.NewFieldChange("trigger_type", nil, string(constants.AgentTriggerTypeManual)),
					audit.NewFieldChange("agent_definition_id", nil, def.ID),
				},
			}); auditErr != nil {
				return auditErr
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
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentRuns, types.ActionUpdate); apiErr != nil {
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
		// A run can be stopped while it is doing or waiting to do work: actively running/pending, or paused awaiting the user (a chat run between turns, or one blocked on tool approval). Only the terminal states (completed/failed/cancelled/timed_out) reject — there is nothing left to stop.
		if !domain.RunStatusIsCancellable(run.StatusCode) {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError("Run cannot be cancelled (status: "+run.StatusCode+")."))
		}

		// Cancelling a run that is paused on tool approval is the human *denying* the gated tool(s): the deny control in the UI resolves the approval by cancelling the run. Capture it (and the actor) so we can audit the denial and mark the pending actions rejected — the mirror of the approval path in
		// ContinueRun.
		wasApprovalDenial := run.StatusCode == domain.RunStatusAwaitingApproval

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
			if updateErr := txRunRepo.MarkCancelledByUser(txCtx, params.AgentRunID); updateErr != nil {
				return updateErr
			}

			// Audit the denial — who rejected which gated tool(s) — and mark those pending actions rejected, written atomically with the status→cancelled transition via the same tx outbox. An empty slug set means "deny all pending"; the audit records the concrete denied set when known.
			if wasApprovalDenial {
				txActionRepo := txSvc.repos.NewAgentActionRepo()
				txOutbox := txSvc.repos.NewOutboxRepo()

				deniedSlugs := make([]string, 0)
				seen := make(map[string]bool)
				if pending, listErr := txActionRepo.ListByRun(txCtx, params.AgentRunID); listErr == nil {
					for _, a := range pending {
						if a.StatusCode != domain.ActionStatusPendingReview {
							continue
						}
						if !seen[a.ToolSlug] {
							seen[a.ToolSlug] = true
							deniedSlugs = append(deniedSlugs, a.ToolSlug)
						}
						_ = txActionRepo.MarkReviewed(txCtx, sqlc.MarkAgentActionReviewedParams{
							ID:                  a.ID,
							StatusCode:          domain.ActionStatusRejected,
							ReviewedBy:          agentdb.PgText(actorID),
							ReviewedByActorType: agentdb.PgText(actorType),
							ReviewedByActorName: agentdb.PgText(actorName),
						})
					}
				}

				metadata := map[string]any{}
				if len(deniedSlugs) > 0 {
					metadata["denied_tool_slugs"] = deniedSlugs
				} else {
					metadata["denied_all_pending"] = true
				}
				if auditErr := audit.NewPublisher().Publish(txCtx, txOutbox, audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionDeny,
					ResourceType: constants.ObjectTypeAgentRun,
					ResourceID:   params.AgentRunID,
					Metadata:     metadata,
				}); auditErr != nil {
					return auditErr
				}
			} else {
				// Plain cancel (not an approval denial): audit the status→cancelled transition atomically with the update. The denial path above publishes its own governance event, so only the plain case is instrumented here to avoid double-publishing.
				if auditErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeAgentRun,
					ResourceID:   params.AgentRunID,
					Changes:      []audit.FieldChange{audit.NewFieldChange("status_code", run.StatusCode, domain.RunStatusCancelled)},
				}); auditErr != nil {
					return auditErr
				}
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
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentRuns, types.ActionUpdate); apiErr != nil {
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

		// Resuming from awaiting_approval is a human approval decision (a tool gated on review was let through), distinct from a plain message continuation out of awaiting_input. Capture it now to audit the approval below.
		wasApproval := run.StatusCode == domain.RunStatusAwaitingApproval

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

			// "Approve all" is the only case where empty slugs means approve: a real approval (the run was awaiting_approval) with no specific slugs, no per-call ids, no rejections, and no typed message. Anything else
			// (per-tool approval names slugs or call ids; a rejection names them; a console continuation carries a message) must not blanket-approve.
			approveAllPending := wasApproval &&
				len(params.ApprovedToolSlugs) == 0 && len(params.ApprovedToolCallIDs) == 0 &&
				len(params.RejectedToolSlugs) == 0 && len(params.RejectedToolCallIDs) == 0 &&
				strings.TrimSpace(params.Message) == ""

			data := messaging.AgentContinueRunData{
				AgentRunID:          params.AgentRunID,
				AccountID:           accountID,
				Message:             params.Message,
				ApprovedToolSlugs:   params.ApprovedToolSlugs,
				ApproveAllPending:   approveAllPending,
				RejectedToolSlugs:   params.RejectedToolSlugs,
				ApprovedToolCallIDs: params.ApprovedToolCallIDs,
				RejectedToolCallIDs: params.RejectedToolCallIDs,
				ActorID:             actorID,
				ActorType:           actorType,
				ActorName:           actorName,
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

			// Audit the human review decision(s) on this run — who let which gated tool(s) run and who denied which. Attributed to the deciding user (ctx identity) and written atomically with the status→running transition via the same tx outbox. A single resume can both approve some tools and reject others, so each decision is its own event. Only emitted for a real review resume — a plain message continuation out of awaiting_input is neither an approval nor a rejection.
			if wasApproval {
				if len(params.ApprovedToolSlugs) > 0 || len(params.ApprovedToolCallIDs) > 0 || approveAllPending {
					metadata := map[string]any{}
					if len(params.ApprovedToolSlugs) > 0 {
						metadata["approved_tool_slugs"] = params.ApprovedToolSlugs
					}
					if len(params.ApprovedToolCallIDs) > 0 {
						metadata["approved_tool_call_ids"] = params.ApprovedToolCallIDs
					}
					if len(params.ApprovedToolSlugs) == 0 && len(params.ApprovedToolCallIDs) == 0 {
						metadata["approved_all_pending"] = true
					}
					if auditErr := audit.NewPublisher().Publish(txCtx, txOutbox, audit.EventData{
						ServiceName:  domain.ServiceName,
						Action:       constants.AuditActionApprove,
						ResourceType: constants.ObjectTypeAgentRun,
						ResourceID:   params.AgentRunID,
						Metadata:     metadata,
					}); auditErr != nil {
						return auditErr
					}
				}
				if len(params.RejectedToolSlugs) > 0 || len(params.RejectedToolCallIDs) > 0 {
					denyMeta := map[string]any{}
					if len(params.RejectedToolSlugs) > 0 {
						denyMeta["denied_tool_slugs"] = params.RejectedToolSlugs
					}
					if len(params.RejectedToolCallIDs) > 0 {
						denyMeta["denied_tool_call_ids"] = params.RejectedToolCallIDs
					}
					if auditErr := audit.NewPublisher().Publish(txCtx, txOutbox, audit.EventData{
						ServiceName:  domain.ServiceName,
						Action:       constants.AuditActionDeny,
						ResourceType: constants.ObjectTypeAgentRun,
						ResourceID:   params.AgentRunID,
						Metadata:     denyMeta,
					}); auditErr != nil {
						return auditErr
					}
				}
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, params.AgentRunID)
		})

		if apiErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		// Kick the enqueuer so the resume command (e.g. a chat tool-approval) is published immediately rather than waiting out the enqueuer's idle backoff — otherwise the thinking bubble lags ~MaxPollInterval after the user approves. Post-commit only: the outbox row must be visible to the poll.
		s.kickOutbox()

		return params.AgentRunID, nil

	default:
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// RetryRun re-attempts a failed run by resuming its existing transcript — no new user message is added, so the agent picks up with full knowledge of what it already did (including any tool results), minimizing duplicate side effects vs. a fresh re-run. The atomic status→running transition (guarded on status='failed' and bounded by retry_count) is the source of truth that prevents double-retry races.
func (s *agentDefSvcImpl) RetryRun(ctx context.Context, params domain.RetryRunParams) (string, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.retry_run")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentRuns, types.ActionUpdate); apiErr != nil {
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
		if runErr != nil || run.AccountID != accountID {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewResourceNotFoundError("Agent run not found."))
		}
		if run.StatusCode != domain.RunStatusFailed {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError(fmt.Sprintf("Only failed runs can be retried (status: %s).", run.StatusCode)))
		}
		if int(run.RetryCount) >= domain.MaxManualRetries {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError(fmt.Sprintf("This run has already been retried the maximum of %d times.", domain.MaxManualRetries)))
		}

		var actorID, actorType, actorName string
		if identity.Actor != nil {
			actorID = identity.Actor.ID
			actorType = string(identity.Type)
			if identity.Actor.Name != nil {
				actorName = *identity.Actor.Name
			}
		}
		// Reply back into the original thread by replying to the trigger message — this also flags the resumed turn as conversation-originated so its reply is posted rather than kept private.
		replyTo := ""
		if run.TriggerMessageID.Valid {
			replyTo = run.TriggerMessageID.String
		}

		length := id.IDLength22
		msgID, msgGenErr := id.GenID(id.MessageIDPrefix, &length)
		if msgGenErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, msgGenErr)
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			txRunRepo := txSvc.repos.NewAgentRunRepo()
			txOutbox := txSvc.repos.NewOutboxRepo()

			if _, markErr := txRunRepo.MarkRetrying(txCtx, params.AgentRunID); markErr != nil {
				return apierror.NewValidationError("This run is no longer in a failed state and can't be retried.")
			}

			data := messaging.AgentContinueRunData{
				AgentRunID:       params.AgentRunID,
				AccountID:        accountID,
				Message:          "", // resume: re-attempt the existing transcript, no new user input
				ActorID:          actorID,
				ActorType:        actorType,
				ActorName:        actorName,
				ReplyToMessageID: replyTo,
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

			// Audit the retry status transition (failed→running) atomically with the mark-retrying update, so the run's lifecycle change is recorded on this Public:false endpoint.
			if auditErr := audit.NewPublisher().Publish(txCtx, txOutbox, audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAgentRun,
				ResourceID:   params.AgentRunID,
				Changes:      []audit.FieldChange{audit.NewFieldChange("status_code", run.StatusCode, domain.RunStatusRunning)},
			}); auditErr != nil {
				return auditErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, params.AgentRunID)
		})

		if apiErr != nil {
			return "", meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		// Kick the enqueuer so the resume command is published immediately rather than waiting out the enqueuer's idle backoff. Post-commit only: the outbox row must be visible to the poll.
		s.kickOutbox()

		return params.AgentRunID, nil

	default:
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// validateEndpointToolSlugs rejects any endpoint_tool_slugs (or endpoint_tool_review key) entry in the agent config JSON that is not a known endpoint-tool. The wildcard
// "*" (grant the whole catalog) is always allowed in the slug list. An empty/absent list or review map is valid.
func validateEndpointToolSlugs(configJSON string) *apierror.APIError {
	if configJSON == "" {
		return nil
	}
	var cfg struct {
		EndpointToolSlugs  []string        `json:"endpoint_tool_slugs"`
		EndpointToolReview map[string]bool `json:"endpoint_tool_review"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return apierror.NewValidationErrorWithParam("Invalid agent config.", "config")
	}
	for _, slug := range cfg.EndpointToolSlugs {
		if slug == "*" {
			continue
		}
		if _, ok := agents.LookupEndpointTool(slug); !ok {
			return apierror.NewValidationErrorWithParam("Tool not found: "+slug, "tools")
		}
	}
	for slug := range cfg.EndpointToolReview {
		if _, ok := agents.LookupEndpointTool(slug); !ok {
			return apierror.NewValidationErrorWithParam("Tool not found: "+slug, "tools")
		}
	}
	return nil
}

// CreateAgentMemory creates a new agent memory record, with idempotency support.
func (s *agentDefSvcImpl) CreateAgentMemory(ctx context.Context, params domain.CreateAgentMemoryParams) (*domain.AgentMemoryInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.create_agent_memory")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentMemories, types.ActionCreate); apiErr != nil {
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

// UpdateAgentMemory updates an existing agent memory record, with idempotency support.
func (s *agentDefSvcImpl) UpdateAgentMemory(ctx context.Context, params domain.UpdateAgentMemoryParams) (*domain.AgentMemoryInfo, *apierror.APIError) {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.update_agent_memory")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentMemories, types.ActionUpdate); apiErr != nil {
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
		// Partial update: only provided fields change. A nil metadata narg → COALESCE keeps the current value (no more "{}" wipe).
		var metadata []byte
		if params.MetadataJSON != nil {
			metadata = []byte(*params.MetadataJSON)
		}

		var expiresAt pgtype.Timestamptz
		if params.ExpiresAt != nil && *params.ExpiresAt != "" {
			t, parseErr := parseTimestamp(*params.ExpiresAt)
			if parseErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationErrorWithParam("Invalid expires_at: "+parseErr.Error(), "expires_at"))
			}
			expiresAt = pgtype.Timestamptz{Time: t, Valid: true}
		}

		var importance pgtype.Float8
		if params.Importance != nil {
			importance = pgtype.Float8{Float64: *params.Importance, Valid: true}
		}

		var result *domain.AgentMemoryInfo
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
			memoryRepo := txSvc.repos.NewAgentMemoryRepo()

			oldMemory, getErr := memoryRepo.GetByID(txCtx, params.MemoryID)
			if getErr != nil {
				if apierror.IsNotFound(getErr) {
					return apierror.NewResourceNotFoundError("Agent memory not found.")
				}
				return getErr
			}
			if oldMemory.AccountID != accountID {
				return apierror.NewResourceNotFoundError("Agent memory not found.")
			}
			old := sqlcMemoryToDomain(oldMemory)

			if updateErr := memoryRepo.Update(txCtx, sqlc.UpdateAgentMemoryParams{
				ID:             params.MemoryID,
				Category:       agentdb.PgTextPtr(params.Category),
				Content:        agentdb.PgTextPtr(params.Content),
				Metadata:       metadata,
				ClearEntity:    params.ClearEntity,
				EntityType:     agentdb.PgTextPtr(params.EntityType),
				EntityID:       agentdb.PgTextPtr(params.EntityID),
				Importance:     importance,
				ClearExpiresAt: params.ClearExpiresAt,
				ExpiresAt:      expiresAt,
				AccountID:      accountID,
			}); updateErr != nil {
				return updateErr
			}

			memory, getErr := memoryRepo.GetByID(txCtx, params.MemoryID)
			if getErr != nil {
				return getErr
			}
			result = sqlcMemoryToDomain(memory)

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
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

// DeleteAgentMemory deletes an agent memory record.
//
//  1. Extract the caller's identity and verify it is an internal actor with a target account.
//  2. Look up the memory; rows that are missing or owned by another account are
//     treated as already deleted (idempotent no-op success).
//  3. Within a transaction, delete the memory and publish a delete audit event.
func (s *agentDefSvcImpl) DeleteAgentMemory(ctx context.Context, params domain.DeleteAgentMemoryParams) *apierror.APIError {
	ctx, span := agentDefSvcTracer.Start(ctx, "service.agent_definition.delete_agent_memory")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAgentMemories, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	accountID := identity.Target.AccountID

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *agentDefSvcImpl) *apierror.APIError {
		memoryRepo := txSvc.repos.NewAgentMemoryRepo()

		oldMemory, getErr := memoryRepo.GetByID(txCtx, params.MemoryID)
		if getErr != nil {
			if apierror.IsNotFound(getErr) {
				// The account-scoped delete has always been a silent no-op for missing rows; keep repeat deletes idempotent successes.
				return nil
			}
			return getErr
		}
		if oldMemory.AccountID != accountID {
			// A memory owned by another account is invisible to the caller; treat it as already deleted rather than leaking its existence.
			return nil
		}
		old := sqlcMemoryToDomain(oldMemory)

		// Snapshot the row into deleted_record before the hard delete so it is recoverable and
		// repeat/racing deletes are distinguishable from "never existed" (deleted-record convention).
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAgentMemory, old.ID, old); apiErr != nil {
			return apiErr
		}

		if deleteErr := memoryRepo.Delete(txCtx, params.MemoryID, accountID); deleteErr != nil {
			return deleteErr
		}

		changes := audit.ComputeChanges(old, (*domain.AgentMemoryInfo)(nil))

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAgentMemory,
			ResourceID:   old.ID,
			Changes:      changes,
		})
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
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

// sqlToAgentDefinitionInfo converts sqlc rows into a domain AgentDefinitionInfo. When includes is non-nil, fields not listed are set to nil.
func sqlToAgentDefinitionInfo(def *sqlc.AgentDefinition, tools []sqlc.ListToolsByAgentDefinitionIDRow, accountStatus *sqlc.AgentAccountStatus, includes []string) *domain.AgentDefinitionInfo {
	var roleID string
	if def.RoleID.Valid {
		roleID = def.RoleID.String
	}

	var domainTools []domain.AgentDefinitionToolInfo
	if includesContains(includes, "tools") {
		domainTools = make([]domain.AgentDefinitionToolInfo, 0, len(tools))
		for _, t := range tools {
			// Linked tools are built-in tools, granted by slug. Display metadata comes from the code catalog (agents.BuiltinTools), not the database.
			info := domain.AgentDefinitionToolInfo{
				ID:            t.ID,
				ToolSlug:      t.ToolSlug,
				Category:      "built_in",
				ConfigSchema:  json.RawMessage(`{}`),
				Config:        t.Config,
				SortOrder:     t.SortOrder,
				RequireReview: t.RequireReview,
			}
			if bt, ok := agents.LookupBuiltinTool(t.ToolSlug); ok {
				info.DisplayName = bt.DisplayName
				info.Description = bt.Description
				info.GroupID = bt.Group.ID
				info.GroupName = bt.Group.Name
				info.RequiredPermissions = bt.RequiredPermissions
			}
			domainTools = append(domainTools, info)
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
		Description:    agentdb.StringFromPgText(def.Description),
		DefinitionType: def.DefinitionType,
		CategoryCode:   def.CategoryCode,
		TriggerType:    def.TriggerType,
		IsEditable:     def.DefinitionType == string(constants.AgentDefinitionTypeCustom),
		Config:         config,
		RoleID:         roleID,
		Tools:          domainTools,
		AccountStatus:  domainAccountStatus,
		CreatedAt:      def.CreatedAt.Time,
		UpdatedAt:      def.UpdatedAt.Time,
	}
}
