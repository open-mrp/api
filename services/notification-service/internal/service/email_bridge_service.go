package service

import (
	"context"
	"strings"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var emailBridgeSvcTracer = tracing.GetTracer("notification-service.email_bridge_service")

type emailBridgeSvcImpl struct {
	repoFactory      domain.RepoFactory
	identityProvider domain.EmailIdentityProvider
}

// NewEmailBridgeSvc constructs the email-bridge service from a repo factory and the SES identity provider.
func NewEmailBridgeSvc(repoFactory domain.RepoFactory, identityProvider domain.EmailIdentityProvider) domain.EmailBridgeSvc {
	return &emailBridgeSvcImpl{repoFactory: repoFactory, identityProvider: identityProvider}
}

// accountID resolves the caller's acting account from the request identity.
func (s *emailBridgeSvcImpl) accountID(ctx context.Context) (string, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return "", apierror.NewAuthenticationError("Authentication is required.")
	}
	if !identity.IsTargetAccountSet() {
		return "", apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}
	return identity.Target.AccountID, nil
}

func (s *emailBridgeSvcImpl) CreateDomain(ctx context.Context, domainName string) (*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.create_domain")
	defer span.End()

	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	domainName = strings.ToLower(strings.TrimSpace(domainName))
	if domainName == "" || !strings.Contains(domainName, ".") || strings.ContainsAny(domainName, "@ ") {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("A valid domain is required.", "domain"))
	}

	// Register the identity with SES first so a persistence failure doesn't leave a domain row with no
	// DKIM tokens; SES registration is idempotent and safe to retry.
	dkimTokens, apiErr := s.identityProvider.RegisterDomain(ctx, domainName)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	domainID, apiErr := id.GenID(id.EmailDomainIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := s.repoFactory.NewEmailDomainRepo().Create(ctx, domainID, accountID, &domain.CreateEmailDomainInput{
		Domain:     domainName,
		DkimTokens: dkimTokens,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
}

func (s *emailBridgeSvcImpl) ListDomains(ctx context.Context) ([]*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.list_domains")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailDomainRepo().ListByAccount(ctx, accountID)
}

func (s *emailBridgeSvcImpl) GetDomain(ctx context.Context, domainID string) (*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.get_domain")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
}

func (s *emailBridgeSvcImpl) VerifyDomain(ctx context.Context, domainID string) (*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.verify_domain")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if dom.Status == domain.EmailDomainStatusVerified {
		return dom, nil
	}
	verified, apiErr := s.identityProvider.DomainVerified(ctx, dom.Domain)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !verified {
		return dom, nil
	}
	if apiErr := s.repoFactory.NewEmailDomainRepo().MarkVerified(ctx, domainID, accountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
}

func (s *emailBridgeSvcImpl) CreateInbox(ctx context.Context, input domain.CreateEmailInboxInput) (*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.create_inbox")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	input.Address = strings.ToLower(strings.TrimSpace(input.Address))
	if input.EmailDomainID == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("An email_domain is required.", "email_domain_id"))
	}
	at := strings.LastIndex(input.Address, "@")
	if at <= 0 || at == len(input.Address)-1 {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("A valid email address is required.", "address"))
	}

	// The inbox address must sit on the caller's own verified domain.
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, input.EmailDomainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !strings.EqualFold(input.Address[at+1:], dom.Domain) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The address must be on the selected domain.", "address"))
	}
	if dom.Status != domain.EmailDomainStatusVerified {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The domain must be verified before adding an inbox.", "email_domain_id"))
	}

	inboxID, apiErr := id.GenID(id.EmailInboxIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := s.repoFactory.NewEmailInboxRepo().Create(ctx, inboxID, accountID, &input); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailInboxRepo().GetByID(ctx, inboxID, accountID)
}

func (s *emailBridgeSvcImpl) ListInboxes(ctx context.Context) ([]*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.list_inboxes")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailInboxRepo().ListByAccount(ctx, accountID)
}

func (s *emailBridgeSvcImpl) GetInbox(ctx context.Context, inboxID string) (*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.get_inbox")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailInboxRepo().GetByID(ctx, inboxID, accountID)
}

func (s *emailBridgeSvcImpl) UpdateInbox(ctx context.Context, inboxID string, input domain.UpdateEmailInboxInput) (*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.update_inbox")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if input.Status == "" {
		input.Status = domain.EmailInboxStatusActive
	}
	if input.Status != domain.EmailInboxStatusActive && input.Status != domain.EmailInboxStatusDisabled {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("Invalid inbox status.", "status"))
	}
	if apiErr := s.repoFactory.NewEmailInboxRepo().Update(ctx, inboxID, accountID, &input); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailInboxRepo().GetByID(ctx, inboxID, accountID)
}

func (s *emailBridgeSvcImpl) DeleteInbox(ctx context.Context, inboxID string) *apierror.APIError {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.delete_inbox")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	deleted, apiErr := s.repoFactory.NewEmailInboxRepo().Delete(ctx, inboxID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !deleted {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Email inbox not found."))
	}
	return nil
}
