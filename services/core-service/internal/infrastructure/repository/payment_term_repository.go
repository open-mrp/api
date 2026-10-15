package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
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
	return &domain.PaymentTerm{
		ID:        row.ID,
		Name:      row.Name,
		Status:    constants.ArchiveStatusFromIsActive(row.IsActive),
		AccountID: accountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

const paymentTermListSelect = `SELECT payment_term.id, payment_term.is_active, payment_term.name, payment_term.account_id,` +
	` payment_term.created_at, payment_term.updated_at FROM payment_term`

func scanPaymentTermListRow(rows *gosql.Rows) (*domain.PaymentTerm, error) {
	var row sqlc.PaymentTerm
	err := rows.Scan(&row.ID, &row.IsActive, &row.Name, &row.AccountID, &row.CreatedAt, &row.UpdatedAt)
	return mapPaymentTermRow(row), err
}

func (r *paymentTermRepoImpl) List(ctx context.Context, params domain.ListPaymentTermsParams) (*domain.ListPaymentTermsResult, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.list")
	defer span.End()

	cur, apiErr := decodeLookupCursor(params.Cursor)
	if apiErr != nil {
		return nil, apiErr
	}
	q := lookupListQuery{selectFrom: paymentTermListSelect, alias: "payment_term"}
	q.and("(payment_term.account_id = ? OR payment_term.account_id IS NULL)", params.AccountID)
	if search := buildPaymentTermSearchParams(params.Query); search.Valid {
		q.and("payment_term.name LIKE ?", search.String)
	}
	if params.IsActive != nil {
		q.and("payment_term.is_active = ?", *params.IsActive)
	}
	query, args := q.sql(cur, params.Limit+1)
	paymentTerms, err := queryLookupRows(ctx, r.queries.DB(), query, args, scanPaymentTermListRow)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(paymentTerms, params.Limit, cursorDirection(cur), paymentTermCreatedAt, paymentTermID)
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

func (r *paymentTermRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetPaymentTermsByIDs(ctx, sqlc.GetPaymentTermsByIDsParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.PaymentTerm, len(rows))
	for i, row := range rows {
		out[i] = mapPaymentTermRow(row)
	}
	return out, nil
}

func (r *paymentTermRepoImpl) Create(ctx context.Context, id string, params domain.CreatePaymentTermParams) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermRepoTracer.Start(ctx, "repository.payment_term.create")
	defer span.End()

	err := r.queries.InsertPaymentTerm(ctx, sqlc.InsertPaymentTermParams{
		ID:        id,
		Name:      params.Name,
		IsActive:  params.Status != constants.ArchiveStatusArchived,
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
		IsActive:  archiveStatusToNullBool(params.Status),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Payment term not found."))
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
		return tracing.Trace(span, apierror.NewNotFoundError("Payment term not found."))
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

// archiveStatusToNullBool is an optional status as the is_active column holds it; nil leaves the column alone.
func archiveStatusToNullBool(s *constants.ArchiveStatus) gosql.NullBool {
	if s == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: s.IsActive(), Valid: true}
}
