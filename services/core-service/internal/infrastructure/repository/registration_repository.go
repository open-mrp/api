package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
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

	billingID, genErr := id.GenID(id.AccountBillingIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err = r.queries.CreateAccountBilling(ctx, sqlc.CreateAccountBillingParams{
		ID:                           billingID,
		AccountPlanID:                planTypeID,
		InternalStripeCustomerID:     nullStr(params.StripeCustomerID),
		InternalStripeSubscriptionID: nullStr(params.StripeSubscriptionID),
		SubscriptionStatus:           sql.NullString{String: subscriptionStatus.String(), Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.CreateAccountForRegistration(ctx, sqlc.CreateAccountForRegistrationParams{
		ID:                   params.ID,
		Name:                 params.Name,
		AccountTypeCode:      string(domain.AccountTypeCompany),
		OnboardingStatusCode: "active",
		AccountBillingID:     sql.NullString{String: billingID, Valid: true},
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

func (r *registrationRepoImpl) CreateAccountBranding(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := registrationRepoTracer.Start(ctx, "repository.registration.create_account_branding")
	defer span.End()

	brandingID, genErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err := r.queries.CreateAccountBranding(ctx, sqlc.CreateAccountBrandingParams{
		ID:             brandingID,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationRepoImpl) CreateSystemProducts(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := registrationRepoTracer.Start(ctx, "repository.registration.create_system_products")
	defer span.End()

	acctID := sql.NullString{String: accountID, Valid: true}

	// 1. Units: Each, Dollar, Day
	eachID, genErr := id.GenID(id.UnitIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err := r.queries.CreateUnit(ctx, sqlc.CreateUnitParams{
		ID: eachID, Name: "Each", Abbreviation: "ea", AccountID: acctID,
		UnitDimensionCode: string(constants.UnitTypeQuantity), RatioNumerator: "1", RatioDenominator: "1", IsBaseUnit: true,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	dollarID, genErr := id.GenID(id.UnitIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateUnit(ctx, sqlc.CreateUnitParams{
		ID: dollarID, Name: "Dollar", Abbreviation: "$", AccountID: acctID,
		UnitDimensionCode: string(constants.UnitTypeCurrency), RatioNumerator: "1", RatioDenominator: "1", IsBaseUnit: true,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	dayID, genErr := id.GenID(id.UnitIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateUnit(ctx, sqlc.CreateUnitParams{
		ID: dayID, Name: "Day", Abbreviation: "d", AccountID: acctID,
		UnitDimensionCode: string(constants.UnitTypeTime), RatioNumerator: "24", RatioDenominator: "1", IsBaseUnit: false,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 2. Unit group: Each Units
	eachGroupID, genErr := id.GenID(id.UnitGroupIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateUnitGroup(ctx, sqlc.CreateUnitGroupParams{
		ID: eachGroupID, Name: "Each Units", BaseUnitID: eachID, AccountID: acctID,
		UnitTypeCode: string(constants.UnitTypeQuantity),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 3. Unit group unit: Each -> Each Units
	eguID, genErr := id.GenID(id.UnitGroupsUnitsIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateUnitGroupUnit(ctx, sqlc.CreateUnitGroupUnitParams{
		ID: eguID, UnitGroupID: eachGroupID, UnitID: eachID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 4. Item categories: Shipping, Credit
	shipCatID, genErr := id.GenID(id.ItemCategoryIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateItemCategory(ctx, sqlc.CreateItemCategoryParams{
		ID: shipCatID, Name: "Shipping", AccountID: acctID,
		ItemCategoryTypeCode: "product_category", UnitGroupID: eachGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditCatID, genErr := id.GenID(id.ItemCategoryIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateItemCategory(ctx, sqlc.CreateItemCategoryParams{
		ID: creditCatID, Name: "Credit", AccountID: acctID,
		ItemCategoryTypeCode: "product_category", UnitGroupID: eachGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 5. Product lines: Shipping, Credit
	shipLineID, genErr := id.GenID(id.ProductLineIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateProductLine(ctx, sqlc.CreateProductLineParams{
		ID: shipLineID, Name: "Shipping",
		Description: sql.NullString{String: "Freight charges", Valid: true},
		AccountID:   acctID, UnitGroupID: eachGroupID,
		IsCommissionExempt: true, IsFreightExempt: true,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditLineID, genErr := id.GenID(id.ProductLineIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateProductLine(ctx, sqlc.CreateProductLineParams{
		ID: creditLineID, Name: "Credit",
		Description: sql.NullString{String: "Credit adjustments", Valid: true},
		AccountID:   acctID, UnitGroupID: eachGroupID,
		IsCommissionExempt: true, IsFreightExempt: true,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 6. Rates: 6 total (unit_value, unit_cost, burn_rate for each product)
	shipUnitValueID, genErr := id.GenID(id.RateIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateRate(ctx, sqlc.CreateRateParams{
		ID: shipUnitValueID, Value: "0", NumeratorUnitID: dollarID, DenominatorUnitID: eachID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	shipUnitCostID, genErr := id.GenID(id.RateIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateRate(ctx, sqlc.CreateRateParams{
		ID: shipUnitCostID, Value: "0", NumeratorUnitID: dollarID, DenominatorUnitID: eachID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	shipBurnRateID, genErr := id.GenID(id.RateIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateRate(ctx, sqlc.CreateRateParams{
		ID: shipBurnRateID, Value: "0", NumeratorUnitID: eachID, DenominatorUnitID: dayID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditUnitValueID, genErr := id.GenID(id.RateIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateRate(ctx, sqlc.CreateRateParams{
		ID: creditUnitValueID, Value: "0", NumeratorUnitID: dollarID, DenominatorUnitID: eachID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditUnitCostID, genErr := id.GenID(id.RateIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateRate(ctx, sqlc.CreateRateParams{
		ID: creditUnitCostID, Value: "0", NumeratorUnitID: dollarID, DenominatorUnitID: eachID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditBurnRateID, genErr := id.GenID(id.RateIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateRate(ctx, sqlc.CreateRateParams{
		ID: creditBurnRateID, Value: "0", NumeratorUnitID: eachID, DenominatorUnitID: dayID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 7. Items: Shipping, Credit
	shipItemID, genErr := id.GenID(id.ItemIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateItem(ctx, sqlc.CreateItemParams{
		ID: shipItemID, Sku: "Shipping",
		Description:    sql.NullString{String: "Freight charges", Valid: true},
		UnitValueID:    shipUnitValueID,
		BurnRateID:     shipBurnRateID,
		AccountID:      accountID,
		ItemTypeCode:   "product",
		UnitCostID:     shipUnitCostID,
		ItemCategoryID: shipCatID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditItemID, genErr := id.GenID(id.ItemIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateItem(ctx, sqlc.CreateItemParams{
		ID: creditItemID, Sku: "Credit",
		Description:    sql.NullString{String: "Credit adjustments", Valid: true},
		UnitValueID:    creditUnitValueID,
		BurnRateID:     creditBurnRateID,
		AccountID:      accountID,
		ItemTypeCode:   "product",
		UnitCostID:     creditUnitCostID,
		ItemCategoryID: creditCatID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// 8. Products: shipping, credit
	shipProductID, genErr := id.GenID(id.ProductIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateProduct(ctx, sqlc.CreateProductParams{
		ID: shipProductID, ItemID: shipItemID, ProductTypeCode: "shipping",
		ProductLineID: sql.NullString{String: shipLineID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	creditProductID, genErr := id.GenID(id.ProductIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateProduct(ctx, sqlc.CreateProductParams{
		ID: creditProductID, ItemID: creditItemID, ProductTypeCode: "credit",
		ProductLineID: sql.NullString{String: creditLineID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
