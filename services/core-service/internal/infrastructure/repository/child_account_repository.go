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

var childAccountRepoTracer = tracing.GetTracer("core-service.child_account_repository")

func childAccountCreatedAt(c *domain.ChildAccount) time.Time { return c.CreatedAt }
func childAccountID(c *domain.ChildAccount) string           { return c.RelationID }

func mapChildAccountForwardRow(row sqlc.ListChildAccountsForwardRow) *domain.ChildAccount {
	var email *string
	if row.Email.Valid {
		email = &row.Email.String
	}
	return &domain.ChildAccount{
		RelationID:     row.RelationID,
		AccountID:      row.AccountID,
		AccountName:    row.AccountName,
		ExternalNumber: row.ExternalNumber,
		Email:          email,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func mapChildAccountBackwardRow(row sqlc.ListChildAccountsBackwardRow) *domain.ChildAccount {
	var email *string
	if row.Email.Valid {
		email = &row.Email.String
	}
	return &domain.ChildAccount{
		RelationID:     row.RelationID,
		AccountID:      row.AccountID,
		AccountName:    row.AccountName,
		ExternalNumber: row.ExternalNumber,
		Email:          email,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (r *accountRelationRepoImpl) ListChildAccounts(ctx context.Context, params domain.ListChildAccountsParams) (*domain.ListChildAccountsResult, *apierror.APIError) {
	ctx, span := childAccountRepoTracer.Start(ctx, "repository.child_account.list")
	defer span.End()

	// Resolve parent counterparty account ID to parent relation ID. If no relation exists, the account has no children — return an empty list.
	parentRelationID, apiErr := r.FindRelationByOwnerAndCounterparty(ctx, params.OwnerAccountID, params.ParentAccountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return &domain.ListChildAccountsResult{
				Items:    []*domain.ChildAccount{},
				PageInfo: pagination.PageInfo{},
			}, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*params.Query) + "%", Valid: true}
	}

	parentRelNullStr := gosql.NullString{String: parentRelationID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListChildAccountsBackward(ctx, sqlc.ListChildAccountsBackwardParams{
				OwnerAccountID:   params.OwnerAccountID,
				ParentRelationID: parentRelNullStr,
				SearchQuery:      searchQuery,
				CursorCreatedAt:  gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:         gosql.NullString{String: cur.ID, Valid: true},
				Limit:            params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.ChildAccount, len(rows))
			for i, row := range rows {
				items[i] = mapChildAccountBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, childAccountCreatedAt, childAccountID)
			return &domain.ListChildAccountsResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListChildAccountsForward(ctx, sqlc.ListChildAccountsForwardParams{
			OwnerAccountID:   params.OwnerAccountID,
			ParentRelationID: parentRelNullStr,
			SearchQuery:      searchQuery,
			CursorCreatedAt:  gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:         gosql.NullString{String: cur.ID, Valid: true},
			Limit:            params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.ChildAccount, len(rows))
		for i, row := range rows {
			items[i] = mapChildAccountForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, childAccountCreatedAt, childAccountID)
		return &domain.ListChildAccountsResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListChildAccountsForward(ctx, sqlc.ListChildAccountsForwardParams{
		OwnerAccountID:   params.OwnerAccountID,
		ParentRelationID: parentRelNullStr,
		SearchQuery:      searchQuery,
		Limit:            params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ChildAccount, len(rows))
	for i, row := range rows {
		items[i] = mapChildAccountForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, childAccountCreatedAt, childAccountID)
	return &domain.ListChildAccountsResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *accountRelationRepoImpl) GetChildAccountDetail(ctx context.Context, ownerAccountID, counterpartyAccountID string) (*domain.ChildAccount, *apierror.APIError) {
	ctx, span := childAccountRepoTracer.Start(ctx, "repository.child_account.get_detail")
	defer span.End()

	row, err := r.queries.GetChildAccountDetail(ctx, sqlc.GetChildAccountDetailParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: counterpartyAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var email *string
	if row.Email.Valid {
		email = &row.Email.String
	}

	return &domain.ChildAccount{
		RelationID:     row.RelationID,
		AccountID:      row.AccountID,
		AccountName:    row.AccountName,
		ExternalNumber: row.ExternalNumber,
		Email:          email,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (r *accountRelationRepoImpl) GetChildAccountsByRelationIDs(ctx context.Context, ownerAccountID string, relationIDs []string) ([]*domain.ChildAccount, *apierror.APIError) {
	ctx, span := childAccountRepoTracer.Start(ctx, "repository.child_account.get_by_relation_ids")
	defer span.End()

	if len(relationIDs) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetChildAccountsByRelationIDs(ctx, sqlc.GetChildAccountsByRelationIDsParams{
		Ids:            relationIDs,
		OwnerAccountID: ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.ChildAccount, len(rows))
	for i, row := range rows {
		var email *string
		if row.Email.Valid {
			email = &row.Email.String
		}
		out[i] = &domain.ChildAccount{
			RelationID:     row.RelationID,
			AccountID:      row.AccountID,
			AccountName:    row.AccountName,
			ExternalNumber: row.ExternalNumber,
			Email:          email,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return out, nil
}

func (r *accountRelationRepoImpl) SetParentRelation(ctx context.Context, ownerAccountID, childRelationID, parentRelationID string) *apierror.APIError {
	ctx, span := childAccountRepoTracer.Start(ctx, "repository.child_account.set_parent_relation")
	defer span.End()

	err := r.queries.SetParentAccountRelation(ctx, sqlc.SetParentAccountRelationParams{
		ParentRelationID: gosql.NullString{String: parentRelationID, Valid: true},
		ChildRelationID:  childRelationID,
		OwnerAccountID:   ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRelationRepoImpl) ClearParentRelation(ctx context.Context, ownerAccountID, childRelationID, parentRelationID string) *apierror.APIError {
	ctx, span := childAccountRepoTracer.Start(ctx, "repository.child_account.clear_parent_relation")
	defer span.End()

	err := r.queries.ClearParentAccountRelation(ctx, sqlc.ClearParentAccountRelationParams{
		ChildRelationID:  childRelationID,
		OwnerAccountID:   ownerAccountID,
		ParentRelationID: gosql.NullString{String: parentRelationID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRelationRepoImpl) GetParentRelationID(ctx context.Context, relationID string) (*string, *apierror.APIError) {
	ctx, span := childAccountRepoTracer.Start(ctx, "repository.child_account.get_parent_relation_id")
	defer span.End()

	parentID, err := r.queries.GetParentAccountRelationID(ctx, relationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if parentID.Valid {
		return &parentID.String, nil
	}
	return nil, nil
}
