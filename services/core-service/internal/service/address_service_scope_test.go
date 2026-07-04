package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// AddressSvcScopeTestSuite exercises the account-scoping logic in
// address_service.go: which account a created address is linked to, and when the
// cross-account EditAccess check runs. This guards the customer-portal regression
// where a customer creating an address (actor = customer account, target = the
// merchant they order from) was blocked by an EditAccess check against the
// merchant and, if it had passed, linked to the wrong account.
type AddressSvcScopeTestSuite struct {
	suite.Suite
	svc domain.AddressSvc

	addressRepo     *repositorymock.MockAddressRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	editAccessMed   *mediatormock.MockEditAccessMed

	ctrl *gomock.Controller
}

func (suite *AddressSvcScopeTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.addressRepo = repositorymock.NewMockAddressRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAddressRepo().Return(suite.addressRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.editAccessMed = mediatormock.NewMockEditAccessMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
		EditAccess:  suite.editAccessMed,
	}).AnyTimes()

	suite.svc = NewAddressSvc(&AddressSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *AddressSvcScopeTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestAddressSvcScopeTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AddressSvcScopeTestSuite))
}

func addressCustomerCtx(targetAccountID, customerAccountID string) context.Context {
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

func addressInternalCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			// No role set -> full address access (see checkAddressWritePermission).
		},
	})
}

// addressInternalCtxWithPerms builds an internal actor on its own account with an
// explicit role + permission set.
func addressInternalCtxWithPerms(accountID string, perms map[string]bool) context.Context {
	roleType := "member"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			RoleType:     &roleType,
			Permissions:  perms,
		},
	})
}

func (suite *AddressSvcScopeTestSuite) expectIdempotencyStartedThenSuccess() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_test", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).
		Return(nil).
		Times(1)
}

// A customer actor creating an address on the portal targets the merchant account
// (external target), but the address must be linked to the customer's OWN account
// and must NOT trigger the EditAccess check against the merchant.
func (suite *AddressSvcScopeTestSuite) TestCreateAddress_CustomerActor_ScopedToOwnAccount() {
	const merchantAccountID = "acct_merchant"
	const customerAccountID = "acct_customer"

	suite.expectIdempotencyStartedThenSuccess()

	// EditAccess must never be consulted for a relation actor.
	suite.editAccessMed.EXPECT().CheckEditAccess(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	var linkedAccountID string
	suite.addressRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, addressID, _, _ string, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
			linkedAccountID = params.AccountID
			return &domain.Address{ID: addressID, Name: params.Name}, nil
		}).
		Times(1)

	ctx := addressCustomerCtx(merchantAccountID, customerAccountID)
	addr, apiErr := suite.svc.CreateAddress(ctx, domain.CreateAddressParams{Name: "Ship To", Country: "US"})

	suite.Require().Nil(apiErr)
	suite.Require().NotNil(addr)
	suite.Equal(customerAccountID, linkedAccountID, "address must be linked to the customer's own account, not the target merchant")
}

// An internal actor operating on its own account links to the target account and
// runs no cross-account check.
func (suite *AddressSvcScopeTestSuite) TestCreateAddress_InternalActor_ScopedToTargetAccount() {
	const accountID = "acct_internal"

	suite.expectIdempotencyStartedThenSuccess()
	suite.editAccessMed.EXPECT().CheckEditAccess(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	var linkedAccountID string
	suite.addressRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, addressID, _, _ string, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
			linkedAccountID = params.AccountID
			return &domain.Address{ID: addressID, Name: params.Name}, nil
		}).
		Times(1)

	ctx := addressInternalCtx(accountID)
	_, apiErr := suite.svc.CreateAddress(ctx, domain.CreateAddressParams{Name: "HQ", Country: "US"})

	suite.Require().Nil(apiErr)
	suite.Equal(accountID, linkedAccountID)
}

// A roled internal actor on its own account that holds customers:update (the
// legacy permission for address writes) but NOT addresses:create must still be
// allowed to create — the downstream check must not be stricter than the
// gateway's OR-gate. This is the customer-portal regression.
func (suite *AddressSvcScopeTestSuite) TestCreateAddress_InternalActor_CustomersUpdateOnly_Allowed() {
	const accountID = "acct_customer_portal"

	suite.expectIdempotencyStartedThenSuccess()
	suite.editAccessMed.EXPECT().CheckEditAccess(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	suite.addressRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, addressID, _, _ string, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
			return &domain.Address{ID: addressID, Name: params.Name}, nil
		}).
		Times(1)

	ctx := addressInternalCtxWithPerms(accountID, map[string]bool{"customers:update": true})
	_, apiErr := suite.svc.CreateAddress(ctx, domain.CreateAddressParams{Name: "Ship To", Country: "US"})

	suite.Require().Nil(apiErr, "customers:update must authorize an own-account address create")
}

// A roled internal actor holding none of the declared write permissions is still
// rejected (the check is not disabled, just aligned with the gateway's OR-set).
func (suite *AddressSvcScopeTestSuite) TestCreateAddress_InternalActor_NoWritePerms_Rejected() {
	const accountID = "acct_customer_portal"

	ctx := addressInternalCtxWithPerms(accountID, map[string]bool{"addresses:read": true})
	_, apiErr := suite.svc.CreateAddress(ctx, domain.CreateAddressParams{Name: "Ship To", Country: "US"})

	suite.Require().NotNil(apiErr)
}
