package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type registrationSvcSetup struct {
	svc           domain.RegistrationFlowSvc
	accounts      *repositorymock.MockAccountRepo
	groups        *repositorymock.MockAccountGroupRepo
	paymentTerms  *repositorymock.MockPaymentTermRepo
	shippingTerms *repositorymock.MockShippingTermRepo
	idempotency   *mediatormock.MockIdempotencyMed
}

// newRegistrationSvcSetup wires no customer registration repo, so a test fails if a refused registration reaches a write.
func newRegistrationSvcSetup(t *testing.T) *registrationSvcSetup {
	ctrl := gomock.NewController(t)
	s := &registrationSvcSetup{
		accounts:      repositorymock.NewMockAccountRepo(ctrl),
		groups:        repositorymock.NewMockAccountGroupRepo(ctrl),
		paymentTerms:  repositorymock.NewMockPaymentTermRepo(ctrl),
		shippingTerms: repositorymock.NewMockShippingTermRepo(ctrl),
		idempotency:   mediatormock.NewMockIdempotencyMed(ctrl),
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAccountRepo().Return(s.accounts).AnyTimes()
	repos.EXPECT().NewAccountGroupRepo().Return(s.groups).AnyTimes()
	repos.EXPECT().NewPaymentTermRepo().Return(s.paymentTerms).AnyTimes()
	repos.EXPECT().NewShippingTermRepo().Return(s.shippingTerms).AnyTimes()
	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: s.idempotency}).AnyTimes()

	s.svc = NewRegistrationFlowSvc(&RegistrationFlowSvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		TxManager:       &stubTxManager{factory: repos},
	})
	return s
}

func registrationCtx(actorType types.IdentityActorType, actorID string, actorAccountID *string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:  actorType,
		Actor: &types.IdentityActor{RelationType: types.IdentityRelationTypeInternal, ID: actorID, AccountID: actorAccountID},
	})
}

func registrationStr(s string) *string { return &s }

func TestRegisterCustomer_NeedsASignedInPerson(t *testing.T) {
	sellerID := "ac_seller"
	s := newRegistrationSvcSetup(t)

	apiErr := s.svc.RegisterCustomer(registrationCtx(types.IdentityActorTypeAPIKey, "ak_admin", &sellerID), domain.RegisterCustomerParams{
		AccountSlug:        "seller",
		IsExistingCustomer: true,
		CustomerData:       domain.CustomerRegistrationData{Number: registrationStr("1001")},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, "an API key has no user to link")
}

func TestRegisterCustomer_ANewCustomersTermsMustBeTheSellers(t *testing.T) {
	s := newRegistrationSvcSetup(t)
	s.accounts.EXPECT().GetBySlug(gomock.Any(), "seller").Return(&domain.PublicAccountBySlug{ID: "ac_seller"}, nil)
	s.idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_reg", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	s.groups.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"acgp_seller"}).Return([]*domain.AccountGroup{{ID: "acgp_seller"}}, nil)
	s.paymentTerms.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"pytm_other"}).Return(nil, nil)

	apiErr := s.svc.RegisterCustomer(registrationCtx(types.IdentityActorTypeUser, "usr_buyer", nil), domain.RegisterCustomerParams{
		AccountSlug: "seller",
		CustomerData: domain.CustomerRegistrationData{
			Name:            registrationStr("Buyer Co"),
			CustomerGroupID: registrationStr("acgp_seller"),
			PaymentTermID:   registrationStr("pytm_other"),
			ShippingTermID:  registrationStr("prepaid"),
			Address:         &domain.CustomerRegistrationAddress{Country: "US"},
		},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "payment_term_id", apiErr.Param)
}

func registrationFlowAdminCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_admin",
			AccountID:    &accountID,
			RoleType:     new(string(constants.RoleTypeAdmin)),
		},
	})
}

// A flow's terms are offered to registrants of this account, so another account's term is refused before anything is written.
func TestCreateRegistrationFlow_OffersOnlyTheAccountsTerms(t *testing.T) {
	s := newRegistrationSvcSetup(t)
	s.paymentTerms.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"pytm_own"}).Return([]*domain.PaymentTerm{{ID: "pytm_own"}}, nil)
	s.shippingTerms.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"prepaid", "shtm_other"}).Return([]*domain.ShippingTerm{{ID: "prepaid"}}, nil)

	_, apiErr := s.svc.CreateRegistrationFlow(registrationFlowAdminCtx("ac_seller"), domain.CreateRegistrationFlowParams{
		Name:            "Wholesale",
		PaymentTermIDs:  []string{"pytm_own"},
		ShippingTermIDs: []string{"prepaid", "shtm_other"},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "shipping_term_ids", apiErr.Param)
}

func TestUpdateRegistrationFlow_OffersOnlyTheAccountsTerms(t *testing.T) {
	s := newRegistrationSvcSetup(t)
	s.paymentTerms.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"pytm_other"}).Return(nil, nil)

	_, apiErr := s.svc.UpdateRegistrationFlow(registrationFlowAdminCtx("ac_seller"), domain.UpdateRegistrationFlowParams{
		RegistrationFlowID: "rgfw_1",
		PaymentTermIDs:     []string{"pytm_other"},
		HasPaymentTermIDs:  true,
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "payment_term_ids", apiErr.Param)
}

func TestDeleteRegistrationFlow_OnlyTheOwnerLearnsItWasDeleted(t *testing.T) {
	t.Parallel()
	assertOnlyTheOwnerLearnsItWasDeleted(t, constants.DeletedRecordResourceTypeRegistrationFlow, "rf_gone", func(h *deletedScopeHarness, accountID string) *apierror.APIError {
		flows := repositorymock.NewMockRegistrationFlowRepo(h.ctrl)
		h.repos.EXPECT().NewRegistrationFlowRepo().Return(flows).AnyTimes()
		flows.EXPECT().Get(gomock.Any(), accountID, "rf_gone").Return(nil, deletedScopeNotFound())
		svc := NewRegistrationFlowSvc(&RegistrationFlowSvcConfig{Repos: h.repos, MediatorFactory: h.mediators, TxManager: &stubTxManager{factory: h.repos}})
		return svc.DeleteRegistrationFlow(deletedScopeCtx(accountID), "rf_gone")
	})
}
