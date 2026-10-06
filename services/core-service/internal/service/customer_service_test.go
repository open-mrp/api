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
	"github.com/open-mrp/api/shared/field"
)

type customerSvcSetup struct {
	svc         domain.CustomerSvc
	customers   *repositorymock.MockCustomerRepo
	salesOrders *repositorymock.MockSalesOrderRepo
	deleted     *repositorymock.MockDeletedRecordRepo
	idempotency *mediatormock.MockIdempotencyMed
	outbox      *recordingOutboxRepo
}

func newCustomerSvcSetup(t *testing.T) *customerSvcSetup {
	ctrl := gomock.NewController(t)
	s := &customerSvcSetup{
		customers:   repositorymock.NewMockCustomerRepo(ctrl),
		salesOrders: repositorymock.NewMockSalesOrderRepo(ctrl),
		deleted:     repositorymock.NewMockDeletedRecordRepo(ctrl),
		idempotency: mediatormock.NewMockIdempotencyMed(ctrl),
		outbox:      &recordingOutboxRepo{},
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCustomerRepo().Return(s.customers).AnyTimes()
	repos.EXPECT().NewSalesOrderRepo().Return(s.salesOrders).AnyTimes()
	repos.EXPECT().NewDeletedRecordRepo().Return(s.deleted).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()
	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: s.idempotency}).AnyTimes()
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
		Note:              field.Set("unrelated edit"),
	})

	require.Nil(t, apiErr)
	got, ok := written.ReceiveCalendarID.Value()
	require.True(t, ok, "the calendar is written back, not cleared")
	assert.Equal(t, calendarID, got)
}

// The deleted-customer snapshot names its owner, and only the owner is told the customer was deleted.
func TestDeleteCustomer_OnlyTheOwnerLearnsACustomerWasDeleted(t *testing.T) {
	notFound := apierror.NewResourceNotFoundError("Customer not found.")

	t.Run("records the snapshot under the owner", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		customer := &domain.Customer{ID: "ac_buyer"}
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(customer, nil)
		s.salesOrders.EXPECT().CountSalesOrdersForBuyerAccounts(gomock.Any(), "ac_seller", []string{"ac_buyer"}).Return(int64(0), nil)
		s.deleted.EXPECT().CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, "ac_buyer", "ac_seller", customer).Return(nil)
		s.customers.EXPECT().Delete(gomock.Any(), "ac_seller", "ac_buyer").Return(nil)

		require.Nil(t, s.svc.DeleteCustomer(customerInternalCtx("ac_seller"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"}))
	})

	t.Run("the owner deleting it again is told it is gone", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", "ac_buyer", gomock.Any()).Return(nil, notFound)
		s.deleted.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, "ac_buyer", "ac_seller").Return(true, nil)

		apiErr := s.svc.DeleteCustomer(customerInternalCtx("ac_seller"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeResourceGone, apiErr.Code)
	})

	t.Run("another tenant is told it was never there", func(t *testing.T) {
		s := newCustomerSvcSetup(t)
		s.customers.EXPECT().Get(gomock.Any(), "ac_other", "ac_buyer", gomock.Any()).Return(nil, notFound)
		s.deleted.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, "ac_buyer", "ac_other").Return(false, nil)

		apiErr := s.svc.DeleteCustomer(customerInternalCtx("ac_other"), domain.DeleteCustomerParams{CustomerAccountID: "ac_buyer"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	})
}

func TestBulkDeleteCustomers_RecordsEachSnapshotUnderTheOwner(t *testing.T) {
	s := newCustomerSvcSetup(t)
	for _, customerID := range []string{"ac_a", "ac_b"} {
		customer := &domain.Customer{ID: customerID}
		s.customers.EXPECT().Get(gomock.Any(), "ac_seller", customerID, gomock.Any()).Return(customer, nil)
		s.deleted.EXPECT().CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeCustomer, customerID, "ac_seller", customer).Return(nil)
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
