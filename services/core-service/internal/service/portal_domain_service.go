package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var portalDomainSvcTracer = tracing.GetTracer("core-service.portal_domain_service")

type portalDomainSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	provider        domain.PortalDomainProvider
}

type PortalDomainSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// Provider (required) is the serving/TLS provider for portal custom domains.
	Provider domain.PortalDomainProvider
}

func (c *PortalDomainSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("portal domain service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("portal domain service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("portal domain service: tx manager is required")
	}
	if c.Provider == nil {
		return fmt.Errorf("portal domain service: provider is required")
	}
	return nil
}

func NewPortalDomainSvc(config *PortalDomainSvcConfig) domain.PortalDomainSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &portalDomainSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		provider:        config.Provider,
	}
}

func (s *portalDomainSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *portalDomainSvcImpl) withTx(ctx context.Context, fn func(context.Context, *portalDomainSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &portalDomainSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			provider:        s.provider,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *portalDomainSvcImpl) CreatePortalDomain(ctx context.Context, domainName string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.create")
	defer span.End()

	identity, accountID, apiErr := portalDomainWriteScope(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	normalized, apiErr := normalizePortalDomain(domainName)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	portalDomainID, apiErr := id.GenID(id.PortalDomainIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PortalDomain](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Phase 1 (atomic): uniqueness checks + pending row + recovery point advance. The provider registration that follows is a foreign mutation, so it must not share this transaction.
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *portalDomainSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPortalDomainRepo()

			existing, apiErr := txRepo.GetByAccountID(txCtx, accountID)
			if apiErr != nil {
				return apiErr
			}
			if existing != nil {
				return apierror.NewConflictErrorWithParam("This account already has a custom portal domain. Remove it before adding another.", "domain")
			}

			taken, apiErr := txRepo.GetByDomain(txCtx, normalized)
			if apiErr != nil {
				return apiErr
			}
			if taken != nil {
				return apierror.NewConflictErrorWithParam("This domain is already in use.", "domain")
			}

			created, apiErr := txRepo.Create(txCtx, portalDomainID, accountID, normalized)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(nil, created)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypePortalDomain,
				ResourceID:   created.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.repos.NewIdempotencyKeyRepo().AdvanceRecoveryPoint(txCtx, idempotencyKey.TypeID, domain.PortalDomainRecoveryPointProviderRegistered)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		result, apiErr := s.completeCreateProviderPhase(ctx, accountID, normalized, idempotencyKey.TypeID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return result, nil

	case domain.PortalDomainRecoveryPointProviderRegistered:
		// A prior attempt persisted the row but crashed before (or during) provider registration. The provider calls are idempotent, so repeat them.
		result, apiErr := s.completeCreateProviderPhase(ctx, accountID, normalized, idempotencyKey.TypeID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// completeCreateProviderPhase registers the domain with the serving provider (foreign mutation, its own atomic phase), then atomically persists the required DNS records and caches the response.
func (s *portalDomainSvcImpl) completeCreateProviderPhase(ctx context.Context, accountID, domainName, idempotencyTypeID string) (*domain.PortalDomain, *apierror.APIError) {
	meds := s.mediators()

	row, apiErr := s.repos.NewPortalDomainRepo().GetByDomain(ctx, domainName)
	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyTypeID, apiErr)
	}
	if row == nil || row.AccountID != accountID {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyTypeID, apierror.NewInvariantViolationError("Portal domain row missing during provider registration phase."))
	}

	state, apiErr := s.provider.AddDomain(ctx, domainName)
	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyTypeID, apiErr)
	}

	var result *domain.PortalDomain
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *portalDomainSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPortalDomainRepo()

		if apiErr := txRepo.UpdateProviderState(txCtx, row.ID, constants.PortalDomainStatusPending, state.DNSRecords); apiErr != nil {
			return apiErr
		}

		// A domain that was previously verified on the provider (e.g. removed and re-added) can come back immediately verified.
		if state.Verified && !state.Misconfigured {
			if apiErr := txRepo.MarkVerified(txCtx, row.ID); apiErr != nil {
				return apiErr
			}
		}

		updated, apiErr := txRepo.GetByID(txCtx, accountID, row.ID)
		if apiErr != nil {
			return apiErr
		}
		if updated == nil {
			return apierror.NewInvariantViolationError("Portal domain row missing after provider registration.")
		}
		result = updated

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyTypeID, result)
	})
	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyTypeID, apiErr)
	}

	return result, nil
}

func (s *portalDomainSvcImpl) GetPortalDomain(ctx context.Context, portalDomainID string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.get")
	defer span.End()

	_, accountID, apiErr := portalDomainReadScope(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	row, apiErr := s.repos.NewPortalDomainRepo().GetByID(ctx, accountID, portalDomainID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if row == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Portal domain not found."))
	}
	return row, nil
}

func (s *portalDomainSvcImpl) ListPortalDomains(ctx context.Context) ([]*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.list")
	defer span.End()

	_, accountID, apiErr := portalDomainReadScope(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPortalDomainRepo().ListByAccount(ctx, accountID)
}

func (s *portalDomainSvcImpl) VerifyPortalDomain(ctx context.Context, portalDomainID string) (*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.verify")
	defer span.End()

	identity, accountID, apiErr := portalDomainWriteScope(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PortalDomain](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		row, apiErr := s.repos.NewPortalDomainRepo().GetByID(ctx, accountID, portalDomainID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		if row == nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apierror.NewResourceNotFoundError("Portal domain not found."))
		}

		if row.Status == constants.PortalDomainStatusVerified {
			var result *domain.PortalDomain
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *portalDomainSvcImpl) *apierror.APIError {
				result = row
				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}
			return result, nil
		}

		// Foreign read: poll the provider for the domain's current verification state.
		state, apiErr := s.provider.GetDomainState(ctx, row.Domain)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var result *domain.PortalDomain
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *portalDomainSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPortalDomainRepo()

			if apiErr := txRepo.UpdateProviderState(txCtx, row.ID, row.Status, state.DNSRecords); apiErr != nil {
				return apiErr
			}

			if state.Verified && !state.Misconfigured {
				if apiErr := txRepo.MarkVerified(txCtx, row.ID); apiErr != nil {
					return apiErr
				}

				updated, apiErr := txRepo.GetByID(txCtx, accountID, row.ID)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypePortalDomain,
					ResourceID:   row.ID,
					Changes:      audit.ComputeChanges(row, updated),
				}); apiErr != nil {
					return apiErr
				}
				result = updated
			} else {
				updated, apiErr := txRepo.GetByID(txCtx, accountID, row.ID)
				if apiErr != nil {
					return apiErr
				}
				result = updated
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *portalDomainSvcImpl) DeletePortalDomain(ctx context.Context, portalDomainID string) *apierror.APIError {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.delete")
	defer span.End()

	_, accountID, apiErr := portalDomainWriteScope(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	row, apiErr := s.repos.NewPortalDomainRepo().GetByID(ctx, accountID, portalDomainID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if row == nil {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Portal domain not found."))
	}

	// Foreign mutation first: detaching from the provider is idempotent, so a crash between this call and the row delete is safely retried end-to-end.
	if apiErr := s.provider.RemoveDomain(ctx, row.Domain); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *portalDomainSvcImpl) *apierror.APIError {
		deleted, apiErr := txSvc.repos.NewPortalDomainRepo().Delete(txCtx, accountID, portalDomainID)
		if apiErr != nil {
			return apiErr
		}
		if !deleted {
			return apierror.NewResourceNotFoundError("Portal domain not found.")
		}

		changes := audit.ComputeChanges(row, (*domain.PortalDomain)(nil))
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypePortalDomain,
			ResourceID:   row.ID,
			Changes:      changes,
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// ResolvePortalHost is an unauthenticated lookup used by the frontend middleware to map a request host to its portal account. Only verified domains resolve.
func (s *portalDomainSvcImpl) ResolvePortalHost(ctx context.Context, domainName string) (*domain.PublicAccountBySlug, *apierror.APIError) {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.resolve_host")
	defer span.End()

	normalized, apiErr := normalizePortalDomain(domainName)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPortalDomainRepo().ResolveVerifiedHost(ctx, normalized)
}

func (s *portalDomainSvcImpl) BatchGetPortalDomainsByIDs(ctx context.Context, ids []string) ([]*domain.PortalDomain, *apierror.APIError) {
	ctx, span := portalDomainSvcTracer.Start(ctx, "service.portal_domain.batch_get_by_ids")
	defer span.End()

	_, accountID, apiErr := portalDomainReadScope(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewPortalDomainRepo().GetByIDs(ctx, accountID, ids)
}

// portalDomainWriteScope authorizes portal domain mutations: internal actors with account-settings update permission, scoped to their target account.
func portalDomainWriteScope(ctx context.Context) (*types.Identity, string, *apierror.APIError) {
	return portalDomainScope(ctx, types.ActionUpdate)
}

// portalDomainReadScope authorizes portal domain reads with the account-settings read permission.
func portalDomainReadScope(ctx context.Context) (*types.Identity, string, *apierror.APIError) {
	return portalDomainScope(ctx, types.ActionRead)
}

func portalDomainScope(ctx context.Context, action types.Action) (*types.Identity, string, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, "", apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, "", apiErr
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, action); apiErr != nil {
		return nil, "", apiErr
	}
	if !identity.IsTargetAccountSet() {
		return nil, "", apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}

	return identity, identity.Target.AccountID, nil
}

// normalizePortalDomain lowercases and validates a customer-supplied portal domain. Augno-owned hosts are rejected so a customer cannot shadow first-party subdomains.
func normalizePortalDomain(domainName string) (string, *apierror.APIError) {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(domainName, ".")))

	if normalized == "" {
		return "", apierror.NewValidationErrorWithParam("Domain is required.", "domain")
	}
	if len(normalized) > 253 {
		return "", apierror.NewValidationErrorWithParam("Domain is too long.", "domain")
	}
	if !strings.Contains(normalized, ".") || strings.ContainsAny(normalized, "@/\\:?#& ") || strings.Contains(normalized, "..") {
		return "", apierror.NewValidationErrorWithParam("Domain must be a valid hostname like shop.example.com.", "domain")
	}
	if normalized == "augno.com" || strings.HasSuffix(normalized, ".augno.com") {
		return "", apierror.NewValidationErrorWithParam("Augno domains cannot be used as a custom portal domain.", "domain")
	}

	return normalized, nil
}
