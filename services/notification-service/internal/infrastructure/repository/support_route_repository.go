package repository

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var supportRouteRepoTracer = tracing.GetTracer("notification-service.support_route_repository")

type supportRouteRepoImpl struct {
	db *sqlc.Queries
}

func NewSupportRouteRepo(db *sqlc.Queries) domain.SupportRouteRepo {
	return &supportRouteRepoImpl{db: db}
}

func (r *supportRouteRepoImpl) Upsert(ctx context.Context, id, accountID, relationAccountID, groupConversationID string) *apierror.APIError {
	ctx, span := supportRouteRepoTracer.Start(ctx, "repository.support_route.upsert")
	defer span.End()
	err := r.db.UpsertSupportRoute(ctx, sqlc.UpsertSupportRouteParams{
		ID:                  id,
		AccountID:           accountID,
		RelationAccountID:   relationAccountID,
		GroupConversationID: groupConversationID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *supportRouteRepoImpl) Get(ctx context.Context, accountID, relationAccountID string) (*domain.SupportRoute, *apierror.APIError) {
	ctx, span := supportRouteRepoTracer.Start(ctx, "repository.support_route.get")
	defer span.End()
	row, err := r.db.GetSupportRoute(ctx, sqlc.GetSupportRouteParams{AccountID: accountID, RelationAccountID: relationAccountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return supportRouteToDomain(row), nil
}

func (r *supportRouteRepoImpl) Resolve(ctx context.Context, accountID, relationAccountID string) (*domain.SupportRoute, *apierror.APIError) {
	ctx, span := supportRouteRepoTracer.Start(ctx, "repository.support_route.resolve")
	defer span.End()
	row, err := r.db.ResolveSupportRoute(ctx, sqlc.ResolveSupportRouteParams{AccountID: accountID, RelationAccountID: relationAccountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return supportRouteToDomain(row), nil
}

func (r *supportRouteRepoImpl) Delete(ctx context.Context, accountID, relationAccountID string) (bool, *apierror.APIError) {
	ctx, span := supportRouteRepoTracer.Start(ctx, "repository.support_route.delete")
	defer span.End()
	rows, err := r.db.DeleteSupportRoute(ctx, sqlc.DeleteSupportRouteParams{AccountID: accountID, RelationAccountID: relationAccountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func supportRouteToDomain(row sqlc.SupportRoute) *domain.SupportRoute {
	return &domain.SupportRoute{
		ID:                  row.ID,
		AccountID:           row.AccountID,
		RelationAccountID:   row.RelationAccountID,
		GroupConversationID: row.GroupConversationID,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}
