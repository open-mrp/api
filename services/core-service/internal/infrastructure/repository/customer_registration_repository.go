package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
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
		// New customer-portal users get the global Customer role so their portal
		// capabilities are permission-driven from the start.
		RoleID: gosql.NullString{String: constants.GlobalCustomerRoleID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRegistrationRepoImpl) AllocateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string) (int64, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.allocate_next_customer_number")
	defer span.End()

	number, apiErr := customerNumbers(r.queries, accountID).next(ctx, sysPropertyID)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return number, nil
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
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, customerNumberDuplicateKeyMapping); apiErr != nil {
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

func (r *customerRegistrationRepoImpl) FindCustomerAccountBySalesOrderNumber(ctx context.Context, ownerAccountID, number string) (string, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.find_by_sales_order_number")
	defer span.End()

	accountID, err := r.queries.FindCustomerAccountBySalesOrderNumber(ctx, sqlc.FindCustomerAccountBySalesOrderNumberParams{
		OwnerAccountID: ownerAccountID,
		Number:         number,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return accountID, nil
}

func (r *customerRegistrationRepoImpl) FindCustomerAccountByInvoiceNumber(ctx context.Context, ownerAccountID, number string) (string, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.find_by_invoice_number")
	defer span.End()

	accountID, err := r.queries.FindCustomerAccountByInvoiceNumber(ctx, sqlc.FindCustomerAccountByInvoiceNumberParams{
		OwnerAccountID: ownerAccountID,
		Number:         number,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return accountID, nil
}

func (r *customerRegistrationRepoImpl) GetCustomerDefaultPostalCodes(ctx context.Context, ownerAccountID, customerAccountID string) ([]string, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.get_default_postal_codes")
	defer span.End()

	row, err := r.queries.GetCustomerDefaultPostalCodes(ctx, sqlc.GetCustomerDefaultPostalCodesParams{
		OwnerAccountID:    ownerAccountID,
		CustomerAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var codes []string
	for _, code := range []gosql.NullString{row.BillingPostalCode, row.ShippingPostalCode} {
		if code.Valid && code.String != "" {
			codes = append(codes, code.String)
		}
	}
	return codes, nil
}

func (r *customerRegistrationRepoImpl) ListCustomerDefaultAddressesInPostalArea(ctx context.Context, ownerAccountID, postalPrefix string) ([]domain.CustomerDefaultAddresses, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.list_default_addresses_in_postal_area")
	defer span.End()

	rows, err := r.queries.ListCustomerDefaultAddressesInPostalArea(ctx, sqlc.ListCustomerDefaultAddressesInPostalAreaParams{
		OwnerAccountID: ownerAccountID,
		PostalPrefix:   gosql.NullString{String: postalPrefix, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.CustomerDefaultAddresses, len(rows))
	for i, row := range rows {
		out[i] = domain.CustomerDefaultAddresses{
			CustomerMatch:       domain.CustomerMatch{AccountID: row.CounterpartyAccountID, Name: row.Name, Number: row.ExternalNumber},
			BillingStreetLine1:  nullStringPtr(row.BillingStreetLine1),
			BillingPostalCode:   nullStringPtr(row.BillingPostalCode),
			ShippingStreetLine1: nullStringPtr(row.ShippingStreetLine1),
			ShippingPostalCode:  nullStringPtr(row.ShippingPostalCode),
		}
	}
	return out, nil
}

func (r *customerRegistrationRepoImpl) ListCustomersWithUserEmailDomain(ctx context.Context, ownerAccountID, emailDomain string) ([]domain.CustomerMatch, *apierror.APIError) {
	ctx, span := customerRegistrationRepoTracer.Start(ctx, "repository.customer_registration.list_with_user_email_domain")
	defer span.End()

	rows, err := r.queries.ListCustomersWithUserEmailDomain(ctx, sqlc.ListCustomersWithUserEmailDomainParams{
		OwnerAccountID: ownerAccountID,
		EmailDomain:    gosql.NullString{String: emailDomain, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.CustomerMatch, len(rows))
	for i, row := range rows {
		out[i] = domain.CustomerMatch{AccountID: row.CounterpartyAccountID, Name: row.Name, Number: row.ExternalNumber}
	}
	return out, nil
}
