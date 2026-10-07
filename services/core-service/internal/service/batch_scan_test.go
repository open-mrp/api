package service

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
)

// The expected strings are what the dashboard's scanning service printed for the same float64, so the
// material check reads the same after the move to Go. They include the cases a naive port gets wrong:
// exact binary ties (1.03125), values whose decimal looks like a tie but whose float is not (7.00005),
// and results that round to zero.
func TestFormatQuantityMatchesTheDashboard(t *testing.T) {
	t.Parallel()

	tenth, fifth := 0.1, 0.2
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{12, "12"},
		{-12, "-12"},
		{12.5, "12.5"},
		{1.03125, "1.0313"},
		{-1.03125, "-1.0313"},
		{0.09375, "0.0938"},
		{1.00005, "1.0001"},
		{7.00005, "7"},
		{33.333333333333336, "33.3333"},
		{2.0 / 3, "0.6667"},
		{-2.0 / 3, "-0.6667"},
		{0.00004, "0"},
		{-0.00004, "0"},
		{0.00005, "0.0001"},
		{99999.99995, "99999.9999"},
		{tenth + fifth, "0.3"},
		{1234567890.12345, "1234567890.1235"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{math.NaN(), "NaN"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatQuantity(c.in), "formatQuantity(%v)", c.in)
	}
}

func TestEnforceBatchScansPerPeriodLimit(t *testing.T) {
	const acct = "ac_test"
	max := func(n int32) *int32 { return &n }
	plan := "acpl_test"

	repos := func(t *testing.T, isSandbox bool, planID *string, limit *int32, count int64) domain.RepoFactory {
		ctrl := gomock.NewController(t)
		accountRepo := repositorymock.NewMockAccountRepo(ctrl)
		accountRepo.EXPECT().GetAccountContext(gomock.Any(), acct).
			Return(&domain.AccountContext{AccountID: acct, IsSandbox: isSandbox}, nil).AnyTimes()
		accountRepo.EXPECT().GetPlanIDAndPeriodEnd(gomock.Any(), acct).Return(planID, nil, nil).AnyTimes()
		if planID != nil {
			accountRepo.EXPECT().ListPlanLimits(gomock.Any(), *planID).
				Return(map[string]*int32{string(constants.AccountPlanLimitBatchesMaximum): limit}, nil).AnyTimes()
		}
		batchRepo := repositorymock.NewMockBatchRepo(ctrl)
		batchRepo.EXPECT().CountScannedSince(gomock.Any(), acct, gomock.Any()).Return(count, nil).AnyTimes()

		factory := factorymock.NewMockRepoFactory(ctrl)
		factory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
		factory.EXPECT().NewBatchRepo().Return(batchRepo).AnyTimes()
		return factory
	}

	t.Run("sandbox is exempt", func(t *testing.T) {
		assert.Nil(t, enforceBatchScansPerPeriodLimit(context.Background(), repos(t, true, &plan, max(1), 99), acct))
	})
	t.Run("no plan is exempt", func(t *testing.T) {
		assert.Nil(t, enforceBatchScansPerPeriodLimit(context.Background(), repos(t, false, nil, nil, 99), acct))
	})
	t.Run("unlimited is exempt", func(t *testing.T) {
		assert.Nil(t, enforceBatchScansPerPeriodLimit(context.Background(), repos(t, false, &plan, nil, 99), acct))
	})
	t.Run("under the limit passes", func(t *testing.T) {
		assert.Nil(t, enforceBatchScansPerPeriodLimit(context.Background(), repos(t, false, &plan, max(10), 9), acct))
	})
	t.Run("at the limit is rejected with the dashboard's message", func(t *testing.T) {
		apiErr := enforceBatchScansPerPeriodLimit(context.Background(), repos(t, false, &plan, max(10), 10), acct)
		require.NotNil(t, apiErr)
		assert.Equal(t, "Your plan allows a maximum of 10 batch scans per billing period", apiErr.PublicMessage)
	})
	t.Run("a limit of one is singular", func(t *testing.T) {
		apiErr := enforceBatchScansPerPeriodLimit(context.Background(), repos(t, false, &plan, max(1), 1), acct)
		require.NotNil(t, apiErr)
		assert.Equal(t, "Your plan allows a maximum of 1 batch scan per billing period", apiErr.PublicMessage)
	})
}

func TestMeterBatchesSkipsSandboxes(t *testing.T) {
	const acct = "ac_test"
	svc := func(t *testing.T, isSandbox bool, withPublisher bool) *batchSvcImpl {
		ctrl := gomock.NewController(t)
		accountRepo := repositorymock.NewMockAccountRepo(ctrl)
		accountRepo.EXPECT().GetAccountContext(gomock.Any(), acct).
			Return(&domain.AccountContext{AccountID: acct, IsSandbox: isSandbox}, nil).AnyTimes()
		factory := factorymock.NewMockRepoFactory(ctrl)
		factory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
		s := &batchSvcImpl{repos: factory}
		if withPublisher {
			s.billingPub = publishermock.NewMockBillingPublisher(ctrl)
		}
		return s
	}

	meter, apiErr := svc(t, false, true).meterBatches(context.Background(), acct)
	require.Nil(t, apiErr)
	assert.True(t, meter, "a live account's created batches are metered")

	meter, apiErr = svc(t, true, true).meterBatches(context.Background(), acct)
	require.Nil(t, apiErr)
	assert.False(t, meter, "a sandbox is not billed")

	meter, apiErr = svc(t, false, false).meterBatches(context.Background(), acct)
	require.Nil(t, apiErr)
	assert.False(t, meter, "nothing is metered without a publisher")
}
