package service

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

// portalRegistrationSessionTTL bounds how long an incomplete session can be resumed. Applied logically at read time (no cleanup job), mirroring the SaaS registration session.
const portalRegistrationSessionTTL = constants.PortalRegistrationSessionTTL

const (
	defaultPortalRegistrationSessionPageSize int32 = 20
	maxPortalRegistrationSessionPageSize     int32 = 100
)

var portalRegSessionSvcTracer = tracing.GetTracer("core-service.portal_registration_session_service")

type PortalRegistrationSessionSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
	// Registrar (required) registers the buyer as a customer on completion (the registration-flow service).
	Registrar domain.CustomerRegistrar
}

func (c *PortalRegistrationSessionSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("portal registration session service: repos is required")
	}
	if c.Registrar == nil {
		return fmt.Errorf("portal registration session service: registrar is required")
	}
	return nil
}

type portalRegistrationSessionSvcImpl struct {
	repos     domain.RepoFactory
	registrar domain.CustomerRegistrar
}

func NewPortalRegistrationSessionSvc(config *PortalRegistrationSessionSvcConfig) domain.PortalRegistrationSessionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &portalRegistrationSessionSvcImpl{repos: config.Repos, registrar: config.Registrar}
}

func (s *portalRegistrationSessionSvcImpl) authedUserID(ctx context.Context) (string, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return "", apiErr
	}
	if identity.Actor == nil {
		return "", apierror.NewAuthenticationError("Actor is required.")
	}
	return identity.Actor.ID, nil
}

// getOwned fetches a session and enforces that the caller owns it.
func (s *portalRegistrationSessionSvcImpl) getOwned(ctx context.Context, userID, typeID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	session, apiErr := s.repos.NewPortalRegistrationSessionRepo().GetByTypeID(ctx, typeID)
	if apiErr != nil {
		return nil, apiErr
	}
	if session == nil {
		return nil, apierror.NewResourceNotFoundError("Registration session not found.")
	}
	if session.UserID != userID {
		return nil, apierror.NewAuthorizationError("You are not authorized to access this registration session.")
	}
	return session, nil
}

func (s *portalRegistrationSessionSvcImpl) CreateOrResumeSession(ctx context.Context, sellerSlug string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionSvcTracer.Start(ctx, "service.portal_registration_session.create_or_resume")
	defer span.End()

	userID, apiErr := s.authedUserID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	seller, apiErr := s.repos.NewAccountRepo().GetBySlug(ctx, sellerSlug)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewPortalRegistrationSessionRepo()

	// Resume the newest non-expired incomplete session for this (buyer, seller) pair.
	existing, apiErr := repo.GetIncomplete(ctx, userID, seller.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if existing != nil && time.Since(existing.CreatedAt) < portalRegistrationSessionTTL {
		return existing, nil
	}

	sessionID, apiErr := id.GenID(id.PortalRegistrationSessionIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return repo.Create(ctx, sessionID, domain.CreatePortalRegistrationSessionParams{
		UserID:          userID,
		SellerAccountID: seller.ID,
		SellerSlug:      seller.Slug,
		Step:            constants.PortalRegistrationStepCustomerDetails,
	})
}

func (s *portalRegistrationSessionSvcImpl) GetSession(ctx context.Context, typeID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionSvcTracer.Start(ctx, "service.portal_registration_session.get")
	defer span.End()

	userID, apiErr := s.authedUserID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	session, apiErr := s.getOwned(ctx, userID, typeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// An expired, still-incomplete session reads as not found.
	if session.CompletedAt == nil && session.AbandonedAt == nil && time.Since(session.CreatedAt) >= portalRegistrationSessionTTL {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Registration session has expired."))
	}
	return session, nil
}

// ListSessions returns the seller account's buyer-registration sessions for the customer-service follow-up view. Seller-facing: requires an internal actor with account read permission and is scoped to the caller's account (a different authorization model from the buyer-owned single-session reads above).
func (s *portalRegistrationSessionSvcImpl) ListSessions(ctx context.Context, params domain.ListPortalRegistrationSessionsParams) (*domain.ListPortalRegistrationSessionsResult, *apierror.APIError) {
	ctx, span := portalRegSessionSvcTracer.Start(ctx, "service.portal_registration_session.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.StatusFilter != nil && !constants.PortalRegistrationStatus(*params.StatusFilter).IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid registration status filter."))
	}

	if params.Limit <= 0 {
		params.Limit = defaultPortalRegistrationSessionPageSize
	}
	if params.Limit > maxPortalRegistrationSessionPageSize {
		params.Limit = maxPortalRegistrationSessionPageSize
	}

	params.SellerAccountID = identity.Target.AccountID
	// Single-source the resume TTL: an incomplete session created before this instant reads as expired.
	params.ExpiryThreshold = time.Now().Add(-portalRegistrationSessionTTL)

	return s.repos.NewPortalRegistrationSessionRepo().ListSessions(ctx, params)
}

func (s *portalRegistrationSessionSvcImpl) UpdateSession(ctx context.Context, params domain.UpdatePortalRegistrationSessionParams) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionSvcTracer.Start(ctx, "service.portal_registration_session.update")
	defer span.End()

	userID, apiErr := s.authedUserID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	session, apiErr := s.getOwned(ctx, userID, params.TypeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if session.CompletedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Registration is already complete."))
	}
	if session.AbandonedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Registration session was abandoned."))
	}
	// Forward-only step transitions.
	if params.Step.IsValid() && session.Step.IsAfter(params.Step) {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot move to an earlier registration step."))
	}
	return s.repos.NewPortalRegistrationSessionRepo().Update(ctx, params)
}

func (s *portalRegistrationSessionSvcImpl) CompleteSession(ctx context.Context, typeID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionSvcTracer.Start(ctx, "service.portal_registration_session.complete")
	defer span.End()

	userID, apiErr := s.authedUserID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	session, apiErr := s.getOwned(ctx, userID, typeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if session.CompletedAt != nil {
		return session, nil // idempotent
	}
	if session.AbandonedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Registration session was abandoned."))
	}

	// Register the buyer as a customer from the accumulated session data (reuses one-shot logic).
	if apiErr := s.registrar.RegisterCustomer(ctx, registerCustomerParamsFromSession(session)); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPortalRegistrationSessionRepo().Complete(ctx, typeID, "")
}

func (s *portalRegistrationSessionSvcImpl) AbandonSession(ctx context.Context, typeID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionSvcTracer.Start(ctx, "service.portal_registration_session.abandon")
	defer span.End()

	userID, apiErr := s.authedUserID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	session, apiErr := s.getOwned(ctx, userID, typeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if session.CompletedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot abandon a completed registration."))
	}
	if apiErr := s.repos.NewPortalRegistrationSessionRepo().Abandon(ctx, typeID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewPortalRegistrationSessionRepo().GetByTypeID(ctx, typeID)
}

func registerCustomerParamsFromSession(session *domain.PortalRegistrationSession) domain.RegisterCustomerParams {
	d := session.SessionData
	data := domain.CustomerRegistrationData{
		Number:          portalRegStrPtr(d.CustomerNumber),
		Name:            portalRegStrPtr(d.CustomerName),
		CustomerGroupID: portalRegStrPtr(d.CustomerGroupID),
		Phone:           portalRegStrPtr(d.Phone),
		ShippingTermID:  portalRegStrPtr(d.ShippingTermID),
		PaymentTermID:   portalRegStrPtr(d.PaymentTermID),
	}
	if d.AddressStreet1 != "" || d.AddressName != "" {
		data.Address = &domain.CustomerRegistrationAddress{
			Name:        portalRegStrPtr(d.AddressName),
			StreetLine1: d.AddressStreet1,
			StreetLine2: portalRegStrPtr(d.AddressStreet2),
			Locality:    d.AddressLocality,
			State:       d.AddressState,
			PostalCode:  d.AddressPostalCode,
			Country:     d.AddressCountry,
		}
	}

	isExisting := false
	if session.IsExistingCustomer != nil {
		isExisting = *session.IsExistingCustomer
	}

	return domain.RegisterCustomerParams{
		AccountSlug:        session.SellerSlug,
		IsExistingCustomer: isExisting,
		CustomerData:       data,
	}
}

func portalRegStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
