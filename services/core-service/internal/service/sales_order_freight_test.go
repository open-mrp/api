package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testFreightAccountID = "ac_freight"
	testFreightOrderID   = "so_freight"
)

type freightHarness struct {
	svc       *salesOrderSvcImpl
	orderRepo *repositorymock.MockSalesOrderRepo
}

// Only the sales order repo is reachable: any product, carrier or Shippo call fails the test.
func newFreightHarness(ctrl *gomock.Controller) *freightHarness {
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	orderRepo := repositorymock.NewMockSalesOrderRepo(ctrl)
	repoFactory.EXPECT().NewSalesOrderRepo().Return(orderRepo).AnyTimes()

	return &freightHarness{
		svc:       &salesOrderSvcImpl{repos: repoFactory, txManager: &stubTxManager{factory: repoFactory}},
		orderRepo: orderRepo,
	}
}

func freightLines() []*domain.SalesOrderLine {
	shipping := systemProductCodeShipping
	return []*domain.SalesOrderLine{
		{ID: "sol_product"},
		{ID: "sol_freight", ProductTypeCode: &shipping},
	}
}

func TestFinishPendingFreight_ExistingFreightLineSettlesWithoutQuoting(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newFreightHarness(ctrl)

	gomock.InOrder(
		h.orderRepo.EXPECT().IsFreightPending(gomock.Any(), testFreightAccountID, testFreightOrderID, false).Return(true, nil),
		h.orderRepo.EXPECT().GetLines(gomock.Any(), testFreightOrderID).Return(freightLines(), nil),
		h.orderRepo.EXPECT().IsFreightPending(gomock.Any(), testFreightAccountID, testFreightOrderID, true).Return(true, nil),
		h.orderRepo.EXPECT().ClearFreightPending(gomock.Any(), testFreightAccountID, testFreightOrderID).Return(nil),
	)

	require.Nil(t, h.svc.FinishPendingFreight(context.Background(), testFreightAccountID, testFreightOrderID))
}

func TestFinishPendingFreight_ConcurrentlyClearedFlagWritesNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newFreightHarness(ctrl)

	gomock.InOrder(
		h.orderRepo.EXPECT().IsFreightPending(gomock.Any(), testFreightAccountID, testFreightOrderID, false).Return(true, nil),
		h.orderRepo.EXPECT().GetLines(gomock.Any(), testFreightOrderID).Return(freightLines(), nil),
		h.orderRepo.EXPECT().IsFreightPending(gomock.Any(), testFreightAccountID, testFreightOrderID, true).Return(false, nil),
	)

	require.Nil(t, h.svc.FinishPendingFreight(context.Background(), testFreightAccountID, testFreightOrderID))
}

func TestFinishPendingFreight_NotPendingIsANoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newFreightHarness(ctrl)
	h.orderRepo.EXPECT().IsFreightPending(gomock.Any(), testFreightAccountID, testFreightOrderID, false).Return(false, nil)

	require.Nil(t, h.svc.FinishPendingFreight(context.Background(), testFreightAccountID, testFreightOrderID))
}
