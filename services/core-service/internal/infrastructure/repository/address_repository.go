package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
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
	timezone gosql.NullString,
	receiveCalendarID gosql.NullString,
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
	if timezone.Valid {
		addr.Geolocation.Timezone = &timezone.String
	}
	if receiveCalendarID.Valid {
		addr.ReceiveCalendarID = &receiveCalendarID.String
	}

	return addr
}

func mapGetAddressRow(row sqlc.GetAddressRow) *domain.Address {
	return mapAddressRow(
		row.ID, row.Name, row.Phone, row.Email, row.IsDropShip,
		row.CreatedAt, row.UpdatedAt,
		row.GeolocationID, row.StreetLine1, row.StreetLine2, row.Locality, row.State, row.PostalCode,
		row.Country, row.GooglePlaceID, row.Latitude, row.Longitude, row.Timezone, row.ReceiveCalendarID,
	)
}

func mapGetAddressesByIDsRow(row sqlc.GetAddressesByIDsRow) *domain.Address {
	return mapAddressRow(
		row.ID, row.Name, row.Phone, row.Email, row.IsDropShip,
		row.CreatedAt, row.UpdatedAt,
		row.GeolocationID, row.StreetLine1, row.StreetLine2, row.Locality, row.State, row.PostalCode,
		row.Country, row.GooglePlaceID, row.Latitude, row.Longitude, row.Timezone, row.ReceiveCalendarID,
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

	dir := pagination.DirectionForward
	var cursor *pagination.StringCursor
	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursor, cursorDir, dir = &cur, &cur.Direction, cur.Direction
	}

	query, args := buildAddressListQuery(params, dir, cursor, params.Limit+1)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	addresses, err := scanAddressListRows(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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

// GetOrderAddress reads an address a sales order names, whichever account (if any) lists it.
func (r *addressRepoImpl) GetOrderAddress(ctx context.Context, addressID string) (*domain.Address, *apierror.APIError) {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.get_order_address")
	defer span.End()

	row, err := r.queries.GetOrderAddress(ctx, addressID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapGetAddressRow(sqlc.GetAddressRow(row)), nil
}

// CreateOrderAddress stores an address given inline on a sales order, unlinked from any account so no address book
// lists it.
func (r *addressRepoImpl) CreateOrderAddress(ctx context.Context, addressID, geolocationID string, params domain.CreateAddressParams) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.create_order_address")
	defer span.End()

	if err := r.queries.InsertGeolocation(ctx, sqlc.InsertGeolocationParams{
		ID:          geolocationID,
		StreetLine1: toNullString(params.StreetLine1),
		StreetLine2: toNullString(params.StreetLine2),
		Locality:    toNullString(params.Locality),
		State:       toNullString(params.State),
		PostalCode:  toNullString(params.PostalCode),
		Country:     params.Country,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.InsertAddress(ctx, sqlc.InsertAddressParams{
		ID:            addressID,
		Name:          params.Name,
		Phone:         toNullString(params.Phone),
		Email:         toNullString(params.Email),
		GeolocationID: geolocationID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *addressRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Address, *apierror.APIError) {

	ctx, span := addressRepoTracer.Start(ctx, "repository.address.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetAddressesByIDs(ctx, sqlc.GetAddressesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Address, len(rows))
	for i, row := range rows {
		out[i] = mapGetAddressesByIDsRow(row)
	}
	return out, nil
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
		ID:                addressID,
		Name:              params.Name,
		Phone:             toNullString(params.Phone),
		Email:             toNullString(params.Email),
		IsDropShip:        params.IsDropShip,
		ReceiveCalendarID: toNullString(params.ReceiveCalendarID),
		GeolocationID:     geolocationID,
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
		ID:                params.AddressID,
		Name:              toNullString(params.Name),
		Phone:             field.StringToNullString(params.Phone),
		Email:             field.StringToNullString(params.Email),
		IsDropShip:        toNullBool(params.IsDropShip),
		ReceiveCalendarID: field.StringToNullString(params.ReceiveCalendarID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Address not found."))
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
		StreetLine2: field.StringToNullString(params.StreetLine2),
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

func (r *addressRepoImpl) SwitchAccountDefaultAddressToRelation(ctx context.Context, addressID string) *apierror.APIError {
	ctx, span := addressRepoTracer.Start(ctx, "repository.address.switch_account_default_to_relation")
	defer span.End()

	err := r.queries.SwitchAccountDefaultAddressToRelation(ctx, sqlc.SwitchAccountDefaultAddressToRelationParams{
		AddressID: gosql.NullString{String: addressID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func toNullBool(b *bool) gosql.NullBool {
	if b == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: *b, Valid: true}
}
