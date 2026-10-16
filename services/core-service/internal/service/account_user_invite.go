package service

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/idemresponse"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/ptrutil"
	"github.com/open-mrp/apikit/tracing"
)

// signInLinkHours is how long the emailed sign-in link works; it matches auth-service's SignInLinkTTL.
const signInLinkHours = 72

// InviteAccountUser emails a member a sign-in link and records that they were invited. Inviting again sends a new link.
//
// Only active members with an email can be invited. A supplier's contacts can't be: suppliers have no portal yet.
func (s *accountUserSvcImpl) InviteAccountUser(ctx context.Context, accountUserID string) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.invite")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountUserUpdatePermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	accountID := identity.Target.AccountID
	if identity.IsExternalTarget() {
		if apiErr := s.mediators().EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), accountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.AccountUserDetail](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Data != nil {
			cached.Data.ImageURL = s.resolveImageURL(ctx, accountID, cached.Data.UserID, cached.Data.ImageURL)
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := s.invite(ctx, identity, accountID, accountUserID)
		if apiErr != nil {
			return nil, tracing.Trace(span, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr))
		}
		if apiErr := meds.Idempotency.CacheSuccessResponse(ctx, idempotencyKey.TypeID, result); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result.ImageURL = s.resolveImageURL(ctx, accountID, result.UserID, result.ImageURL)
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *accountUserSvcImpl) invite(ctx context.Context, identity *types.Identity, accountID, accountUserID string) (*domain.AccountUserDetail, *apierror.APIError) {
	current, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, accountID, accountUserID, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	if current.StatusCode != constants.AccountUserStatusActive {
		return nil, apperr.NewInvalidStateTransitionError("Only an active account user can be invited. Activate them first.")
	}
	if current.Email == nil || strings.TrimSpace(*current.Email) == "" {
		return nil, apierror.NewValidationError("This account user has no email address to send a sign-in link to.")
	}
	if identity.IsTargetSupplierAccount() {
		return nil, apierror.NewValidationError("A supplier's contacts can't be invited: suppliers have no portal to sign in to.")
	}

	token, apiErr := s.authClient.IssueSignInToken(ctx, current.UserID)
	if apiErr != nil {
		return nil, apiErr
	}

	var result *domain.AccountUserDetail
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountUserSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		repo := txSvc.repos.NewAccountUserRepo()
		if apiErr := repo.SetInvitedAt(txCtx, current.ID); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.sendSignInLink(txCtx, identity, current.Name, *current.Email, token); apiErr != nil {
			return apiErr
		}
		next, apiErr := repo.GetDetailByAccountAndID(txCtx, accountID, current.ID, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.attachNotificationTypes(txCtx, identity, next); apiErr != nil {
			return apiErr
		}
		result = next
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeAccountUser,
			ResourceID:   current.ID,
			Changes:      audit.ComputeChanges(current, next),
		})
	})
	return result, apiErr
}

// sendSignInLink emails a member their sign-in link. Your own team signs in to the dashboard; a customer's users sign
// in to your portal under its slug on the shared portal host, so the token never travels to a custom domain.
func (s *accountUserSvcImpl) sendSignInLink(ctx context.Context, identity *types.Identity, name *string, email, token string) *apierror.APIError {
	emailParams := map[string]any{
		"Name":               ptrutil.ValOrDefault(name, "there"),
		"LinkExpiresInHours": signInLinkHours,
	}
	subject := "Welcome to OpenMRP"
	base := strings.TrimRight(s.frontendURL, "/")

	if identity.IsExternalTarget() {
		actorAccountID := *identity.ActorAccountID()
		accountRepo := s.repos.NewAccountRepo()
		accountName, _ := accountRepo.GetName(ctx, actorAccountID)
		logoRef, _ := accountRepo.GetBrandingLogoKey(ctx, actorAccountID)
		slug, _ := accountRepo.GetPortalSlug(ctx, actorAccountID)

		emailParams["AccountName"] = accountName
		emailParams["IsBranded"] = true
		// The stored value is an object key, so it has to be signed before it can be an <img src>.
		if logoURL := s.branding.LogoURL(ctx, ptrutil.Deref(logoRef)); logoURL != "" {
			emailParams["LogoURL"] = logoURL
		}
		base = slugPortalBase(s.portalURL, slug)
		subject = "Welcome to the " + accountName + " platform"
	}
	if base != "" {
		emailParams["LoginLink"] = base + string(constants.DashboardPathResetPassword) + "?t=" + token
	}

	return s.notificationPublisher.PublishSendEmail(ctx, messaging.EmailSendData{
		To:         []string{email},
		Subject:    subject,
		TemplateID: constants.EmailTemplateNewUserWelcome,
		Params:     emailParams,
		// Scope the audit log to the originating (actor) account so the sender can confirm the email went out, even when inviting a user of another account.
		AccountID: identity.ActorAccountID(),
		SentByID:  &identity.Actor.ID,
	})
}
