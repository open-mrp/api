package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testLabelAccountID  = "ac_labels"
	testLabelCustomerID = "ac_labels_buyer"
	testLabelShipmentID = "sh_labels"
	testLabelOrderID    = "so_labels"
	testLabelCaseID     = "shc_labels_1"
	testLabelCaseNumber = "1001"
	testShippoAccountID = "shippo_carrier_account"
)

// Holds task leases in memory, standing in for the task_leases table the dispatch claim uses.
type memLeases struct {
	mu      sync.Mutex
	holders map[string]string
}

func newMemLeases() *memLeases { return &memLeases{holders: map[string]string{}} }

func (m *memLeases) Acquire(_ context.Context, name, holder string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, held := m.holders[name]; held && current != holder {
		return false, nil
	}
	m.holders[name] = holder
	return true, nil
}

func (m *memLeases) Renew(_ context.Context, name, holder string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.holders[name] == holder, nil
}

func (m *memLeases) Release(_ context.Context, name, holder string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holders[name] == holder {
		delete(m.holders, name)
	}
	return nil
}

func (m *memLeases) held(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.holders[name]
	return ok
}

// Wires the repos, Shippo client and shipment a label purchase or refund touches.
type labelHarness struct {
	t   *testing.T
	svc *shipmentSvcImpl

	repoFactory     *factorymock.MockRepoFactory
	carrierRepo     *repositorymock.MockCarrierRepo
	integrationRepo *repositorymock.MockAccountIntegrationRepo
	caseRepo        *repositorymock.MockShippingCaseRepo
	shipmentRepo    *repositorymock.MockShipmentRepo
	orderRepo       *repositorymock.MockSalesOrderRepo
	orderLineRepo   *repositorymock.MockSalesOrderLineRepo
	idempotencyRepo *repositorymock.MockIdempotencyKeyRepo
	accountUserRepo *repositorymock.MockAccountUserRepo
	accountRepo     *repositorymock.MockAccountRepo
	invoiceRepo     *repositorymock.MockInvoiceRepo
	serviceLevels   *repositorymock.MockServiceLevelRepo
	shippoClient    *clientmock.MockShippoClient
	leases          *memLeases

	shipment *domain.Shipment
}

func newLabelHarness(t *testing.T, ctrl *gomock.Controller) *labelHarness {
	t.Helper()

	h := &labelHarness{
		t:               t,
		repoFactory:     factorymock.NewMockRepoFactory(ctrl),
		carrierRepo:     repositorymock.NewMockCarrierRepo(ctrl),
		integrationRepo: repositorymock.NewMockAccountIntegrationRepo(ctrl),
		caseRepo:        repositorymock.NewMockShippingCaseRepo(ctrl),
		shipmentRepo:    repositorymock.NewMockShipmentRepo(ctrl),
		orderRepo:       repositorymock.NewMockSalesOrderRepo(ctrl),
		orderLineRepo:   repositorymock.NewMockSalesOrderLineRepo(ctrl),
		idempotencyRepo: repositorymock.NewMockIdempotencyKeyRepo(ctrl),
		accountUserRepo: repositorymock.NewMockAccountUserRepo(ctrl),
		accountRepo:     repositorymock.NewMockAccountRepo(ctrl),
		invoiceRepo:     repositorymock.NewMockInvoiceRepo(ctrl),
		serviceLevels:   repositorymock.NewMockServiceLevelRepo(ctrl),
		shippoClient:    clientmock.NewMockShippoClient(ctrl),
		leases:          newMemLeases(),
	}

	h.repoFactory.EXPECT().NewCarrierRepo().Return(h.carrierRepo).AnyTimes()
	h.repoFactory.EXPECT().NewAccountIntegrationRepo().Return(h.integrationRepo).AnyTimes()
	h.repoFactory.EXPECT().NewShippingCaseRepo().Return(h.caseRepo).AnyTimes()
	h.repoFactory.EXPECT().NewShipmentRepo().Return(h.shipmentRepo).AnyTimes()
	h.repoFactory.EXPECT().NewSalesOrderRepo().Return(h.orderRepo).AnyTimes()
	h.repoFactory.EXPECT().NewSalesOrderLineRepo().Return(h.orderLineRepo).AnyTimes()
	h.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(h.idempotencyRepo).AnyTimes()
	h.repoFactory.EXPECT().NewAccountUserRepo().Return(h.accountUserRepo).AnyTimes()
	h.repoFactory.EXPECT().NewAccountRepo().Return(h.accountRepo).AnyTimes()
	h.repoFactory.EXPECT().NewInvoiceRepo().Return(h.invoiceRepo).AnyTimes()
	h.repoFactory.EXPECT().NewServiceLevelRepo().Return(h.serviceLevels).AnyTimes()
	h.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()
	h.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("acu_test", nil).AnyTimes()

	shippoFactory := clientmock.NewMockShippoClientFactory(ctrl)
	shippoFactory.EXPECT().Build(gomock.Any()).Return(h.shippoClient, nil).AnyTimes()

	serviceLevelToken := "ups_ground"
	h.shipment = &domain.Shipment{
		ID:                         testLabelShipmentID,
		Number:                     "SO-100",
		AccountID:                  testLabelAccountID,
		CustomerID:                 testLabelCustomerID,
		SalesOrderID:               testLabelOrderID,
		SalesOrderNumber:           "SO-100",
		StatusCode:                 "packed",
		CarrierID:                  "car_ups",
		ServiceLevelToken:          &serviceLevelToken,
		ShippingAddressName:        strPtr("Buyer Dock"),
		ShippingAddressStreetLine1: strPtr("185 Berry St"),
		ShippingAddressLocality:    strPtr("San Francisco"),
		ShippingAddressState:       strPtr("CA"),
		ShippingAddressPostalCode:  strPtr("94107"),
		ShippingAddressCountry:     strPtr("US"),
	}

	h.svc = &shipmentSvcImpl{
		repos:          h.repoFactory,
		txManager:      &stubTxManager{factory: h.repoFactory},
		shippoFactory:  shippoFactory,
		encryptionKey:  testEncryptionKey(),
		dispatchLeases: h.leases,
	}

	return h
}

func (h *labelHarness) expectAccountContext(isSandbox bool) {
	h.accountRepo.EXPECT().GetAccountContext(gomock.Any(), testLabelAccountID).
		Return(&domain.AccountContext{AccountID: testLabelAccountID, IsSandbox: isSandbox}, nil).AnyTimes()
}

func (h *labelHarness) expectShippoCarrier() {
	shippoAccount := testShippoAccountID
	h.carrierRepo.EXPECT().
		Get(gomock.Any(), domain.GetCarrierParams{AccountID: testLabelAccountID, CarrierID: "car_ups"}).
		Return(&domain.Carrier{ID: "car_ups", ShippoCarrierAccountID: &shippoAccount}, nil)
}

func (h *labelHarness) expectShippoCredentials(isActive bool) {
	h.integrationRepo.EXPECT().HasIntegration(gomock.Any(), testLabelAccountID, gomock.Any()).Return(true, nil)
	h.integrationRepo.EXPECT().GetEncryptedCredentials(gomock.Any(), testLabelAccountID, gomock.Any()).
		Return(sealShippoCreds(h.t, testEncryptionKey(), testLabelAccountID, `{"api_key":"shippo_test_key"}`), isActive, nil)
}

func (h *labelHarness) expectCases(cases ...*domain.ShippingCase) {
	h.caseRepo.EXPECT().ListByShipment(gomock.Any(), testLabelShipmentID).Return(cases, nil)
}

func (h *labelHarness) expectOrigin(origin *domain.ShippingAddress) {
	h.orderRepo.EXPECT().GetAccountOriginAddress(gomock.Any(), testLabelAccountID).Return(origin, nil)
}

func (h *labelHarness) expectBrandingPhones(sellerPhone, buyerPhone *string) {
	h.accountRepo.EXPECT().GetByIDs(gomock.Any(), []string{testLabelAccountID, testLabelCustomerID}).
		Return([]*domain.Account{
			{ID: testLabelAccountID, Branding: &domain.AccountBranding{PhoneNumber: sellerPhone}},
			{ID: testLabelCustomerID, Branding: &domain.AccountBranding{PhoneNumber: buyerPhone}},
		}, nil)
}

func weighedCase() *domain.ShippingCase {
	return &domain.ShippingCase{ID: testLabelCaseID, Number: testLabelCaseNumber, FreightWeightValue: "12.5"}
}

func testOrigin() *domain.ShippingAddress {
	return &domain.ShippingAddress{
		Name: "OpenMRP", Street1: "215 Clayton St", City: "San Francisco", State: "CA", Zip: "94117", Country: "US",
	}
}

// Stubs every read a purchase plan makes up to the point it is ready to buy.
func (h *labelHarness) expectPlannablePurchase() {
	h.expectAccountContext(false)
	h.expectShippoCarrier()
	h.expectCases(weighedCase())
	h.expectShippoCredentials(true)
	h.expectOrigin(testOrigin())
	h.expectBrandingPhones(strPtr("555-0100"), strPtr("555-0199"))
}

// --- planning: every check runs before the carrier is called ---

func TestPlanShipmentLabels_SandboxUsesPlaceholderTracking(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(true)

	// The carrier, case and Shippo mocks carry no expectations: a sandbox ship must never reach Shippo.
	purchase, labels, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.Nil(t, apiErr)
	assert.Nil(t, purchase)
	require.NotNil(t, labels.MasterTracking)
	assert.Equal(t, sandboxTrackingNumber(testLabelShipmentID), *labels.MasterTracking)
	assert.False(t, labels.CostRecorded, "a sandbox ship records a zero freight cost, as legacy did")
}

func TestPlanShipmentLabels_NonShippoCarrierBuysNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(false)
	h.carrierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Carrier{ID: "car_ltl"}, nil)

	purchase, labels, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.Nil(t, apiErr)
	assert.Nil(t, purchase)
	assert.Nil(t, labels.MasterTracking, "the shipment keeps whatever tracking it was given")
	assert.False(t, labels.CostRecorded)
}

func TestPlanShipmentLabels_BuildsTheLabelRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.shipment.CustomerPONumber = strPtr("PO-77")
	h.expectPlannablePurchase()

	purchase, _, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.Nil(t, apiErr)
	require.NotNil(t, purchase)

	params := purchase.params
	assert.Equal(t, testShippoAccountID, params.CarrierAccountObjectID)
	assert.Equal(t, "ups_ground", params.ServiceLevelToken)
	assert.Equal(t, testLabelShipmentID, params.Metadata, "the purchase names its shipment for reconciliation")
	assert.Equal(t, "555-0199", *params.ToAddress.Phone, "the recipient is reached at the buyer's own number first")
	assert.Equal(t, "555-0100", *params.FromAddress.Phone, "the sender falls back to the seller's number")
	assert.Equal(t, "Buyer Dock", *params.ToAddress.Company, "the address name prints as the company")
	assert.Equal(t, "OpenMRP", *params.FromAddress.Company)

	require.Len(t, params.Parcels, 1)
	assert.Equal(t, "PO#PO-77", params.Parcels[0].Reference1)
	assert.Equal(t, "SO#SO-100", params.Parcels[0].Reference2)
	assert.Equal(t, testLabelCaseID, params.Parcels[0].Metadata)
	assert.Nil(t, params.Billing)
}

func TestLabelParcels_ReferenceTheCaseWithoutAPO(t *testing.T) {
	shipment := &domain.Shipment{SalesOrderNumber: "SO-100"}
	parcels := labelParcels(shipment, []*domain.ShippingCase{weighedCase()})

	require.Len(t, parcels, 1)
	assert.Equal(t, "SO#SO-100", parcels[0].Reference1)
	assert.Equal(t, "C#"+testLabelCaseNumber, parcels[0].Reference2)
}

func TestLabelPhones_FallBackInLegacyOrder(t *testing.T) {
	tests := []struct {
		name     string
		seller   *string
		buyer    *string
		shipTo   *string
		origin   *string
		wantTo   string
		wantFrom string
	}{
		{name: "buyer first", seller: strPtr("1"), buyer: strPtr("2"), shipTo: strPtr("3"), origin: strPtr("4"), wantTo: "2", wantFrom: "4"},
		{name: "ship-to when the buyer has none", seller: strPtr("1"), shipTo: strPtr("3"), wantTo: "3", wantFrom: "1"},
		{name: "seller when neither has one", seller: strPtr("1"), buyer: strPtr(" "), wantTo: "1", wantFrom: "1"},
		{name: "placeholder when nobody has one", wantTo: labelPhonePlaceholder, wantFrom: labelPhonePlaceholder},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			h := newLabelHarness(t, ctrl)
			h.shipment.ShippingAddressPhone = tc.shipTo
			h.expectBrandingPhones(tc.seller, tc.buyer)

			to, from, apiErr := h.svc.labelPhones(context.Background(), h.shipment, domain.ShippingAddress{Phone: tc.origin})
			require.Nil(t, apiErr)
			assert.Equal(t, tc.wantTo, to)
			assert.Equal(t, tc.wantFrom, from)
		})
	}
}

func TestPlanShipmentLabels_RefusesAnUnweighedCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(false)
	h.expectShippoCarrier()
	h.expectCases(weighedCase(), &domain.ShippingCase{ID: "shc_2", Number: "1002", FreightWeightValue: "0"})
	h.expectShippoCredentials(true)

	purchase, _, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.NotNil(t, apiErr, "a case with no weight cannot be rated, so its label is refused before buying")
	assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
	assert.Nil(t, purchase)
}

func TestPlanShipmentLabels_MissingOriginRefusesToBuy(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(false)
	h.expectShippoCarrier()
	h.expectCases(weighedCase())
	h.expectShippoCredentials(true)
	h.expectOrigin(nil)

	purchase, _, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.NotNil(t, apiErr)
	assert.Nil(t, purchase)
}

func TestPlanShipmentLabels_InactiveIntegrationIsRefused(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(false)
	h.expectShippoCarrier()
	h.expectCases(weighedCase())
	h.expectShippoCredentials(false)

	_, _, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.NotNil(t, apiErr)
	assert.Equal(t, "Shippo integration is inactive.", apiErr.PublicMessage)
}

func TestPlanShipmentLabels_ReusesLabelsAlreadyBought(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.shipment.MasterTrackingNumber = strPtr("1Z-MASTER")
	h.expectAccountContext(false)
	h.expectShippoCarrier()
	bought := weighedCase()
	bought.ShippoTransactionID = strPtr("txn_1")
	bought.TrackingNumber = strPtr("1Z-CASE-1")
	h.expectCases(bought)

	// No credential or Shippo expectations: a second purchase fails the test.
	purchase, labels, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.Nil(t, apiErr)
	assert.Nil(t, purchase, "labels already bought must never be bought again")
	require.NotNil(t, labels.MasterTracking)
	assert.Equal(t, "1Z-MASTER", *labels.MasterTracking)
	assert.True(t, labels.CostRecorded, "the earlier purchase recorded the carrier's charge")
}

func TestPlanShipmentLabels_RefusesPartlyBoughtLabels(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(false)
	h.expectShippoCarrier()
	bought := weighedCase()
	bought.ShippoTransactionID = strPtr("txn_1")
	h.expectCases(bought, &domain.ShippingCase{ID: "shc_2", Number: "1002", FreightWeightValue: "4"})

	purchase, _, apiErr := h.svc.planShipmentLabels(context.Background(), h.shipment)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)
	assert.Nil(t, purchase)
}

func TestShipmentThirdPartyBilling_UsesTheOrdersOwnBilling(t *testing.T) {
	shipment := &domain.Shipment{
		OrderCarrierBillingType:    strPtr("third_party"),
		OrderCarrierBillingAccount: strPtr("ACCT-9"),
		BillingAddressCountry:      strPtr("US"),
		BillingAddressZip:          strPtr("43215"),
	}
	billing := shipmentThirdPartyBilling(shipment)
	require.NotNil(t, billing)
	assert.Equal(t, domain.ShippingBilling{Type: "THIRD_PARTY", Account: "ACCT-9", Country: "US", Zip: "43215"}, *billing,
		"the third party bills against the order's billing address, not the seller's origin")

	// The customer's default billing shows on the shipment but does not bill a label the order did not name.
	inherited := &domain.Shipment{CarrierBillingType: strPtr("third_party"), CarrierBillingAccount: strPtr("ACCT-9")}
	assert.Nil(t, shipmentThirdPartyBilling(inherited))
}

// --- buying: the purchase is recorded before anything else can fail ---

func testPurchase(h *labelHarness) *labelPurchase {
	return &labelPurchase{
		client: h.shippoClient,
		cases:  []*domain.ShippingCase{weighedCase()},
		params: domain.CreateLabelParams{Metadata: testLabelShipmentID, Parcels: []domain.Parcel{{Weight: "12.5"}}},
	}
}

// Stubs a whole successful purchase: the carrier call plus every write it produces.
func (h *labelHarness) expectRecordedPurchase(result *domain.LabelResult) {
	h.shippoClient.EXPECT().CreateTransactionInstantLabel(gomock.Any(), gomock.Any()).Return(result, nil)

	pkg := result.Packages[0]
	h.caseRepo.EXPECT().
		UpdateWithShipmentInfo(gomock.Any(), testLabelCaseID, pkg.TrackingNumber, pkg.ShippoTransactionID, pkg.LabelURL).
		Return(nil)

	if result.MasterTrackingNumber != "" {
		h.shipmentRepo.EXPECT().
			SetMasterTracking(gomock.Any(), testLabelAccountID, testLabelShipmentID, result.MasterTrackingNumber).
			Return(nil)
	}

	if result.NegotiatedRate <= 0 {
		// A zero rate still writes back, so a stale freight cost clears rather than lingering.
		h.orderRepo.EXPECT().GetLines(gomock.Any(), testLabelOrderID).Return([]*domain.SalesOrderLine{
			{ID: "sol_freight", ProductTypeCode: strPtr("shipping")},
		}, nil)
		h.orderLineRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, nil)

		h.idempotencyRepo.EXPECT().
			AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.RecoveryPointShipLabelsCreated).
			Return(nil)
	}
}

func TestBuyShipmentLabels_RecordsTrackingAndLabels(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: "https://shippo/label1.png", ShippoTransactionID: "txn_1"},
		},
	})

	labels, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr)
	assert.Nil(t, labels.MasterTracking, "the master tracking is recorded with the purchase, not handed back")
	assert.True(t, labels.CostRecorded)
}

func TestBuyShipmentLabels_OutlivesTheCallersDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	h.shippoClient.EXPECT().CreateTransactionInstantLabel(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ domain.CreateLabelParams) (*domain.LabelResult, *apierror.APIError) {
			assert.NoError(t, ctx.Err(), "a caller that gave up must not cut the purchase off")
			return nil, apierror.NewValidationError("SHIPPO: stop here")
		})

	_, apiErr := h.svc.buyShipmentLabels(ctx, h.shipment, testPurchase(h), "idk_test")
	require.NotNil(t, apiErr)
}

func TestBuyShipmentLabels_WritesBackNegotiatedRate(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		NegotiatedRate:       18.456,
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: "https://shippo/label1.png", ShippoTransactionID: "txn_1"},
		},
	})

	h.orderRepo.EXPECT().GetLines(gomock.Any(), testLabelOrderID).Return([]*domain.SalesOrderLine{
		{ID: "sol_product", ProductTypeCode: strPtr("sale")},
		{
			ID:                         "sol_freight",
			ProductTypeCode:            strPtr("shipping"),
			UnitPriceNumeratorUnitID:   "unt_usd",
			UnitPriceDenominatorUnitID: "unt_each",
		},
	}, nil)

	h.orderLineRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			assert.Equal(t, "sol_freight", params.SalesOrderLineID)
			require.NotNil(t, params.UnitCostValue)
			assert.Equal(t, "18.46", *params.UnitCostValue, "the carrier's negotiated rate lands on the freight line cost, rounded to cents")
			require.NotNil(t, params.UnitCostNumeratorUnitID)
			assert.Equal(t, "unt_usd", *params.UnitCostNumeratorUnitID)
			require.NotNil(t, params.UnitCostDenominatorUnitID)
			assert.Equal(t, "unt_each", *params.UnitCostDenominatorUnitID)
			return nil, nil
		})

	h.idempotencyRepo.EXPECT().
		AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.RecoveryPointShipLabelsCreated).
		Return(nil)

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr)
}

func TestBuyShipmentLabels_UnrecordablePurchaseIsNotRetried(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.shippoClient.EXPECT().CreateTransactionInstantLabel(gomock.Any(), gomock.Any()).Return(&domain.LabelResult{
		Packages: []domain.LabelPackage{{TrackingNumber: "1Z-CASE-1", ShippoTransactionID: "txn_1"}},
	}, nil)
	// The database keeps failing: each attempt rolls back at its first write.
	h.caseRepo.EXPECT().UpdateWithShipmentInfo(gomock.Any(), testLabelCaseID, gomock.Any(), "txn_1", gomock.Any()).
		Return(apierror.NewInternalError(nil, "connection reset")).Times(recordLabelAttempts)

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.NotNil(t, apiErr)
	assert.False(t, apiErr.IsTransient, "a retry would find no recorded label and buy it again, so the error must stick to the key")
}

func TestBuyShipmentLabels_CarrierFailureRecordsNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.shippoClient.EXPECT().CreateTransactionInstantLabel(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewValidationError("SHIPPO: Transaction status is ERROR - invalid address"))

	// No case, shipment or recovery-point write is expected: a shipment with no label is not shipped.
	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.NotNil(t, apiErr)
}

// Stands in for the carrier's label host, counting hits so a test can prove the fetch was skipped.
func labelServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *int) {
	t.Helper()

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	return server, &hits
}

func TestBuyShipmentLabels_UploadsLabelToBucket(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	gif := []byte("GIF89a-label-bytes")
	server, hits := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(gif)
	})

	store := &capturingObjectStore{}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr)

	assert.Equal(t, 1, *hits)
	assert.Equal(t, 1, store.uploads)
	assert.Equal(t, "augno-shipping-labels", store.bucket)
	assert.Equal(t, "shipping-labels/"+testLabelAccountID+"/"+testLabelCaseNumber+".gif", store.key,
		"upload, void's delete and the label read must agree on one key")
	assert.Equal(t, "image/gif", store.contentType)
	assert.Equal(t, gif, store.body)
}

func TestBuyShipmentLabels_FetchFailureStillShips(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, _ := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	store := &capturingObjectStore{}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr, "the label is already bought, so an unreachable label host must not fail the ship")
	assert.Zero(t, store.uploads)
}

func TestBuyShipmentLabels_UploadFailureStillShips(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, _ := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GIF89a-label-bytes"))
	})

	store := &capturingObjectStore{uploadErr: apierror.NewInternalError(nil, "AccessDenied")}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr, "the shippo-hosted url stays as the fallback when the bucket refuses the label")
	assert.Equal(t, 1, store.uploads)
}

func TestBuyShipmentLabels_NoObjectStoreSkipsUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, hits := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GIF89a-label-bytes"))
	})

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr)
	assert.Zero(t, *hits, "an unconfigured bucket must not even reach out for the label")
}

func TestBuyShipmentLabels_EmptyBucketSkipsUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, hits := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GIF89a-label-bytes"))
	})

	store := &capturingObjectStore{}
	h.svc.s3Client = store

	h.expectRecordedPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.buyShipmentLabels(context.Background(), h.shipment, testPurchase(h), "idk_test")
	require.Nil(t, apiErr)
	assert.Zero(t, *hits)
	assert.Zero(t, store.uploads)
}

// --- the ship action: claim, checks, purchase, atomic phase ---

func shipmentShipCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			Permissions:  map[string]bool{"shipments:update": true},
		},
	})
}

// Builds the shipment service over the harness, with a mocked idempotency mediator.
func (h *labelHarness) shipService(ctrl *gomock.Controller) (domain.ShipmentSvc, *mediatormock.MockIdempotencyMed) {
	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)
	mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotencyMed}).AnyTimes()

	return NewShipmentSvc(&ShipmentSvcConfig{
		Repos:           h.repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: h.repoFactory},
		ShippoFactory:   h.svc.shippoFactory,
		EncryptionKey:   testEncryptionKey(),
		DispatchLeases:  h.leases,
	}), idempotencyMed
}

// Proves a ship retried after a successful purchase resumes past it: the carrier, credential and
// Shippo mocks carry no expectations, so any second purchase attempt fails the test.
func TestShipShipment_ResumesAfterLabelsCreatedWithoutRebuying(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	svc, idempotencyMed := h.shipService(ctrl)

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: string(domain.RecoveryPointShipLabelsCreated),
	}, nil)

	h.shipmentRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(h.shipment, nil)
	h.invoiceRepo.EXPECT().IsDuplicateNumber(gomock.Any(), testLabelAccountID, h.shipment.Number).Return(false, nil)
	// The letterhead logo is fetched before the transaction opens; this account has none.
	h.accountRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	// Stop the atomic phase at its first write; reaching it at all proves the purchase was skipped.
	stopped := apierror.NewInternalError(nil, "stop after entering the atomic phase")
	h.shipmentRepo.EXPECT().MarkShipped(gomock.Any(), testLabelAccountID, testLabelShipmentID, "acu_test").Return(stopped)
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), "idk_test", stopped).Return(stopped)

	_, apiErr := svc.ShipShipment(shipmentShipCtx(testLabelAccountID), domain.ShipShipmentParams{ShipmentID: testLabelShipmentID})
	require.NotNil(t, apiErr)
	assert.False(t, h.leases.held("shipment-dispatch:"+testLabelShipmentID), "the claim is released when the ship ends")
}

// A duplicate invoice number would fail the ship's transaction after the labels were already paid
// for, so it is refused before the carrier is ever called, and left uncached so it can be fixed.
func TestShipShipment_DuplicateInvoiceNumberIsRefusedBeforeBuying(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	svc, idempotencyMed := h.shipService(ctrl)

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: string(domain.RecoveryPointStarted),
	}, nil)
	h.shipmentRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(h.shipment, nil)
	h.expectAccountContext(false)
	h.accountRepo.EXPECT().GetPlanIDAndPeriodEnd(gomock.Any(), testLabelAccountID).Return(nil, nil, nil)
	h.invoiceRepo.EXPECT().IsDuplicateNumber(gomock.Any(), testLabelAccountID, h.shipment.Number).Return(true, nil)

	_, apiErr := svc.ShipShipment(shipmentShipCtx(testLabelAccountID), domain.ShipShipmentParams{ShipmentID: testLabelShipmentID})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)
}

// Two ships of one shipment at once: the first holds the claim while it buys, so the second is
// refused without reaching the carrier, and the carrier is called exactly once.
func TestShipShipment_ConcurrentShipIsRefusedWhileTheFirstBuys(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	svc, idempotencyMed := h.shipService(ctrl)

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{
		TypeID:        "idk_first",
		RecoveryPoint: string(domain.RecoveryPointStarted),
	}, nil)
	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{
		TypeID:        "idk_second",
		RecoveryPoint: string(domain.RecoveryPointStarted),
	}, nil)

	h.shipmentRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(h.shipment, nil)
	h.accountRepo.EXPECT().GetPlanIDAndPeriodEnd(gomock.Any(), testLabelAccountID).Return(nil, nil, nil)
	h.invoiceRepo.EXPECT().IsDuplicateNumber(gomock.Any(), testLabelAccountID, h.shipment.Number).Return(false, nil)
	h.expectPlannablePurchase()

	buying := make(chan struct{})
	finish := make(chan struct{})
	carrierRefusal := apierror.NewValidationError("SHIPPO: Transaction status is ERROR - stop here")
	h.shippoClient.EXPECT().CreateTransactionInstantLabel(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.CreateLabelParams) (*domain.LabelResult, *apierror.APIError) {
			close(buying)
			<-finish
			return nil, carrierRefusal
		}).Times(1)
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), "idk_first", carrierRefusal).Return(carrierRefusal)

	var firstErr *apierror.APIError
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, firstErr = svc.ShipShipment(shipmentShipCtx(testLabelAccountID), domain.ShipShipmentParams{ShipmentID: testLabelShipmentID})
	}()

	<-buying
	_, secondErr := svc.ShipShipment(shipmentShipCtx(testLabelAccountID), domain.ShipShipmentParams{ShipmentID: testLabelShipmentID})
	require.NotNil(t, secondErr)
	assert.Equal(t, apierror.ErrorCodeResourceConflict, secondErr.Code)

	close(finish)
	<-done
	require.NotNil(t, firstErr)
	assert.False(t, h.leases.held("shipment-dispatch:"+testLabelShipmentID))
}

func TestClaimDispatch_HoldsTheShipmentUntilReleased(t *testing.T) {
	svc := &shipmentSvcImpl{dispatchLeases: newMemLeases()}

	release, apiErr := svc.claimDispatch(context.Background(), testLabelShipmentID)
	require.Nil(t, apiErr)

	_, apiErr = svc.claimDispatch(context.Background(), testLabelShipmentID)
	require.NotNil(t, apiErr, "each attempt holds its own claim, so even a replay of the same request conflicts")
	assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)

	release()
	release, apiErr = svc.claimDispatch(context.Background(), testLabelShipmentID)
	require.Nil(t, apiErr)
	release()
}

func TestDeleteShipment_RefusesAShippedShipment(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	svc, _ := h.shipService(ctrl)

	h.shipment.StatusCode = "shipped"
	h.shipmentRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(h.shipment, nil)

	apiErr := svc.DeleteShipment(shipmentShipCtx(testLabelAccountID), domain.DeleteShipmentParams{ShipmentID: testLabelShipmentID})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceConflict, apiErr.Code)
}

// --- void: refunds ---

func TestRefundShippingLabels_Sandbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)
	h.expectAccountContext(true)

	// No case listing, no refund, no recovery-point advance: sandbox never bought a label.
	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
}

func TestRefundShippingLabels_RefundsEachTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectAccountContext(false)
	h.expectVoidCases()

	h.expectShippoCredentials(true)
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_1").Return(nil)
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_2").Return(nil)

	h.idempotencyRepo.EXPECT().
		AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.RecoveryPointVoidLabelsRefunded).
		Return(nil)

	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
}

// Clearing a case whose refund the carrier refused would erase the only record of a label that is
// still charged, so the void stops there — no recovery point, no deleted label — and can be retried.
func TestRefundShippingLabels_RefundFailureAbortsTheVoid(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	store := &labelDeleteRecorder{}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectAccountContext(false)
	h.expectVoidCases()

	h.expectShippoCredentials(true)
	refusal := apierror.NewValidationError("SHIPPO: Refund failed for transaction txn_1.")
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_1").Return(refusal)

	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Equal(t, refusal, apiErr)
	assert.Empty(t, store.deleted, "the stored label stays while the void is refused")
}

func TestRefundShippingLabels_NoIntegrationLeftToRefundAbortsTheVoid(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectAccountContext(false)
	h.expectVoidCases()
	h.integrationRepo.EXPECT().HasIntegration(gomock.Any(), testLabelAccountID, gomock.Any()).Return(false, nil)

	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.NotNil(t, apiErr)
}

// Records the label objects a void deletes, so the test can assert on the keys that were dropped.
type labelDeleteRecorder struct {
	capturingObjectStore
	deleted []string
	err     *apierror.APIError
}

func (r *labelDeleteRecorder) Delete(_ context.Context, bucket, key string) *apierror.APIError {
	r.deleted = append(r.deleted, bucket+"/"+key)
	return r.err
}

func TestRefundShippingLabels_DeletesStoredLabels(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	store := &labelDeleteRecorder{}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectAccountContext(false)
	h.expectVoidCases()
	h.expectShippoCredentials(true)
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	h.idempotencyRepo.EXPECT().
		AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.RecoveryPointVoidLabelsRefunded).
		Return(nil)

	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
	assert.Equal(t, []string{
		"augno-shipping-labels/shipping-labels/" + testLabelAccountID + "/1001.gif",
		"augno-shipping-labels/shipping-labels/" + testLabelAccountID + "/1002.gif",
	}, store.deleted)
}

func (h *labelHarness) expectVoidCases() {
	txn1, txn2 := "txn_1", "txn_2"
	h.caseRepo.EXPECT().ListByShipment(gomock.Any(), testLabelShipmentID).
		Return([]*domain.ShippingCase{
			{ID: "shc_1", Number: "1001", ShippoTransactionID: &txn1},
			{ID: "shc_2", Number: "1002", ShippoTransactionID: &txn2},
		}, nil)
}
