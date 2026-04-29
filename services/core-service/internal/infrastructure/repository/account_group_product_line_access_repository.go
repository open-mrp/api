package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var accountGroupProductLineAccessRepoTracer = tracing.GetTracer("core-service.account_group_product_line_access_repository")

type accountGroupProductLineAccessRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountGroupProductLineAccessRepo(queries *sqlc.Queries) domain.AccountGroupProductLineAccessRepo {
	return &accountGroupProductLineAccessRepoImpl{queries: queries}
}

func accessCreatedAt(a *domain.AccountGroupProductLineAccess) time.Time { return a.CreatedAt }
func accessAccountGroupID(a *domain.AccountGroupProductLineAccess) string {
	return a.AccountGroupID
}

func buildAccessSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

// groupForwardRows groups flat forward rows into domain objects, preserving order.
func groupForwardRows(rows []sqlc.ListAccountGroupProductLineAccessForwardRow) []*domain.AccountGroupProductLineAccess {
	var result []*domain.AccountGroupProductLineAccess
	seen := map[string]int{} // accountGroupID -> index in result

	for _, row := range rows {
		idx, ok := seen[row.AccountGroupID]
		if !ok {
			idx = len(result)
			seen[row.AccountGroupID] = idx
			result = append(result, &domain.AccountGroupProductLineAccess{
				AccountGroupID:   row.AccountGroupID,
				AccountGroupName: row.AccountGroupName,
				CreatedAt:        row.AccountGroupCreatedAt,
				UpdatedAt:        row.AccountGroupUpdatedAt,
			})
		}
		result[idx].ProductLines = append(result[idx].ProductLines, domain.ProductLineInfo{
			ID:   row.ProductLineID,
			Name: row.ProductLineName,
		})
	}

	return result
}

// groupBackwardRows groups flat backward rows into domain objects, preserving order.
func groupBackwardRows(rows []sqlc.ListAccountGroupProductLineAccessBackwardRow) []*domain.AccountGroupProductLineAccess {
	var result []*domain.AccountGroupProductLineAccess
	seen := map[string]int{}

	for _, row := range rows {
		idx, ok := seen[row.AccountGroupID]
		if !ok {
			idx = len(result)
			seen[row.AccountGroupID] = idx
			result = append(result, &domain.AccountGroupProductLineAccess{
				AccountGroupID:   row.AccountGroupID,
				AccountGroupName: row.AccountGroupName,
				CreatedAt:        row.AccountGroupCreatedAt,
				UpdatedAt:        row.AccountGroupUpdatedAt,
			})
		}
		result[idx].ProductLines = append(result[idx].ProductLines, domain.ProductLineInfo{
			ID:   row.ProductLineID,
			Name: row.ProductLineName,
		})
	}

	return result
}

// rowLimitMultiplier is used to over-fetch rows so we get enough unique account groups
// after grouping. Since each account group may have multiple product lines, we fetch
// more rows than the requested limit.
const rowLimitMultiplier int32 = 20

func (r *accountGroupProductLineAccessRepoImpl) List(ctx context.Context, params domain.ListAccountGroupProductLineAccessParams) (*domain.ListAccountGroupProductLineAccessResult, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessRepoTracer.Start(ctx, "repository.account_group_product_line_access.list")
	defer span.End()

	searchQuery := buildAccessSearchParams(params.Query)
	rowLimit := (params.Limit + 1) * rowLimitMultiplier

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAccountGroupProductLineAccessBackward(ctx, sqlc.ListAccountGroupProductLineAccessBackwardParams{
				OwnerAccountID:  params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           rowLimit,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			groups := groupBackwardRows(rows)
			result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, accessCreatedAt, accessAccountGroupID)
			return &domain.ListAccountGroupProductLineAccessResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListAccountGroupProductLineAccessForward(ctx, sqlc.ListAccountGroupProductLineAccessForwardParams{
			OwnerAccountID:  params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           rowLimit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		groups := groupForwardRows(rows)
		result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, accessCreatedAt, accessAccountGroupID)
		return &domain.ListAccountGroupProductLineAccessResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListAccountGroupProductLineAccessForward(ctx, sqlc.ListAccountGroupProductLineAccessForwardParams{
		OwnerAccountID: params.AccountID,
		SearchQuery:    searchQuery,
		Limit:          rowLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	groups := groupForwardRows(rows)
	result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, accessCreatedAt, accessAccountGroupID)
	return &domain.ListAccountGroupProductLineAccessResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *accountGroupProductLineAccessRepoImpl) Get(ctx context.Context, accountID, accountGroupID string) (*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessRepoTracer.Start(ctx, "repository.account_group_product_line_access.get")
	defer span.End()

	// Verify account group exists and belongs to this account.
	agRow, err := r.queries.GetAccountGroupByIDAndAccount(ctx, sqlc.GetAccountGroupByIDAndAccountParams{
		ID:             accountGroupID,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Get the product lines for this account group.
	plRows, err := r.queries.GetAccountGroupProductLineAccess(ctx, accountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productLines := make([]domain.ProductLineInfo, len(plRows))
	for i, row := range plRows {
		productLines[i] = domain.ProductLineInfo{
			ID:   row.ProductLineID,
			Name: row.ProductLineName,
		}
	}

	return &domain.AccountGroupProductLineAccess{
		AccountGroupID:   agRow.ID,
		AccountGroupName: agRow.Name,
		ProductLines:     productLines,
		CreatedAt:        agRow.CreatedAt,
		UpdatedAt:        agRow.UpdatedAt,
	}, nil
}

func (r *accountGroupProductLineAccessRepoImpl) Create(ctx context.Context, params domain.CreateAccountGroupProductLineAccessParams) (*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessRepoTracer.Start(ctx, "repository.account_group_product_line_access.create")
	defer span.End()

	// Verify account group exists and belongs to this account.
	_, err := r.queries.GetAccountGroupByIDAndAccount(ctx, sqlc.GetAccountGroupByIDAndAccountParams{
		ID:             params.AccountGroupID,
		OwnerAccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Verify no existing product line access mapping exists for this account group.
	count, err := r.queries.CountAccountGroupProductLinesByAccountGroupID(ctx, params.AccountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if count > 0 {
		return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Product line access already exists for this account group. Use update instead.", "account_group_id"))
	}

	// Verify each product line exists and belongs to this account.
	for _, plID := range params.ProductLineIDs {
		plCount, err := r.queries.ProductLineExistsByIDAndAccount(ctx, sqlc.ProductLineExistsByIDAndAccountParams{
			ID:        plID,
			AccountID: gosql.NullString{String: params.AccountID, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if plCount == 0 {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product line not found: "+plID))
		}
	}

	// Insert each product line mapping.
	for _, plID := range params.ProductLineIDs {
		newID, apiErr := id.GenID(id.AccountGroupProductLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		err := r.queries.InsertAccountGroupProductLine(ctx, sqlc.InsertAccountGroupProductLineParams{
			ID:             newID,
			AccountGroupID: params.AccountGroupID,
			ProductLineID:  plID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Re-fetch and return the created access.
	return r.Get(ctx, params.AccountID, params.AccountGroupID)
}

func (r *accountGroupProductLineAccessRepoImpl) Update(ctx context.Context, params domain.UpdateAccountGroupProductLineAccessParams) (*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessRepoTracer.Start(ctx, "repository.account_group_product_line_access.update")
	defer span.End()

	// Verify account group exists and belongs to this account.
	_, err := r.queries.GetAccountGroupByIDAndAccount(ctx, sqlc.GetAccountGroupByIDAndAccountParams{
		ID:             params.AccountGroupID,
		OwnerAccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Verify existing mapping exists.
	count, err := r.queries.CountAccountGroupProductLinesByAccountGroupID(ctx, params.AccountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if count == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("No product line access found for this account group."))
	}

	// Verify each product line exists and belongs to this account.
	for _, plID := range params.ProductLineIDs {
		plCount, err := r.queries.ProductLineExistsByIDAndAccount(ctx, sqlc.ProductLineExistsByIDAndAccountParams{
			ID:        plID,
			AccountID: gosql.NullString{String: params.AccountID, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if plCount == 0 {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product line not found: "+plID))
		}
	}

	// Delete old rows.
	_, err = r.queries.DeleteAccountGroupProductLinesByAccountGroupID(ctx, params.AccountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Insert new rows.
	for _, plID := range params.ProductLineIDs {
		newID, apiErr := id.GenID(id.AccountGroupProductLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		err := r.queries.InsertAccountGroupProductLine(ctx, sqlc.InsertAccountGroupProductLineParams{
			ID:             newID,
			AccountGroupID: params.AccountGroupID,
			ProductLineID:  plID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Re-fetch and return the updated access.
	return r.Get(ctx, params.AccountID, params.AccountGroupID)
}

func (r *accountGroupProductLineAccessRepoImpl) Delete(ctx context.Context, accountID, accountGroupID string) *apierror.APIError {
	ctx, span := accountGroupProductLineAccessRepoTracer.Start(ctx, "repository.account_group_product_line_access.delete")
	defer span.End()

	// Verify account group exists and belongs to this account.
	_, err := r.queries.GetAccountGroupByIDAndAccount(ctx, sqlc.GetAccountGroupByIDAndAccountParams{
		ID:             accountGroupID,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Verify mapping exists.
	count, err := r.queries.CountAccountGroupProductLinesByAccountGroupID(ctx, accountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if count == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("No product line access found for this account group."))
	}

	// Delete all rows.
	_, err = r.queries.DeleteAccountGroupProductLinesByAccountGroupID(ctx, accountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountGroupProductLineAccessRepoImpl) ExistsByAccountGroupID(ctx context.Context, accountGroupID string) (bool, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessRepoTracer.Start(ctx, "repository.account_group_product_line_access.exists_by_account_group_id")
	defer span.End()

	count, err := r.queries.CountAccountGroupProductLinesByAccountGroupID(ctx, accountGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
