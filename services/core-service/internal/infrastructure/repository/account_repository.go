package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var accountRepoTracer = tracing.GetTracer("core-service.account_repository")

type accountRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountRepo(queries *sqlc.Queries) domain.AccountRepo {
	return &accountRepoImpl{queries: queries}
}

func (r *accountRepoImpl) Create(ctx context.Context, id, name string, accountTypeCode domain.AccountType) *apierror.APIError {
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

	return &domain.AccountContext{
		AccountID:      row.ID,
		IsSandbox:      isSandbox,
		OwnerAccountID: db.StringFromNullString(row.OwnerAccountID),
		AccountMode:    accountMode,
	}, nil
}

func (r *accountRepoImpl) GetAccountNames(ctx context.Context, ids []string) (map[string]string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_account_names")
	defer span.End()

	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	rows, err := r.queries.GetAccountNames(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

func (r *accountRepoImpl) IsSelfManaged(ctx context.Context, accountID string) (bool, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.is_self_managed")
	defer span.End()

	selfManaged, err := r.queries.IsAccountSelfManaged(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return selfManaged, nil
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
			FaviconURL:      db.StringFromNullString(row.BrandingFaviconUrl),
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

func (r *accountRepoImpl) GetByIDs(ctx context.Context, ids []string) ([]*domain.Account, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetAccountsByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Account, len(rows))
	for i, row := range rows {
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
		out[i] = account
	}
	return out, nil
}

func (r *accountRepoImpl) GetBySlug(ctx context.Context, slug string) (*domain.PublicAccountBySlug, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_by_slug")
	defer span.End()

	// Unauthenticated portal branding/profile loads funnel through this read, and internet scanners probe the route constantly, so a dropped database connection (Vitess tablet failover, killed connection) must not surface as a 500. The read is idempotent, making a short retry safe.
	var row sqlc.GetPublicAccountBySlugRow
	err := db.WithConnRetry(ctx, nil, "account.get_by_slug", func() error {
		var queryErr error
		row, queryErr = r.queries.GetPublicAccountBySlug(ctx, slug)
		return queryErr
	})
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
		FaviconURL:              db.StringFromNullString(row.FaviconUrl),
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

func (r *accountRepoImpl) UpdateBranding(ctx context.Context, accountID string, branding domain.AccountBranding) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_branding")
	defer span.End()

	brandingID, genErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err := r.queries.UpsertAccountBranding(ctx, sqlc.UpsertAccountBrandingParams{
		ID:              brandingID,
		SupportEmail:    db.NullStringPtr(branding.SupportEmail),
		PhoneNumber:     db.NullStringPtr(branding.PhoneNumber),
		FacebookHandle:  db.NullStringPtr(branding.FacebookHandle),
		InstagramHandle: db.NullStringPtr(branding.InstagramHandle),
		LinkedinHandle:  db.NullStringPtr(branding.LinkedInHandle),
		TwitterHandle:   db.NullStringPtr(branding.TwitterHandle),
		WebsiteUrl:      db.NullStringPtr(branding.WebsiteURL),
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) SetDefaultAddresses(ctx context.Context, accountID string, billingAddressID, shippingAddressID *string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.set_default_addresses")
	defer span.End()

	err := r.queries.SetAccountDefaultAddresses(ctx, sqlc.SetAccountDefaultAddressesParams{
		BillingAddressID:  ptrToNullString(billingAddressID),
		ShippingAddressID: ptrToNullString(shippingAddressID),
		AccountID:         accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *accountRepoImpl) HasAddress(ctx context.Context, accountID, addressID string) (bool, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.has_address")
	defer span.End()

	linked, err := r.queries.AccountHasAddress(ctx, sqlc.AccountHasAddressParams{AccountID: accountID, AddressID: addressID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return linked, nil
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
	if rowsAffected > 0 {
		return nil
	}

	// No row changed: the portal already has this slug, or the account has no portal at all, as
	// accounts made before portals existed do. Their first slug creates it, as registration would.
	_, err = r.queries.GetAccountPortalSlugByAccountID(ctx, accountID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	portalID, genErr := id.GenID(id.AccountPortalIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}
	err = r.queries.CreateAccountPortal(ctx, sqlc.CreateAccountPortalParams{
		ID:             portalID,
		OwnerAccountID: accountID,
		Slug:           slug,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
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

	brandingID, genErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err := r.queries.UpsertAccountBrandingLogoURL(ctx, sqlc.UpsertAccountBrandingLogoURLParams{
		ID:        brandingID,
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

func (r *accountRepoImpl) UpdateBrandingFaviconURL(ctx context.Context, accountID, faviconURL string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_branding_favicon_url")
	defer span.End()

	brandingID, genErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, genErr)
	}

	err := r.queries.UpsertAccountBrandingFaviconURL(ctx, sqlc.UpsertAccountBrandingFaviconURLParams{
		ID:         brandingID,
		FaviconUrl: sql.NullString{String: faviconURL, Valid: true},
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetBrandingFaviconKey(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_branding_favicon_key")
	defer span.End()

	result, err := r.queries.GetAccountBrandingFaviconKey(ctx, accountID)
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
