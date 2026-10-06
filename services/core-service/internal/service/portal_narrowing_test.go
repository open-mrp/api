package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A customer or supplier portal reads the seller's account only for what is its own, and a supplier relation opens nothing a customer relation does not.

const inclSupplierID = "ac_supplier"

type portalCase struct {
	ctx func(included bool) context.Context
	own string
}

func portalCases() map[string]portalCase {
	return map[string]portalCase{
		"customer portal": {ctx: inclPortal, own: inclBuyerID},
		"supplier portal": {ctx: func(included bool) context.Context {
			return inclCtx(types.IdentityRelationTypeSupplier, inclSupplierID, included)
		}, own: inclSupplierID},
	}
}

func (h *inclHarness) allowCounterpartyReads() {
	h.readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), gomock.Any(), inclSellerID).Return(nil).AnyTimes()
}

func requireNotFound(t *testing.T, apiErr *apierror.APIError, msgAndArgs ...any) {
	t.Helper()
	require.NotNil(t, apiErr, msgAndArgs...)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code, msgAndArgs...)
}

func (h *inclHarness) salesOrders() (domain.SalesOrderSvc, *repositorymock.MockSalesOrderRepo) {
	repo := repositorymock.NewMockSalesOrderRepo(h.ctrl)
	h.repos.EXPECT().NewSalesOrderRepo().Return(repo).AnyTimes()
	return NewSalesOrderSvc(&SalesOrderSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}}), repo
}

func TestPortalNarrowing_SalesOrderListIsWhatThePortalBought(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		svc, repo := h.salesOrders()
		repo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, params domain.ListSalesOrdersParams) (*domain.ListSalesOrdersResult, *apierror.APIError) {
			assert.Equal(t, inclSellerID, params.AccountID, name)
			require.NotNil(t, params.BuyerAccountID, name)
			assert.Equal(t, portal.own, *params.BuyerAccountID, name)
			return &domain.ListSalesOrdersResult{}, nil
		})

		other := inclOtherID
		_, apiErr := svc.ListSalesOrders(portal.ctx(false), domain.ListSalesOrdersParams{Limit: 10, BuyerAccountID: &other})
		require.Nil(t, apiErr, name)
	}
}

func TestPortalNarrowing_SalesOrderRetrieveIsScopedToTheBuyer(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		svc, repo := h.salesOrders()
		repo.EXPECT().GetForCustomer(gomock.Any(), inclSellerID, portal.own, "so_other").Return(nil, apierror.NewResourceNotFoundError("Sales order not found."))

		_, apiErr := svc.GetSalesOrder(portal.ctx(false), domain.GetSalesOrderParams{SalesOrderID: "so_other"})
		requireNotFound(t, apiErr, name)
	}
}

// Loading an order an authorized request includes reads along the relation; the portal's own batch reads keep the buyer filter.
func TestPortalNarrowing_SalesOrderBatchKeepsTheBuyerFilterOutsideIncludes(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		svc, repo := h.salesOrders()

		own := portal.own
		repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, &own, []string{"so_1"}).Return(nil, nil)
		_, apiErr := svc.BatchGetSalesOrders(portal.ctx(false), []string{"so_1"}, nil)
		require.Nil(t, apiErr, name)

		repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, (*string)(nil), []string{"so_1"}).Return(nil, nil)
		_, apiErr = svc.BatchGetSalesOrders(portal.ctx(true), []string{"so_1"}, nil)
		require.Nil(t, apiErr, name)
	}
}
