package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var registrationRepoTracer = tracing.GetTracer("core-service.registration_repository")

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

type registrationRepoImpl struct {
	queries *sqlc.Queries
}

func NewRegistrationRepo(queries *sqlc.Queries) domain.RegistrationRepo {
	return &registrationRepoImpl{queries: queries}
}

func (r *registrationRepoImpl) CreateAccountForRegistration(ctx context.Context, params domain.CreateAccountParams) *apierror.APIError {
	ctx, span := registrationRepoTracer.Start(ctx, "repository.registration.create_account")
	defer span.End()

	planTypeID, err := r.queries.GetAccountPlanTypeIDByCode(ctx, params.PlanCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	subscriptionStatus := constants.SubscriptionStatusTrialing
	if params.PlanCode == string(constants.PlanCodeFree) {
		subscriptionStatus = constants.SubscriptionStatusActive
	}

	err = r.queries.CreateAccountForRegistration(ctx, sqlc.CreateAccountForRegistrationParams{
		ID:                           params.ID,
		Name:                         params.Name,
		AccountTypeCode:              string(domain.AccountTypeCompany),
		OnboardingStatusCode:         "active",
		PlanCode:                     params.PlanCode,
		AccountPlanID:                sql.NullString{String: planTypeID, Valid: true},
		InternalStripeCustomerID:     nullStr(params.StripeCustomerID),
		InternalStripeSubscriptionID: nullStr(params.StripeSubscriptionID),
		SubscriptionStatus:           sql.NullString{String: subscriptionStatus.String(), Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationRepoImpl) CreateAccountUser(ctx context.Context, accountID, userID, roleID string) *apierror.APIError {
	ctx, span := registrationRepoTracer.Start(ctx, "repository.registration.create_account_user")
	defer span.End()

	auID, genErr := id.GenID(id.AccountUserIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err := r.queries.CreateAccountUserForRegistration(ctx, sqlc.CreateAccountUserForRegistrationParams{
		ID:        auID,
		AccountID: accountID,
		UserID:    userID,
		RoleID:    sql.NullString{String: roleID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationRepoImpl) CreateBusinessAddress(ctx context.Context, accountID, accountName string, address domain.RegistrationAddress) *apierror.APIError {
	ctx, span := registrationRepoTracer.Start(ctx, "repository.registration.create_business_address")
	defer span.End()

	geoID, genErr := id.GenID(id.GeolocationIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	country := address.Country
	if country == "" {
		country = "US"
	}

	err := r.queries.CreateGeolocation(ctx, sqlc.CreateGeolocationParams{
		ID:          geoID,
		StreetLine1: nullStr(address.Line1),
		StreetLine2: nullStr(address.Line2),
		Locality:    nullStr(address.City),
		State:       nullStr(address.State),
		PostalCode:  nullStr(address.PostalCode),
		Country:     country,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	addrID, genErr := id.GenID(id.AddressIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err = r.queries.CreateAddress(ctx, sqlc.CreateAddressParams{
		ID:            addrID,
		Name:          fmt.Sprintf("%s Business Address", accountName),
		GeolocationID: geoID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	aaID, genErr := id.GenID(id.AccountAddressIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err = r.queries.CreateAccountAddress(ctx, sqlc.CreateAccountAddressParams{
		ID:        aaID,
		AccountID: accountID,
		AddressID: addrID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.SetAccountDefaultBillingAddress(ctx, sqlc.SetAccountDefaultBillingAddressParams{
		DefaultBillingAddressID: sql.NullString{String: addrID, Valid: true},
		ID:                      accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.SetAccountDefaultShippingAddress(ctx, sqlc.SetAccountDefaultShippingAddressParams{
		DefaultShippingAddressID: sql.NullString{String: addrID, Valid: true},
		ID:                       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationRepoImpl) CreateAccountPortal(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := registrationRepoTracer.Start(ctx, "repository.registration.create_account_portal")
	defer span.End()

	portalID, genErr := id.GenID(id.AccountPortalIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	slug := strings.ReplaceAll(accountID, "_", "-")

	err := r.queries.CreateAccountPortal(ctx, sqlc.CreateAccountPortalParams{
		ID:             portalID,
		OwnerAccountID: accountID,
		Slug:           slug,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
