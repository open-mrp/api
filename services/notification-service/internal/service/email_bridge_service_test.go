package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const ebTestAccountID = "ac_eb_test"

func emailBridgeCtx() context.Context {
	acct := ebTestAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: ebTestAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "us_eb_test",
			AccountID:    &acct,
		},
	})
}

// emailBridgeRelationActorCtx is a customer of the target account — the shape the gateway's coarse
// permission gate deliberately waves through (APIEndpoint.authorize returns nil for relation actors
// and defers to the service), so the service is the only thing standing between a counterparty and
// the merchant's mail configuration.
func emailBridgeRelationActorCtx() context.Context {
	buyer := "ac_eb_buyer"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: ebTestAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "us_eb_buyer",
			AccountID:    &buyer,
		},
	})
}

// A counterparty must not be able to read or rewrite the merchant's mail configuration. Repointing the
// sender would send the merchant's invoices and checkout links from — and route their replies to — an
// address the counterparty controls; registering a domain would seed the identity to do it with.
func TestEmailBridge_RejectsRelationActors(t *testing.T) {
	svc := newEmailBridgeSvc(t, repositorymock.NewMockEmailDomainRepo(gomock.NewController(t)), nil, &stubIdentityProvider{verified: true})
	ctx := emailBridgeRelationActorCtx()

	t.Run("SetSender", func(t *testing.T) {
		_, apiErr := svc.UpdateSender(ctx, domain.UpdateAccountEmailSenderInput{EmailDomainID: new("emdn_1"), LocalPart: new("orders")})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeInsufficientPermissions, apiErr.Code)
	})
	t.Run("GetSender", func(t *testing.T) {
		_, apiErr := svc.GetSender(ctx)
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeInsufficientPermissions, apiErr.Code)
	})
	t.Run("DeleteSender", func(t *testing.T) {
		_, apiErr := svc.DeleteSender(ctx)
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeInsufficientPermissions, apiErr.Code)
	})
	t.Run("CreateDomain", func(t *testing.T) {
		_, apiErr := svc.CreateDomain(ctx, "attacker.example.com")
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeInsufficientPermissions, apiErr.Code)
	})
	t.Run("CreateInbox", func(t *testing.T) {
		_, apiErr := svc.CreateInbox(ctx, domain.CreateEmailInboxInput{EmailDomainID: "emdn_1", Address: "x@theirco.com"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeInsufficientPermissions, apiErr.Code)
	})
}

// stubIdentityProvider is a controllable EmailIdentityProvider for service tests.
type stubIdentityProvider struct {
	tokens   []string
	verified bool
}

func (p *stubIdentityProvider) RegisterDomain(_ context.Context, _ string) ([]string, *apierror.APIError) {
	return p.tokens, nil
}

func (p *stubIdentityProvider) DomainVerified(_ context.Context, _ string) (bool, *apierror.APIError) {
	return p.verified, nil
}

func (p *stubIdentityProvider) DeleteDomain(_ context.Context, _ string) *apierror.APIError {
	return nil
}

func (p *stubIdentityProvider) SetMailFromDomain(_ context.Context, _, mailFromSubdomain string) (domain.MailFromRecords, *apierror.APIError) {
	return p.MailFromRecordsFor(mailFromSubdomain), nil
}

func (p *stubIdentityProvider) MailFromRecordsFor(mailFromSubdomain string) domain.MailFromRecords {
	return domain.MailFromRecords{
		Subdomain: mailFromSubdomain,
		MXRecord:  "10 feedback-smtp.us-east-1.amazonses.com",
		SPFRecord: "v=spf1 include:amazonses.com ~all",
	}
}

func newEmailBridgeSvc(t *testing.T, domainRepo *repositorymock.MockEmailDomainRepo, inboxRepo *repositorymock.MockEmailInboxRepo, provider domain.EmailIdentityProvider) *emailBridgeSvcImpl {
	t.Helper()
	ctrl := gomock.NewController(t)
	factory := factorymock.NewMockRepoFactory(ctrl)
	if domainRepo != nil {
		factory.EXPECT().NewEmailDomainRepo().Return(domainRepo).AnyTimes()
	}
	if inboxRepo != nil {
		factory.EXPECT().NewEmailInboxRepo().Return(inboxRepo).AnyTimes()
	}
	// CreateDomain records the MAIL FROM subdomain, and DeleteDomain clears any sender bound to the domain; both are incidental to what these tests assert.
	senderRepo := repositorymock.NewMockAccountEmailSenderRepo(ctrl)
	senderRepo.EXPECT().GetByAccount(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	senderRepo.EXPECT().DeleteByDomain(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	factory.EXPECT().NewAccountEmailSenderRepo().Return(senderRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()
	if domainRepo != nil {
		domainRepo.EXPECT().SetMailFromDomain(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	}
	return &emailBridgeSvcImpl{repoFactory: factory, txManager: passthroughTx{f: factory}, identityProvider: provider}
}

func TestCreateInbox_RejectsAddressOffDomain(t *testing.T) {
	domainRepo := repositorymock.NewMockEmailDomainRepo(gomock.NewController(t))
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusVerified,
	}, nil)
	svc := newEmailBridgeSvc(t, domainRepo, nil, &stubIdentityProvider{})

	_, apiErr := svc.CreateInbox(emailBridgeCtx(), domain.CreateEmailInboxInput{
		EmailDomainID: "emdn_1",
		Address:       "support@elsewhere.com",
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "address", apiErr.Param)
}

func TestCreateInbox_RejectsUnverifiedDomain(t *testing.T) {
	domainRepo := repositorymock.NewMockEmailDomainRepo(gomock.NewController(t))
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusPending,
	}, nil)
	svc := newEmailBridgeSvc(t, domainRepo, nil, &stubIdentityProvider{})

	_, apiErr := svc.CreateInbox(emailBridgeCtx(), domain.CreateEmailInboxInput{
		EmailDomainID: "emdn_1",
		Address:       "support@theirco.com",
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "email_domain_id", apiErr.Param)
}

func TestVerifyDomain_FlipsWhenSESConfirms(t *testing.T) {
	domainRepo := repositorymock.NewMockEmailDomainRepo(gomock.NewController(t))
	gomock.InOrder(
		domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
			ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusPending,
		}, nil),
		domainRepo.EXPECT().MarkVerified(gomock.Any(), "emdn_1", ebTestAccountID).Return(nil),
		domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
			ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusVerified,
		}, nil),
	)
	svc := newEmailBridgeSvc(t, domainRepo, nil, &stubIdentityProvider{verified: true})

	dom, apiErr := svc.VerifyDomain(emailBridgeCtx(), "emdn_1")
	require.Nil(t, apiErr)
	assert.Equal(t, domain.EmailDomainStatusVerified, dom.Status)
}

func TestDeleteDomain_BlocksWhenInboxesExist(t *testing.T) {
	domainRepo := repositorymock.NewMockEmailDomainRepo(gomock.NewController(t))
	inboxRepo := repositorymock.NewMockEmailInboxRepo(gomock.NewController(t))
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusVerified,
	}, nil)
	inboxRepo.EXPECT().CountByDomain(gomock.Any(), ebTestAccountID, "emdn_1").Return(int64(2), nil)
	// The SES identity must not be deleted and the row must not be removed while inboxes remain.
	svc := newEmailBridgeSvc(t, domainRepo, inboxRepo, &stubIdentityProvider{})

	apiErr := svc.DeleteDomain(emailBridgeCtx(), "emdn_1")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeResourceConflict, apiErr.Code)
}

func TestDeleteDomain_DeletesWhenNoInboxes(t *testing.T) {
	domainRepo := repositorymock.NewMockEmailDomainRepo(gomock.NewController(t))
	inboxRepo := repositorymock.NewMockEmailInboxRepo(gomock.NewController(t))
	gomock.InOrder(
		domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
			ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusVerified,
		}, nil),
		inboxRepo.EXPECT().CountByDomain(gomock.Any(), ebTestAccountID, "emdn_1").Return(int64(0), nil),
		domainRepo.EXPECT().Delete(gomock.Any(), "emdn_1", ebTestAccountID).Return(true, nil),
	)
	svc := newEmailBridgeSvc(t, domainRepo, inboxRepo, &stubIdentityProvider{})

	apiErr := svc.DeleteDomain(emailBridgeCtx(), "emdn_1")
	require.Nil(t, apiErr)
}

func TestVerifyDomain_NoFlipWhenSESPending(t *testing.T) {
	domainRepo := repositorymock.NewMockEmailDomainRepo(gomock.NewController(t))
	// Verify writes to the row (MAIL FROM) before returning it, so it re-reads rather than returning the copy it started from.
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusPending,
	}, nil).AnyTimes()
	svc := newEmailBridgeSvc(t, domainRepo, nil, &stubIdentityProvider{verified: false})

	dom, apiErr := svc.VerifyDomain(emailBridgeCtx(), "emdn_1")
	require.Nil(t, apiErr)
	assert.Equal(t, domain.EmailDomainStatusPending, dom.Status)
}

// A domain registered before custom MAIL FROM existed, or one whose SES call failed at creation, keeps the amazonses.com Return-Path — which is what makes a merchant's mail read as "via amazonses.com". Verify is the only retry path a customer has, so it has to repair that rather than only re-poll DKIM.
func TestVerifyDomain_ConfiguresMissingMailFrom(t *testing.T) {
	ctrl := gomock.NewController(t)
	domainRepo := repositorymock.NewMockEmailDomainRepo(ctrl)
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID:     "emdn_1",
		Domain: "theirco.com",
		Status: domain.EmailDomainStatusVerified,
		// No MailFromDomain — the state this repairs.
	}, nil).AnyTimes()
	domainRepo.EXPECT().
		SetMailFromDomain(gomock.Any(), "emdn_1", ebTestAccountID, "mail.theirco.com").
		Return(nil).
		Times(1)

	svc := newEmailBridgeSvc(t, nil, nil, &stubIdentityProvider{verified: true})
	svc.repoFactory = repoFactoryWithDomainRepo(t, ctrl, domainRepo)
	svc.txManager = passthroughTx{f: svc.repoFactory}

	_, apiErr := svc.VerifyDomain(emailBridgeCtx(), "emdn_1")
	require.Nil(t, apiErr)
}

// An already-configured domain must not be re-pushed to SES on every verify poll.
func TestVerifyDomain_LeavesConfiguredMailFromAlone(t *testing.T) {
	ctrl := gomock.NewController(t)
	existing := "mail.theirco.com"
	domainRepo := repositorymock.NewMockEmailDomainRepo(ctrl)
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID: "emdn_1", Domain: "theirco.com", Status: domain.EmailDomainStatusVerified, MailFromDomain: &existing,
	}, nil).AnyTimes()
	domainRepo.EXPECT().SetMailFromDomain(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := newEmailBridgeSvc(t, nil, nil, &stubIdentityProvider{verified: true})
	svc.repoFactory = repoFactoryWithDomainRepo(t, ctrl, domainRepo)
	svc.txManager = passthroughTx{f: svc.repoFactory}

	_, apiErr := svc.VerifyDomain(emailBridgeCtx(), "emdn_1")
	require.Nil(t, apiErr)
}

// The background recheck verifies a pending domain in its own account, and touches each domain it checks first so one SES keeps failing on can't starve the rest.
func TestRecheckPendingDomains(t *testing.T) {
	ctrl := gomock.NewController(t)
	mailFrom := "mail.theirco.com"
	pending := &domain.EmailDomain{ID: "emdn_p", AccountID: "ac_other", Domain: "theirco.com", Status: domain.EmailDomainStatusPending, MailFromDomain: &mailFrom}
	domainRepo := repositorymock.NewMockEmailDomainRepo(ctrl)
	gomock.InOrder(
		domainRepo.EXPECT().ListPending(gomock.Any(), gomock.Any(), int32(20)).Return([]*domain.EmailDomain{pending}, nil),
		domainRepo.EXPECT().Touch(gomock.Any(), "emdn_p").Return(nil),
		domainRepo.EXPECT().MarkVerified(gomock.Any(), "emdn_p", "ac_other").Return(nil),
	)
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_p", "ac_other").Return(&domain.EmailDomain{
		ID: "emdn_p", AccountID: "ac_other", Domain: "theirco.com", Status: domain.EmailDomainStatusVerified, MailFromDomain: &mailFrom,
	}, nil)

	svc := newEmailBridgeSvc(t, nil, nil, &stubIdentityProvider{verified: true})
	svc.repoFactory = repoFactoryWithDomainRepo(t, ctrl, domainRepo)
	svc.txManager = passthroughTx{f: svc.repoFactory}

	n, apiErr := svc.RecheckPendingDomains(context.Background(), 5*time.Minute, 20)
	require.Nil(t, apiErr)
	assert.Equal(t, 1, n)
}

// repoFactoryWithDomainRepo builds a factory over a domain repo whose SetMailFromDomain expectations are the assertion, rather than the permissive AnyTimes() the shared helper installs.
func repoFactoryWithDomainRepo(t *testing.T, ctrl *gomock.Controller, domainRepo *repositorymock.MockEmailDomainRepo) domain.RepoFactory {
	t.Helper()
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewEmailDomainRepo().Return(domainRepo).AnyTimes()
	senderRepo := repositorymock.NewMockAccountEmailSenderRepo(ctrl)
	senderRepo.EXPECT().DeleteByDomain(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	factory.EXPECT().NewAccountEmailSenderRepo().Return(senderRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()
	return factory
}

// The sender is an upsert, so the first save is the sender's creation and a later one replaces the same row.
func TestSetSender_AuditsCreateThenUpdate(t *testing.T) {
	cases := []struct {
		name       string
		existing   *domain.AccountEmailSender
		wantAction constants.AuditAction
	}{
		{"no sender yet", nil, constants.AuditActionCreate},
		{"replacing a sender", &domain.AccountEmailSender{ID: "emsn_1", EmailDomainID: "emdn_1", LocalPart: "billing"}, constants.AuditActionUpdate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			domainRepo := repositorymock.NewMockEmailDomainRepo(ctrl)
			domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
				ID: "emdn_1", Domain: "theirco.com", Status: domain.EmailDomainStatusVerified,
			}, nil).MaxTimes(1)
			saved := &domain.AccountEmailSender{ID: "emsn_1", EmailDomainID: "emdn_1", LocalPart: "orders", Domain: "theirco.com"}
			senderRepo := repositorymock.NewMockAccountEmailSenderRepo(ctrl)
			gomock.InOrder(
				senderRepo.EXPECT().GetByAccount(gomock.Any(), ebTestAccountID).Return(tc.existing, nil),
				senderRepo.EXPECT().GetByAccount(gomock.Any(), ebTestAccountID).Return(tc.existing, nil),
				senderRepo.EXPECT().Upsert(gomock.Any(), gomock.Any(), ebTestAccountID, gomock.Any()).Return(nil),
				senderRepo.EXPECT().GetByAccount(gomock.Any(), ebTestAccountID).Return(saved, nil),
			)
			outbox := &fakeOutboxRepo{}
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewEmailDomainRepo().Return(domainRepo).AnyTimes()
			factory.EXPECT().NewAccountEmailSenderRepo().Return(senderRepo).AnyTimes()
			factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()
			svc := &emailBridgeSvcImpl{repoFactory: factory, txManager: passthroughTx{f: factory}, identityProvider: &stubIdentityProvider{}}

			_, apiErr := svc.UpdateSender(emailBridgeCtx(), domain.UpdateAccountEmailSenderInput{EmailDomainID: new("emdn_1"), LocalPart: new("orders")})
			require.Nil(t, apiErr)

			event := onlyAuditEvent(t, outbox)
			assert.Equal(t, string(tc.wantAction), event.Action)
			assert.Equal(t, string(constants.ObjectTypeEmailSender), event.ResourceType)
			assert.Equal(t, "emsn_1", event.ResourceID)
		})
	}
}

type auditEventPayload struct {
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

func onlyAuditEvent(t *testing.T, outbox *fakeOutboxRepo) auditEventPayload {
	t.Helper()
	var events []auditEventPayload
	for _, in := range outbox.inputs {
		if in.MessageType != string(contracts.PlatformEventAuditLogged) {
			continue
		}
		var event auditEventPayload
		require.NoError(t, json.Unmarshal(in.Payload.Data, &event))
		events = append(events, event)
	}
	require.Len(t, events, 1)
	return events[0]
}

// Creating a sender needs a domain and a mailbox name; both are reported at once.
func TestUpdateSender_CreateRequiresDomainAndLocalPart(t *testing.T) {
	ctrl := gomock.NewController(t)
	senderRepo := repositorymock.NewMockAccountEmailSenderRepo(ctrl)
	senderRepo.EXPECT().GetByAccount(gomock.Any(), ebTestAccountID).Return(nil, nil)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAccountEmailSenderRepo().Return(senderRepo).AnyTimes()
	svc := &emailBridgeSvcImpl{repoFactory: factory, txManager: passthroughTx{f: factory}, identityProvider: &stubIdentityProvider{}}

	_, apiErr := svc.UpdateSender(emailBridgeCtx(), domain.UpdateAccountEmailSenderInput{FromName: new("Acme")})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	require.Len(t, apiErr.Errors, 2)
	assert.Equal(t, "email_domain_id", apiErr.Errors[0].Param)
	assert.Equal(t, "local_part", apiErr.Errors[1].Param)
}

// A PATCH that only renames keeps the domain and mailbox, and doesn't re-check a domain it isn't changing.
func TestUpdateSender_MergesIntoCurrentSender(t *testing.T) {
	ctrl := gomock.NewController(t)
	current := &domain.AccountEmailSender{ID: "emsn_1", EmailDomainID: "emdn_1", LocalPart: "billing", ReplyTo: new("ap@theirco.com"), Domain: "theirco.com"}
	senderRepo := repositorymock.NewMockAccountEmailSenderRepo(ctrl)
	senderRepo.EXPECT().GetByAccount(gomock.Any(), ebTestAccountID).Return(current, nil).Times(3)
	var saved *domain.UpsertAccountEmailSenderInput
	senderRepo.EXPECT().Upsert(gomock.Any(), gomock.Any(), ebTestAccountID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ string, in *domain.UpsertAccountEmailSenderInput) *apierror.APIError {
			saved = in
			return nil
		})
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAccountEmailSenderRepo().Return(senderRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()
	svc := &emailBridgeSvcImpl{repoFactory: factory, txManager: passthroughTx{f: factory}, identityProvider: &stubIdentityProvider{}}

	_, apiErr := svc.UpdateSender(emailBridgeCtx(), domain.UpdateAccountEmailSenderInput{FromName: new("Acme Billing"), ClearReplyTo: true})
	require.Nil(t, apiErr)
	require.NotNil(t, saved)
	assert.Equal(t, "emdn_1", saved.EmailDomainID)
	assert.Equal(t, "billing", saved.LocalPart)
	assert.Equal(t, "Acme Billing", *saved.FromName)
	assert.Nil(t, saved.ReplyTo)
}

func TestEmailDomainDNSRecords(t *testing.T) {
	mailFrom := "mail.theirco.com"
	dom := &domain.EmailDomain{
		Domain:            "theirco.com",
		DkimTokens:        []string{"abc", "def"},
		MailFromDomain:    &mailFrom,
		MailFromMXRecord:  "10 feedback-smtp.us-east-1.amazonses.com",
		MailFromSPFRecord: "v=spf1 include:amazonses.com ~all",
	}
	assert.Equal(t, []domain.DNSRecord{
		{Type: "CNAME", Name: "abc._domainkey.theirco.com", Value: "abc.dkim.amazonses.com"},
		{Type: "CNAME", Name: "def._domainkey.theirco.com", Value: "def.dkim.amazonses.com"},
		{Type: "MX", Name: "mail.theirco.com", Value: "10 feedback-smtp.us-east-1.amazonses.com"},
		{Type: "TXT", Name: "mail.theirco.com", Value: "v=spf1 include:amazonses.com ~all"},
	}, dom.DNSRecords())
}
