package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var paymentTermRepoTracer = tracing.GetTracer("core-service.payment_term_repository")

type paymentTermRepoImpl struct {
	queries *sqlc.Queries
}

func NewPaymentTermRepo(queries *sqlc.Queries) domain.PaymentTermRepo {
	return &paymentTermRepoImpl{queries: queries}
}

func paymentTermCreatedAt(pt *domain.PaymentTerm) time.Time { return pt.CreatedAt }
func paymentTermID(pt *domain.PaymentTerm) string           { return pt.ID }

func buildPaymentTermSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapPaymentTermRow(row sqlc.PaymentTerm) *domain.PaymentTerm {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	status := constants.PaymentTermStatusInactive
	if row.IsActive {
		status = constants.PaymentTermStatusActive
	}
	return &domain.PaymentTerm{
		ID:        row.ID,
		Name:      row.Name,
		Status:    status,
		AccountID: accountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *paymentTermRepoImpl) List(ctx context.Context, params domain.ListPaymentTermsParams) (*domain.ListPaymentTermsResult, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.list")
	defer span.End()

	searchQuery := buildPaymentTermSearchParams(params.Query)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPaymentTermsBackward(ctx, sqlc.ListPaymentTermsBackwardParams{
				AccountID:       accountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			paymentTerms := make([]*domain.PaymentTerm, len(rows))
			for i, row := range rows {
				paymentTerms[i] = mapPaymentTermRow(row)
			}
			result, pageInfo := pagination.BuildPageString(paymentTerms, params.Limit, cursorDir, paymentTermCreatedAt, paymentTermID)
			return &domain.ListPaymentTermsResult{PaymentTerms: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListPaymentTermsForward(ctx, sqlc.ListPaymentTermsForwardParams{
			AccountID:       accountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		paymentTerms := make([]*domain.PaymentTerm, len(rows))
		for i, row := range rows {
			paymentTerms[i] = mapPaymentTermRow(row)
		}
		result, pageInfo := pagination.BuildPageString(paymentTerms, params.Limit, cursorDir, paymentTermCreatedAt, paymentTermID)
		return &domain.ListPaymentTermsResult{PaymentTerms: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPaymentTermsForward(ctx, sqlc.ListPaymentTermsForwardParams{
		AccountID:   accountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	paymentTerms := make([]*domain.PaymentTerm, len(rows))
	for i, row := range rows {
		paymentTerms[i] = mapPaymentTermRow(row)
	}
	result, pageInfo := pagination.BuildPageString(paymentTerms, params.Limit, cursorDir, paymentTermCreatedAt, paymentTermID)
	return &domain.ListPaymentTermsResult{PaymentTerms: result, PageInfo: pageInfo}, nil
}

func (r *paymentTermRepoImpl) Get(ctx context.Context, params domain.GetPaymentTermParams) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.get")
	defer span.End()

	row, err := r.queries.GetPaymentTerm(ctx, sqlc.GetPaymentTermParams{
		ID:        params.PaymentTermID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapPaymentTermRow(row), nil
}

func (r *paymentTermRepoImpl) Create(ctx context.Context, id string, params domain.CreatePaymentTermParams) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.create")
	defer span.End()

	err := r.queries.InsertPaymentTerm(ctx, sqlc.InsertPaymentTermParams{
		ID:        id,
		Name:      params.Name,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetPaymentTermParams{AccountID: params.AccountID, PaymentTermID: id})
}

func (r *paymentTermRepoImpl) Update(ctx context.Context, params domain.UpdatePaymentTermParams) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.update")
	defer span.End()

	result, err := r.queries.UpdatePaymentTerm(ctx, sqlc.UpdatePaymentTermParams{
		ID:        params.PaymentTermID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
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
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Payment term not found."))
	}

	return r.Get(ctx, domain.GetPaymentTermParams{AccountID: params.AccountID, PaymentTermID: params.PaymentTermID})
}

func (r *paymentTermRepoImpl) Delete(ctx context.Context, params domain.DeletePaymentTermParams) *apierror.APIError {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.delete")
	defer span.End()

	result, err := r.queries.DeletePaymentTerm(ctx, sqlc.DeletePaymentTermParams{
		ID:        params.PaymentTermID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Payment term not found."))
	}

	return nil
}

func (r *paymentTermRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.exists_by_name")
	defer span.End()

	count, err := r.queries.CountPaymentTermsByName(ctx, sqlc.CountPaymentTermsByNameParams{
		Name:      name,
		AccountID: gosql.NullString{String: accountID, Valid: true},
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
