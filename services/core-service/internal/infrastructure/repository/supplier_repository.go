package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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

func mapSupplierForwardRow(row sqlc.ListSuppliersForwardRow) *domain.SupplierSummary {
	return &domain.SupplierSummary{
		ID:            row.AccountID,
		Name:          row.AccountName,
		Number:        row.ExternalNumber,
		MaterialCount: row.MaterialCount,
		CreatedAt:     row.CreatedAt,
	}
}

func mapSupplierBackwardRow(row sqlc.ListSuppliersBackwardRow) *domain.SupplierSummary {
	return &domain.SupplierSummary{
		ID:            row.AccountID,
		Name:          row.AccountName,
		Number:        row.ExternalNumber,
		MaterialCount: row.MaterialCount,
		CreatedAt:     row.CreatedAt,
	}
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
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
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
				items[i] = mapSupplierBackwardRow(row)
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
			items[i] = mapSupplierForwardRow(row)
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
		items[i] = mapSupplierForwardRow(row)
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
