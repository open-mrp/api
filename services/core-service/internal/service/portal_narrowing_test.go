package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
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

func (h *inclHarness) customers() (domain.CustomerSvc, *repositorymock.MockCustomerRepo) {
	repo := repositorymock.NewMockCustomerRepo(h.ctrl)
	h.repos.EXPECT().NewCustomerRepo().Return(repo).AnyTimes()
	return NewCustomerSvc(&CustomerSvcConfig{Repos: h.repos, MediatorFactory: h.meds, JobSvcFactory: factorymock.NewMockJobSvcFactory(h.ctrl), TxManager: &stubTxManager{factory: h.repos}}), repo
}

// The repo mock has no expectation for another account's record, so reading one fails the test.
func TestPortalNarrowing_CustomerRecordIsThePortalsOwn(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		svc, repo := h.customers()

		_, apiErr := svc.GetCustomer(portal.ctx(false), inclOtherID, nil)
		requireNotFound(t, apiErr, name)

		_, apiErr = svc.GetFrequentlyOrderedProducts(portal.ctx(false), inclOtherID)
		requireNotFound(t, apiErr, name)

		repo.EXPECT().Get(gomock.Any(), inclSellerID, portal.own, gomock.Any()).Return(&domain.Customer{}, nil)
		_, apiErr = svc.GetCustomer(portal.ctx(false), portal.own, nil)
		require.Nil(t, apiErr, name)

		repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, []string{portal.own}).Return(nil, nil)
		_, apiErr = svc.BatchGetCustomers(portal.ctx(false), []string{inclOtherID, portal.own})
		require.Nil(t, apiErr, name)
	}
}

func (h *inclHarness) shipments() (domain.ShipmentSvc, domain.ShipmentLineSvc, *repositorymock.MockShipmentRepo, *repositorymock.MockShipmentLineRepo) {
	shipmentRepo := repositorymock.NewMockShipmentRepo(h.ctrl)
	lineRepo := repositorymock.NewMockShipmentLineRepo(h.ctrl)
	h.repos.EXPECT().NewShipmentRepo().Return(shipmentRepo).AnyTimes()
	h.repos.EXPECT().NewShipmentLineRepo().Return(lineRepo).AnyTimes()
	shipments := NewShipmentSvc(&ShipmentSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}, DispatchLeases: newMemLeases()})
	lines := NewShipmentLineSvc(&ShipmentLineSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}})
	return shipments, lines, shipmentRepo, lineRepo
}

func TestPortalNarrowing_AnotherBuyersShipmentIsNotFound(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		shipments, lines, shipmentRepo, _ := h.shipments()
		shipmentRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Shipment{ID: "shp_other", CustomerID: inclOtherID}, nil).AnyTimes()

		_, apiErr := shipments.GetShipment(portal.ctx(false), domain.GetShipmentParams{ShipmentID: "shp_other"})
		requireNotFound(t, apiErr, name+": the shipment")

		_, apiErr = lines.ListShipmentLines(portal.ctx(false), domain.ListShipmentLinesParams{ShipmentID: "shp_other", Limit: 10})
		requireNotFound(t, apiErr, name+": its lines")

		_, apiErr = lines.GetShipmentLine(portal.ctx(false), inclSellerID, "shp_other", "shln_1")
		requireNotFound(t, apiErr, name+": one of its lines")
	}
}

func TestPortalNarrowing_ThePortalsOwnShipmentLinesAreRead(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		_, lines, shipmentRepo, lineRepo := h.shipments()
		shipmentRepo.EXPECT().Get(gomock.Any(), domain.GetShipmentParams{AccountID: inclSellerID, ShipmentID: "shp_own"}).Return(&domain.Shipment{ID: "shp_own", CustomerID: portal.own}, nil)
		lineRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(&domain.ListShipmentLinesResult{}, nil)

		_, apiErr := lines.ListShipmentLines(portal.ctx(false), domain.ListShipmentLinesParams{ShipmentID: "shp_own", Limit: 10})
		require.Nil(t, apiErr, name)
	}
}

// Lines an authorized request includes, and the seller's staff, only need the shipment to be in the account.
func TestPortalNarrowing_IncludedAndStaffShipmentLinesNeedOnlyTheAccount(t *testing.T) {
	t.Parallel()

	for name, ctx := range map[string]context.Context{
		"customer portal include": inclPortal(true),
		"staff":                   inclStaff(false, "shipments:read"),
	} {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		_, lines, shipmentRepo, lineRepo := h.shipments()
		shipmentRepo.EXPECT().IsInAccount(gomock.Any(), inclSellerID, "shp_other").Return(true, nil)
		lineRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(&domain.ListShipmentLinesResult{}, nil)

		_, apiErr := lines.ListShipmentLines(ctx, domain.ListShipmentLinesParams{ShipmentID: "shp_other", Limit: 10})
		require.Nil(t, apiErr, name)
	}
}

func (h *inclHarness) accountPrices() (domain.AccountPriceSvc, *repositorymock.MockAccountPriceRepo) {
	repo := repositorymock.NewMockAccountPriceRepo(h.ctrl)
	h.repos.EXPECT().NewAccountPriceRepo().Return(repo).AnyTimes()
	return NewAccountPriceSvc(&AccountPriceSvcConfig{Repos: h.repos, MediatorFactory: h.meds, JobSvcFactory: NewJobSvcFactory(), TxManager: &stubTxManager{factory: h.repos}}), repo
}

func TestPortalNarrowing_AccountPricesAreThePortalsOwn(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		svc, repo := h.accountPrices()
		repo.EXPECT().ResolveRecipientAccountIDs(gomock.Any(), inclSellerID, portal.own).Return([]string{portal.own}, nil).Times(2)
		repo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, params domain.ListAccountPricesParams) (*domain.ListAccountPricesResult, *apierror.APIError) {
			assert.Equal(t, []string{portal.own}, params.RecipientAccountIDs, name)
			return &domain.ListAccountPricesResult{}, nil
		})

		_, apiErr := svc.ListAccountPrices(portal.ctx(false), domain.ListAccountPricesParams{Limit: 10, RecipientAccountIDs: []string{inclOtherID}})
		require.Nil(t, apiErr, name)

		repo.EXPECT().Get(gomock.Any(), inclSellerID, "acpr_other").Return(&domain.AccountPrice{ID: "acpr_other", RecipientAccountID: inclOtherID}, nil)
		_, apiErr = svc.GetAccountPrice(portal.ctx(false), "acpr_other")
		requireNotFound(t, apiErr, name)
	}
}

func (h *inclHarness) volumeDiscounts() (domain.VolumeDiscountSvc, *repositorymock.MockVolumeDiscountRepo) {
	repo := repositorymock.NewMockVolumeDiscountRepo(h.ctrl)
	h.repos.EXPECT().NewVolumeDiscountRepo().Return(repo).AnyTimes()
	return NewVolumeDiscountSvc(&VolumeDiscountSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}}), repo
}

func TestPortalNarrowing_VolumeDiscountsAreTheOnesThePortalsListingCarries(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		h.allowCounterpartyReads()
		svc, repo := h.volumeDiscounts()

		repo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, params domain.ListVolumeDiscountsParams) (*domain.ListVolumeDiscountsResult, *apierror.APIError) {
			require.NotNil(t, params.CustomerAccountID, name)
			assert.Equal(t, portal.own, *params.CustomerAccountID, name)
			return &domain.ListVolumeDiscountsResult{}, nil
		})
		_, apiErr := svc.ListVolumeDiscounts(portal.ctx(false), domain.ListVolumeDiscountsParams{Limit: 10})
		require.Nil(t, apiErr, name)

		repo.EXPECT().AppliesToCustomer(gomock.Any(), inclSellerID, portal.own, "qudi_other_group").Return(false, nil)
		_, apiErr = svc.GetVolumeDiscount(portal.ctx(false), domain.GetVolumeDiscountParams{VolumeDiscountID: "qudi_other_group"})
		requireNotFound(t, apiErr, name)

		repo.EXPECT().AppliesToCustomer(gomock.Any(), inclSellerID, portal.own, "qudi_everyone").Return(true, nil)
		repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.VolumeDiscount{ID: "qudi_everyone"}, nil)
		got, apiErr := svc.GetVolumeDiscount(portal.ctx(false), domain.GetVolumeDiscountParams{VolumeDiscountID: "qudi_everyone"})
		require.Nil(t, apiErr, name)
		assert.Equal(t, "qudi_everyone", got.ID, name)
	}
}

func TestPortalNarrowing_StaffOpenAnyVolumeDiscount(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	svc, repo := h.volumeDiscounts()
	repo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.VolumeDiscount{ID: "qudi_other_group"}, nil)

	_, apiErr := svc.GetVolumeDiscount(inclStaff(false, "discounts:read"), domain.GetVolumeDiscountParams{VolumeDiscountID: "qudi_other_group"})
	require.Nil(t, apiErr)
}

func (h *inclHarness) jobs() (domain.JobSvc, *repositorymock.MockJobRepo) {
	repo := repositorymock.NewMockJobRepo(h.ctrl)
	accountUsers := repositorymock.NewMockAccountUserRepo(h.ctrl)
	accountUsers.EXPECT().ResolveAccountUserID(gomock.Any(), inclSellerID, gomock.Any()).Return("", apierror.NewResourceNotFoundError("Account user not found.")).AnyTimes()
	h.repos.EXPECT().NewJobRepo().Return(repo).AnyTimes()
	h.repos.EXPECT().NewAccountUserRepo().Return(accountUsers).AnyTimes()
	return NewJobSvc(&JobSvcConfig{Repos: h.repos}), repo
}

// A portal's job is attributed to its own actor; the seller's staff raise the rest.
func TestPortalNarrowing_JobsAreTheOnesThePortalRaised(t *testing.T) {
	t.Parallel()

	for name, portal := range portalCases() {
		h := newInclHarness(t)
		svc, repo := h.jobs()
		staffJob := &domain.Job{ID: "jb_staff", CreatedByID: new("acus_staff")}
		systemJob := &domain.Job{ID: "jb_system"}
		ownJob := &domain.Job{ID: "jb_own", CreatedByID: new("us_actor")}
		for _, job := range []*domain.Job{staffJob, systemJob, ownJob} {
			repo.EXPECT().Get(gomock.Any(), job.ID, inclSellerID).Return(job, nil)
		}

		for _, id := range []string{"jb_staff", "jb_system"} {
			_, apiErr := svc.GetJob(portal.ctx(false), id)
			requireNotFound(t, apiErr, name+": "+id)
			assert.Equal(t, "Resource not found.", apiErr.PublicMessage, "%s: reads exactly as a missing job", name)
		}

		got, apiErr := svc.GetJob(portal.ctx(false), "jb_own")
		require.Nil(t, apiErr, name)
		assert.Equal(t, "jb_own", got.ID, name)
	}
}

func TestPortalNarrowing_StaffReadEveryJobInTheAccount(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	svc, repo := h.jobs()
	repo.EXPECT().Get(gomock.Any(), "jb_other", inclSellerID).Return(&domain.Job{ID: "jb_other", CreatedByID: new("acus_someone_else")}, nil)

	got, apiErr := svc.GetJob(inclStaff(false, "jobs:read"), "jb_other")
	require.Nil(t, apiErr)
	assert.Equal(t, "jb_other", got.ID)
}
