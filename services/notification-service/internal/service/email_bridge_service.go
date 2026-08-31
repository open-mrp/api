package service

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
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
		return "", apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required.")
	}
	return identity.Target.AccountID, nil
}

// withMailFromRecords fills in the DNS records the customer publishes for the domain's envelope Return-Path. Rendered on every read rather than stored, because the MX host names the SES region and a stored copy would go stale if that ever moved.
func (s *emailBridgeSvcImpl) withMailFromRecords(dom *domain.EmailDomain) *domain.EmailDomain {
	if dom == nil || dom.MailFromDomain == nil || *dom.MailFromDomain == "" {
		return dom
	}
	records := s.identityProvider.MailFromRecordsFor(*dom.MailFromDomain)
	dom.MailFromMXRecord = records.MXRecord
	dom.MailFromSPFRecord = records.SPFRecord
	return dom
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

	// Point the envelope Return-Path at a subdomain the customer controls. DKIM alone leaves the Return-Path
	// on amazonses.com, and mail clients annotate a sender whose Return-Path domain does not match its From
	// domain — so without this a merchant's mail arrives as "orders@theirdomain.com via amazonses.com".
	// Best-effort: SES falls back to the default Return-Path until the records are published, so a failure
	// here costs the annotation, not the domain, and the customer can retry through VerifyDomain.
	if records, apiErr := s.identityProvider.SetMailFromDomain(ctx, domainName, domain.MailFromSubdomain(domainName)); apiErr == nil {
		if apiErr := s.repoFactory.NewEmailDomainRepo().SetMailFromDomain(ctx, domainID, accountID, records.Subdomain); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.withMailFromRecords(dom), nil
}

func (s *emailBridgeSvcImpl) ListDomains(ctx context.Context) ([]*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.list_domains")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	domains, apiErr := s.repoFactory.NewEmailDomainRepo().ListByAccount(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, dom := range domains {
		s.withMailFromRecords(dom)
	}
	return domains, nil
}

func (s *emailBridgeSvcImpl) GetDomain(ctx context.Context, domainID string) (*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.get_domain")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.withMailFromRecords(dom), nil
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
		return s.withMailFromRecords(dom), nil
	}
	verified, apiErr := s.identityProvider.DomainVerified(ctx, dom.Domain)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !verified {
		return s.withMailFromRecords(dom), nil
	}
	if apiErr := s.repoFactory.NewEmailDomainRepo().MarkVerified(ctx, domainID, accountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	verifiedDomain, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.withMailFromRecords(verifiedDomain), nil
}

func (s *emailBridgeSvcImpl) DeleteDomain(ctx context.Context, domainID string) *apierror.APIError {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.delete_domain")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Resolve the domain first so we can enforce ownership and read its name for the SES identity delete.
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// A domain with inboxes on it can't be deleted — the inboxes would be left orphaned and unroutable.
	inboxCount, apiErr := s.repoFactory.NewEmailInboxRepo().CountByDomain(ctx, accountID, domainID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if inboxCount > 0 {
		return tracing.Trace(span, apierror.NewConflictErrorWithParam("Delete the inboxes on this domain before deleting the domain.", "id"))
	}

	// Delete the SES identity first (foreign mutation). It is idempotent, so if the row delete below fails
	// the whole operation can be retried safely.
	if apiErr := s.identityProvider.DeleteDomain(ctx, dom.Domain); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// A sender bound to this domain would otherwise survive it and resolve to an address SES no longer knows, failing every merchant send instead of falling back to the platform address.
	if _, apiErr := s.repoFactory.NewAccountEmailSenderRepo().DeleteByDomain(ctx, domainID, accountID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	deleted, apiErr := s.repoFactory.NewEmailDomainRepo().Delete(ctx, domainID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !deleted {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Email domain not found."))
	}
	return nil
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

	if apiErr := s.validateInboxGroup(ctx, accountID, input.GroupID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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
	if apiErr := s.validateInboxGroup(ctx, accountID, input.GroupID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := s.repoFactory.NewEmailInboxRepo().Update(ctx, inboxID, accountID, &input); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewEmailInboxRepo().GetByID(ctx, inboxID, accountID)
}

// validateInboxGroup ensures a group_id set on an inbox names a roster the caller's account owns, so a new
// email thread can seat its members. A blank/nil group_id is allowed (no team seeded). The account scope on
// MessagingGroupRepo.Get is what prevents pointing an inbox at another tenant's roster.
func (s *emailBridgeSvcImpl) validateInboxGroup(ctx context.Context, accountID string, groupID *string) *apierror.APIError {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return nil
	}
	if _, apiErr := s.repoFactory.NewMessagingGroupRepo().Get(ctx, strings.TrimSpace(*groupID), accountID); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return apierror.NewParameterInvalidError("The group does not exist.", "group_id")
		}
		return apiErr
	}
	return nil
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

// ── email_sender ──

func (s *emailBridgeSvcImpl) GetSender(ctx context.Context) (*domain.AccountEmailSender, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.get_sender")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	sender, apiErr := s.repoFactory.NewAccountEmailSenderRepo().GetByAccount(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The repo reports "no sender configured" as a nil row because that is the ordinary state on the send path. Read as a resource, its absence is a 404.
	if sender == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("No email sender is configured for this account."))
	}
	return sender, nil
}

func (s *emailBridgeSvcImpl) SetSender(ctx context.Context, input domain.UpsertAccountEmailSenderInput) (*domain.AccountEmailSender, *apierror.APIError) {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.set_sender")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	input.LocalPart = strings.ToLower(strings.TrimSpace(input.LocalPart))
	if !isValidLocalPart(input.LocalPart) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("A valid mailbox name is required, for example \"orders\".", "local_part"))
	}

	// The domain has to be this account's, and verified — SES rejects an identity it has not confirmed, so accepting an unverified one here would bounce every merchant email later instead of failing now.
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, input.EmailDomainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if dom.Status != domain.EmailDomainStatusVerified {
		return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Verify this domain before sending from it.", "email_domain_id"))
	}

	senderID, apiErr := id.GenID(id.EmailSenderIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := s.repoFactory.NewAccountEmailSenderRepo().Upsert(ctx, senderID, accountID, &input); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.GetSender(ctx)
}

func (s *emailBridgeSvcImpl) DeleteSender(ctx context.Context) *apierror.APIError {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.delete_sender")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	deleted, apiErr := s.repoFactory.NewAccountEmailSenderRepo().Delete(ctx, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !deleted {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("No email sender is configured for this account."))
	}
	return nil
}

// isValidLocalPart accepts the conservative subset of RFC 5322 local parts that survives every mail system in practice: letters, digits, and the separators dot, hyphen, underscore and plus, not leading or trailing. Quoted forms and the rarer specials are rejected rather than escaped.
func isValidLocalPart(localPart string) bool {
	const maxLocalPartLength = 64
	if localPart == "" || len(localPart) > maxLocalPartLength {
		return false
	}
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") || strings.Contains(localPart, "..") {
		return false
	}
	for _, r := range localPart {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '.' || r == '-' || r == '_' || r == '+':
		default:
			return false
		}
	}
	return true
}
