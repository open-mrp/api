package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var supplierRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.supplier")

type supplierRepoImpl struct {
	queries *sqlc.Queries
}

func NewSupplierRepo(queries *sqlc.Queries) domain.SupplierRepo {
	return &supplierRepoImpl{queries: queries}
}

func supplierCreatedAt(s *domain.Supplier) time.Time { return s.CreatedAt }
func supplierID(s *domain.Supplier) string           { return s.ID }

// supplierInUseQuery probes, each from its own key, for the owner's purchase orders from the supplier and for the
// supplier's items.
const supplierInUseQuery = `SELECT 'purchase orders' FROM DUAL WHERE EXISTS (
    SELECT 1 FROM sales_order FORCE INDEX (sales_order_owner_account_id_seller_account_id_sales_order_t_idx)
    WHERE owner_account_id = ? AND seller_account_id = ?)
UNION ALL
SELECT 'supplier items' FROM DUAL WHERE EXISTS (
    SELECT 1 FROM supplier_material FORCE INDEX (supplier_material_owner_supplier_created_idx)
    WHERE owner_account_id = ? AND supplier_account_id = ?)`

// mapSupplierRow reads a supplier, with the default addresses the caller asked for. A supplier's default addresses
// belong to the supplier's own account, so the gateway's account-scoped address loader cannot reach them.
func mapSupplierRow(row sqlc.GetSupplierRow, includes []string) *domain.Supplier {
	s := &domain.Supplier{
		ID:                    row.AccountID,
		Name:                  row.AccountName,
		Number:                row.ExternalNumber,
		ArchiveStatus:         constants.ArchiveStatusFromIsActive(row.IsActive),
		Instructions:          nullStringPtr(row.Instructions),
		Email:                 nullStringPtr(row.Email),
		Phone:                 nullStringPtr(row.PhoneNumber),
		URL:                   nullStringPtr(row.WebsiteUrl),
		CarrierBillingAccount: nullStringPtr(row.CarrierBillingAccount),
		DefaultCarrierID:      nullStringPtr(row.DefaultCarrierID),
		DefaultServiceLevelID: nullStringPtr(row.DefaultCarrierOptionID),
		PaymentTermID:         nullStringPtr(row.PaymentTermID),
		ShippingTermID:        nullStringPtr(row.ShippingTermID),
		BillToAddressID:       nullStringPtr(row.DefaultBillingAddressID),
		ShipToAddressID:       nullStringPtr(row.DefaultShippingAddressID),
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	if row.CarrierBillingType.Valid && row.CarrierBillingType.String != "" {
		t := constants.CarrierBillingType(row.CarrierBillingType.String)
		s.CarrierBillingType = &t
	}
	if slices.Contains(includes, domain.SupplierIncludeBillToAddress) && row.DefaultBillingAddressID.Valid {
		s.BillToAddress = buildCustomerAddress(
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
	if slices.Contains(includes, domain.SupplierIncludeShipToAddress) && row.DefaultShippingAddressID.Valid {
		s.ShipToAddress = buildCustomerAddress(
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
	return s
}

// readByIDs reads the owner's suppliers among ids, keyed by ID, counting their items when asked.
func (r *supplierRepoImpl) readByIDs(ctx context.Context, ownerAccountID string, ids, includes []string) (map[string]*domain.Supplier, *apierror.APIError) {
	byID := make(map[string]*domain.Supplier, len(ids))
	if len(ids) == 0 {
		return byID, nil
	}
	rows, err := r.queries.ListSuppliersByIDs(ctx, sqlc.ListSuppliersByIDsParams{OwnerAccountID: ownerAccountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		byID[row.AccountID] = mapSupplierRow(sqlc.GetSupplierRow(row), includes)
	}
	if apiErr := r.countMaterials(ctx, ownerAccountID, byID, includes); apiErr != nil {
		return nil, apiErr
	}
	return byID, nil
}

// countMaterials sets each supplier's item count when the caller asked for it.
func (r *supplierRepoImpl) countMaterials(ctx context.Context, ownerAccountID string, byID map[string]*domain.Supplier, includes []string) *apierror.APIError {
	if !slices.Contains(includes, domain.SupplierIncludeMaterialCount) || len(byID) == 0 {
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id, s := range byID {
		ids = append(ids, id)
		s.MaterialCount = new(int64)
	}
	rows, err := r.queries.CountSupplierMaterials(ctx, sqlc.CountSupplierMaterialsParams{OwnerAccountID: ownerAccountID, SupplierIds: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	for _, row := range rows {
		if s, ok := byID[row.SupplierAccountID]; ok {
			*s.MaterialCount = row.MaterialCount
		}
	}
	return nil
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

	byID, apiErr := r.readByIDs(ctx, params.OwnerAccountID, ids, params.Includes)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, pageInfo := pagination.BuildPageString(inPageOrder(ids, byID), params.Limit, cursorDirection(cursor), supplierCreatedAt, supplierID)
	return &domain.ListSuppliersResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *supplierRepoImpl) GetByIDs(ctx context.Context, ownerAccountID string, ids, includes []string) ([]*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.get_by_ids")
	defer span.End()

	byID, apiErr := r.readByIDs(ctx, ownerAccountID, ids, includes)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Supplier, 0, len(byID))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			out = append(out, s)
			delete(byID, id)
		}
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

	s := mapSupplierRow(row, params.Includes)
	if apiErr := r.countMaterials(ctx, params.OwnerAccountID, map[string]*domain.Supplier{s.ID: s}, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s, nil
}

func (r *supplierRepoImpl) Create(ctx context.Context, accountID, relationID string, params domain.CreateSupplierParams, billToAddressID, shipToAddressID *string) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.create")
	defer span.End()

	err := r.queries.InsertSupplierAccount(ctx, sqlc.InsertSupplierAccountParams{
		ID:                       accountID,
		Name:                     params.Name,
		DefaultBillingAddressID:  ptrToNullString(billToAddressID),
		DefaultShippingAddressID: ptrToNullString(shipToAddressID),
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
		IsActive:                 params.ArchiveStatus.IsActive(),
		Instructions:             ptrToNullString(params.Instructions),
		CarrierBillingType:       carrierBillingTypeNull(params.CarrierBillingType),
		CarrierBillingAccount:    ptrToNullString(params.CarrierBillingAccount),
		DefaultCarrierID:         ptrToNullString(params.DefaultCarrierID),
		DefaultCarrierOptionID:   ptrToNullString(params.DefaultServiceLevelID),
		PaymentTermID:            ptrToNullString(params.PaymentTermID),
		ShippingTermID:           ptrToNullString(params.ShippingTermID),
		DefaultBillingAddressID:  ptrToNullString(billToAddressID),
		DefaultShippingAddressID: ptrToNullString(shipToAddressID),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, supplierNumberDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: accountID, Includes: params.Includes})
}

func (r *supplierRepoImpl) Update(ctx context.Context, w domain.SupplierRelationWrite, includes []string) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.update")
	defer span.End()

	err := r.queries.UpdateSupplierRelation(ctx, sqlc.UpdateSupplierRelationParams{
		Alias:                    gosql.NullString{String: w.Name, Valid: true},
		ExternalNumber:           w.Number,
		IsActive:                 w.IsActive,
		Instructions:             ptrToNullString(w.Instructions),
		CarrierBillingType:       carrierBillingTypeNull(w.CarrierBillingType),
		CarrierBillingAccount:    ptrToNullString(w.CarrierBillingAccount),
		DefaultCarrierID:         ptrToNullString(w.DefaultCarrierID),
		DefaultCarrierOptionID:   ptrToNullString(w.DefaultServiceLevelID),
		PaymentTermID:            ptrToNullString(w.PaymentTermID),
		ShippingTermID:           ptrToNullString(w.ShippingTermID),
		DefaultBillingAddressID:  ptrToNullString(w.BillToAddressID),
		DefaultShippingAddressID: ptrToNullString(w.ShipToAddressID),
		OwnerAccountID:           w.OwnerAccountID,
		CounterpartyAccountID:    w.SupplierID,
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, supplierNumberDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetSupplierParams{OwnerAccountID: w.OwnerAccountID, SupplierID: w.SupplierID, Includes: includes})
}

func (r *supplierRepoImpl) InUseBy(ctx context.Context, ownerAccountID, supplierAccountID string) ([]string, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.in_use_by")
	defer span.End()

	labels, err := selectStrings(ctx, r.queries.DB(), supplierInUseQuery, ownerAccountID, supplierAccountID, ownerAccountID, supplierAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return labels, nil
}

func (r *supplierRepoImpl) Delete(ctx context.Context, ownerAccountID, supplierAccountID string) *apierror.APIError {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.delete")
	defer span.End()

	err := r.queries.DeleteSupplierAccountUsers(ctx, supplierAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.DeleteSupplierAccountAddresses(ctx, supplierAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.DeleteSupplierRelation(ctx, sqlc.DeleteSupplierRelationParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: supplierAccountID,
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

func (r *supplierRepoImpl) LockNumbers(ctx context.Context, ownerAccountID string) *apierror.APIError {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.lock_numbers")
	defer span.End()

	_, err := r.queries.LockSupplierNumbers(ctx, ownerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *supplierRepoImpl) ExistsByNumber(ctx context.Context, ownerAccountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.exists_by_number")
	defer span.End()

	exists, err := r.queries.SupplierExistsByNumber(ctx, sqlc.SupplierExistsByNumberParams{
		OwnerAccountID:        ownerAccountID,
		ExternalNumber:        number,
		ExcludeCounterpartyID: ptrToNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func carrierBillingTypeNull(t *constants.CarrierBillingType) gosql.NullString {
	if t == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: string(*t), Valid: true}
}

func (r *supplierRepoImpl) AllocateNextNumber(ctx context.Context, sysPropertyID, ownerAccountID string) (int64, *apierror.APIError) {
	ctx, span := supplierRepoTracer.Start(ctx, "repository.supplier.allocate_next_number")
	defer span.End()

	number, apiErr := supplierNumbers(r.queries, ownerAccountID).next(ctx, sysPropertyID)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return number, nil
}
