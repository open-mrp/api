package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var addressRepoTracer = tracing.GetTracer("core-service.address_repository")

type addressRepoImpl struct {
	queries *sqlc.Queries
}

func NewAddressRepo(queries *sqlc.Queries) domain.AddressRepo {
	return &addressRepoImpl{queries: queries}
}

func addressCreatedAt(a *domain.Address) time.Time { return a.CreatedAt }
func addressID(a *domain.Address) string           { return a.ID }

func mapAddressRow(
	id string,
	name string,
	phone gosql.NullString,
	email gosql.NullString,
	isDropShip bool,
	createdAt, updatedAt time.Time,
	geoID string,
	streetLine1, streetLine2, locality, state, postalCode gosql.NullString,
	country string,
	googlePlaceID gosql.NullString,
	latitude, longitude gosql.NullFloat64,
) *domain.Address {
	addr := &domain.Address{
		ID:         id,
		Name:       name,
		IsDropShip: isDropShip,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Geolocation: &domain.Geolocation{
			ID:      geoID,
			Country: country,
		},
	}

	if phone.Valid {
		addr.Phone = &phone.String
	}
	if email.Valid {
		addr.Email = &email.String
	}

	if streetLine1.Valid {
		addr.Geolocation.StreetLine1 = &streetLine1.String
	}
	if streetLine2.Valid {
		addr.Geolocation.StreetLine2 = &streetLine2.String
	}
	if locality.Valid {
		addr.Geolocation.Locality = &locality.String
	}
	if state.Valid {
		addr.Geolocation.State = &state.String
	}
	if postalCode.Valid {
		addr.Geolocation.PostalCode = &postalCode.String
	}
	if googlePlaceID.Valid {
		addr.Geolocation.GooglePlaceID = &googlePlaceID.String
	}
	if latitude.Valid {
		addr.Geolocation.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		addr.Geolocation.Longitude = &longitude.Float64
	}

	return addr
}

func mapForwardAddressRow(row sqlc.ListAddressesForwardRow) *domain.Address {
	return mapAddressRow(
		row.ID, row.Name, row.Phone, row.Email, row.IsDropShip,
		row.CreatedAt, row.UpdatedAt,
		row.GeolocationID, row.StreetLine1, row.StreetLine2, row.Locality, row.State, row.PostalCode,
		row.Country, row.GooglePlaceID, row.Latitude, row.Longitude,
	)
}

func mapBackwardAddressRow(row sqlc.ListAddressesBackwardRow) *domain.Address {
	return mapAddressRow(
		row.ID, row.Name, row.Phone, row.Email, row.IsDropShip,
		row.CreatedAt, row.UpdatedAt,
		row.GeolocationID, row.StreetLine1, row.StreetLine2, row.Locality, row.State, row.PostalCode,
		row.Country, row.GooglePlaceID, row.Latitude, row.Longitude,
	)
}

func mapGetAddressRow(row sqlc.GetAddressRow) *domain.Address {
	return mapAddressRow(
		row.ID, row.Name, row.Phone, row.Email, row.IsDropShip,
		row.CreatedAt, row.UpdatedAt,
		row.GeolocationID, row.StreetLine1, row.StreetLine2, row.Locality, row.State, row.PostalCode,
		row.Country, row.GooglePlaceID, row.Latitude, row.Longitude,
	)
}

func buildAddressSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *addressRepoImpl) List(ctx context.Context, params domain.ListAddressesParams) (*domain.ListAddressesResult, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.list")
	defer span.End()

	searchQuery := buildAddressSearchParams(params.Query)
	dropShip := gosql.NullBool{}
	if params.DropShip != nil {
		dropShip = gosql.NullBool{Bool: *params.DropShip, Valid: true}
	}
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAddressesBackward(ctx, sqlc.ListAddressesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				DropShip:        dropShip,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			addresses := make([]*domain.Address, len(rows))
			for i, row := range rows {
				addresses[i] = mapBackwardAddressRow(row)
			}
			result, pageInfo := pagination.BuildPageString(addresses, params.Limit, cursorDir, addressCreatedAt, addressID)
			return &domain.ListAddressesResult{Addresses: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListAddressesForward(ctx, sqlc.ListAddressesForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			DropShip:        dropShip,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		addresses := make([]*domain.Address, len(rows))
		for i, row := range rows {
			addresses[i] = mapForwardAddressRow(row)
		}
		result, pageInfo := pagination.BuildPageString(addresses, params.Limit, cursorDir, addressCreatedAt, addressID)
		return &domain.ListAddressesResult{Addresses: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListAddressesForward(ctx, sqlc.ListAddressesForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		DropShip:    dropShip,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	addresses := make([]*domain.Address, len(rows))
	for i, row := range rows {
		addresses[i] = mapForwardAddressRow(row)
	}
	result, pageInfo := pagination.BuildPageString(addresses, params.Limit, cursorDir, addressCreatedAt, addressID)
	return &domain.ListAddressesResult{Addresses: result, PageInfo: pageInfo}, nil
}

func (r *addressRepoImpl) Get(ctx context.Context, params domain.GetAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.get")
	defer span.End()

	row, err := r.queries.GetAddress(ctx, sqlc.GetAddressParams{
		ID:        params.AddressID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetAddressRow(row), nil
}

func (r *addressRepoImpl) Create(ctx context.Context, addressID, geolocationID, accountAddressID string, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.create")
	defer span.End()

	// Insert geolocation
	if err := r.queries.InsertGeolocation(ctx, sqlc.InsertGeolocationParams{
		ID:          geolocationID,
		StreetLine1: toNullString(params.StreetLine1),
		StreetLine2: toNullString(params.StreetLine2),
		Locality:    toNullString(params.Locality),
		State:       toNullString(params.State),
		PostalCode:  toNullString(params.PostalCode),
		Country:     params.Country,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert address
	if err := r.queries.InsertAddress(ctx, sqlc.InsertAddressParams{
		ID:            addressID,
		Name:          params.Name,
		Phone:         toNullString(params.Phone),
		Email:         toNullString(params.Email),
		IsDropShip:    params.IsDropShip,
		GeolocationID: geolocationID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert account_address
	if err := r.queries.InsertAccountAddress(ctx, sqlc.InsertAccountAddressParams{
		ID:        accountAddressID,
		AccountID: params.AccountID,
		AddressID: addressID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, domain.GetAddressParams{
		AccountID: params.AccountID,
		AddressID: addressID,
	})
}

func (r *addressRepoImpl) Update(ctx context.Context, params domain.UpdateAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.update")
	defer span.End()

	result, err := r.queries.UpdateAddress(ctx, sqlc.UpdateAddressParams{
		ID:         params.AddressID,
		Name:       toNullString(params.Name),
		Phone:      toNullString(params.Phone),
		Email:      toNullString(params.Email),
		IsDropShip: toNullBool(params.IsDropShip),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Address not found."))
	}

	return r.Get(ctx, domain.GetAddressParams{
		AccountID: params.AccountID,
		AddressID: params.AddressID,
	})
}

func (r *addressRepoImpl) Delete(ctx context.Context, params domain.DeleteAddressParams) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.delete")
	defer span.End()

	// Delete account_address records
	if err := r.queries.DeleteAccountAddressByAddressID(ctx, params.AddressID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete address
	if err := r.queries.DeleteAddress(ctx, params.AddressID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *addressRepoImpl) IsInAccount(ctx context.Context, accountID, addressID string) (bool, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.is_in_account")
	defer span.End()

	exists, err := r.queries.CheckAddressInAccount(ctx, sqlc.CheckAddressInAccountParams{
		AccountID: accountID,
		AddressID: addressID,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check address in account."))
	}
	return exists, nil
}

func (r *addressRepoImpl) GetGeolocationSharedCount(ctx context.Context, geolocationID string) (int64, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.get_geolocation_shared_count")
	defer span.End()

	count, err := r.queries.CountAddressesByGeolocationID(ctx, geolocationID)
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to count addresses by geolocation."))
	}
	return count, nil
}

func (r *addressRepoImpl) GetGeolocationIDByAddressID(ctx context.Context, addressID string) (string, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.get_geolocation_id")
	defer span.End()

	geoID, err := r.queries.GetGeolocationIDByAddressID(ctx, addressID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return geoID, nil
}

func (r *addressRepoImpl) CreateGeolocation(ctx context.Context, id string, params domain.CreateAddressParams) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.create_geolocation")
	defer span.End()

	err := r.queries.InsertGeolocation(ctx, sqlc.InsertGeolocationParams{
		ID:          id,
		StreetLine1: toNullString(params.StreetLine1),
		StreetLine2: toNullString(params.StreetLine2),
		Locality:    toNullString(params.Locality),
		State:       toNullString(params.State),
		PostalCode:  toNullString(params.PostalCode),
		Country:     params.Country,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *addressRepoImpl) UpdateGeolocation(ctx context.Context, geolocationID string, params domain.UpdateAddressParams) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.update_geolocation")
	defer span.End()

	_, err := r.queries.UpdateGeolocation(ctx, sqlc.UpdateGeolocationParams{
		ID:          geolocationID,
		StreetLine1: toNullString(params.StreetLine1),
		StreetLine2: toNullString(params.StreetLine2),
		Locality:    toNullString(params.Locality),
		State:       toNullString(params.State),
		PostalCode:  toNullString(params.PostalCode),
		Country:     toNullString(params.Country),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *addressRepoImpl) RelinkGeolocation(ctx context.Context, addressID, geolocationID string) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.relink_geolocation")
	defer span.End()

	_, err := r.queries.UpdateAddressGeolocationID(ctx, sqlc.UpdateAddressGeolocationIDParams{
		ID:            addressID,
		GeolocationID: geolocationID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *addressRepoImpl) CheckAddressNotInUse(ctx context.Context, addressID string) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.check_not_in_use")
	defer span.End()

	// Check sales orders
	orderNumber, err := r.queries.CheckAddressUsedInSalesOrder(ctx, sqlc.CheckAddressUsedInSalesOrderParams{AddressID: addressID})
	if err != nil && err != gosql.ErrNoRows {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check address usage in sales orders."))
	}
	if err == nil {
		return apierror.NewValidationError(fmt.Sprintf("Cannot delete address that is used as a billing or shipping address in a sales order: %s", orderNumber))
	}

	// Check invoices
	invoiceNumber, err := r.queries.CheckAddressUsedInInvoice(ctx, addressID)
	if err != nil && err != gosql.ErrNoRows {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check address usage in invoices."))
	}
	if err == nil {
		return apierror.NewValidationError(fmt.Sprintf("Cannot delete address that is used as a billing address in an invoice: %s", invoiceNumber))
	}

	// Check shipments
	shipmentNumber, err := r.queries.CheckAddressUsedInShipment(ctx, addressID)
	if err != nil && err != gosql.ErrNoRows {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check address usage in shipments."))
	}
	if err == nil {
		return apierror.NewValidationError(fmt.Sprintf("Cannot delete address that is used as a shipping address in a shipment: %s", shipmentNumber))
	}

	// Check account defaults
	accountName, err := r.queries.CheckAddressUsedAsAccountDefault(ctx, sqlc.CheckAddressUsedAsAccountDefaultParams{AddressID: gosql.NullString{String: addressID, Valid: true}})
	if err != nil && err != gosql.ErrNoRows {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check address usage as account default."))
	}
	if err == nil {
		return apierror.NewValidationError(fmt.Sprintf("Cannot delete address that is used as a default billing or shipping address in an account: %s", accountName))
	}

	return nil
}

func toNullBool(b *bool) gosql.NullBool {
	if b == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: *b, Valid: true}
}
