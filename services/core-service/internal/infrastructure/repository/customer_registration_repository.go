package repository

import (
	"context"
	gosql "database/sql"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

var customerRegistrationRepoTracer = tracing.GetTracer("core-service.customer_registration_repository")

type customerRegistrationRepoImpl struct {
	queries *sqlc.Queries
}

func NewCustomerRegistrationRepo(queries *sqlc.Queries) domain.CustomerRegistrationRepo {
	return &customerRegistrationRepoImpl{queries: queries}
}

func (r *customerRegistrationRepoImpl) FindCustomerAccountByExternalNumber(ctx context.Context, ownerAccountID, externalNumber string) (string, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.find_by_external_number")
	defer span.End()

	accountID, err := r.queries.FindCustomerAccountByExternalNumber(ctx, sqlc.FindCustomerAccountByExternalNumberParams{
		OwnerAccountID: ownerAccountID,
		ExternalNumber: externalNumber,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}

func (r *customerRegistrationRepoImpl) CreateAccountUserLink(ctx context.Context, linkID, accountID, userID string) *apierror.APIError {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.create_account_user_link")
	defer span.End()

	err := r.queries.InsertAccountUserForCustomerRegistration(ctx, sqlc.InsertAccountUserForCustomerRegistrationParams{
		ID:        linkID,
		AccountID: accountID,
		UserID:    userID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRegistrationRepoImpl) GetNextCustomerNumber(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.get_next_customer_number")
	defer span.End()

	raw, err := r.queries.GetNextCustomerNumber(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	// COALESCE returns interface{} — convert to int64
	switch v := raw.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case []uint8:
		var n int64
		for _, b := range v {
			n = n*10 + int64(b-'0')
		}
		return n, nil
	default:
		return 0, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("unexpected type %T for customer number", raw), "Failed to parse customer number."))
	}
}

func (r *customerRegistrationRepoImpl) UpdateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string, value int64) *apierror.APIError {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.update_next_customer_number")
	defer span.End()

	err := r.queries.UpdateNextCustomerNumber(ctx, sqlc.UpdateNextCustomerNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
		Value:     safeconv.Int64ToInt32(value),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRegistrationRepoImpl) CreateNewCustomerAccount(ctx context.Context, params domain.CreateNewCustomerAccountParams) (string, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.create_new_customer_account")
	defer span.End()

	// Generate IDs for all entities
	geoID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	addressID, apiErr := id.GenID(id.AddressIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	accountID, apiErr := id.GenID(id.AccountIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	brandingID, apiErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	relationID, apiErr := id.GenID(id.AccountRelationIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	accountAddressID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Create geolocation
	err := r.queries.InsertGeolocationForCustomer(ctx, sqlc.InsertGeolocationForCustomerParams{
		ID:          geoID,
		StreetLine1: gosql.NullString{String: params.Address.StreetLine1, Valid: true},
		StreetLine2: gosql.NullString{String: stringPtrVal(params.Address.StreetLine2), Valid: params.Address.StreetLine2 != nil},
		Locality:    gosql.NullString{String: params.Address.Locality, Valid: true},
		State:       gosql.NullString{String: params.Address.State, Valid: true},
		PostalCode:  gosql.NullString{String: params.Address.PostalCode, Valid: true},
		Country:     params.Address.Country,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Create address
	addressName := ""
	if params.Address.Name != nil {
		addressName = *params.Address.Name
	}
	err = r.queries.InsertAddressForCustomer(ctx, sqlc.InsertAddressForCustomerParams{
		ID:            addressID,
		Name:          addressName,
		GeolocationID: geoID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Create account (billing + shipping address default to the same)
	err = r.queries.InsertAccountForCustomer(ctx, sqlc.InsertAccountForCustomerParams{
		ID:                       accountID,
		Name:                     params.CustomerName,
		DefaultBillingAddressID:  gosql.NullString{String: addressID, Valid: true},
		DefaultShippingAddressID: gosql.NullString{String: addressID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Create account branding (support_email = user email)
	err = r.queries.InsertAccountBrandingForCustomer(ctx, sqlc.InsertAccountBrandingForCustomerParams{
		ID:             brandingID,
		OwnerAccountID: accountID,
		SupportEmail:   gosql.NullString{String: params.Email, Valid: true},
		PhoneNumber:    gosql.NullString{String: stringPtrVal(params.Phone), Valid: params.Phone != nil},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Create account relation
	err = r.queries.InsertAccountRelationForCustomer(ctx, sqlc.InsertAccountRelationForCustomerParams{
		ID:                       relationID,
		OwnerAccountID:           params.AccountID,
		CounterpartyAccountID:    accountID,
		Alias:                    gosql.NullString{String: params.CustomerName, Valid: true},
		ExternalNumber:           params.CustomerNumber,
		ShippingTermID:           gosql.NullString{String: params.ShippingTermID, Valid: true},
		PaymentTermID:            gosql.NullString{String: params.PaymentTermID, Valid: true},
		DefaultBillingAddressID:  gosql.NullString{String: addressID, Valid: true},
		DefaultShippingAddressID: gosql.NullString{String: addressID, Valid: true},
		AccountGroupID:           gosql.NullString{String: params.CustomerGroupID, Valid: true},
		StripeEmail:              gosql.NullString{String: params.Email, Valid: true},
		DefaultCarrierOptionID:   gosql.NullString{String: "ground", Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Create account address link
	err = r.queries.InsertAccountAddressForCustomer(ctx, sqlc.InsertAccountAddressForCustomerParams{
		ID:        accountAddressID,
		AccountID: accountID,
		AddressID: addressID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}

func (r *customerRegistrationRepoImpl) GetUserEmailByID(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.get_user_email")
	defer span.End()

	email, err := r.queries.GetUserEmailByID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return email.String, nil
}

func stringPtrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
