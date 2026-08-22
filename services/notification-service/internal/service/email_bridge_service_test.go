package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

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
	return &emailBridgeSvcImpl{repoFactory: factory, identityProvider: provider}
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
	assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)
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
	domainRepo.EXPECT().GetByID(gomock.Any(), "emdn_1", ebTestAccountID).Return(&domain.EmailDomain{
		ID: "emdn_1", AccountID: ebTestAccountID, Domain: "theirco.com", Status: domain.EmailDomainStatusPending,
	}, nil)
	svc := newEmailBridgeSvc(t, domainRepo, nil, &stubIdentityProvider{verified: false})

	dom, apiErr := svc.VerifyDomain(emailBridgeCtx(), "emdn_1")
	require.Nil(t, apiErr)
	assert.Equal(t, domain.EmailDomainStatusPending, dom.Status)
}
