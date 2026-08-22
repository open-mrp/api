package repository

import (
	"context"

	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var cfgRepoTracer = tracing.GetTracer("agent-service.agent_config_repository")

type agentConfigRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentConfigRepo(queries *sqlc.Queries) *agentConfigRepoImpl {
	return &agentConfigRepoImpl{queries: queries}
}

func (r *agentConfigRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentConfig, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, cfgRepoTracer, "repository.agent_config.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentConfigByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentConfigRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentConfigParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, cfgRepoTracer, "repository.agent_config.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentConfig(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentConfigRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]sqlc.AgentConfig, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, cfgRepoTracer, "repository.agent_config.list_by_account")
	defer span.End()
	rows, err := r.queries.ListAgentConfigsByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentConfigRepoImpl) ListEnabledWithSchedule(ctx context.Context) ([]sqlc.ListEnabledConfigsWithScheduleRow, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, cfgRepoTracer, "repository.agent_config.list_enabled_with_schedule")
	defer span.End()
	rows, err := r.queries.ListEnabledConfigsWithSchedule(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentConfigRepoImpl) GetByAccountAndDefinition(ctx context.Context, accountID, definitionID string) (*sqlc.AgentConfig, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, cfgRepoTracer, "repository.agent_config.get_by_account_and_definition")
	defer span.End()
	row, err := r.queries.GetAgentConfigByAccountAndDefinition(ctx, sqlc.GetAgentConfigByAccountAndDefinitionParams{
		AccountID:         accountID,
		AgentDefinitionID: definitionID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}
