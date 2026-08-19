package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var registrationFlowRepoTracer = tracing.GetTracer("core-service.registration_flow_repository")

type registrationFlowRepoImpl struct {
	queries *sqlc.Queries
}

func NewRegistrationFlowRepo(queries *sqlc.Queries) domain.RegistrationFlowRepo {
	return &registrationFlowRepoImpl{queries: queries}
}

func registrationFlowCreatedAt(rf *domain.RegistrationFlow) time.Time { return rf.CreatedAt }
func registrationFlowID(rf *domain.RegistrationFlow) string           { return rf.ID }

func buildRegistrationFlowSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapRegistrationFlowRow(row sqlc.RegistrationFlow) *domain.RegistrationFlow {
	return &domain.RegistrationFlow{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *registrationFlowRepoImpl) loadOptions(ctx context.Context, flowID string) ([]*domain.RegistrationFlowOption, []*domain.RegistrationFlowOption, []*domain.RegistrationFlowOption, *apierror.APIError) {
	ptRows, err := r.queries.ListPaymentTermOptionsByFlowID(ctx, flowID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, nil, nil, apiErr
	}
	stRows, err := r.queries.ListShippingTermOptionsByFlowID(ctx, flowID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, nil, nil, apiErr
	}
	agRows, err := r.queries.ListAccountGroupOptionsByFlowID(ctx, gosql.NullString{String: flowID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, nil, nil, apiErr
	}

	customerGroupOptions := make([]*domain.RegistrationFlowOption, len(agRows))
	for i, row := range agRows {
		customerGroupOptions[i] = &domain.RegistrationFlowOption{ID: row.ID, Name: row.Name}
	}

	paymentTermOptions := make([]*domain.RegistrationFlowOption, len(ptRows))
	for i, row := range ptRows {
		paymentTermOptions[i] = &domain.RegistrationFlowOption{ID: row.ID, Name: row.Name}
	}

	shippingTermOptions := make([]*domain.RegistrationFlowOption, len(stRows))
	for i, row := range stRows {
		shippingTermOptions[i] = &domain.RegistrationFlowOption{ID: row.ID, Name: row.Name}
	}

	return customerGroupOptions, paymentTermOptions, shippingTermOptions, nil
}

func (r *registrationFlowRepoImpl) enrichFlow(ctx context.Context, flow *domain.RegistrationFlow) *apierror.APIError {
	cg, pt, st, apiErr := r.loadOptions(ctx, flow.ID)
	if apiErr != nil {
		return apiErr
	}
	flow.CustomerGroupOptions = cg
	flow.PaymentTermOptions = pt
	flow.ShippingTermOptions = st
	return nil
}

func (r *registrationFlowRepoImpl) enrichFlows(ctx context.Context, flows []*domain.RegistrationFlow) *apierror.APIError {
	for _, flow := range flows {
		if apiErr := r.enrichFlow(ctx, flow); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *registrationFlowRepoImpl) List(ctx context.Context, params domain.ListRegistrationFlowsParams) (*domain.ListRegistrationFlowsResult, *apierror.APIError) {
	ctx, span := registrationFlowRepoTracer.Start(ctx, "repository.registration_flow.list")
	defer span.End()

	searchQuery := buildRegistrationFlowSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListRegistrationFlowsBackward(ctx, sqlc.ListRegistrationFlowsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			flows := make([]*domain.RegistrationFlow, len(rows))
			for i, row := range rows {
				flows[i] = mapRegistrationFlowRow(row)
			}
			if apiErr := r.enrichFlows(ctx, flows); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			result, pageInfo := pagination.BuildPageString(flows, params.Limit, cursorDir, registrationFlowCreatedAt, registrationFlowID)
			return &domain.ListRegistrationFlowsResult{RegistrationFlows: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListRegistrationFlowsForward(ctx, sqlc.ListRegistrationFlowsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		flows := make([]*domain.RegistrationFlow, len(rows))
		for i, row := range rows {
			flows[i] = mapRegistrationFlowRow(row)
		}
		if apiErr := r.enrichFlows(ctx, flows); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result, pageInfo := pagination.BuildPageString(flows, params.Limit, cursorDir, registrationFlowCreatedAt, registrationFlowID)
		return &domain.ListRegistrationFlowsResult{RegistrationFlows: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListRegistrationFlowsForward(ctx, sqlc.ListRegistrationFlowsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	flows := make([]*domain.RegistrationFlow, len(rows))
	for i, row := range rows {
		flows[i] = mapRegistrationFlowRow(row)
	}
	if apiErr := r.enrichFlows(ctx, flows); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(flows, params.Limit, cursorDir, registrationFlowCreatedAt, registrationFlowID)
	return &domain.ListRegistrationFlowsResult{RegistrationFlows: result, PageInfo: pageInfo}, nil
}

func (r *registrationFlowRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowRepoTracer.Start(ctx, "repository.registration_flow.get")
	defer span.End()

	row, err := r.queries.GetRegistrationFlow(ctx, sqlc.GetRegistrationFlowParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	flow := mapRegistrationFlowRow(row)
	if apiErr := r.enrichFlow(ctx, flow); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return flow, nil
}

func (r *registrationFlowRepoImpl) GetByAccountID(ctx context.Context, accountID string) ([]*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowRepoTracer.Start(ctx, "repository.registration_flow.get_by_account_id")
	defer span.End()

	rows, err := r.queries.ListRegistrationFlowsByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	flows := make([]*domain.RegistrationFlow, len(rows))
	for i, row := range rows {
		flows[i] = mapRegistrationFlowRow(row)
	}
	if apiErr := r.enrichFlows(ctx, flows); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return flows, nil
}

func (r *registrationFlowRepoImpl) writeOptions(ctx context.Context, flowID, accountID string, params domain.CreateRegistrationFlowParams) *apierror.APIError {
	for _, ptID := range params.PaymentTermIDs {
		if err := r.queries.InsertPaymentTermOption(ctx, sqlc.InsertPaymentTermOptionParams{
			PaymentTermID:      ptID,
			RegistrationFlowID: flowID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return apiErr
			}
		}
	}

	for _, stID := range params.ShippingTermIDs {
		if err := r.queries.InsertShippingTermOption(ctx, sqlc.InsertShippingTermOptionParams{
			RegistrationFlowID: flowID,
			ShippingTermID:     stID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return apiErr
			}
		}
	}

	for _, agID := range params.CustomerGroupIDs {
		if err := r.queries.SetAccountGroupRegistrationFlowID(ctx, sqlc.SetAccountGroupRegistrationFlowIDParams{
			RegistrationFlowID: gosql.NullString{String: flowID, Valid: true},
			AccountGroupID:     agID,
			AccountID:          accountID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return apiErr
			}
		}
	}

	return nil
}

func (r *registrationFlowRepoImpl) clearOptions(ctx context.Context, flowID, accountID string) *apierror.APIError {
	if err := r.queries.DeletePaymentTermOptionsByFlowID(ctx, flowID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
	}

	if err := r.queries.DeleteShippingTermOptionsByFlowID(ctx, flowID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
	}

	if err := r.queries.ClearAccountGroupRegistrationFlowID(ctx, sqlc.ClearAccountGroupRegistrationFlowIDParams{
		RegistrationFlowID: gosql.NullString{String: flowID, Valid: true},
		AccountID:          accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

func (r *registrationFlowRepoImpl) Create(ctx context.Context, id string, params domain.CreateRegistrationFlowParams) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowRepoTracer.Start(ctx, "repository.registration_flow.create")
	defer span.End()

	err := r.queries.InsertRegistrationFlow(ctx, sqlc.InsertRegistrationFlowParams{
		ID:        id,
		Name:      params.Name,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := r.writeOptions(ctx, id, params.AccountID, params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, id)
}

func (r *registrationFlowRepoImpl) Update(ctx context.Context, params domain.UpdateRegistrationFlowParams) (*domain.RegistrationFlow, *apierror.APIError) {
	ctx, span := registrationFlowRepoTracer.Start(ctx, "repository.registration_flow.update")
	defer span.End()

	result, err := r.queries.UpdateRegistrationFlow(ctx, sqlc.UpdateRegistrationFlowParams{
		ID:        params.RegistrationFlowID,
		AccountID: params.AccountID,
		Name:      toNullString(params.Name),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Registration flow not found."))
	}

	// Only clear and rewrite options for fields that were explicitly provided.
	if params.HasPaymentTermIDs {
		if err := r.queries.DeletePaymentTermOptionsByFlowID(ctx, params.RegistrationFlowID); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		for _, ptID := range params.PaymentTermIDs {
			if err := r.queries.InsertPaymentTermOption(ctx, sqlc.InsertPaymentTermOptionParams{
				PaymentTermID:      ptID,
				RegistrationFlowID: params.RegistrationFlowID,
			}); err != nil {
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}
	}

	if params.HasShippingTermIDs {
		if err := r.queries.DeleteShippingTermOptionsByFlowID(ctx, params.RegistrationFlowID); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		for _, stID := range params.ShippingTermIDs {
			if err := r.queries.InsertShippingTermOption(ctx, sqlc.InsertShippingTermOptionParams{
				RegistrationFlowID: params.RegistrationFlowID,
				ShippingTermID:     stID,
			}); err != nil {
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}
	}

	if params.HasCustomerGroupIDs {
		if err := r.queries.ClearAccountGroupRegistrationFlowID(ctx, sqlc.ClearAccountGroupRegistrationFlowIDParams{
			RegistrationFlowID: gosql.NullString{String: params.RegistrationFlowID, Valid: true},
			AccountID:          params.AccountID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		for _, agID := range params.CustomerGroupIDs {
			if err := r.queries.SetAccountGroupRegistrationFlowID(ctx, sqlc.SetAccountGroupRegistrationFlowIDParams{
				RegistrationFlowID: gosql.NullString{String: params.RegistrationFlowID, Valid: true},
				AccountGroupID:     agID,
				AccountID:          params.AccountID,
			}); err != nil {
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}
	}

	return r.Get(ctx, params.AccountID, params.RegistrationFlowID)
}

func (r *registrationFlowRepoImpl) Delete(ctx context.Context, params domain.DeleteRegistrationFlowParams) *apierror.APIError {
	ctx, span := registrationFlowRepoTracer.Start(ctx, "repository.registration_flow.delete")
	defer span.End()

	if apiErr := r.clearOptions(ctx, params.RegistrationFlowID, params.AccountID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	result, err := r.queries.DeleteRegistrationFlow(ctx, sqlc.DeleteRegistrationFlowParams{
		ID:        params.RegistrationFlowID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Registration flow not found."))
	}

	return nil
}
