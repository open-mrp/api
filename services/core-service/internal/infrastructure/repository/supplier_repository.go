package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var supplierRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.supplier")

type supplierRepoImpl struct {
	queries *sqlc.Queries
}

func NewSupplierRepo(queries *sqlc.Queries) domain.SupplierRepo {
	return &supplierRepoImpl{queries: queries}
}

func supplierSummaryCreatedAt(s *domain.SupplierSummary) time.Time { return s.CreatedAt }
func supplierSummaryID(s *domain.SupplierSummary) string           { return s.ID }

func mapSupplierForwardRow(row sqlc.ListSuppliersForwardRow, includes []string) *domain.SupplierSummary {
	summary := &domain.SupplierSummary{
		ID:            row.AccountID,
		Name:          row.AccountName,
		Number:        row.ExternalNumber,
		MaterialCount: row.MaterialCount,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.Notes.Valid {
		summary.Note = &row.Notes.String
	}
	if row.DefaultBillingAddressID.Valid {
		summary.BillToAddressID = &row.DefaultBillingAddressID.String
	}
	if row.DefaultShippingAddressID.Valid {
		summary.ShipToAddressID = &row.DefaultShippingAddressID.String
	}
	// A supplier's default addresses belong to the supplier's own account, so the gateway's
	// account-scoped address loader cannot reach them. The list joins them here, as Get does.
	if slices.Contains(includes, "bill_to_address") && row.DefaultBillingAddressID.Valid {
		summary.BillToAddress = buildCustomerAddress(
			row.DefaultBillingAddressID.String,
			row.DefaultBillingAddressName.String,
			row.DefaultBillingAddressPhone,
			row.DefaultBillingAddressEmail,
			row.DefaultBillingIsDropShip.Bool,
			row.DefaultBillingGeolocationID,
			row.DefaultBillingStreetLine1,
			row.DefaultBillingStreetLine2,
			row.DefaultBillingLocality,
			row.DefaultBillingState,
			row.DefaultBillingPostalCode,
			row.DefaultBillingCountry,
			row.DefaultBillingAddressCreatedAt.Time,
			row.DefaultBillingAddressUpdatedAt.Time,
		)
	}
	if slices.Contains(includes, "ship_to_address") && row.DefaultShippingAddressID.Valid {
		summary.ShipToAddress = buildCustomerAddress(
			row.DefaultShippingAddressID.String,
			row.DefaultShippingAddressName.String,
			row.DefaultShippingAddressPhone,
			row.DefaultShippingAddressEmail,
			row.DefaultShippingIsDropShip.Bool,
			row.DefaultShippingGeolocationID,
			row.DefaultShippingStreetLine1,
			row.DefaultShippingStreetLine2,
			row.DefaultShippingLocality,
			row.DefaultShippingState,
			row.DefaultShippingPostalCode,
			row.DefaultShippingCountry,
			row.DefaultShippingAddressCreatedAt.Time,
			row.DefaultShippingAddressUpdatedAt.Time,
		)
	}
	return summary
}

// The two list queries select the same columns in the same order, so a backward page is mapped
// through the forward mapper rather than a second copy that can drift from it — as it had, dropping
// the note, the updated timestamp and both addresses from every backward page.
func mapSupplierBackwardRow(row sqlc.ListSuppliersBackwardRow, includes []string) *domain.SupplierSummary {
	return mapSupplierForwardRow(sqlc.ListSuppliersForwardRow(row), includes)
}

func (r *supplierRepoImpl) List(ctx context.Context, params domain.ListSuppliersParams) (*domain.ListSuppliersResult, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.list")
	defer span.End()

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*params.Query) + "%", Valid: true}
	}

	startDate := gosql.NullTime{}
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}

	endDate := gosql.NullTime{}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	hasItemFilter := len(params.ItemIDs) > 0
	itemIDs := params.ItemIDs
	if !hasItemFilter {
		itemIDs = []string{}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSuppliersBackward(ctx, sqlc.ListSuppliersBackwardParams{
				OwnerAccountID:  params.OwnerAccountID,
				SearchQuery:     searchQuery,
				StartDate:       startDate,
				EndDate:         endDate,
				HasItemFilter:   hasItemFilter,
				ItemIds:         itemIDs,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.SupplierSummary, len(rows))
			for i, row := range rows {
				items[i] = mapSupplierBackwardRow(row, params.Includes)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, supplierSummaryCreatedAt, supplierSummaryID)
			return &domain.ListSuppliersResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListSuppliersForward(ctx, sqlc.ListSuppliersForwardParams{
			OwnerAccountID:  params.OwnerAccountID,
			SearchQuery:     searchQuery,
			StartDate:       startDate,
			EndDate:         endDate,
			HasItemFilter:   hasItemFilter,
			ItemIds:         itemIDs,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.SupplierSummary, len(rows))
		for i, row := range rows {
			items[i] = mapSupplierForwardRow(row, params.Includes)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, supplierSummaryCreatedAt, supplierSummaryID)
		return &domain.ListSuppliersResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListSuppliersForward(ctx, sqlc.ListSuppliersForwardParams{
		OwnerAccountID: params.OwnerAccountID,
		SearchQuery:    searchQuery,
		StartDate:      startDate,
		EndDate:        endDate,
		HasItemFilter:  hasItemFilter,
		ItemIds:        itemIDs,
		Limit:          params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.SupplierSummary, len(rows))
	for i, row := range rows {
		items[i] = mapSupplierForwardRow(row, params.Includes)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, supplierSummaryCreatedAt, supplierSummaryID)
	return &domain.ListSuppliersResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *supplierRepoImpl) Get(ctx context.Context, params domain.GetSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.get")
	defer span.End()

	row, err := r.queries.GetSupplier(ctx, sqlc.GetSupplierParams{
		OwnerAccountID:        params.OwnerAccountID,
		CounterpartyAccountID: params.SupplierID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var billToAddress *domain.CustomerAddress
	if slices.Contains(params.Includes, "bill_to_address") && row.DefaultBillingAddressID.Valid {
		billToAddress = buildCustomerAddress(
			row.DefaultBillingAddressID.String,
			row.DefaultBillingAddressName.String,
			row.DefaultBillingAddressPhone,
			row.DefaultBillingAddressEmail,
			row.DefaultBillingIsDropShip.Bool,
			row.DefaultBillingGeolocationID,
			row.DefaultBillingStreetLine1,
			row.DefaultBillingStreetLine2,
			row.DefaultBillingLocality,
			row.DefaultBillingState,
			row.DefaultBillingPostalCode,
			row.DefaultBillingCountry,
			row.DefaultBillingAddressCreatedAt.Time,
			row.DefaultBillingAddressUpdatedAt.Time,
		)
	}

	var shipToAddress *domain.CustomerAddress
	if slices.Contains(params.Includes, "ship_to_address") && row.DefaultShippingAddressID.Valid {
		shipToAddress = buildCustomerAddress(
			row.DefaultShippingAddressID.String,
			row.DefaultShippingAddressName.String,
			row.DefaultShippingAddressPhone,
			row.DefaultShippingAddressEmail,
			row.DefaultShippingIsDropShip.Bool,
			row.DefaultShippingGeolocationID,
			row.DefaultShippingStreetLine1,
			row.DefaultShippingStreetLine2,
			row.DefaultShippingLocality,
			row.DefaultShippingState,
			row.DefaultShippingPostalCode,
			row.DefaultShippingCountry,
			row.DefaultShippingAddressCreatedAt.Time,
			row.DefaultShippingAddressUpdatedAt.Time,
		)
	}

	return &domain.Supplier{
		ID:            row.AccountID,
		Name:          row.AccountName,
		Number:        row.ExternalNumber,
		Note:          nullStringPtr(row.Notes),
		BillToAddress: billToAddress,
		ShipToAddress: shipToAddress,
		MaterialCount: row.MaterialCount,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *supplierRepoImpl) Create(ctx context.Context, accountID, relationID string, params domain.CreateSupplierParams, billToAddressID, shipToAddressID *string) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.create")
	defer span.End()

	notes := gosql.NullString{}
	if params.Note != nil {
		notes = gosql.NullString{String: *params.Note, Valid: true}
	}

	billAddr := gosql.NullString{}
	if billToAddressID != nil {
		billAddr = gosql.NullString{String: *billToAddressID, Valid: true}
	}

	shipAddr := gosql.NullString{}
	if shipToAddressID != nil {
		shipAddr = gosql.NullString{String: *shipToAddressID, Valid: true}
	}

	err := r.queries.InsertSupplierAccount(ctx, sqlc.InsertSupplierAccountParams{
		ID:                       accountID,
		Name:                     params.Name,
		DefaultBillingAddressID:  billAddr,
		DefaultShippingAddressID: shipAddr,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	err = r.queries.InsertSupplierRelation(ctx, sqlc.InsertSupplierRelationParams{
		ID:                       relationID,
		OwnerAccountID:           params.OwnerAccountID,
		CounterpartyAccountID:    accountID,
		Alias:                    gosql.NullString{String: params.Name, Valid: true},
		ExternalNumber:           params.Number,
		Notes:                    notes,
		DefaultBillingAddressID:  billAddr,
		DefaultShippingAddressID: shipAddr,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: accountID, Includes: params.Includes})
}

func (r *supplierRepoImpl) Update(ctx context.Context, params domain.UpdateSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.update")
	defer span.End()

	externalNumber := gosql.NullString{}
	if params.Number != nil {
		externalNumber = gosql.NullString{String: *params.Number, Valid: true}
	}

	notes := gosql.NullString{}
	if params.Note != nil {
		notes = gosql.NullString{String: *params.Note, Valid: true}
	}

	billAddrID := gosql.NullString{}
	if params.BillToAddressID != nil {
		billAddrID = gosql.NullString{String: *params.BillToAddressID, Valid: true}
	}

	shipAddrID := gosql.NullString{}
	if params.ShipToAddressID != nil {
		shipAddrID = gosql.NullString{String: *params.ShipToAddressID, Valid: true}
	}

	err := r.queries.UpdateSupplierRelation(ctx, sqlc.UpdateSupplierRelationParams{
		ExternalNumber:           externalNumber,
		UpdateNotes:              params.UpdateNote,
		Notes:                    notes,
		DefaultBillingAddressID:  billAddrID,
		DefaultShippingAddressID: shipAddrID,
		OwnerAccountID:           params.OwnerAccountID,
		CounterpartyAccountID:    params.SupplierID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.Name != nil {
		err = r.queries.UpdateSupplierAccountName(ctx, sqlc.UpdateSupplierAccountNameParams{
			Name: *params.Name,
			ID:   params.SupplierID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierID, Includes: params.Includes})
}

func (r *supplierRepoImpl) Delete(ctx context.Context, ownerAccountID, supplierAccountID string) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.delete")
	defer span.End()

	// Fetch with all sub-resources for audit trail.
	supplier, apiErr := r.Get(ctx, domain.GetSupplierParams{OwnerAccountID: ownerAccountID, SupplierID: supplierAccountID, Includes: []string{"bill_to_address", "ship_to_address"}})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	err := r.queries.DeleteSupplierAccountUsers(ctx, supplierAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	err = r.queries.DeleteSupplierAccountAddresses(ctx, supplierAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	err = r.queries.DeleteSupplierRelation(ctx, sqlc.DeleteSupplierRelationParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: supplierAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return supplier, nil
}

func (r *supplierRepoImpl) BulkDelete(ctx context.Context, ownerAccountID string, supplierAccountIDs []string) *apierror.APIError {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.bulk_delete")
	defer span.End()

	err := r.queries.BulkDeleteSupplierAccountUsers(ctx, supplierAccountIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.BulkDeleteSupplierAccountAddresses(ctx, supplierAccountIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.BulkDeleteSupplierRelations(ctx, sqlc.BulkDeleteSupplierRelationsParams{
		OwnerAccountID:         ownerAccountID,
		CounterpartyAccountIds: supplierAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *supplierRepoImpl) FindByNames(ctx context.Context, ownerAccountID string, names []string) ([]*domain.SupplierNameMatch, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.find_by_names")
	defer span.End()

	if len(names) == 0 {
		return nil, nil
	}

	rows, err := r.queries.FindSuppliersByNames(ctx, sqlc.FindSuppliersByNamesParams{
		OwnerAccountID: ownerAccountID,
		Names:          names,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	matches := make([]*domain.SupplierNameMatch, len(rows))
	for i, row := range rows {
		matches[i] = &domain.SupplierNameMatch{
			AccountID: row.AccountID,
			Name:      row.AccountName,
		}
	}
	return matches, nil
}

func (r *supplierRepoImpl) ExistsByNumber(ctx context.Context, ownerAccountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.exists_by_number")
	defer span.End()

	excludeCounterpartyID := gosql.NullString{}
	if excludeID != nil {
		excludeCounterpartyID = gosql.NullString{String: *excludeID, Valid: true}
	}

	exists, err := r.queries.SupplierExistsByNumber(ctx, sqlc.SupplierExistsByNumberParams{
		OwnerAccountID:        ownerAccountID,
		ExternalNumber:        number,
		ExcludeCounterpartyID: excludeCounterpartyID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}
