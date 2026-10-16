package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/field"
)

type customerSvcSetup struct {
	svc           domain.CustomerSvc
	customers     *repositorymock.MockCustomerRepo
	salesOrders   *repositorymock.MockSalesOrderRepo
	deleted       *repositorymock.MockDeletedRecordRepo
	carriers      *repositorymock.MockCarrierRepo
	serviceLevels *repositorymock.MockServiceLevelRepo
	idempotency   *mediatormock.MockIdempotencyMed
	sites         *mediatormock.MockSiteMed
	addresses     *repositorymock.MockAddressRepo
	outbox        *recordingOutboxRepo
}

func newCustomerSvcSetup(t *testing.T) *customerSvcSetup {
	ctrl := gomock.NewController(t)
	s := &customerSvcSetup{
		customers:     repositorymock.NewMockCustomerRepo(ctrl),
		salesOrders:   repositorymock.NewMockSalesOrderRepo(ctrl),
		deleted:       repositorymock.NewMockDeletedRecordRepo(ctrl),
		carriers:      repositorymock.NewMockCarrierRepo(ctrl),
		serviceLevels: repositorymock.NewMockServiceLevelRepo(ctrl),
		idempotency:   mediatormock.NewMockIdempotencyMed(ctrl),
		sites:         mediatormock.NewMockSiteMed(ctrl),
		outbox:        &recordingOutboxRepo{},
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCustomerRepo().Return(s.customers).AnyTimes()
	repos.EXPECT().NewSalesOrderRepo().Return(s.salesOrders).AnyTimes()
	repos.EXPECT().NewDeletedRecordRepo().Return(s.deleted).AnyTimes()
	repos.EXPECT().NewCarrierRepo().Return(s.carriers).AnyTimes()
	repos.EXPECT().NewServiceLevelRepo().Return(s.serviceLevels).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()
	s.addresses = repositorymock.NewMockAddressRepo(ctrl)
	repos.EXPECT().NewAddressRepo().Return(s.addresses).AnyTimes()
	// A deleted customer's own price list goes with it; these customers have none.
	priceLists := repositorymock.NewMockPriceListRepo(ctrl)
	priceLists.EXPECT().GetBySource(gomock.Any(), gomock.Any(), constants.PriceListSourceCustomer, gomock.Any()).Return(nil, nil).AnyTimes()
	priceLists.EXPECT().DeleteCustomerAssignments(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	repos.EXPECT().NewPriceListRepo().Return(priceLists).AnyTimes()
	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: s.idempotency, Site: s.sites}).AnyTimes()
	jobs := factorymock.NewMockJobSvcFactory(ctrl)

	s.svc = NewCustomerSvc(&CustomerSvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		JobSvcFactory:   jobs,
		TxManager:       &stubTxManager{factory: repos},
	})
	return s
}

func (s *customerSvcSetup) expectWrite() {
	s.idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_cust", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	s.idempotency.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_cust", gomock.Any()).Return(nil).AnyTimes()
	s.idempotency.EXPECT().CacheErrorResponse(gomock.Any(), "idk_cust", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()
}

func customerInternalCtx(accountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test",
			AccountID:    &accountID,
			RoleType:     &adminCode,
		},
	})
}

// The receive calendar is a default like the lead time: an edit that does not send it must write it back unchanged.
func TestUpdateCustomer_KeepsTheReceiveCalendarItWasNotSent(t *testing.T) {
	s := newCustomerSvcSetup(t)
	s.expectWrite()
	calendarID := "opcal_dock"
	old := &domain.Customer{ID: "ac_buyer", Name: "Buyer Co", Number: "1001", ReceiveCalendarID: &calendarID}
	s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(old, nil).Times(2)
	s.customers.EXPECT().GetRelationID(gomock.Any(), "ac_seller", "ac_buyer").Return("acre_buyer", nil)

	var written domain.UpdateCustomerParams
	s.customers.EXPECT().Update(gomock.Any(), "acre_buyer", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params domain.UpdateCustomerParams) *apierror.APIError {
			written = params
			return nil
		})

	_, apiErr := s.svc.UpdateCustomer(customerInternalCtx("ac_seller"), domain.UpdateCustomerParams{
		CustomerAccountID: "ac_buyer",
		Instructions:      field.Set("unrelated edit"),
	})

	require.Nil(t, apiErr)
	got, ok := written.ReceiveCalendarID.Value()
	require.True(t, ok, "the calendar is written back, not cleared")
	assert.Equal(t, calendarID, got)
}

// The deleted-customer snapshot names its owner, and only the owner is told the customer was deleted.
func TestDeleteCustomer_OnlyTheOwnerLearnsACustomerWasDeleted(t *testing.T) {
	notFound := apierror.NewNotFoundError("Customer not found.")

	t.Run("records the snapshot under the owner", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		customer := &domain.Customer{ID: "ac_buyer"}
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(customer, nil)
		s.customers.EXPECT().GetRelationID(gomock.Any(), "ac_seller", "ac_buyer").Return("acre_buyer", nil)
		s.customers.EXPECT().InUseBy(gomock.Any(), "ac_seller", "ac_buyer", "acre_buyer").Return(nil, nil)
		s.deleted.EXPECT().CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, "ac_buyer", "ac_seller", customer).Return(nil)
		s.customers.EXPECT().Delete(gomock.Any(), "ac_seller", "ac_buyer").Return(nil)

		require.Nil(t, s.svc.DeleteCustomer(customerInternalCtx("ac_seller"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"}))
	})

	t.Run("a customer still in use is refused and kept", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(&domain.Customer{ID: "ac_buyer"}, nil)
		s.customers.EXPECT().GetRelationID(gomock.Any(), "ac_seller", "ac_buyer").Return("acre_buyer", nil)
		s.customers.EXPECT().InUseBy(gomock.Any(), "ac_seller", "ac_buyer", "acre_buyer").Return([]string{"sales orders", "child customers"}, nil)

		apiErr := s.svc.DeleteCustomer(customerInternalCtx("ac_seller"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeResourceInUse, apiErr.Code)
		assert.Contains(t, apiErr.PublicMessage, "sales orders and child customers")
	})

	t.Run("the owner deleting it again is told it is gone", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(nil, notFound)
		s.deleted.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, "ac_buyer", "ac_seller").Return(true, nil)

		apiErr := s.svc.DeleteCustomer(customerInternalCtx("ac_seller"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeResourceGone, apiErr.Code)
	})

	t.Run("another tenant is told it was never there", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		s.customers.EXPECT().Get(gomock.Any(), "ac_other", "ac_buyer", gomock.Any()).Return(nil, notFound)
		s.deleted.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, "ac_buyer", "ac_other").Return(false, nil)

		apiErr := s.svc.DeleteCustomer(customerInternalCtx("ac_other"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.CodeResourceNotFound, apiErr.Code)
	})
}

func TestBulkDeleteCustomers_RecordsEachSnapshotUnderTheOwner(t *testing.T) {
	s := newCustomerSvcSetup(t)
	for _, customerID := range []string{"ac_a", "ac_b"} {
		customer := &domain.Customer{ID: customerID}
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", customerID, gomock.Any()).Return(customer, nil)
		s.deleted.EXPECT().CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, customerID, "ac_seller", customer).Return(nil)
		s.customers.EXPECT().GetRelationID(gomock.Any(), "ac_seller", customerID).Return("acre_"+customerID, nil)
	}
	s.salesOrders.EXPECT().CountSalesOrdersForBuyerAccounts(gomock.Any(), "ac_seller", []string{"ac_a", "ac_b"}).Return(int64(0), nil)
	s.customers.EXPECT().BulkDelete(gomock.Any(), "ac_seller", []string{"ac_a", "ac_b"}).Return(nil)

	require.Nil(t, s.svc.BulkDeleteCustomers(customerInternalCtx("ac_seller"), domain.BulkDeleteCustomersParams{CustomerIDs: []string{"ac_a", "ac_b"}}))
}

// No unique index guards customer numbers: an edit that sets one queues on the owner's row before its
// first read, or two edits made at once both find the number free.
func TestUpdateCustomer_LocksTheOwnersNumbersBeforeItsFirstRead(t *testing.T) {
	s := newCustomerSvcSetup(t)
	s.expectWrite()
	number := "C-2"
	gomock.InOrder(
		s.customers.EXPECT().LockNumbers(gomock.Any(), "ac_seller").Return(nil),
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(&domain.Customer{ID: "ac_buyer", Number: "C-1"}, nil),
		s.customers.EXPECT().ExistsByNumber(gomock.Any(), "ac_seller", number, gomock.Any()).Return(true, nil),
	)

	_, apiErr := s.svc.UpdateCustomer(customerInternalCtx("ac_seller"), domain.UpdateCustomerParams{
		CustomerAccountID: "ac_buyer",
		Number:            &number,
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, "number", apiErr.Param)
}

// Every create takes a number, typed in or allocated, so every create queues on the owner's row first.
func TestCreateCustomer_ChecksATypedNumberOnlyUnderTheOwnersLock(t *testing.T) {
	s := newCustomerSvcSetup(t)
	s.expectWrite()
	number := "C-1"
	gomock.InOrder(
		s.customers.EXPECT().LockNumbers(gomock.Any(), "ac_seller").Return(nil),
		s.customers.EXPECT().ExistsByNumber(gomock.Any(), "ac_seller", number, nil).Return(true, nil),
	)

	_, apiErr := s.svc.CreateCustomer(customerInternalCtx("ac_seller"), domain.CreateCustomerParams{
		Name:          "Buyer Co",
		Number:        &number,
		BillToAddress: &domain.CreateAddressParams{},
		ShipToAddress: &domain.CreateAddressParams{},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, "number", apiErr.Param)
}

// expectRoutingOwned answers the ownership reads for the default carrier and service level as the account's own.
func (s *customerSvcSetup) expectRoutingOwned(carrierID, serviceLevelID string) {
	if carrierID != "" {
		s.carriers.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{carrierID}).Return([]*domain.Carrier{{ID: carrierID, IsActive: true}}, nil)
	}
	if serviceLevelID != "" {
		s.carriers.EXPECT().GetOptionsByIDs(gomock.Any(), "ac_seller", []string{serviceLevelID}).Return([]*domain.ServiceLevel{{ID: serviceLevelID, IsActive: true}}, nil)
	}
}

func TestCreateCustomer_RefusesADefaultServiceLevelOffTheDefaultCarrier(t *testing.T) {
	s := newCustomerSvcSetup(t)
	s.expectWrite()
	s.customers.EXPECT().LockNumbers(gomock.Any(), "ac_seller").Return(nil)
	s.expectRoutingOwned("cr_own", "crop_other")
	s.serviceLevels.EXPECT().IsInCarrier(gomock.Any(), "crop_other", "cr_own").Return(false, nil)

	_, apiErr := s.svc.CreateCustomer(customerInternalCtx("ac_seller"), domain.CreateCustomerParams{
		Name:                  "Buyer Co",
		DefaultCarrierID:      new("cr_own"),
		DefaultServiceLevelID: new("crop_other"),
		BillToAddress:         &domain.CreateAddressParams{},
		ShipToAddress:         &domain.CreateAddressParams{},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	assert.Equal(t, "defaults.service_level_id", apiErr.Param)
}

// The pair is checked as the customer will hold it, so a default carrier or service level sent alone is checked against the other one already held.
func TestUpdateCustomer_RefusesADefaultServiceLevelOffTheDefaultCarrier(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		params                   domain.UpdateCustomerParams
		carrierID, levelID       string
		checkLevel, checkCarrier string
	}{
		{"carrier alone", domain.UpdateCustomerParams{DefaultCarrierID: new("cr_new")}, "cr_new", "", "crop_own", "cr_new"},
		{"service level alone", domain.UpdateCustomerParams{DefaultServiceLevelID: field.Set("crop_other")}, "", "crop_other", "crop_other", "cr_own"},
		{"both", domain.UpdateCustomerParams{DefaultCarrierID: new("cr_new"), DefaultServiceLevelID: field.Set("crop_other")}, "cr_new", "crop_other", "crop_other", "cr_new"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newCustomerSvcSetup(t)
			s.expectWrite()
			s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).
				Return(&domain.Customer{ID: "ac_buyer", Number: "1001", DefaultCarrierID: new("cr_own"), DefaultServiceLevelID: new("crop_own")}, nil)
			s.expectRoutingOwned(tc.carrierID, tc.levelID)
			s.serviceLevels.EXPECT().IsInCarrier(gomock.Any(), tc.checkLevel, tc.checkCarrier).Return(false, nil)

			params := tc.params
			params.CustomerAccountID = "ac_buyer"
			_, apiErr := s.svc.UpdateCustomer(customerInternalCtx("ac_seller"), params)

			require.NotNil(t, apiErr)
			assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
			assert.Equal(t, "defaults.service_level_id", apiErr.Param)
		})
	}
}

// Clearing the default service level alongside a new default carrier leaves nothing to check, and an edit that leaves a pair alone does not re-check it.
func TestUpdateCustomer_TakesANewCarrierWithItsServiceLevelClearedAndLeavesAnUntouchedPair(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params domain.UpdateCustomerParams
		owned  string
	}{
		{"new carrier, level cleared", domain.UpdateCustomerParams{DefaultCarrierID: new("cr_new"), DefaultServiceLevelID: field.Clear[string]()}, "cr_new"},
		{"pair untouched", domain.UpdateCustomerParams{Instructions: field.Set("unrelated edit")}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newCustomerSvcSetup(t)
			s.expectWrite()
			// The held pair is already mismatched, so a re-check of it would refuse the edit.
			old := &domain.Customer{ID: "ac_buyer", Name: "Buyer Co", Number: "1001", DefaultCarrierID: new("cr_own"), DefaultServiceLevelID: new("crop_other")}
			s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(old, nil).Times(2)
			s.customers.EXPECT().GetRelationID(gomock.Any(), "ac_seller", "ac_buyer").Return("acre_buyer", nil)
			s.customers.EXPECT().Update(gomock.Any(), "acre_buyer", gomock.Any()).Return(nil)
			s.expectRoutingOwned(tc.owned, "")

			params := tc.params
			params.CustomerAccountID = "ac_buyer"
			_, apiErr := s.svc.UpdateCustomer(customerInternalCtx("ac_seller"), params)

			require.Nil(t, apiErr)
		})
	}
}

// Every account starts with its default site, so a customer's is made in the transaction that creates the customer:
// a create whose site can't be made is refused rather than leaving an account without one.
func TestCreateCustomer_GivesTheNewAccountItsDefaultSite(t *testing.T) {
	s := newCustomerSvcSetup(t)
	s.expectWrite()
	s.customers.EXPECT().LockNumbers(gomock.Any(), "ac_seller").Return(nil)
	s.expectRoutingOwned("cr_own", "")
	s.customers.EXPECT().ExistsByNumber(gomock.Any(), "ac_seller", "C-9", nil).Return(false, nil)
	s.addresses.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.Address{}, nil)

	var accountID string
	s.customers.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "C-9").
		DoAndReturn(func(_ context.Context, id, _, _ string, _ domain.CreateCustomerParams, _ string) (*domain.Customer, *apierror.APIError) {
			accountID = id
			return &domain.Customer{ID: id}, nil
		})
	siteErr := apierror.NewInternalError(errors.New("boom"), "site insert failed")
	s.sites.EXPECT().CreateDefaultSite(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) (*domain.Location, *apierror.APIError) {
			assert.Equal(t, accountID, id, "the site belongs to the new customer's account")
			return nil, siteErr
		})

	_, apiErr := s.svc.CreateCustomer(customerInternalCtx("ac_seller"), domain.CreateCustomerParams{
		Name:             "Buyer Co",
		Number:           new("C-9"),
		DefaultCarrierID: new("cr_own"),
		BillToAddress:    &domain.CreateAddressParams{Name: "HQ", Country: "US"},
		ShipToAddress:    &domain.CreateAddressParams{Name: "HQ", Country: "US"},
	})
	require.NotNil(t, apiErr)
	assert.Same(t, siteErr, apiErr)
	assert.NotEmpty(t, accountID)
}
