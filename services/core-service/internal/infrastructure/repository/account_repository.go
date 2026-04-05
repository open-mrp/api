package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountRepoTracer = tracing.GetTracer("core-service.account_repository")

type accountRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountRepo(queries *sqlc.Queries) domain.AccountRepo {
	return &accountRepoImpl{queries: queries}
}

func (r *accountRepoImpl) Create(ctx context.Context, id, name string, accountTypeCode domain.AccountType, planCode constants.PlanCode) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.create")
	defer span.End()

	err := r.queries.CreateAccount(ctx, sqlc.CreateAccountParams{
		ID:              id,
		Name:            name,
		AccountTypeCode: string(accountTypeCode),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetPlanCode(ctx context.Context, id string) (constants.PlanCode, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_plan_code")
	defer span.End()

	planCodeStr, err := r.queries.GetAccountPlanCode(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	planCode := constants.PlanCode(planCodeStr)
	if !planCode.IsValid() {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Invalid plan code: %s", planCodeStr)))
	}

	return planCode, nil
}

func (r *accountRepoImpl) Delete(ctx context.Context, id string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.delete")
	defer span.End()

	result, err := r.queries.DeleteAccountByIDIfSandbox(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected"))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Account is not a sandbox account or does not exist."))
	}

	return nil
}

func (r *accountRepoImpl) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_account_context")
	defer span.End()

	row, err := r.queries.GetAccountContext(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	isSandbox := row.AccountTypeCode == string(constants.AccountTypeCodeSandbox)
	accountMode := constants.AccountModeProduction
	if isSandbox {
		accountMode = constants.AccountModeSandbox
	}

	var subscriptionStatus *string
	if s := db.StringFromInterface(row.SubscriptionStatus); s != "" {
		subscriptionStatus = &s
	}

	planCode := db.StringFromInterface(row.PlanCode)

	var spendingCap *int64
	if v, ok := db.Int64FromInterface(row.AgentMonthlySpendingCapCents); ok {
		spendingCap = &v
	}

	return &domain.AccountContext{
		AccountID:                    row.ID,
		IsSandbox:                    isSandbox,
		OwnerAccountID:               db.StringFromNullString(row.OwnerAccountID),
		AccountMode:                  accountMode,
		SubscriptionStatus:           subscriptionStatus,
		PlanCode:                     planCode,
		AgentMonthlySpendingCapCents: spendingCap,
	}, nil
}

func (r *accountRepoImpl) GetPlanTypeIDByCode(ctx context.Context, planCode string) (string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_plan_type_id_by_code")
	defer span.End()

	typeID, err := r.queries.GetAccountPlanTypeIDByCode(ctx, planCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return typeID, nil
}

func (r *accountRepoImpl) UpdateSubscription(ctx context.Context, accountID string, status *string, planCode string, accountPlanID *string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_subscription")
	defer span.End()

	err := r.queries.UpdateAccountSubscription(ctx, sqlc.UpdateAccountSubscriptionParams{
		AccountID:                       accountID,
		SubscriptionStatus:              db.NullStringPtr(status),
		AccountPlanID:                   db.NullStringPtr(accountPlanID),
		InternalStripeSubscriptionID:    db.NullStringPtr(stripeSubID),
		SubscriptionCurrentPeriodEnd:    db.NullTimePtr(periodEnd),
		InternalStripeCustomerID:        db.NullStringPtr(stripeCustomerID),
		StripeBillingProfileID:          db.NullStringPtr(billingProfileID),
		StripeBillingCadenceID:          db.NullStringPtr(billingCadenceID),
		StripePricingPlanSubscriptionID: db.NullStringPtr(pricingPlanSubscriptionID),
		ServicingStatus:                 db.NullStringPtr(servicingStatus),
		CollectionStatus:                db.NullStringPtr(collectionStatus),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) ClearStripeCustomer(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.clear_stripe_customer")
	defer span.End()

	err := r.queries.ClearAccountStripeCustomer(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) ClearPricingPlanSubscription(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.clear_pricing_plan_subscription")
	defer span.End()

	err := r.queries.ClearAccountPricingPlanSubscription(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetSandboxLimit(ctx context.Context, accountID string) (*int32, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_sandbox_limit")
	defer span.End()

	result, err := r.queries.GetSandboxLimitByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	val := result.Int32
	return &val, nil
}

func (r *accountRepoImpl) GetSeatLimitByPlanCode(ctx context.Context, planCode string) (*int32, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_seat_limit_by_plan_code")
	defer span.End()

	result, err := r.queries.GetSeatLimitByPlanCode(ctx, planCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	val := result.Int32
	return &val, nil
}

func (r *accountRepoImpl) CountNonSandboxByPlanCode(ctx context.Context, planCode string) (int64, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.count_non_sandbox_by_plan_code")
	defer span.End()

	count, err := r.queries.CountNonSandboxAccountsByPlanCode(ctx, planCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *accountRepoImpl) UpdateAgentSpendingCap(ctx context.Context, accountID string, capCents *int64) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_agent_spending_cap")
	defer span.End()

	err := r.queries.UpdateAgentSpendingCap(ctx, sqlc.UpdateAgentSpendingCapParams{
		AgentMonthlySpendingCapCents: db.NullInt64Ptr(capCents),
		AccountID:                    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetAgentSpendingCap(ctx context.Context, accountID string) (*int64, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_agent_spending_cap")
	defer span.End()

	result, err := r.queries.GetAgentSpendingCap(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	return &result.Int64, nil
}

func (r *accountRepoImpl) HasActiveBillingPlan(ctx context.Context, accountID string) (bool, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.has_active_billing_plan")
	defer span.End()

	hasPlan, err := r.queries.HasActiveBillingPlan(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return hasPlan, nil
}

func (r *accountRepoImpl) GetName(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_name")
	defer span.End()

	name, err := r.queries.GetAccountNameByID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return name, nil
}

func (r *accountRepoImpl) GetBrandingLogoURL(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_branding_logo_url")
	defer span.End()

	result, err := r.queries.GetAccountBrandingByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	return &result.String, nil
}

func (r *accountRepoImpl) GetPortalSlug(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_portal_slug")
	defer span.End()

	slug, err := r.queries.GetAccountPortalSlugByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &slug, nil
}

func (r *accountRepoImpl) GetByID(ctx context.Context, accountID string) (*domain.Account, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_by_id")
	defer span.End()

	row, err := r.queries.GetAccountByID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	account := &domain.Account{
		ID:                       row.ID,
		Name:                     row.Name,
		DefaultBillingAddressID:  db.StringFromNullString(row.DefaultBillingAddressID),
		DefaultShippingAddressID: db.StringFromNullString(row.DefaultShippingAddressID),
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}

	if row.BrandingID.Valid {
		account.Branding = &domain.AccountBranding{
			ID:              row.BrandingID.String,
			SupportEmail:    db.StringFromNullString(row.BrandingSupportEmail),
			PhoneNumber:     db.StringFromNullString(row.BrandingPhoneNumber),
			LogoURL:         db.StringFromNullString(row.BrandingLogoUrl),
			FacebookHandle:  db.StringFromNullString(row.BrandingFacebookHandle),
			InstagramHandle: db.StringFromNullString(row.BrandingInstagramHandle),
			LinkedInHandle:  db.StringFromNullString(row.BrandingLinkedinHandle),
			TwitterHandle:   db.StringFromNullString(row.BrandingTwitterHandle),
			WebsiteURL:      db.StringFromNullString(row.BrandingWebsiteUrl),
			CreatedAt:       row.BrandingCreatedAt.Time,
			UpdatedAt:       row.BrandingUpdatedAt.Time,
		}
	}

	if row.PortalID.Valid {
		account.Portal = &domain.AccountPortal{
			ID:        row.PortalID.String,
			Slug:      row.PortalSlug.String,
			CreatedAt: row.PortalCreatedAt.Time,
			UpdatedAt: row.PortalUpdatedAt.Time,
		}
	}

	return account, nil
}

func (r *accountRepoImpl) GetBySlug(ctx context.Context, slug string) (*domain.PublicAccountBySlug, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_by_slug")
	defer span.End()

	row, err := r.queries.GetPublicAccountBySlug(ctx, slug)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.PublicAccountBySlug{
		ID:                      row.ID,
		Name:                    row.Name,
		Slug:                    row.Slug,
		DefaultBillingAddressID: db.StringFromNullString(row.DefaultBillingAddressID),
		SupportEmail:            db.StringFromNullString(row.SupportEmail),
		LogoURL:                 db.StringFromNullString(row.LogoUrl),
	}, nil
}

func (r *accountRepoImpl) UpdateName(ctx context.Context, accountID, name string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_name")
	defer span.End()

	result, err := r.queries.UpdateAccountName(ctx, sqlc.UpdateAccountNameParams{
		Name:      name,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected"))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Account not found."))
	}

	return nil
}

func (r *accountRepoImpl) UpdateBranding(ctx context.Context, accountID string, params domain.UpdateAccountParams) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_branding")
	defer span.End()

	_, err := r.queries.UpdateAccountBranding(ctx, sqlc.UpdateAccountBrandingParams{
		SupportEmail:    db.NullStringPtr(params.SupportEmail),
		PhoneNumber:     db.NullStringPtr(params.PhoneNumber),
		FacebookHandle:  db.NullStringPtr(params.FacebookHandle),
		InstagramHandle: db.NullStringPtr(params.InstagramHandle),
		LinkedinHandle:  db.NullStringPtr(params.LinkedInHandle),
		TwitterHandle:   db.NullStringPtr(params.TwitterHandle),
		WebsiteUrl:      db.NullStringPtr(params.WebsiteURL),
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) UpdatePortalSlug(ctx context.Context, accountID, slug string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_portal_slug")
	defer span.End()

	result, err := r.queries.UpdateAccountPortalSlug(ctx, sqlc.UpdateAccountPortalSlugParams{
		Slug:      slug,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected"))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Account portal not found."))
	}

	return nil
}

func (r *accountRepoImpl) ExistsPortalSlug(ctx context.Context, slug, excludeAccountID string) (bool, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.exists_portal_slug")
	defer span.End()

	exists, err := r.queries.ExistsPortalSlug(ctx, sqlc.ExistsPortalSlugParams{
		Slug:             slug,
		ExcludeAccountID: excludeAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *accountRepoImpl) UpdateBrandingLogoURL(ctx context.Context, accountID, logoURL string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_branding_logo_url")
	defer span.End()

	err := r.queries.UpdateAccountBrandingLogoURL(ctx, sqlc.UpdateAccountBrandingLogoURLParams{
		LogoUrl:   sql.NullString{String: logoURL, Valid: true},
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetBrandingLogoKey(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_branding_logo_key")
	defer span.End()

	result, err := r.queries.GetAccountBrandingLogoKey(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	return &result.String, nil
}

func (r *accountRepoImpl) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_by_stripe_customer_id")
	defer span.End()

	row, err := r.queries.GetAccountByStripeCustomerID(ctx, sql.NullString{String: stripeCustomerID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", "", tracing.Trace(span, apiErr)
	}

	return row.ID, row.PlanCode, nil
}
