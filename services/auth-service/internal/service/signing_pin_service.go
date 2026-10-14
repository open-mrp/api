package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/event"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/auth-service/internal/mediator"
	"github.com/open-mrp/api/services/auth-service/internal/signingpin"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/trace"
)

var signingPinSvcTracer = tracing.GetTracer("auth-service.signing_pin_service")

const (
	signingPinEventSet                = "set"
	signingPinEventTemporarySet       = "temporary_set"
	signingPinEventCleared            = "cleared"
	signingPinEventChanged            = "changed"
	signingPinEventVerificationFailed = "verification_failed"
	signingPinEventLocked             = "locked"
)

type signingPinSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	coreClient      domain.AuthCoreClient
	txManager       TransactionManager
}

type SigningPinSvcConfig struct {
	// Repos (required) is the repository factory for auth persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// CoreClient (required) resolves a signer's role and permissions, the same way an identity's are.
	CoreClient domain.AuthCoreClient

	// TxManager (optional; default: nil) wraps multi-step operations in database transactions. It is not validated at construction; transactional code paths panic at runtime if it is unset.
	TxManager TransactionManager
}

func (c *SigningPinSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("signing pin service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("signing pin service: mediator factory is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("signing pin service: core client is required")
	}
	return nil
}

func NewSigningPinSvc(config *SigningPinSvcConfig) domain.SigningPinSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &signingPinSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		coreClient:      config.CoreClient,
		txManager:       config.TxManager,
	}
}

func BuildSigningPinSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL, portalURL string, coreClient domain.AuthCoreClient) *SigningPinSvcConfig {
	repoFactory := repository.NewRepoFactory(queries)

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: event.NewOutboxNotificationPublisher(),
		FrontendURL:           frontendURL,
		PortalURL:             portalURL,
		CoreClient:            coreClient,
	})

	return &SigningPinSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		CoreClient:      coreClient,
	}
}

func (s *signingPinSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *signingPinSvcImpl) withTx(ctx context.Context, fn func(context.Context, *signingPinSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &signingPinSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			coreClient:      s.coreClient,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// SetSigningPin sets the caller's own signing PIN, replacing any PIN they had, including a temporary one.
//
// 1. Require a person signed in as themselves to an account: API keys, agents and scanning station logins are refused.
// 2. Check the PIN rules.
// 3. Upsert an idempotency key; return the cached response if already finished.
// 4. Store the PIN and publish an audit event in one transaction, caching the success with it.
func (s *signingPinSvcImpl) SetSigningPin(ctx context.Context, pin string) *apierror.APIError {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.set")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsUser(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if identity.IsScanner() {
		return tracing.Trace(span, apierror.NewAuthorizationError("A scanning station login can't set a signing PIN. Sign in as yourself to set yours."))
	}
	if apiErr := signingpin.Validate(pin, "pin"); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountUser, apiErr := s.repos.NewUserRepo().FindAccountUserByUserID(ctx, identity.Target.AccountID, identity.Actor.ID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return s.saveIdempotently(ctx, identity, accountUser, pin, false, signingPinEventSet)
}

// SetTemporarySigningPin sets a PIN for an account user in the caller's account that must be changed on first use.
//
// 1. Require team:update in the caller's own account.
// 2. Check the PIN rules and find the account user in the caller's account.
// 3. Upsert an idempotency key; return the cached response if already finished.
// 4. Store the PIN as temporary and publish an audit event in one transaction, caching the success with it.
func (s *signingPinSvcImpl) SetTemporarySigningPin(ctx context.Context, accountUserID, pin string) *apierror.APIError {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.set_temporary")
	defer span.End()

	identity, apiErr := managerIdentity(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := signingpin.Validate(pin, "pin"); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountUser, apiErr := s.repos.NewUserRepo().FindAccountUser(ctx, identity.Target.AccountID, accountUserID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return s.saveIdempotently(ctx, identity, accountUser, pin, true, signingPinEventTemporarySet)
}

// ClearSigningPin removes an account user's signing PIN, so they must set a new one before signing.
//
// 1. Require team:update in the caller's own account.
// 2. Find the account user in the caller's account.
// 3. Upsert an idempotency key; return the cached response if already finished.
// 4. Delete the PIN and, when there was one, publish an audit event in the same transaction. Clearing a person with no PIN succeeds.
func (s *signingPinSvcImpl) ClearSigningPin(ctx context.Context, accountUserID string) *apierror.APIError {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.clear")
	defer span.End()

	identity, apiErr := managerIdentity(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountUser, apiErr := s.repos.NewUserRepo().FindAccountUser(ctx, identity.Target.AccountID, accountUserID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, requestIdentity(identity))
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		return cachedEmptyResponse(ctx, span, idempotencyKey)

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *signingPinSvcImpl) *apierror.APIError {
			deleted, apiErr := txSvc.repos.NewSigningPinRepo().Delete(txCtx, accountUser.UserID)
			if apiErr != nil {
				return apiErr
			}
			if deleted {
				if apiErr := txSvc.publishAudit(txCtx, constants.AuditActionDelete, accountUser, signingPinEventCleared, nil); apiErr != nil {
					return apiErr
				}
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// ChangeSigningPin replaces an account user's PIN after checking their current one; it is how a temporary PIN gets replaced at a station.
//
// 1. Require an internal caller in the account; the person proves themselves with the current PIN, not the caller's login.
// 2. Check the new PIN's rules and find the active account user in the caller's account.
// 3. Upsert an idempotency key; return the cached response if already finished.
// 4. Check the current PIN; a wrong one counts toward the lockout and fails, and the failure is cached so a retry does not count twice.
// 5. Refuse a new PIN equal to the current one.
// 6. Store the new PIN as permanent and publish an audit event in one transaction, caching the success with it.
func (s *signingPinSvcImpl) ChangeSigningPin(ctx context.Context, accountUserID, currentPin, newPin string) *apierror.APIError {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.change")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := signingpin.Validate(newPin, "new_pin"); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountUser, apiErr := s.findActiveAccountUser(ctx, identity.Target.AccountID, accountUserID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, requestIdentity(identity))
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		return cachedEmptyResponse(ctx, span, idempotencyKey)

	case domain.RecoveryPointStarted:
		check, apiErr := meds.SigningPin.Check(ctx, accountUser.UserID, currentPin)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		if !check.Matched {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, s.recordFailure(ctx, accountUser, check.Pin))
		}
		if newPin == currentPin {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationErrorWithParam("The new signing PIN must be different from the current one.", "new_pin"))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *signingPinSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			if apiErr := txMeds.SigningPin.Save(txCtx, accountUser.UserID, newPin, false); apiErr != nil {
				return apiErr
			}
			if apiErr := txSvc.publishAudit(txCtx, constants.AuditActionUpdate, accountUser, signingPinEventChanged, map[string]any{"was_temporary": check.Pin.IsTemporary}); apiErr != nil {
				return apiErr
			}
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// VerifySigningPin checks an account user's signing PIN and returns who signed, with the role and permissions they hold now.
//
// It takes no idempotency key: it is a check whose only writes are the lockout count, and a cached success must never stand in for a PIN entered later.
//
// 1. Require an internal caller in the account, and find the active account user in it.
// 2. Fail with signing_pin_not_set or signing_pin_locked before comparing.
// 3. A wrong PIN counts toward the lockout and fails with signing_pin_invalid, or signing_pin_locked on the attempt that locks.
// 4. A correct PIN clears the count; a temporary one then fails with signing_pin_change_required.
// 5. Resolve the signer's role and permissions the way an identity's are resolved.
func (s *signingPinSvcImpl) VerifySigningPin(ctx context.Context, accountUserID, pin string) (*domain.VerifiedSigner, *apierror.APIError) {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.verify")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountUser, apiErr := s.findActiveAccountUser(ctx, identity.Target.AccountID, accountUserID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()
	check, apiErr := meds.SigningPin.Check(ctx, accountUser.UserID, pin)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !check.Matched {
		return nil, tracing.Trace(span, s.recordFailure(ctx, accountUser, check.Pin))
	}
	if apiErr := meds.SigningPin.ResetFailures(ctx, check.Pin); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if check.Pin.IsTemporary {
		return nil, tracing.Trace(span, apierror.NewSigningPINChangeRequiredError(mediator.ErrSigningPinChangeRequired))
	}

	access, found, apiErr := s.coreClient.GetUserAccountAccess(ctx, accountUser.UserID, accountUser.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !found || access == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
	}

	return &domain.VerifiedSigner{
		AccountUserID: accountUser.ID,
		UserID:        accountUser.UserID,
		Name:          accountUser.DisplayName(),
		RoleID:        access.RoleID,
		RoleName:      access.RoleName,
		RoleTypeCode:  access.RoleType,
		DepartmentID:  accountUser.DepartmentID,
		Permissions:   signerPermissions(access),
	}, nil
}

// saveIdempotently stores pin for the account user under the request's idempotency key and records event in the audit trail.
func (s *signingPinSvcImpl) saveIdempotently(ctx context.Context, identity *types.Identity, accountUser *domain.SigningAccountUser, pin string, isTemporary bool, event string) *apierror.APIError {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.save")
	defer span.End()

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, requestIdentity(identity))
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		return cachedEmptyResponse(ctx, span, idempotencyKey)

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *signingPinSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			if apiErr := txMeds.SigningPin.Save(txCtx, accountUser.UserID, pin, isTemporary); apiErr != nil {
				return apiErr
			}
			if apiErr := txSvc.publishAudit(txCtx, constants.AuditActionUpsert, accountUser, event, nil); apiErr != nil {
				return apiErr
			}
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// recordFailure commits the wrong attempt and its audit events even though the request fails, then returns the error to report.
func (s *signingPinSvcImpl) recordFailure(ctx context.Context, accountUser *domain.SigningAccountUser, pin *domain.SigningPin) *apierror.APIError {
	ctx, span := signingPinSvcTracer.Start(ctx, "service.signing_pin.record_failure")
	defer span.End()

	var failure *domain.SigningPinFailure
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *signingPinSvcImpl) *apierror.APIError {
		f, apiErr := txSvc.mediators().SigningPin.RecordFailure(txCtx, pin)
		if apiErr != nil {
			return apiErr
		}
		failure = f

		if f.LockedUntil == nil {
			return txSvc.publishAudit(txCtx, constants.AuditActionUpdate, accountUser, signingPinEventVerificationFailed, map[string]any{"failed_attempts": f.FailedAttempts})
		}
		if apiErr := txSvc.publishAudit(txCtx, constants.AuditActionUpdate, accountUser, signingPinEventVerificationFailed, nil); apiErr != nil {
			return apiErr
		}
		return txSvc.publishAudit(txCtx, constants.AuditActionUpdate, accountUser, signingPinEventLocked, map[string]any{"locked_until": f.LockedUntil.UTC()})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if failure.LockedUntil != nil {
		return apierror.NewSigningPINLockedError(mediator.ErrSigningPinLocked)
	}
	return apierror.NewSigningPINInvalidError(mediator.ErrSigningPinInvalid)
}

// publishAudit records a signing PIN event against the account user it belongs to; the PIN and its hash are never part of it.
func (s *signingPinSvcImpl) publishAudit(ctx context.Context, action constants.AuditAction, accountUser *domain.SigningAccountUser, event string, extra map[string]any) *apierror.APIError {
	metadata := map[string]any{"signing_pin_event": event}
	for k, v := range extra {
		metadata[k] = v
	}
	return audit.NewPublisher().Publish(ctx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       action,
		ResourceType: constants.ObjectTypeSigningPIN,
		ResourceID:   accountUser.ID,
		Metadata:     metadata,
	})
}

func (s *signingPinSvcImpl) findActiveAccountUser(ctx context.Context, accountID, accountUserID string) (*domain.SigningAccountUser, *apierror.APIError) {
	accountUser, apiErr := s.repos.NewUserRepo().FindAccountUser(ctx, accountID, accountUserID)
	if apiErr != nil {
		return nil, apiErr
	}
	if !accountUser.IsActive() {
		return nil, apierror.NewConflictErrorWithParam("This account user is not active, so they can't sign.", "account_user_id")
	}
	return accountUser, nil
}

// managerIdentity is the caller allowed to manage other people's signing PINs in its own account.
func managerIdentity(ctx context.Context) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	// A PIN stands for a person's signature, so an agent may not hand one out or take one away.
	if identity.Type == types.IdentityActorTypeAgent {
		return nil, apierror.NewAuthorizationError("Agents can't set or clear signing PINs.")
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionUpdate); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

func requestIdentity(identity *types.Identity) *domain.RequestIdentity {
	return &domain.RequestIdentity{
		ActorID:         identity.Actor.ID,
		IdentityType:    identity.Type,
		TargetAccountID: &identity.Target.AccountID,
	}
}

func cachedEmptyResponse(ctx context.Context, span trace.Span, key *domain.IdempotencyKey) *apierror.APIError {
	cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, key.ResponseCode, key.ResponseBody)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
	}
	return cached.Error
}

// signerPermissions lists the granted permissions as "<domain>:<action>"; an admin's list is empty because the role holds every permission.
func signerPermissions(access *domain.AccountUserAccess) []string {
	if access.RoleType != nil && *access.RoleType == string(constants.RoleTypeAdmin) {
		return []string{}
	}
	permissions := make([]string, 0, len(access.Permissions))
	for permission, granted := range access.Permissions {
		if granted {
			permissions = append(permissions, permission)
		}
	}
	sort.Strings(permissions)
	return permissions
}
