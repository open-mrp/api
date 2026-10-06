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
	deleted     *repositorymock.MockDeletedRecordRepo
	idempotency *mediatormock.MockIdempotencyMed
	outbox      *recordingOutboxRepo
}

func newCustomerSvcSetup(t *testing.T) *customerSvcSetup {
	ctrl := gomock.NewController(t)
	s := &customerSvcSetup{
		customers:   repositorymock.NewMockCustomerRepo(ctrl),
		deleted:     repositorymock.NewMockDeletedRecordRepo(ctrl),
		idempotency: mediatormock.NewMockIdempotencyMed(ctrl),
		outbox:      &recordingOutboxRepo{},
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCustomerRepo().Return(s.customers).AnyTimes()
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
