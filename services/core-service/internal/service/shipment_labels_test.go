package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
	testLabelShipmentID = "sh_labels"
	testLabelOrderID    = "so_labels"
	testLabelCaseID     = "shc_labels_1"
	testLabelCaseNumber = "1001"
	testShippoAccountID = "shippo_carrier_account"
)

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
	shippoClient    *clientmock.MockShippoClient

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
		shippoClient:    clientmock.NewMockShippoClient(ctrl),
	}

	h.repoFactory.EXPECT().NewCarrierRepo().Return(h.carrierRepo).AnyTimes()
	h.repoFactory.EXPECT().NewAccountIntegrationRepo().Return(h.integrationRepo).AnyTimes()
	h.repoFactory.EXPECT().NewShippingCaseRepo().Return(h.caseRepo).AnyTimes()
	h.repoFactory.EXPECT().NewShipmentRepo().Return(h.shipmentRepo).AnyTimes()
	h.repoFactory.EXPECT().NewSalesOrderRepo().Return(h.orderRepo).AnyTimes()
	h.repoFactory.EXPECT().NewSalesOrderLineRepo().Return(h.orderLineRepo).AnyTimes()
	h.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(h.idempotencyRepo).AnyTimes()
	h.repoFactory.EXPECT().NewAccountUserRepo().Return(h.accountUserRepo).AnyTimes()
	h.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("acu_test", nil).AnyTimes()

	shippoFactory := clientmock.NewMockShippoClientFactory(ctrl)
	shippoFactory.EXPECT().Build(gomock.Any()).Return(h.shippoClient).AnyTimes()

	serviceLevelToken := "ups_ground"
	h.shipment = &domain.Shipment{
		ID:                         testLabelShipmentID,
		AccountID:                  testLabelAccountID,
		SalesOrderID:               testLabelOrderID,
		CarrierID:                  "car_ups",
		ServiceLevelToken:          &serviceLevelToken,
		ShippingAddressStreetLine1: strPtr("185 Berry St"),
		ShippingAddressLocality:    strPtr("San Francisco"),
		ShippingAddressState:       strPtr("CA"),
		ShippingAddressPostalCode:  strPtr("94107"),
		ShippingAddressCountry:     strPtr("US"),
	}

	h.svc = &shipmentSvcImpl{
		repos:         h.repoFactory,
		shippoFactory: shippoFactory,
		encryptionKey: testEncryptionKey(),
	}

	return h
}

// Stubs the reads leading up to the carrier round-trip: a Shippo carrier, live credentials, one case and an origin.
func (h *labelHarness) expectPurchasePreconditions() {
	shippoAccount := testShippoAccountID
	h.carrierRepo.EXPECT().
		Get(gomock.Any(), domain.GetCarrierParams{AccountID: testLabelAccountID, CarrierID: "car_ups"}).
		Return(&domain.Carrier{ID: "car_ups", ShippoCarrierAccountID: &shippoAccount}, nil)

	h.expectShippoCredentials()

	h.caseRepo.EXPECT().ListByShipment(gomock.Any(), testLabelShipmentID).
		Return([]*domain.ShippingCase{{ID: testLabelCaseID, Number: testLabelCaseNumber, FreightWeightValue: "12.5"}}, nil)

	h.orderRepo.EXPECT().GetAccountOriginAddress(gomock.Any(), testLabelAccountID).
		Return(&domain.ShippingAddress{
			Name: "OpenMRP", Street1: "215 Clayton St", City: "San Francisco", State: "CA", Zip: "94117", Country: "US",
		}, nil)
}

func (h *labelHarness) expectShippoCredentials() {
	h.integrationRepo.EXPECT().HasIntegration(gomock.Any(), testLabelAccountID, gomock.Any()).Return(true, nil)
	h.integrationRepo.EXPECT().GetEncryptedCredentials(gomock.Any(), testLabelAccountID, gomock.Any()).
		Return(sealShippoCreds(h.t, testEncryptionKey(), testLabelAccountID, `{"api_key":"shippo_live_key"}`), true, nil)
}

// Stubs a whole successful purchase: the carrier call plus every write it produces.
func (h *labelHarness) expectSuccessfulPurchase(result *domain.LabelResult) {
	h.expectPurchasePreconditions()

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

func TestPurchaseShippingLabels_PersistsTrackingAndLabels(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: "https://shippo/label1.png", ShippoTransactionID: "txn_1"},
		},
	})

	got, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
	assert.Nil(t, got, "the master tracking is persisted here, not handed back to the caller")
}

func TestPurchaseShippingLabels_WritesBackNegotiatedRate(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectSuccessfulPurchase(&domain.LabelResult{
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

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
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

func TestPurchaseShippingLabels_UploadsLabelToBucket(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	gif := []byte("GIF89a-label-bytes")
	server, hits := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(gif)
	})

	store := &capturingObjectStore{}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)

	assert.Equal(t, 1, *hits)
	assert.Equal(t, 1, store.uploads)
	assert.Equal(t, "augno-shipping-labels", store.bucket)
	assert.Equal(t, "shipping-labels/"+testLabelAccountID+"/"+testLabelCaseNumber+".gif", store.key,
		"upload, void's delete and the label read must agree on one key")
	assert.Equal(t, "image/gif", store.contentType)
	assert.Equal(t, gif, store.body)
}

func TestPurchaseShippingLabels_FetchFailureStillShips(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, _ := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	store := &capturingObjectStore{}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr, "the label is already bought, so an unreachable label host must not fail the ship")
	assert.Zero(t, store.uploads)
}

func TestPurchaseShippingLabels_UploadFailureStillShips(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, _ := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GIF89a-label-bytes"))
	})

	store := &capturingObjectStore{uploadErr: apierror.NewInternalError(nil, "AccessDenied")}
	h.svc.s3Client = store
	h.svc.shippingLabelsBucket = "augno-shipping-labels"

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr, "the shippo-hosted url stays as the fallback when the bucket refuses the label")
	assert.Equal(t, 1, store.uploads)
}

func TestPurchaseShippingLabels_NoObjectStoreSkipsUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, hits := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GIF89a-label-bytes"))
	})

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
	assert.Zero(t, *hits, "an unconfigured bucket must not even reach out for the label")
}

func TestPurchaseShippingLabels_EmptyBucketSkipsUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	server, hits := labelServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GIF89a-label-bytes"))
	})

	store := &capturingObjectStore{}
	h.svc.s3Client = store

	h.expectSuccessfulPurchase(&domain.LabelResult{
		MasterTrackingNumber: "1Z-MASTER",
		Packages: []domain.LabelPackage{
			{TrackingNumber: "1Z-CASE-1", LabelURL: server.URL + "/label1.gif", ShippoTransactionID: "txn_1"},
		},
	})

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
	assert.Zero(t, *hits)
	assert.Zero(t, store.uploads)
}

func TestPurchaseShippingLabels_NonShippoCarrierBuysNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	existing := "MANUAL-TRACK"
	h.shipment.MasterTrackingNumber = &existing

	h.carrierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.Carrier{ID: "car_ltl"}, nil)

	got, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
	require.NotNil(t, got)
	assert.Equal(t, existing, *got)
}

func TestPurchaseShippingLabels_CarrierFailureFailsTheShip(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectPurchasePreconditions()
	h.shippoClient.EXPECT().CreateTransactionInstantLabel(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewValidationError("SHIPPO: Transaction status is ERROR - invalid address"))

	// No case, shipment or recovery-point write is expected: a shipment with no label is not shipped.
	got, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	assert.Nil(t, got)
	require.NotNil(t, apiErr)
}

func TestPurchaseShippingLabels_MissingOriginRefusesToBuy(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	shippoAccount := testShippoAccountID
	h.carrierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.Carrier{ID: "car_ups", ShippoCarrierAccountID: &shippoAccount}, nil)
	h.expectShippoCredentials()
	h.caseRepo.EXPECT().ListByShipment(gomock.Any(), testLabelShipmentID).
		Return([]*domain.ShippingCase{{ID: testLabelCaseID, Number: testLabelCaseNumber, FreightWeightValue: "12.5"}}, nil)
	h.orderRepo.EXPECT().GetAccountOriginAddress(gomock.Any(), testLabelAccountID).Return(nil, nil)

	_, apiErr := h.svc.purchaseShippingLabels(context.Background(), h.shipment, "idk_test")
	require.NotNil(t, apiErr)
}

// Proves a ship retried after a successful purchase resumes past it: the carrier, credential and
// Shippo mocks carry no expectations, so any second purchase attempt fails the test.
func TestShipShipment_ResumesAfterLabelsCreatedWithoutRebuying(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)
	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: string(domain.RecoveryPointShipLabelsCreated),
	}, nil)

	mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotencyMed}).AnyTimes()

	h.shipmentRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(h.shipment, nil)

	// The letterhead logo is fetched before the transaction opens; this account has none.
	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	h.repoFactory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()

	// Stop the atomic phase at its first write; reaching it at all proves the purchase was skipped.
	stopped := apierror.NewInternalError(nil, "stop after entering the atomic phase")
	h.caseRepo.EXPECT().MarkShippedByShipment(gomock.Any(), testLabelShipmentID).Return(stopped)
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), "idk_test", stopped).Return(stopped)

	svc := NewShipmentSvc(&ShipmentSvcConfig{
		Repos:           h.repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: h.repoFactory},
		ShippoFactory:   h.svc.shippoFactory,
		EncryptionKey:   testEncryptionKey(),
	})

	_, apiErr := svc.ShipShipment(shipmentShipCtx(testLabelAccountID), domain.ShipShipmentParams{ShipmentID: testLabelShipmentID})
	require.NotNil(t, apiErr)
}

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

func TestRefundShippingLabels_Sandbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetAccountContext(gomock.Any(), testLabelAccountID).
		Return(&domain.AccountContext{AccountID: testLabelAccountID, IsSandbox: true}, nil)
	h.repoFactory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()

	// No case listing, no refund, no recovery-point advance: sandbox never bought a label.
	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
}

func TestRefundShippingLabels_RefundsEachTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectVoidAccountContext(ctrl, false)
	h.expectVoidCases()

	h.expectShippoCredentials()
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_1").Return(nil)
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_2").Return(nil)

	h.idempotencyRepo.EXPECT().
		AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.RecoveryPointVoidLabelsRefunded).
		Return(nil)

	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr)
}

func TestRefundShippingLabels_RefundFailureStillVoids(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := newLabelHarness(t, ctrl)

	h.expectVoidAccountContext(ctrl, false)
	h.expectVoidCases()

	h.expectShippoCredentials()
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_1").
		Return(apierror.NewValidationError("SHIPPO: Refund failed for transaction txn_1."))
	h.shippoClient.EXPECT().RefundTransaction(gomock.Any(), "txn_2").Return(nil)

	h.idempotencyRepo.EXPECT().
		AdvanceRecoveryPoint(gomock.Any(), "idk_test", domain.RecoveryPointVoidLabelsRefunded).
		Return(nil)

	apiErr := h.svc.refundShippingLabels(context.Background(), h.shipment, "idk_test")
	require.Nil(t, apiErr, "a carrier that refuses a refund must not strand the shipment in shipped state")
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

	h.expectVoidAccountContext(ctrl, false)
	h.expectVoidCases()
	h.expectShippoCredentials()
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

func (h *labelHarness) expectVoidAccountContext(ctrl *gomock.Controller, isSandbox bool) {
	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetAccountContext(gomock.Any(), testLabelAccountID).
		Return(&domain.AccountContext{AccountID: testLabelAccountID, IsSandbox: isSandbox}, nil)
	h.repoFactory.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
}

func (h *labelHarness) expectVoidCases() {
	txn1, txn2 := "txn_1", "txn_2"
	h.caseRepo.EXPECT().ListByShipment(gomock.Any(), testLabelShipmentID).
		Return([]*domain.ShippingCase{
			{ID: "shc_1", Number: "1001", ShippoTransactionID: &txn1},
			{ID: "shc_2", Number: "1002", ShippoTransactionID: &txn2},
		}, nil)
}
