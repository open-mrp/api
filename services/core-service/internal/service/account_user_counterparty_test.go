package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	counterpartySellerID     = "ac_seller"
	counterpartyCustomerID   = "ac_customer"
	counterpartyActorID      = "us_actor"
	counterpartyAccountUser  = "acus_contact"
	counterpartyContactUser  = "us_contact"
	counterpartyOtherContact = "acus_other"
)

// sellerCtx is a seller's staff member acting in the account its relation names, or in the seller's own account when
// relation is empty, holding exactly perms.
func sellerCtx(relation types.IdentityRelationType, perms ...string) context.Context {
	seller := counterpartySellerID
	granted := make(map[string]bool, len(perms))
	for _, p := range perms {
		granted[p] = true
	}
	target := &types.IdentityTarget{AccountID: counterpartySellerID}
	if relation != "" {
		target = &types.IdentityTarget{AccountID: counterpartyCustomerID, RelationType: &relation}
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: target,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           counterpartyActorID,
			AccountID:    &seller,
			Permissions:  granted,
		},
	})
}

func mustIdentity(t *testing.T, ctx context.Context) *types.Identity {
	t.Helper()
	identity, ok := appctx.GetIdentityFromContext(ctx)
	require.True(t, ok)
	return identity
}

// Each action on a customer's or supplier's users takes that domain's permission of the same name; the seller's own
// team takes the team permission.
func TestAccountUserPermissions_FollowTheTargetAndAction(t *testing.T) {
	t.Parallel()

	checks := map[types.Action]func(*types.Identity) *apierror.APIError{
		types.ActionCreate: checkAccountUserCreatePermission,
		types.ActionUpdate: checkAccountUserUpdatePermission,
		types.ActionDelete: checkAccountUserDeletePermission,
	}
	domains := map[types.IdentityRelationType]types.PermissionDomain{
		types.IdentityRelationTypeCustomer: types.PermissionDomainCustomers,
		types.IdentityRelationTypeSupplier: types.PermissionDomainSuppliers,
		"":                                 types.PermissionDomainTeamUsers,
	}
	for relation, permDomain := range domains {
		for action, check := range checks {
			granted := string(permDomain) + ":" + string(action)
			assert.Nil(t, check(mustIdentity(t, sellerCtx(relation, granted))), "%q target with %s", relation, granted)

			// Holding every other action of the right domain, or this action of every other domain, is not enough.
			var others []string
			for otherAction := range checks {
				if otherAction != action {
					others = append(others, string(permDomain)+":"+string(otherAction))
				}
			}
			for otherDomain := range domains {
				if d := domains[otherDomain]; d != permDomain {
					others = append(others, string(d)+":"+string(action))
				}
			}
			apiErr := check(mustIdentity(t, sellerCtx(relation, others...)))
			if assert.NotNil(t, apiErr, "%q target %s without %s", relation, action, granted) {
				assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
			}
		}
	}
}

// The link in a welcome email goes to the dashboard for the seller's own staff and to the seller's portal for a
// customer's users, on its verified custom domain when there is one.
func TestWelcomeLoginLink(t *testing.T) {
	t.Parallel()

	slug := "acme"
	blank := "  "
	cases := []struct {
		name, frontendURL, domain string
		slug                      *string
		want                      string
	}{
		{name: "own staff sign in to the dashboard", frontendURL: "https://app.example.com", want: "https://app.example.com/auth/login"},
		{name: "a trailing slash is not doubled", frontendURL: "https://app.example.com/", want: "https://app.example.com/auth/login"},
		{name: "portal users sign in under the slug", frontendURL: "https://app.example.com", slug: &slug, want: "https://app.example.com/acme/auth/login"},
		{name: "a verified custom domain serves the portal without the slug", frontendURL: "https://app.example.com", domain: "portal.example.com", slug: &slug, want: "https://portal.example.com/auth/login"},
		{name: "a blank slug falls back to the dashboard", frontendURL: "https://app.example.com", slug: &blank, want: "https://app.example.com/auth/login"},
		{name: "no frontend and no domain gives no link", slug: &slug, want: ""},
		{name: "a custom domain needs no frontend", domain: "portal.example.com", want: "https://portal.example.com/auth/login"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, welcomeLoginLink(tc.frontendURL, tc.domain, tc.slug), tc.name)
	}
}

// Toggling preferences only writes the difference: an enabled type already on is not inserted again (a restored
// contact keeps their old rows), a disabled one is deleted, and an unlisted one is left alone.
func TestApplyNotificationPreferences_WritesOnlyTheDifference(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	relationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)

	relationRepo.EXPECT().ListNotificationPreferences(gomock.Any(), "ar_rel", counterpartyAccountUser).Return([]domain.NotificationPreference{
		{ID: "arnp_ack", NotificationTypeCode: string(constants.AccountRelationNotificationTypeOrderAcknowledgement)},
		{ID: "arnp_po", NotificationTypeCode: string(constants.AccountRelationNotificationTypePurchaseOrderSubmission)},
	}, nil)
	relationRepo.EXPECT().CreateNotificationPreference(gomock.Any(), gomock.Any(), "ar_rel", counterpartyAccountUser, string(constants.AccountRelationNotificationTypeInvoice)).Return(nil)
	relationRepo.EXPECT().DeleteNotificationPreference(gomock.Any(), "ar_rel", counterpartyAccountUser, string(constants.AccountRelationNotificationTypePurchaseOrderSubmission)).Return(nil)

	apiErr := applyNotificationPreferences(context.Background(), relationRepo, "ar_rel", counterpartyAccountUser, []domain.NotificationPreferenceItem{
		{NotificationTypeCode: string(constants.AccountRelationNotificationTypeOrderAcknowledgement), Enabled: true},
		{NotificationTypeCode: string(constants.AccountRelationNotificationTypeInvoice), Enabled: true},
		{NotificationTypeCode: string(constants.AccountRelationNotificationTypeInvoice), Enabled: true},
		{NotificationTypeCode: string(constants.AccountRelationNotificationTypePurchaseOrderSubmission), Enabled: false},
	})
	require.Nil(t, apiErr)
}

func newCounterpartyAccountUserSvc(ctrl *gomock.Controller, repos domain.RepoFactory, meds domain.Mediators, billing domain.BillingPublisher) domain.AccountUserSvc {
	mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(meds).AnyTimes()
	if billing == nil {
		billing = publishermock.NewMockBillingPublisher(ctrl)
	}
	return NewAccountUserSvc(&AccountUserSvcConfig{
		Repos:                 repos,
		MediatorFactory:       mediatorFactory,
		TxManager:             &stubTxManager{factory: repos},
		NotificationPublisher: publishermock.NewMockNotificationPublisher(ctrl),
		BillingPublisher:      billing,
		S3Client:              &capturingObjectStore{},
		PlatformMode:          constants.PlatformModeTest,
	})
}

// A customer's users carry the notification types the seller sends them, an empty list when none, read for the whole
// batch at once; the seller's own users carry none, and no preference lookup is made for them.
func TestBatchGetAccountUsersByIDs_NotificationTypes(t *testing.T) {
	t.Parallel()

	t.Run("customer contacts", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
		relationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
		repos := factorymock.NewMockRepoFactory(ctrl)
		repos.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()
		repos.EXPECT().NewAccountRelationRepo().Return(relationRepo).AnyTimes()
		readAccess := mediatormock.NewMockReadAccessMed(ctrl)
		readAccess.EXPECT().CheckReadAccess(gomock.Any(), counterpartySellerID, counterpartyCustomerID).Return(nil)

		ids := []string{counterpartyAccountUser, counterpartyOtherContact}
		accountUserRepo.EXPECT().GetByIDs(gomock.Any(), counterpartyCustomerID, ids).Return([]*domain.AccountUserDetail{
			{ID: counterpartyAccountUser}, {ID: counterpartyOtherContact},
		}, nil)
		relationRepo.EXPECT().ListNotificationTypesForRecipients(gomock.Any(), counterpartySellerID, counterpartyCustomerID, ids).
			Return(map[string][]string{counterpartyAccountUser: {"invoice", "order_acknowledgement"}}, nil).Times(1)

		svc := newCounterpartyAccountUserSvc(ctrl, repos, domain.Mediators{ReadAccess: readAccess}, nil)
		users, apiErr := svc.BatchGetAccountUsersByIDs(sellerCtx(types.IdentityRelationTypeCustomer, "customers:read"), ids)
		require.Nil(t, apiErr)
		require.Len(t, users, 2)
		assert.Equal(t, []string{"invoice", "order_acknowledgement"}, users[0].NotificationTypes)
		assert.NotNil(t, users[1].NotificationTypes, "a contact with nothing enabled has an empty list, not none")
		assert.Empty(t, users[1].NotificationTypes)
	})

	t.Run("own team", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
		repos := factorymock.NewMockRepoFactory(ctrl)
		repos.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()
		accountUserRepo.EXPECT().GetByIDs(gomock.Any(), counterpartySellerID, []string{counterpartyAccountUser}).
			Return([]*domain.AccountUserDetail{{ID: counterpartyAccountUser}}, nil)

		svc := newCounterpartyAccountUserSvc(ctrl, repos, domain.Mediators{}, nil)
		users, apiErr := svc.BatchGetAccountUsersByIDs(sellerCtx("", "team:read"), []string{counterpartyAccountUser})
		require.Nil(t, apiErr)
		require.Len(t, users, 1)
		assert.Nil(t, users[0].NotificationTypes)
	})
}

// Removing and restoring a customer's contact checks the caller's own membership in the seller's account (they have
// none in the customer's), and restoring does not count the contact against the seller's seats.
func TestUpdateAccountUserStatus_CustomerContact(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, current constants.AccountUserStatus) (*repositorymock.MockAccountUserRepo, domain.AccountUserSvc) {
		ctrl := gomock.NewController(t)
		accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
		deletedRepo := repositorymock.NewMockDeletedRecordRepo(ctrl)
		// No account repo: a seat-limit lookup would fail the test as an unexpected call.
		repos := factorymock.NewMockRepoFactory(ctrl)
		repos.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()
		repos.EXPECT().NewDeletedRecordRepo().Return(deletedRepo).AnyTimes()
		repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()
		editAccess := mediatormock.NewMockEditAccessMed(ctrl)
		editAccess.EXPECT().CheckEditAccess(gomock.Any(), counterpartySellerID, counterpartyCustomerID).Return(nil)
		billing := publishermock.NewMockBillingPublisher(ctrl)
		billing.EXPECT().PublishReportSeatChange(gomock.Any(), counterpartyCustomerID).Return(nil)

		accountUserRepo.EXPECT().GetDetail(gomock.Any(), counterpartySellerID, counterpartyActorID, gomock.Any()).
			Return(&domain.AccountUserDetail{ID: "acus_actor", UserID: counterpartyActorID, StatusCode: constants.AccountUserStatusActive}, nil)
		accountUserRepo.EXPECT().GetDetailByAccountAndID(gomock.Any(), counterpartyCustomerID, counterpartyAccountUser, gomock.Any()).
			Return(&domain.AccountUserDetail{ID: counterpartyAccountUser, UserID: counterpartyContactUser, StatusCode: current}, nil).AnyTimes()
		deletedRepo.EXPECT().Create(gomock.Any(), gomock.Any(), counterpartyAccountUser, gomock.Any()).Return(nil).AnyTimes()

		return accountUserRepo, newCounterpartyAccountUserSvc(ctrl, repos, domain.Mediators{EditAccess: editAccess}, billing)
	}

	t.Run("remove", func(t *testing.T) {
		t.Parallel()
		accountUserRepo, svc := setup(t, constants.AccountUserStatusActive)
		accountUserRepo.EXPECT().SoftDelete(gomock.Any(), counterpartyAccountUser).Return(nil)

		apiErr := svc.UpdateAccountUserStatus(sellerCtx(types.IdentityRelationTypeCustomer, "customers:delete"), counterpartyAccountUser, constants.AccountUserStatusRemoved)
		require.Nil(t, apiErr, apierror.Describe(apiErr))
	})

	t.Run("restore", func(t *testing.T) {
		t.Parallel()
		accountUserRepo, svc := setup(t, constants.AccountUserStatusRemoved)
		accountUserRepo.EXPECT().UpdateStatus(gomock.Any(), counterpartyAccountUser, constants.AccountUserStatusActive).Return(nil)

		apiErr := svc.UpdateAccountUserStatus(sellerCtx(types.IdentityRelationTypeCustomer, "customers:update"), counterpartyAccountUser, constants.AccountUserStatusActive)
		require.Nil(t, apiErr, apierror.Describe(apiErr))
	})
}

// Removing a customer's contact takes customers:delete; customers:update alone is refused before anything is read.
func TestUpdateAccountUserStatus_RemoveNeedsDelete(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := newCounterpartyAccountUserSvc(ctrl, factorymock.NewMockRepoFactory(ctrl), domain.Mediators{}, nil)

	apiErr := svc.UpdateAccountUserStatus(sellerCtx(types.IdentityRelationTypeCustomer, "customers:update"), counterpartyAccountUser, constants.AccountUserStatusRemoved)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// Reading a customer's users' profiles takes customers:read, not the seller's team permission, and still requires a
// relation to the customer.
func TestBatchGetUsersByIDs_FollowsTheTarget(t *testing.T) {
	t.Parallel()

	t.Run("customer contacts with customers:read", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := repositorymock.NewMockUserRepo(ctrl)
		repos := factorymock.NewMockRepoFactory(ctrl)
		repos.EXPECT().NewUserRepo().Return(userRepo).AnyTimes()
		readAccess := mediatormock.NewMockReadAccessMed(ctrl)
		readAccess.EXPECT().CheckReadAccess(gomock.Any(), counterpartySellerID, counterpartyCustomerID).Return(nil)
		mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
		mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{ReadAccess: readAccess}).AnyTimes()
		userRepo.EXPECT().GetByIDs(gomock.Any(), counterpartyCustomerID, []string{counterpartyContactUser}).
			Return([]*domain.UserRecord{{ID: counterpartyContactUser}}, nil)

		svc := &userSvcImpl{repos: repos, mediatorFactory: mediatorFactory}
		users, apiErr := svc.BatchGetUsersByIDs(sellerCtx(types.IdentityRelationTypeCustomer, "customers:read"), []string{counterpartyContactUser})
		require.Nil(t, apiErr, apierror.Describe(apiErr))
		require.Len(t, users, 1)
	})

	t.Run("customer without a relation", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		readAccess := mediatormock.NewMockReadAccessMed(ctrl)
		readAccess.EXPECT().CheckReadAccess(gomock.Any(), counterpartySellerID, counterpartyCustomerID).
			Return(apierror.NewAuthorizationError("You cannot access this account."))
		mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
		mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{ReadAccess: readAccess}).AnyTimes()

		svc := &userSvcImpl{repos: factorymock.NewMockRepoFactory(ctrl), mediatorFactory: mediatorFactory}
		_, apiErr := svc.BatchGetUsersByIDs(sellerCtx(types.IdentityRelationTypeCustomer, "customers:read"), []string{counterpartyContactUser})
		require.NotNil(t, apiErr)
	})

	t.Run("own team needs team:read", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		svc := &userSvcImpl{repos: factorymock.NewMockRepoFactory(ctrl)}
		_, apiErr := svc.BatchGetUsersByIDs(sellerCtx("", "customers:read"), []string{counterpartyContactUser})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
	})
}
