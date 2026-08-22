package service

// Covers the account-plan cap helpers in account_service.go and the invoice enforcement that
// consumes them from invoice_service.go.

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEnforceInvoicesPerPeriodLimit(t *testing.T) {
	const acct = "ac_test"
	max := func(n int32) *int32 { return &n }

	// build wires an account repo (context/plan/limits) and invoice count into a service.
	build := func(t *testing.T, isSandbox bool, planID *string, limit *int32, count int64) *shipmentSvcImpl {
		ctrl := gomock.NewController(t)
		accountRepo := repositorymock.NewMockAccountRepo(ctrl)
		accountRepo.EXPECT().GetAccountContext(gomock.Any(), acct).
			Return(&domain.AccountContext{AccountID: acct, IsSandbox: isSandbox}, nil).AnyTimes()
		if !isSandbox {
			accountRepo.EXPECT().GetPlanIDAndPeriodEnd(gomock.Any(), acct).Return(planID, nil, nil).AnyTimes()
			if planID != nil {
				accountRepo.EXPECT().ListPlanLimits(gomock.Any(), *planID).
					Return(map[string]*int32{string(constants.AccountPlanLimitInvoicesMaximum): limit}, nil).AnyTimes()
			}
		}
		invoiceRepo := repositorymock.NewMockInvoiceRepo(ctrl)
		invoiceRepo.EXPECT().CountSince(gomock.Any(), acct, gomock.Any()).Return(count, nil).AnyTimes()

		repoFactory := factorymock.NewMockRepoFactory(ctrl)
		repoFactory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
		repoFactory.EXPECT().NewInvoiceRepo().Return(invoiceRepo).AnyTimes()
		return &shipmentSvcImpl{repos: repoFactory}
	}

	plan := "acpl_test"

	t.Run("sandbox is exempt", func(t *testing.T) {
		assert.Nil(t, enforceInvoicesPerPeriodLimit(context.Background(), build(t, true, &plan, max(1), 99).repos, acct))
	})
	t.Run("no plan is exempt", func(t *testing.T) {
		assert.Nil(t, enforceInvoicesPerPeriodLimit(context.Background(), build(t, false, nil, nil, 99).repos, acct))
	})
	t.Run("unlimited (nil limit) is exempt", func(t *testing.T) {
		assert.Nil(t, enforceInvoicesPerPeriodLimit(context.Background(), build(t, false, &plan, nil, 99).repos, acct))
	})
	t.Run("under the limit passes", func(t *testing.T) {
		assert.Nil(t, enforceInvoicesPerPeriodLimit(context.Background(), build(t, false, &plan, max(10), 9).repos, acct))
	})
	t.Run("at the limit is rejected", func(t *testing.T) {
		apiErr := enforceInvoicesPerPeriodLimit(context.Background(), build(t, false, &plan, max(10), 10).repos, acct)
		assert.NotNil(t, apiErr)
		assert.Contains(t, apiErr.PublicMessage, "maximum of 10 invoices")
	})
}
