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

// supplierAddresses builds the default addresses a caller asked for. A supplier's default addresses belong to the
// supplier's own account, so the gateway's account-scoped address loader cannot reach them; they are joined here.
func supplierAddresses(row sqlc.GetSupplierRow, includes []string) (billTo, shipTo *domain.CustomerAddress) {
	if slices.Contains(includes, "bill_to_address") && row.DefaultBillingAddressID.Valid {
		billTo = buildCustomerAddress(
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
		shipTo = buildCustomerAddress(
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
	return billTo, shipTo
}

func mapSupplierRow(row sqlc.GetSupplierRow, includes []string) *domain.Supplier {
	billTo, shipTo := supplierAddresses(row, includes)
	return &domain.Supplier{
		ID:            row.AccountID,
		Name:          row.AccountName,
		Number:        row.ExternalNumber,
		Note:          nullStringPtr(row.Notes),
		BillToAddress: billTo,
		ShipToAddress: shipTo,
		MaterialCount: row.MaterialCount,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

// The by-IDs query selects GetSupplier's columns, so a listed row is read through the same mapper.
func mapSupplierSummaryRow(row sqlc.ListSuppliersByIDsRow, includes []string) *domain.SupplierSummary {
	full := sqlc.GetSupplierRow(row)
	billTo, shipTo := supplierAddresses(full, includes)
	return &domain.SupplierSummary{
		ID:              row.AccountID,
		Name:            row.AccountName,
		Number:          row.ExternalNumber,
		Note:            nullStringPtr(row.Notes),
		BillToAddressID: nullStringPtr(row.DefaultBillingAddressID),
		ShipToAddressID: nullStringPtr(row.DefaultShippingAddressID),
		BillToAddress:   billTo,
		ShipToAddress:   shipTo,
		MaterialCount:   row.MaterialCount,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (r *supplierRepoImpl) List(ctx context.Context, params domain.ListSuppliersParams) (*domain.ListSuppliersResult, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.list")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	query, args := supplierListPageQuery(params, cursor, params.Limit+1)
	ids, err := selectStrings(ctx, r.queries.DB(), query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	byID := make(map[string]*domain.SupplierSummary, len(ids))
	if len(ids) > 0 {
		rows, err := r.queries.ListSuppliersByIDs(ctx, sqlc.ListSuppliersByIDsParams{
			OwnerAccountID: params.OwnerAccountID,
			Ids:            ids,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range rows {
			byID[row.AccountID] = mapSupplierSummaryRow(row, params.Includes)
		}
	}

	result, pageInfo := pagination.BuildPageString(inPageOrder(ids, byID), params.Limit, cursorDirection(cursor), supplierSummaryCreatedAt, supplierSummaryID)
	return &domain.ListSuppliersResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *supplierRepoImpl) GetByIDs(ctx context.Context, ownerAccountID string, ids []string) ([]*domain.SupplierSummary, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.queries.ListSuppliersByIDs(ctx, sqlc.ListSuppliersByIDsParams{
		OwnerAccountID: ownerAccountID,
		Ids:            ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.SupplierSummary, len(rows))
	for i, row := range rows {
		out[i] = mapSupplierSummaryRow(row, nil)
	}
	return out, nil
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

	return mapSupplierRow(row, params.Includes), nil
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

	err := r.queries.UpdateSupplierRelation(ctx, sqlc.UpdateSupplierRelationParams{
		Alias:                    ptrToNullString(params.Name),
		ExternalNumber:           ptrToNullString(params.Number),
		UpdateNotes:              params.UpdateNote,
		Notes:                    ptrToNullString(params.Note),
		DefaultBillingAddressID:  ptrToNullString(params.BillToAddressID),
		DefaultShippingAddressID: ptrToNullString(params.ShipToAddressID),
		OwnerAccountID:           params.OwnerAccountID,
		CounterpartyAccountID:    params.SupplierID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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

	err = r.queries.DeleteSupplierMaterialsBySuppliers(ctx, sqlc.DeleteSupplierMaterialsBySuppliersParams{
		OwnerAccountID:     ownerAccountID,
		SupplierAccountIds: []string{supplierAccountID},
	})
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

	err = r.queries.DeleteSupplierMaterialsBySuppliers(ctx, sqlc.DeleteSupplierMaterialsBySuppliersParams{
		OwnerAccountID:     ownerAccountID,
		SupplierAccountIds: supplierAccountIDs,
	})
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

	aliasNames := make([]gosql.NullString, len(names))
	for i, name := range names {
		aliasNames[i] = gosql.NullString{String: name, Valid: true}
	}
	rows, err := r.queries.FindSuppliersByNames(ctx, sqlc.FindSuppliersByNamesParams{
		OwnerAccountID: ownerAccountID,
		AliasNames:     aliasNames,
		AccountNames:   names,
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
