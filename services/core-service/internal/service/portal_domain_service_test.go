package service

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	clientmock "github.com/augno/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PortalDomainSvcTestSuite struct {
	suite.Suite
	svc domain.PortalDomainSvc

	portalDomainRepo *repositorymock.MockPortalDomainRepo
	idempotencyRepo  *repositorymock.MockIdempotencyKeyRepo
	repoFactory      *factorymock.MockRepoFactory
	mediatorFactory  *factorymock.MockMediatorFactory
	idempotencyMed   *mediatormock.MockIdempotencyMed
	provider         *clientmock.MockPortalDomainProvider

	ctrl *gomock.Controller
}

func (suite *PortalDomainSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.portalDomainRepo = repositorymock.NewMockPortalDomainRepo(suite.ctrl)
	suite.idempotencyRepo = repositorymock.NewMockIdempotencyKeyRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewPortalDomainRepo().Return(suite.portalDomainRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(suite.idempotencyRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.provider = clientmock.NewMockPortalDomainProvider(suite.ctrl)

	suite.svc = NewPortalDomainSvc(&PortalDomainSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		Provider:        suite.provider,
	})
}

func (suite *PortalDomainSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestPortalDomainSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PortalDomainSvcTestSuite))
}

const (
	testPortalAccountID = "acct_merchant"
	testPortalDomainID  = "podn_test"
	testPortalHost      = "shop.acme.com"
)

func portalDomainInternalCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			Permissions: map[string]bool{
				types.Permission{Domain: types.PermissionDomainAccount, Action: types.ActionRead}.String():   true,
				types.Permission{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}.String(): true,
			},
		},
	})
}

func portalDomainCustomerCtx(targetAccountID string) context.Context {
	customerAccountID := "acct_customer"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_customer",
			AccountID:    &customerAccountID,
			Permissions:  map[string]bool{},
		},
	})
}

func pendingPortalDomain() *domain.PortalDomain {
	return &domain.PortalDomain{
		ID:        testPortalDomainID,
		AccountID: testPortalAccountID,
		Domain:    testPortalHost,
		Status:    constants.PortalDomainStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (suite *PortalDomainSvcTestSuite) expectIdempotencyKey(recoveryPoint domain.RecoveryPoint) {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_test", RecoveryPoint: string(recoveryPoint)}, nil).
		Times(1)
}

func (suite *PortalDomainSvcTestSuite) TestCreatePortalDomain_HappyPath() {
	ctx := portalDomainInternalCtx(testPortalAccountID)
	routing := []domain.PortalDNSRecord{{Type: "CNAME", Name: testPortalHost, Value: "cname.vercel-dns.com", Reason: "routing"}}

	suite.expectIdempotencyKey(domain.RecoveryPointStarted)

	// Phase 1: uniqueness checks + insert + recovery point advance.
	suite.portalDomainRepo.EXPECT().GetByAccountID(gomock.Any(), testPortalAccountID).Return(nil, nil)
	suite.portalDomainRepo.EXPECT().GetByDomain(gomock.Any(), testPortalHost).Return(nil, nil).Times(1)
	suite.portalDomainRepo.EXPECT().Create(gomock.Any(), gomock.Any(), testPortalAccountID, testPortalHost).Return(pendingPortalDomain(), nil)
	suite.idempotencyRepo.EXPECT().AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.PortalDomainRecoveryPointProviderRegistered).Return(nil)

	// Provider phase: row re-read + idempotent provider registration.
	suite.portalDomainRepo.EXPECT().GetByDomain(gomock.Any(), testPortalHost).Return(pendingPortalDomain(), nil).Times(1)
	suite.provider.EXPECT().AddDomain(gomock.Any(), testPortalHost).Return(&domain.PortalDomainProviderState{
		Verified:      false,
		Misconfigured: true,
		DNSRecords:    routing,
	}, nil)

	// Final atomic phase: persist DNS records + cache response.
	suite.portalDomainRepo.EXPECT().UpdateProviderState(gomock.Any(), testPortalDomainID, constants.PortalDomainStatusPending, routing).Return(nil)
	withRecords := pendingPortalDomain()
	withRecords.DNSRecords = routing
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(withRecords, nil)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil)

	result, apiErr := suite.svc.CreatePortalDomain(ctx, "Shop.Acme.COM")
	suite.Nil(apiErr)
	suite.Equal(testPortalHost, result.Domain)
	suite.Equal(constants.PortalDomainStatusPending, result.Status)
	suite.Len(result.DNSRecords, 1)
}

func (suite *PortalDomainSvcTestSuite) TestCreatePortalDomain_ResumesProviderPhase() {
	ctx := portalDomainInternalCtx(testPortalAccountID)
	routing := []domain.PortalDNSRecord{{Type: "CNAME", Name: testPortalHost, Value: "cname.vercel-dns.com", Reason: "routing"}}

	// A prior attempt crashed after phase 1: the row exists and the recovery point is provider_registered. No uniqueness checks or insert may run again.
	suite.expectIdempotencyKey(domain.PortalDomainRecoveryPointProviderRegistered)

	suite.portalDomainRepo.EXPECT().GetByDomain(gomock.Any(), testPortalHost).Return(pendingPortalDomain(), nil)
	suite.provider.EXPECT().AddDomain(gomock.Any(), testPortalHost).Return(&domain.PortalDomainProviderState{Misconfigured: true, DNSRecords: routing}, nil)
	suite.portalDomainRepo.EXPECT().UpdateProviderState(gomock.Any(), testPortalDomainID, constants.PortalDomainStatusPending, routing).Return(nil)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(pendingPortalDomain(), nil)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil)

	result, apiErr := suite.svc.CreatePortalDomain(ctx, testPortalHost)
	suite.Nil(apiErr)
	suite.Equal(testPortalDomainID, result.ID)
}

func (suite *PortalDomainSvcTestSuite) TestCreatePortalDomain_ConflictWhenAccountHasDomain() {
	ctx := portalDomainInternalCtx(testPortalAccountID)

	suite.expectIdempotencyKey(domain.RecoveryPointStarted)
	suite.portalDomainRepo.EXPECT().GetByAccountID(gomock.Any(), testPortalAccountID).Return(pendingPortalDomain(), nil)
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr })

	_, apiErr := suite.svc.CreatePortalDomain(ctx, "portal.acme.com")
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *PortalDomainSvcTestSuite) TestCreatePortalDomain_RejectsInvalidAndAugnoDomains() {
	ctx := portalDomainInternalCtx(testPortalAccountID)

	for _, bad := range []string{"", "not-a-domain", "shop.augno.com", "augno.com", "http://shop.acme.com", "shop..acme.com"} {
		_, apiErr := suite.svc.CreatePortalDomain(ctx, bad)
		suite.NotNil(apiErr, "expected validation error for %q", bad)
		suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code, "domain %q", bad)
	}
}

func (suite *PortalDomainSvcTestSuite) TestCreatePortalDomain_RejectsCustomerActor() {
	_, apiErr := suite.svc.CreatePortalDomain(portalDomainCustomerCtx(testPortalAccountID), testPortalHost)
	suite.NotNil(apiErr)
}

func (suite *PortalDomainSvcTestSuite) TestVerifyPortalDomain_MarksVerifiedWhenProviderReady() {
	ctx := portalDomainInternalCtx(testPortalAccountID)
	routing := []domain.PortalDNSRecord{{Type: "CNAME", Name: testPortalHost, Value: "cname.vercel-dns.com", Reason: "routing"}}

	suite.expectIdempotencyKey(domain.RecoveryPointStarted)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(pendingPortalDomain(), nil).Times(1)
	suite.provider.EXPECT().GetDomainState(gomock.Any(), testPortalHost).Return(&domain.PortalDomainProviderState{Verified: true, Misconfigured: false, Serving: true, DNSRecords: routing}, nil)

	suite.portalDomainRepo.EXPECT().UpdateProviderState(gomock.Any(), testPortalDomainID, constants.PortalDomainStatusVerified, routing).Return(nil)
	suite.portalDomainRepo.EXPECT().MarkVerified(gomock.Any(), testPortalDomainID).Return(nil)
	verified := pendingPortalDomain()
	verified.Status = constants.PortalDomainStatusVerified
	now := time.Now().UTC()
	verified.VerifiedAt = &now
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(verified, nil).Times(1)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil)

	result, apiErr := suite.svc.VerifyPortalDomain(ctx, testPortalDomainID)
	suite.Nil(apiErr)
	suite.Equal(constants.PortalDomainStatusVerified, result.Status)
	suite.NotNil(result.VerifiedAt)
}

func (suite *PortalDomainSvcTestSuite) TestVerifyPortalDomain_SecuringWhileCertificateIssues() {
	ctx := portalDomainInternalCtx(testPortalAccountID)
	routing := []domain.PortalDNSRecord{{Type: "CNAME", Name: testPortalHost, Value: "cname.vercel-dns.com", Reason: "routing"}}

	// DNS is correct (verified + routing) but the domain does not yet answer over HTTPS: the TLS certificate is still issuing, so the domain moves to securing, not verified, and verified_at is not stamped.
	suite.expectIdempotencyKey(domain.RecoveryPointStarted)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(pendingPortalDomain(), nil).Times(1)
	suite.provider.EXPECT().GetDomainState(gomock.Any(), testPortalHost).Return(&domain.PortalDomainProviderState{Verified: true, Misconfigured: false, Serving: false, DNSRecords: routing}, nil)

	suite.portalDomainRepo.EXPECT().UpdateProviderState(gomock.Any(), testPortalDomainID, constants.PortalDomainStatusSecuring, routing).Return(nil)
	// No MarkVerified while securing.
	securing := pendingPortalDomain()
	securing.Status = constants.PortalDomainStatusSecuring
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(securing, nil).Times(1)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil)

	result, apiErr := suite.svc.VerifyPortalDomain(ctx, testPortalDomainID)
	suite.Nil(apiErr)
	suite.Equal(constants.PortalDomainStatusSecuring, result.Status)
	suite.Nil(result.VerifiedAt)
}

func (suite *PortalDomainSvcTestSuite) TestVerifyPortalDomain_StaysPendingWhenMisconfigured() {
	ctx := portalDomainInternalCtx(testPortalAccountID)
	routing := []domain.PortalDNSRecord{{Type: "CNAME", Name: testPortalHost, Value: "cname.vercel-dns.com", Reason: "routing"}}

	suite.expectIdempotencyKey(domain.RecoveryPointStarted)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(pendingPortalDomain(), nil).Times(1)
	suite.provider.EXPECT().GetDomainState(gomock.Any(), testPortalHost).Return(&domain.PortalDomainProviderState{Verified: false, Misconfigured: true, DNSRecords: routing}, nil)
	suite.portalDomainRepo.EXPECT().UpdateProviderState(gomock.Any(), testPortalDomainID, constants.PortalDomainStatusPending, routing).Return(nil)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(pendingPortalDomain(), nil).Times(1)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil)

	result, apiErr := suite.svc.VerifyPortalDomain(ctx, testPortalDomainID)
	suite.Nil(apiErr)
	suite.Equal(constants.PortalDomainStatusPending, result.Status)
}

func (suite *PortalDomainSvcTestSuite) TestVerifyPortalDomain_ShortCircuitsWhenAlreadyVerified() {
	ctx := portalDomainInternalCtx(testPortalAccountID)

	verified := pendingPortalDomain()
	verified.Status = constants.PortalDomainStatusVerified

	suite.expectIdempotencyKey(domain.RecoveryPointStarted)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(verified, nil)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil)
	// No provider call may happen.

	result, apiErr := suite.svc.VerifyPortalDomain(ctx, testPortalDomainID)
	suite.Nil(apiErr)
	suite.Equal(constants.PortalDomainStatusVerified, result.Status)
}

func (suite *PortalDomainSvcTestSuite) TestVerifyPortalDomain_NotFound() {
	ctx := portalDomainInternalCtx(testPortalAccountID)

	suite.expectIdempotencyKey(domain.RecoveryPointStarted)
	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, "podn_missing").Return(nil, nil)
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr })

	_, apiErr := suite.svc.VerifyPortalDomain(ctx, "podn_missing")
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

func (suite *PortalDomainSvcTestSuite) TestDeletePortalDomain_RemovesProviderFirst() {
	ctx := portalDomainInternalCtx(testPortalAccountID)

	gomock.InOrder(
		suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(pendingPortalDomain(), nil),
		suite.provider.EXPECT().RemoveDomain(gomock.Any(), testPortalHost).Return(nil),
		suite.portalDomainRepo.EXPECT().Delete(gomock.Any(), testPortalAccountID, testPortalDomainID).Return(true, nil),
	)

	apiErr := suite.svc.DeletePortalDomain(ctx, testPortalDomainID)
	suite.Nil(apiErr)
}

func (suite *PortalDomainSvcTestSuite) TestDeletePortalDomain_NotFound() {
	ctx := portalDomainInternalCtx(testPortalAccountID)

	suite.portalDomainRepo.EXPECT().GetByID(gomock.Any(), testPortalAccountID, "podn_missing").Return(nil, nil)

	apiErr := suite.svc.DeletePortalDomain(ctx, "podn_missing")
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

func (suite *PortalDomainSvcTestSuite) TestResolvePortalHost_NormalizesAndResolves() {
	account := &domain.PublicAccountBySlug{ID: testPortalAccountID, Name: "Acme", Slug: "acme"}
	suite.portalDomainRepo.EXPECT().ResolveVerifiedHost(gomock.Any(), testPortalHost).Return(account, nil)

	// No identity in context: the resolver is unauthenticated.
	result, apiErr := suite.svc.ResolvePortalHost(context.Background(), "SHOP.ACME.COM.")
	suite.Nil(apiErr)
	suite.Equal("acme", result.Slug)
}
