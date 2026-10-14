package service

import (
	"context"
	"net/mail"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

var emailBridgeSvcTracer = tracing.GetTracer("notification-service.email_bridge_service")

type emailBridgeSvcImpl struct {
	repoFactory      domain.RepoFactory
	txManager        TransactionManager
	identityProvider domain.EmailIdentityProvider
}

// NewEmailBridgeSvc constructs the email-bridge service from a repo factory, a transaction manager and the SES identity provider.
func NewEmailBridgeSvc(repoFactory domain.RepoFactory, txManager TransactionManager, identityProvider domain.EmailIdentityProvider) domain.EmailBridgeSvc {
	return &emailBridgeSvcImpl{repoFactory: repoFactory, txManager: txManager, identityProvider: identityProvider}
}

func publishEmailBridgeAudit(ctx context.Context, f domain.RepoFactory, action constants.AuditAction, resourceType constants.ObjectType, resourceID string, changes []audit.FieldChange) *apierror.APIError {
	return audit.NewPublisher().Publish(ctx, f.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      changes,
	})
}

// accountID resolves the caller's acting account from the request identity, and is the sole authorization point for every email-bridge operation: registering and verifying domains, provisioning inboxes, and choosing the address customer-facing mail is sent from.
//
// It requires an INTERNAL actor of the target account. The gateway's coarse permission gate cannot be relied on here — APIEndpoint.authorize returns nil for customer- and supplier-relation actors by design, deferring their authorization to the service — so without this check a counterparty targeting the merchant's account would pass the gate and reach these methods. That would let a customer register a domain on the merchant's account, or repoint the From and Reply-To used for the merchant's invoices, checkout links, purchase orders, and statements at a mailbox of their choosing.
//
// This is merchant configuration with no counterparty-facing use, so requiring an internal actor costs no legitimate caller. Agents qualify: they run as same-account internal actors (see the agent identity built in agent-service's runner).
func (s *emailBridgeSvcImpl) accountID(ctx context.Context) (string, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return "", apierror.NewAuthenticationError("Authentication is required.")
	}
	if !identity.IsTargetAccountSet() {
		return "", apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", apiErr
	}
	return identity.Target.AccountID, nil
}

// readAccountID is accountID for a read, which also admits an actor loading what an authorized request includes.
func (s *emailBridgeSvcImpl) readAccountID(ctx context.Context) (string, *apierror.APIError) {
	if identity, ok := appctx.Identity[*types.Identity](ctx); ok && identity.IsIncludeRead() {
		return identity.Target.AccountID, nil
	}
	return s.accountID(ctx)
}

// ensureMailFromDomain points the identity's envelope Return-Path at a subdomain the customer controls. DKIM alone leaves the Return-Path on amazonses.com, and mail clients annotate a sender whose Return-Path domain does not match its From domain — so without this a merchant's mail arrives as "orders@theirdomain.com via amazonses.com" however well it is signed.
//
// It returns the subdomain the caller must record, or nil when one is already configured or SES refused. The SES call is best-effort: SES falls back to the default Return-Path until the customer publishes the records, so a failure here costs the annotation rather than the domain. Both CreateDomain and VerifyDomain route through here, so a domain that missed it — registered before this existed, or failed at creation — is repaired by verifying again. Callers must not swallow a failure to record it: that would leave SES and the database disagreeing about which subdomain the customer has to publish.
func (s *emailBridgeSvcImpl) ensureMailFromDomain(ctx context.Context, domainName string, existing *string) *string {
	if existing != nil && *existing != "" {
		return nil
	}
	records, apiErr := s.identityProvider.SetMailFromDomain(ctx, domainName, domain.MailFromSubdomain(domainName))
	if apiErr != nil {
		return nil
	}
	return &records.Subdomain
}

// reloadDomain re-reads a domain and renders its DNS records. Verify writes to the row before returning it (marking it verified, configuring MAIL FROM), so the copy read at the top of the call is already stale.
func (s *emailBridgeSvcImpl) reloadDomain(ctx context.Context, domainID, accountID string) (*domain.EmailDomain, *apierror.APIError) {
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, domainID, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	return s.withMailFromRecords(dom), nil
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
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("domain", "A valid domain is required."))
	}

	// Register the identity with SES first so a persistence failure doesn't leave a domain row with no
	// DKIM tokens; SES registration is idempotent and safe to retry.
	dkimTokens, apiErr := s.identityProvider.RegisterDomain(ctx, domainName)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	mailFrom := s.ensureMailFromDomain(ctx, domainName, nil)

	domainID, apiErr := id.GenID(id.EmailDomainIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var dom *domain.EmailDomain
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewEmailDomainRepo()
		if apiErr := repo.Create(txCtx, domainID, accountID, &domain.CreateEmailDomainInput{
			Domain:     domainName,
			DkimTokens: dkimTokens,
		}); apiErr != nil {
			return apiErr
		}
		if mailFrom != nil {
			if apiErr := repo.SetMailFromDomain(txCtx, domainID, accountID, *mailFrom); apiErr != nil {
				return apiErr
			}
		}
		created, apiErr := repo.GetByID(txCtx, domainID, accountID)
		if apiErr != nil {
			return apiErr
		}
		dom = created
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionCreate, constants.ObjectTypeEmailDomain, domainID, audit.ComputeChanges(nil, created))
	})
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
	accountID, apiErr := s.readAccountID(ctx)
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
	// Verify doubles as the repair path for the best-effort MAIL FROM setup in CreateDomain: a domain
	// registered before custom MAIL FROM existed, or one whose SES call failed at creation, would
	// otherwise keep the amazonses.com Return-Path forever with no way to fix it through the API.
	mailFrom := s.ensureMailFromDomain(ctx, dom.Domain, dom.MailFromDomain)

	markVerified := false
	if dom.Status != domain.EmailDomainStatusVerified {
		markVerified, apiErr = s.identityProvider.DomainVerified(ctx, dom.Domain)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if mailFrom == nil && !markVerified {
		return s.reloadDomain(ctx, domainID, accountID)
	}

	var verifiedDom *domain.EmailDomain
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewEmailDomainRepo()
		if mailFrom != nil {
			if apiErr := repo.SetMailFromDomain(txCtx, domainID, accountID, *mailFrom); apiErr != nil {
				return apiErr
			}
		}
		if markVerified {
			if apiErr := repo.MarkVerified(txCtx, domainID, accountID); apiErr != nil {
				return apiErr
			}
		}
		after, apiErr := repo.GetByID(txCtx, domainID, accountID)
		if apiErr != nil {
			return apiErr
		}
		verifiedDom = after
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionUpdate, constants.ObjectTypeEmailDomain, domainID, audit.ComputeChanges(dom, after))
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.withMailFromRecords(verifiedDom), nil
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
		return tracing.Trace(span, apierror.NewConflictErrorWithParam("id", "Delete the inboxes on this domain before deleting the domain."))
	}

	// Delete the SES identity first (foreign mutation). It is idempotent, so if the row delete below fails
	// the whole operation can be retried safely.
	if apiErr := s.identityProvider.DeleteDomain(ctx, dom.Domain); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := deleteDomainSender(txCtx, f, domainID, accountID); apiErr != nil {
			return apiErr
		}
		deleted, apiErr := f.NewEmailDomainRepo().Delete(txCtx, domainID, accountID)
		if apiErr != nil {
			return apiErr
		}
		if !deleted {
			return apierror.NewNotFoundError("Email domain not found.")
		}
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionDelete, constants.ObjectTypeEmailDomain, domainID, audit.ComputeChanges(dom, (*domain.EmailDomain)(nil)))
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// deleteDomainSender removes the account's sender when it is bound to the domain being deleted. Left in place it would resolve to an address SES no longer knows, failing every merchant send instead of falling back to the platform address.
func deleteDomainSender(ctx context.Context, f domain.RepoFactory, domainID, accountID string) *apierror.APIError {
	senderRepo := f.NewAccountEmailSenderRepo()
	sender, apiErr := senderRepo.GetByAccount(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}
	deleted, apiErr := senderRepo.DeleteByDomain(ctx, domainID, accountID)
	if apiErr != nil {
		return apiErr
	}
	if !deleted || sender == nil || sender.EmailDomainID != domainID {
		return nil
	}
	return publishEmailBridgeAudit(ctx, f, constants.AuditActionDelete, constants.ObjectTypeEmailSender, sender.ID, audit.ComputeChanges(sender, (*domain.AccountEmailSender)(nil)))
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
		return nil, tracing.Trace(span, apierror.NewMissingFieldError("email_domain_id", "An email_domain is required."))
	}
	at := strings.LastIndex(input.Address, "@")
	if at <= 0 || at == len(input.Address)-1 {
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("address", "A valid email address is required."))
	}

	// The inbox address must sit on the caller's own verified domain.
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, input.EmailDomainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !strings.EqualFold(input.Address[at+1:], dom.Domain) {
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("address", "The address must be on the selected domain."))
	}
	if dom.Status != domain.EmailDomainStatusVerified {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("email_domain_id", "The domain must be verified before adding an inbox."))
	}

	if apiErr := s.validateInboxGroup(ctx, accountID, input.GroupID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	inboxID, apiErr := id.GenID(id.EmailInboxIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var inbox *domain.EmailInbox
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewEmailInboxRepo()
		if apiErr := repo.Create(txCtx, inboxID, accountID, &input); apiErr != nil {
			return apiErr
		}
		created, apiErr := repo.GetByID(txCtx, inboxID, accountID)
		if apiErr != nil {
			return apiErr
		}
		inbox = created
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionCreate, constants.ObjectTypeEmailInbox, inboxID, audit.ComputeChanges(nil, created))
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return inbox, nil
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
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("status", "Invalid inbox status."))
	}
	if apiErr := s.validateInboxGroup(ctx, accountID, input.GroupID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var inbox *domain.EmailInbox
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewEmailInboxRepo()
		before, apiErr := repo.GetByID(txCtx, inboxID, accountID)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repo.Update(txCtx, inboxID, accountID, &input); apiErr != nil {
			return apiErr
		}
		after, apiErr := repo.GetByID(txCtx, inboxID, accountID)
		if apiErr != nil {
			return apiErr
		}
		inbox = after
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionUpdate, constants.ObjectTypeEmailInbox, inboxID, audit.ComputeChanges(before, after))
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return inbox, nil
}

// validateInboxGroup ensures a group_id set on an inbox names a roster the caller's account owns, so a new
// email thread can seat its members. A blank/nil group_id is allowed (no team seeded). The account scope on
// MessagingGroupRepo.Get is what prevents pointing an inbox at another tenant's roster.
func (s *emailBridgeSvcImpl) validateInboxGroup(ctx context.Context, accountID string, groupID *string) *apierror.APIError {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return nil
	}
	if _, apiErr := s.repoFactory.NewMessagingGroupRepo().Get(ctx, strings.TrimSpace(*groupID), accountID); apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return apierror.NewParameterInvalidError("group_id", "The group does not exist.")
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
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewEmailInboxRepo()
		existing, apiErr := repo.GetByID(txCtx, inboxID, accountID)
		if apiErr != nil {
			if apiErr.Code == apierror.CodeResourceNotFound {
				return apierror.NewNotFoundError("Email inbox not found.")
			}
			return apiErr
		}
		deleted, apiErr := repo.Delete(txCtx, inboxID, accountID)
		if apiErr != nil {
			return apiErr
		}
		if !deleted {
			return apierror.NewNotFoundError("Email inbox not found.")
		}
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionDelete, constants.ObjectTypeEmailInbox, inboxID, audit.ComputeChanges(existing, (*domain.EmailInbox)(nil)))
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
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
		return nil, tracing.Trace(span, apierror.NewNotFoundError("No email sender is configured for this account."))
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
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("local_part", "A valid mailbox name is required, for example \"orders\"."))
	}

	// The display name is written into a header of every merchant-facing message. A control character there is never legitimate and is how header injection starts, so it is refused at the door rather than only neutralized at render time.
	if input.FromName != nil {
		trimmed := strings.TrimSpace(*input.FromName)
		if containsControlChar(trimmed) {
			return nil, tracing.Trace(span, apierror.NewInvalidFormatError("from_name", "The display name cannot contain line breaks or control characters."))
		}
		input.FromName = &trimmed
	}
	if input.ReplyTo != nil {
		trimmed := strings.TrimSpace(*input.ReplyTo)
		if trimmed != "" && !isValidEmailAddress(trimmed) {
			return nil, tracing.Trace(span, apierror.NewInvalidFormatError("reply_to", "A valid reply-to email address is required."))
		}
		input.ReplyTo = &trimmed
	}

	// The domain has to be this account's, and verified — SES rejects an identity it has not confirmed, so accepting an unverified one here would bounce every merchant email later instead of failing now.
	dom, apiErr := s.repoFactory.NewEmailDomainRepo().GetByID(ctx, input.EmailDomainID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if dom.Status != domain.EmailDomainStatusVerified {
		return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("email_domain_id", "Verify this domain before sending from it."))
	}

	senderID, apiErr := id.GenID(id.EmailSenderIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var sender *domain.AccountEmailSender
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewAccountEmailSenderRepo()
		before, apiErr := repo.GetByAccount(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repo.Upsert(txCtx, senderID, accountID, &input); apiErr != nil {
			return apiErr
		}
		after, apiErr := repo.GetByAccount(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		if after == nil {
			return apierror.NewInvariantViolationError("The email sender was not found after it was saved.")
		}
		sender = after
		// The upsert keeps an existing row's id, so a replacement reads as an update of the same sender.
		action := constants.AuditActionUpdate
		if before == nil {
			action = constants.AuditActionCreate
		}
		return publishEmailBridgeAudit(txCtx, f, action, constants.ObjectTypeEmailSender, after.ID, audit.ComputeChanges(before, after))
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return sender, nil
}

func (s *emailBridgeSvcImpl) DeleteSender(ctx context.Context) *apierror.APIError {
	ctx, span := emailBridgeSvcTracer.Start(ctx, "service.email_bridge.delete_sender")
	defer span.End()
	accountID, apiErr := s.accountID(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewAccountEmailSenderRepo()
		existing, apiErr := repo.GetByAccount(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		deleted, apiErr := repo.Delete(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		if !deleted || existing == nil {
			return apierror.NewNotFoundError("No email sender is configured for this account.")
		}
		return publishEmailBridgeAudit(txCtx, f, constants.AuditActionDelete, constants.ObjectTypeEmailSender, existing.ID, audit.ComputeChanges(existing, (*domain.AccountEmailSender)(nil)))
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
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

// containsControlChar reports whether s holds a C0/C7F control character. CR and LF are the dangerous ones — they end a header — but no control character belongs in a display name, so the whole class is refused.
func containsControlChar(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// isValidEmailAddress reports whether s parses as a single RFC 5322 address. Reply-To is echoed into a header on every merchant send, so it is parsed rather than pattern-matched: mail.ParseAddress rejects the embedded newlines and stray angle brackets a regex tends to let through.
func isValidEmailAddress(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}
