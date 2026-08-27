package repository

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var portalDomainRepoTracer = tracing.GetTracer("core-service.portal_domain_repository")

type portalDomainRepoImpl struct {
	queries *sqlc.Queries
}

func NewPortalDomainRepo(queries *sqlc.Queries) domain.PortalDomainRepo {
	return &portalDomainRepoImpl{queries: queries}
}

func (r *portalDomainRepoImpl) Create(ctx context.Context, portalDomainID, accountID, domainName string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.create")
	defer span.End()

	err := r.queries.CreatePortalDomain(ctx, sqlc.CreatePortalDomainParams{
		ID:        portalDomainID,
		AccountID: accountID,
		Domain:    domainName,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.GetByID(ctx, accountID, portalDomainID)
}

func (r *portalDomainRepoImpl) GetByID(ctx context.Context, accountID, portalDomainID string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.get_by_id")
	defer span.End()

	row, err := r.queries.GetPortalDomainByID(ctx, sqlc.GetPortalDomainByIDParams{ID: portalDomainID, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return portalDomainFromRow(row), nil
}

func (r *portalDomainRepoImpl) GetByAccountID(ctx context.Context, accountID string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.get_by_account_id")
	defer span.End()

	// Second read of the unauthenticated portal branding lookup, so it carries the same connection-drop retry as the account read it follows.
	var row sqlc.PortalDomain
	err := db.WithConnRetry(ctx, nil, "portal_domain.get_by_account_id", func() error {
		var queryErr error
		row, queryErr = r.queries.GetPortalDomainByAccountID(ctx, accountID)
		return queryErr
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return portalDomainFromRow(row), nil
}

func (r *portalDomainRepoImpl) GetByDomain(ctx context.Context, domainName string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.get_by_domain")
	defer span.End()

	row, err := r.queries.GetPortalDomainByDomain(ctx, domainName)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return portalDomainFromRow(row), nil
}

func (r *portalDomainRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.list_by_account")
	defer span.End()

	rows, err := r.queries.ListPortalDomainsByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.PortalDomain, 0, len(rows))
	for _, row := range rows {
		out = append(out, portalDomainFromRow(row))
	}
	return out, nil
}

func (r *portalDomainRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.queries.BatchGetPortalDomainsByIDs(ctx, sqlc.BatchGetPortalDomainsByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.PortalDomain, 0, len(rows))
	for _, row := range rows {
		out = append(out, portalDomainFromRow(row))
	}
	return out, nil
}

func (r *portalDomainRepoImpl) UpdateProviderState(ctx context.Context, portalDomainID string, status constants.PortalDomainStatus, dnsRecords []domain.PortalDNSRecord) *apierror.APIError {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.update_provider_state")
	defer span.End()

	err := r.queries.UpdatePortalDomainProviderState(ctx, sqlc.UpdatePortalDomainProviderStateParams{
		Status:     string(status),
		DnsRecords: marshalDNSRecords(dnsRecords),
		ID:         portalDomainID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *portalDomainRepoImpl) MarkVerified(ctx context.Context, portalDomainID string) *apierror.APIError {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.mark_verified")
	defer span.End()

	err := r.queries.MarkPortalDomainVerified(ctx, portalDomainID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *portalDomainRepoImpl) Delete(ctx context.Context, accountID, portalDomainID string) (bool, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.delete")
	defer span.End()

	rows, err := r.queries.DeletePortalDomain(ctx, sqlc.DeletePortalDomainParams{ID: portalDomainID, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *portalDomainRepoImpl) ResolveVerifiedHost(ctx context.Context, domainName string) (*domain.PublicAccountBySlug, *apierror.APIError) {
	ctx, span := portalDomainRepoTracer.Start(ctx, "repository.portal_domain.resolve_verified_host")
	defer span.End()

	row, err := r.queries.ResolveVerifiedPortalHost(ctx, domainName)
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
		PortalDomain:            &row.Domain,
	}, nil
}

func portalDomainFromRow(row sqlc.PortalDomain) *domain.PortalDomain {
	return &domain.PortalDomain{
		ID:         row.ID,
		AccountID:  row.AccountID,
		Domain:     row.Domain,
		Status:     constants.PortalDomainStatus(row.Status),
		DNSRecords: unmarshalDNSRecords(row.DnsRecords),
		VerifiedAt: db.TimeFromNullTime(row.VerifiedAt),
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

// marshalDNSRecords encodes DNS records for the nullable JSON column; empty → NULL.
func marshalDNSRecords(records []domain.PortalDNSRecord) db.NullableRawMessage {
	if len(records) == 0 {
		return nil
	}
	b, err := json.Marshal(records)
	if err != nil {
		return nil
	}
	return db.NullableRawMessage(b)
}

// unmarshalDNSRecords decodes the nullable JSON column back to DNS records; NULL → nil.
func unmarshalDNSRecords(raw db.NullableRawMessage) []domain.PortalDNSRecord {
	if len(raw) == 0 {
		return nil
	}
	var out []domain.PortalDNSRecord
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
