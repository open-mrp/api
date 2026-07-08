package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	clientmock "github.com/augno/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	publishermock "github.com/augno/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// SalesOrderSvcTestSuite unit-tests sales_order_service.go with every collaborator
// mocked. Exercises business logic only — no real DB, HTTP, or network access.
//
// Lifecycle note: tests use SetupTest (not SetupSuite) so each test gets a fresh
// gomock controller; this keeps expectations from leaking across the ~30 test
// methods here and makes failures easier to localize.
type SalesOrderSvcTestSuite struct {
	suite.Suite
	svc domain.SalesOrderSvc

	// Repos (everything the service touches).
	accountRepo              *repositorymock.MockAccountRepo
	accountUserRepo          *repositorymock.MockAccountUserRepo
	accountIntegrationRepo   *repositorymock.MockAccountIntegrationRepo
	addressRepo              *repositorymock.MockAddressRepo
	batchRepo                *repositorymock.MockBatchRepo
	customerRepo             *repositorymock.MockCustomerRepo
	deletedRecordRepo        *repositorymock.MockDeletedRecordRepo
	invoiceRepo              *repositorymock.MockInvoiceRepo
	inventoryReservationRepo *repositorymock.MockInventoryReservationRepo
	materialDemandRepo       *repositorymock.MockMaterialDemandRepo
	orderDiscountRepo        *repositorymock.MockOrderDiscountRepo
	orderRepo                *repositorymock.MockSalesOrderRepo
	lineRepo                 *repositorymock.MockSalesOrderLineRepo
	pricingRepo              *repositorymock.MockPricingRepo
	pickRepo                 *repositorymock.MockPickRepo
	productRepo              *repositorymock.MockProductRepo
	productionRunQueryRepo   *repositorymock.MockProductionRunQueryRepo
	territoryRepo            *repositorymock.MockTerritoryRepo
	unitRepo                 *repositorymock.MockUnitRepo
	carrierRepo              *repositorymock.MockCarrierRepo
	serviceLevelRepo         *repositorymock.MockServiceLevelRepo
	shippingTermRepo         *repositorymock.MockShippingTermRepo
	paymentTermRepo          *repositorymock.MockPaymentTermRepo
	transactionRepo          *repositorymock.MockTransactionRepo
	opiRepo                  *repositorymock.MockOrderPaymentIntentRepo

	// Collaborators.
	checkoutFactory *clientmock.MockStripeCheckoutClientFactory
	checkoutClient  *clientmock.MockStripeCheckoutClient
	notifier        *publishermock.MockNotificationPublisher

	// Factory + mediators.
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	readAccessMed   *mediatormock.MockReadAccessMed

	ctrl          *gomock.Controller
	encryptionKey []byte
}

func (suite *SalesOrderSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.accountRepo = repositorymock.NewMockAccountRepo(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.accountIntegrationRepo = repositorymock.NewMockAccountIntegrationRepo(suite.ctrl)
	suite.addressRepo = repositorymock.NewMockAddressRepo(suite.ctrl)
	suite.batchRepo = repositorymock.NewMockBatchRepo(suite.ctrl)
	suite.customerRepo = repositorymock.NewMockCustomerRepo(suite.ctrl)
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)
	suite.invoiceRepo = repositorymock.NewMockInvoiceRepo(suite.ctrl)
	suite.inventoryReservationRepo = repositorymock.NewMockInventoryReservationRepo(suite.ctrl)
	suite.materialDemandRepo = repositorymock.NewMockMaterialDemandRepo(suite.ctrl)
	suite.orderDiscountRepo = repositorymock.NewMockOrderDiscountRepo(suite.ctrl)
	suite.orderRepo = repositorymock.NewMockSalesOrderRepo(suite.ctrl)
	suite.lineRepo = repositorymock.NewMockSalesOrderLineRepo(suite.ctrl)
	suite.pricingRepo = repositorymock.NewMockPricingRepo(suite.ctrl)
	suite.pickRepo = repositorymock.NewMockPickRepo(suite.ctrl)
	suite.productRepo = repositorymock.NewMockProductRepo(suite.ctrl)
	suite.productionRunQueryRepo = repositorymock.NewMockProductionRunQueryRepo(suite.ctrl)
	suite.territoryRepo = repositorymock.NewMockTerritoryRepo(suite.ctrl)
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	suite.carrierRepo = repositorymock.NewMockCarrierRepo(suite.ctrl)
	suite.serviceLevelRepo = repositorymock.NewMockServiceLevelRepo(suite.ctrl)
	suite.shippingTermRepo = repositorymock.NewMockShippingTermRepo(suite.ctrl)
	suite.paymentTermRepo = repositorymock.NewMockPaymentTermRepo(suite.ctrl)
	suite.transactionRepo = repositorymock.NewMockTransactionRepo(suite.ctrl)
	suite.opiRepo = repositorymock.NewMockOrderPaymentIntentRepo(suite.ctrl)

	suite.checkoutFactory = clientmock.NewMockStripeCheckoutClientFactory(suite.ctrl)
	suite.checkoutClient = clientmock.NewMockStripeCheckoutClient(suite.ctrl)
	suite.notifier = publishermock.NewMockNotificationPublisher(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAccountRepo().Return(suite.accountRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountIntegrationRepo().Return(suite.accountIntegrationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAddressRepo().Return(suite.addressRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewBatchRepo().Return(suite.batchRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewCustomerRepo().Return(suite.customerRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(suite.deletedRecordRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewInvoiceRepo().Return(suite.invoiceRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewInventoryReservationRepo().Return(suite.inventoryReservationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewMaterialDemandRepo().Return(suite.materialDemandRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOrderDiscountRepo().Return(suite.orderDiscountRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewSalesOrderRepo().Return(suite.orderRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPricingRepo().Return(suite.pricingRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewSalesOrderLineRepo().Return(suite.lineRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPickRepo().Return(suite.pickRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewProductRepo().Return(suite.productRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewProductionRunQueryRepo().Return(suite.productionRunQueryRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewTerritoryRepo().Return(suite.territoryRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewCarrierRepo().Return(suite.carrierRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewServiceLevelRepo().Return(suite.serviceLevelRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewShippingTermRepo().Return(suite.shippingTermRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPaymentTermRepo().Return(suite.paymentTermRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()
	suite.repoFactory.EXPECT().NewTransactionRepo().Return(suite.transactionRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOrderPaymentIntentRepo().Return(suite.opiRepo).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.readAccessMed = mediatormock.NewMockReadAccessMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
		ReadAccess:  suite.readAccessMed,
	}).AnyTimes()

	// Deterministic 32-byte key so the Checkout test can round-trip encrypted creds.
	suite.encryptionKey = make([]byte, 32)
	for i := range suite.encryptionKey {
		suite.encryptionKey[i] = byte(i + 1)
	}

	suite.svc = NewSalesOrderSvc(&SalesOrderSvcConfig{
		Repos:                 suite.repoFactory,
		MediatorFactory:       suite.mediatorFactory,
		TxManager:             &stubTxManager{factory: suite.repoFactory},
		CheckoutClientFactory: suite.checkoutFactory,
		NotificationPublisher: suite.notifier,
		EncryptionKey:         suite.encryptionKey,
		FrontendURL:           "https://dash.test",
	})
}

func (suite *SalesOrderSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestSalesOrderSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SalesOrderSvcTestSuite))
}

// --- Helpers ---

func salesOrderInternalCtx(accountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"sales_orders:read":      true,
				"sales_orders:create":    true,
				"sales_orders:update":    true,
				"sales_orders:delete":    true,
				"production_runs:create": true,
			},
		},
	})
}

func salesOrderCustomerCtx(targetAccountID, customerAccountID string) context.Context {
	return salesOrderCustomerCtxWithPerms(targetAccountID, customerAccountID, map[string]bool{})
}

// salesOrderCustomerCtxWithPerms builds a customer relation-actor identity that
// carries its own-account permissions (mirrors the post-redesign auth mediator,
// which no longer strips a counterparty-side customer's permissions).
func salesOrderCustomerCtxWithPerms(targetAccountID, customerAccountID string, perms map[string]bool) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_customer",
			AccountID:    &customerAccountID,
			Permissions:  perms,
		},
	})
}

func salesOrderIdempotencyCtx(ctx context.Context, handler string) context.Context {
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithHandler(ctx, handler)
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func (suite *SalesOrderSvcTestSuite) expectIdempotencyStarted() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

func (suite *SalesOrderSvcTestSuite) expectCacheSuccess() {
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).
		Return(nil).
		Times(1)
}

func (suite *SalesOrderSvcTestSuite) expectCacheError() {
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

// expectReadAccessAllowed stubs ReadAccessMed.CheckCounterpartyReadAccess to allow access.
// Sales-order portal endpoints use the counterparty check because customers read
// data on the vendor's account (the relation row is stored vendor→customer).
func (suite *SalesOrderSvcTestSuite) expectReadAccessAllowed(actorAccountID, targetAccountID string) {
	suite.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), actorAccountID, targetAccountID).
		Return(nil).
		Times(1)
}

// expectPlanLimitAllows stubs the account-plan invoice-limit guard to let the
// order through (either sandbox or no configured limit).
func (suite *SalesOrderSvcTestSuite) expectPlanLimitAllows() {
	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), gomock.Any()).
		Return(&domain.AccountContext{IsSandbox: true}, nil).
		Times(1)
}

// --- ListSalesOrders ---

func (suite *SalesOrderSvcTestSuite) TestListSalesOrders_InternalActor() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.ListSalesOrdersParams) (*domain.ListSalesOrdersResult, *apierror.APIError) {
			suite.Equal("ac_test", params.AccountID)
			suite.Nil(params.BuyerAccountID, "internal actor must not be scoped to a buyer")
			return &domain.ListSalesOrdersResult{SalesOrders: []*domain.SalesOrder{{ID: "or_1"}}}, nil
		}).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentStatuses(gomock.Any(), "ac_test", []string{"or_1"}).
		Return(map[string]constants.SalesOrderPaymentStatus{"or_1": constants.SalesOrderPaymentStatusPaid}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentIntentIDs(gomock.Any(), "ac_test", []string{"or_1"}).
		Return(map[string][]string{"or_1": {"pi_1"}}, nil).
		Times(1)

	result, apiErr := suite.svc.ListSalesOrders(ctx, domain.ListSalesOrdersParams{Limit: 10})
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Len(result.SalesOrders, 1)
	suite.Equal(constants.SalesOrderPaymentStatusPaid, result.SalesOrders[0].PaymentStatus)
	suite.Equal([]string{"pi_1"}, result.SalesOrders[0].PaymentIntentIDs)
}

func (suite *SalesOrderSvcTestSuite) TestListSalesOrders_CustomerActorScopedToOwnAccount() {
	// Customer actor: the service must force BuyerAccountID = actor's account ID so
	// a customer can only see their own orders regardless of what they request.
	ctx := salesOrderCustomerCtx("ac_target", "ac_customer")

	suite.expectReadAccessAllowed("ac_customer", "ac_target")

	suite.orderRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.ListSalesOrdersParams) (*domain.ListSalesOrdersResult, *apierror.APIError) {
			suite.Equal("ac_target", params.AccountID)
			suite.Require().NotNil(params.BuyerAccountID)
			suite.Equal("ac_customer", *params.BuyerAccountID)
			return &domain.ListSalesOrdersResult{}, nil
		}).
		Times(1)

	_, apiErr := suite.svc.ListSalesOrders(ctx, domain.ListSalesOrdersParams{Limit: 10})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestListSalesOrders_InternalActorRequiresReadPermission() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_test"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr",
			AccountID:    new("ac_test"),
			RoleType:     &customCode,
			Permissions:  map[string]bool{},
		},
	})

	_, apiErr := suite.svc.ListSalesOrders(ctx, domain.ListSalesOrdersParams{Limit: 10})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// --- GetSalesOrder ---

func (suite *SalesOrderSvcTestSuite) TestGetSalesOrder_InternalActor() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentStatuses(gomock.Any(), "ac_test", []string{"or_1"}).
		Return(map[string]constants.SalesOrderPaymentStatus{"or_1": constants.SalesOrderPaymentStatusPaid}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentIntentIDs(gomock.Any(), "ac_test", []string{"or_1"}).
		Return(map[string][]string{"or_1": {"pi_1"}}, nil).
		Times(1)

	result, apiErr := suite.svc.GetSalesOrder(ctx, domain.GetSalesOrderParams{SalesOrderID: "or_1"})
	suite.Nil(apiErr)
	suite.Equal("or_1", result.ID)
	suite.Equal(constants.SalesOrderPaymentStatusPaid, result.PaymentStatus)
	suite.Equal([]string{"pi_1"}, result.PaymentIntentIDs)
}

func (suite *SalesOrderSvcTestSuite) TestGetSalesOrder_CustomerActorUsesGetForCustomer() {
	ctx := salesOrderCustomerCtx("ac_target", "ac_customer")

	suite.expectReadAccessAllowed("ac_customer", "ac_target")

	// Customer paths route through GetForCustomer (enforces ownership at the query level).
	suite.orderRepo.EXPECT().
		GetForCustomer(gomock.Any(), "ac_target", "ac_customer", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentStatuses(gomock.Any(), "ac_target", []string{"or_1"}).
		Return(map[string]constants.SalesOrderPaymentStatus{"or_1": constants.SalesOrderPaymentStatusUnpaid}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentIntentIDs(gomock.Any(), "ac_target", []string{"or_1"}).
		Return(map[string][]string{}, nil).
		Times(1)

	result, apiErr := suite.svc.GetSalesOrder(ctx, domain.GetSalesOrderParams{SalesOrderID: "or_1"})
	suite.Nil(apiErr)
	suite.Equal("or_1", result.ID)
}

func (suite *SalesOrderSvcTestSuite) TestGetSalesOrder_LinesInclude() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentStatuses(gomock.Any(), "ac_test", []string{"or_1"}).
		Return(map[string]constants.SalesOrderPaymentStatus{"or_1": constants.SalesOrderPaymentStatusPaid}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPaymentIntentIDs(gomock.Any(), "ac_test", []string{"or_1"}).
		Return(map[string][]string{}, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		GetLines(gomock.Any(), "or_1").
		Return([]*domain.SalesOrderLine{{ID: "orl_1"}}, nil).
		Times(1)

	result, apiErr := suite.svc.GetSalesOrder(ctx, domain.GetSalesOrderParams{
		SalesOrderID: "or_1",
		Includes:     []string{"lines"},
	})
	suite.Nil(apiErr)
	suite.Len(result.Lines, 1)
}

// --- CreateSalesOrder ---

// customerWithOrderDefaults returns a buyer customer relation populated with the
// carrier / service level / shipping term / payment term defaults the create path
// falls back to when the request omits them. The IDs line up with the reference-
// validation mocks (svcl_default in particular) so a defaulted order validates.
func customerWithOrderDefaults() *domain.Customer {
	// Deliberately distinct from the IDs baseCreateOrderParams supplies (cr_default /
	// shtm_default / pmtm_default) so a test that nils the request fields can prove the
	// value came from the customer default. The service level keeps svcl_default so it
	// matches the specific reference-validation mock.
	carrier := "cr_cust_default"
	serviceLevel := "svcl_default"
	shippingTerm := "shtm_cust_default"
	paymentTerm := "pmtm_cust_default"
	return &domain.Customer{
		DefaultCarrierID:      &carrier,
		DefaultServiceLevelID: &serviceLevel,
		DefaultShippingTermID: &shippingTerm,
		DefaultPaymentTermID:  &paymentTerm,
	}
}

// baseCreateOrderParams returns a valid set of create params (addresses, lines,
// priority, type). Individual tests mutate specific fields.
func baseCreateOrderParams() domain.CreateSalesOrderParams {
	carrierID := "cr_default"
	serviceLevelID := "svcl_default"
	shippingTermID := "shtm_default"
	paymentTermID := "pmtm_default"
	return domain.CreateSalesOrderParams{
		BuyerAccountID:       "ac_buyer",
		SalesOrderStatusCode: string(constants.SalesOrderStatusCodeEstimate),
		PriorityCode:         "normal",
		BillToAddressID:      "ad_bill",
		ShipToAddressID:      "ad_ship",
		CarrierID:            &carrierID,
		ServiceLevelID:       &serviceLevelID,
		ShippingTermID:       &shippingTermID,
		PaymentTermID:        &paymentTermID,
		Lines: []domain.CreateSalesOrderLineInput{
			{
				ProductID:      "prod_1",
				QuantityValue:  "2",
				QuantityUnitID: "un_ea",
			},
		},
	}
}

// basePricingBundle returns a minimal pricing bundle for prod_1 priced in un_ea so
// the create-line resolution (unit-group validation + price/cost) succeeds in tests.
func basePricingBundle() *domain.PricingBundle {
	return &domain.PricingBundle{
		Products: map[string]*domain.PricingProduct{
			"prod_1": {
				ProductID:                  "prod_1",
				ItemID:                     "it_1",
				SKU:                        "SKU-1",
				UnitCost:                   "5",
				UnitCostNumeratorUnitID:    "un_usd",
				UnitCostDenominatorUnitID:  "un_ea",
				UnitValue:                  "10",
				UnitValueNumeratorUnitID:   "un_usd",
				UnitValueDenominatorUnitID: "un_ea",
				CategoryUnitGroupID:        "ug_1",
			},
		},
		Units: map[string]*domain.PricingUnit{
			"un_ea": {ID: "un_ea", IsBaseUnit: true},
		},
		UnitGroupUnits: map[string]map[string]*domain.PricingUnitGroupUnit{
			"ug_1": {"un_ea": {UnitGroupID: "ug_1", UnitID: "un_ea"}},
		},
	}
}

// expectCreateOrderResolutionChain wires the read-only resolution calls that run
// BEFORE the write transaction: address validation, sales-rep resolution, line
// pricing, and the (no-carrier → zero) shipping-rate cascade. It deliberately does
// NOT set up order-number allocation (which now happens inside the transaction) or
// any persistence, so branch tests can drive a failure at allocation time.
func (suite *SalesOrderSvcTestSuite) expectCreateOrderResolutionChain(accountID string) {
	// Bill-to + ship-to addresses are referenced by ID: validated against an account, then fetched.
	suite.addressRepo.EXPECT().IsInAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	suite.addressRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.Address{Geolocation: &domain.Geolocation{State: new("CA"), PostalCode: new("90001")}}, nil).AnyTimes()

	// Sales-rep resolution cascade: commission-exempt checks → customer lookup → zipcode → state.
	// Not commission-exempt → customer has no default rep → zipcode nil → state nil.
	suite.customerRepo.EXPECT().IsCommissionExempt(gomock.Any(), accountID, "ac_buyer").
		Return(false, nil).AnyTimes()
	suite.orderRepo.EXPECT().AreAllLineProductLinesCommissionExempt(gomock.Any(), gomock.Any()).
		Return(false, nil).AnyTimes()
	// The customer carries the carrier / service level / shipping term / payment term defaults that the create path applies when the request omits them (matching the Dashboard create form). These are what let a base order — which supplies none of them — resolve to a readable order.
	suite.customerRepo.EXPECT().Get(gomock.Any(), accountID, "ac_buyer", gomock.Any()).
		Return(customerWithOrderDefaults(), nil).AnyTimes()
	suite.territoryRepo.EXPECT().FindSalesRepByZipcode(gomock.Any(), accountID, int32(90001)).
		Return(nil, nil).AnyTimes()
	suite.territoryRepo.EXPECT().FindSalesRepByState(gomock.Any(), accountID, "CA").
		Return(nil, nil).AnyTimes()

	// Line resolution (pricing + cost + unit-group validation) loads the bundle.
	suite.pricingRepo.EXPECT().LoadPricingBundle(gomock.Any(), gomock.Any()).
		Return(basePricingBundle(), nil).AnyTimes()

	// Shipping-rate cascade inputs (no carrier / no Shippo in unit tests → rate 0).
	suite.orderRepo.EXPECT().GetProductTypesAndLines(gomock.Any(), gomock.Any()).
		Return([]domain.ProductTypeLine{}, nil).AnyTimes()
	suite.orderRepo.EXPECT().GetAccountOriginAddress(gomock.Any(), accountID).
		Return(nil, nil).AnyTimes()
	suite.unitRepo.EXPECT().GetByIDs(gomock.Any(), accountID, gomock.Any()).
		Return(nil, nil).AnyTimes()
}

func (suite *SalesOrderSvcTestSuite) expectCreateOrderReferenceValidationMocks(accountID string) {
	suite.serviceLevelRepo.EXPECT().Get(gomock.Any(), accountID, "svcl_default").
		Return(&domain.ServiceLevel{}, nil).AnyTimes()
	suite.shippingTermRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.ShippingTerm{}, nil).AnyTimes()
	suite.paymentTermRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.PaymentTerm{}, nil).AnyTimes()
	suite.carrierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.Carrier{}, nil).AnyTimes()
}

// expectCreateOrderHappyRepoChain wires up every non-discretionary repo call in
// the create happy path. Tests that exercise the happy path call this to avoid
// re-declaring the whole chain; tests that exercise a specific branch (plan
// limit, duplicate number, etc.) set up only the calls relevant to their check.
func (suite *SalesOrderSvcTestSuite) expectCreateOrderHappyRepoChain(accountID string) {
	suite.expectCreateOrderResolutionChain(accountID)
	suite.expectCreateOrderReferenceValidationMocks(accountID)

	// Order-number allocation + duplicate check now run inside the write transaction.
	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), accountID).Return("1001", nil).Times(1)
	suite.orderRepo.EXPECT().IsDuplicateOrderNumber(gomock.Any(), accountID, "1001", (*string)(nil)).Return(false, nil).Times(1)
	// No CustomerPONumber in base params → no duplicate-PO check performed.

	suite.orderRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, _ domain.CreateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
			return &domain.SalesOrder{ID: id, Number: "1001"}, nil
		}).Times(1)

	// Input lines.
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{}, nil).Times(1)

	// Synthetic shipping line: look up the "shipping" system product + currency unit
	// then create a zero-price line item.
	suite.productRepo.EXPECT().GetSystemProduct(gomock.Any(), accountID, "shipping").
		Return(&domain.SystemProductInfo{ProductID: "prod_ship", ProductSKU: "SHIP", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return("un_usd", nil).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			suite.Equal("0", params.UnitPriceValue, "synthetic shipping line must be emitted at price=0 (rate estimation deferred)")
			suite.Equal("prod_ship", params.ProductID)
			return &domain.SalesOrderLine{}, nil
		}).Times(1)

	// Final re-fetch returns the hydrated order.
	suite.orderRepo.EXPECT().Get(gomock.Any(), accountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, id string) (*domain.SalesOrder, *apierror.APIError) {
			return &domain.SalesOrder{ID: id, Number: "1001"}, nil
		}).Times(1)
}

func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_Success_SynthesizesShippingLineAtZero() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()
	suite.expectCreateOrderHappyRepoChain("ac_test")
	suite.expectCacheSuccess()

	result, apiErr := suite.svc.CreateSalesOrder(ctx, baseCreateOrderParams())
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal("1001", result.Number)
}

func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_RejectsNonInternalActor() {
	ctx := salesOrderCustomerCtx("ac_target", "ac_customer")

	_, apiErr := suite.svc.CreateSalesOrder(ctx, baseCreateOrderParams())
	suite.NotNil(apiErr)
	// CheckIsInternalActor returns an auth / permission-ish error; assert we reject the request.
	suite.NotEqual(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

// A customer entering an order for their own account but WITHOUT purchase_orders:create
// is rejected at the authorization gate (before any repo work).
func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_CustomerWithoutPurchaseOrderPermissionRejected() {
	ctx := salesOrderCustomerCtxWithPerms("ac_target", "ac_customer", map[string]bool{})

	params := baseCreateOrderParams()
	params.BuyerAccountID = "ac_customer" // self-scope satisfied, so we reach the permission check

	_, apiErr := suite.svc.CreateSalesOrder(ctx, params)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// A customer holding purchase_orders:create for their own buyer account clears
// authorization; we prove it by letting the flow proceed to (and fail at) the
// plan-limit stage, which runs only after the auth switch passes.
func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_CustomerWithPurchaseOrderPermissionPassesAuthorization() {
	ctx := salesOrderIdempotencyCtx(
		salesOrderCustomerCtxWithPerms("ac_target", "ac_customer", map[string]bool{"purchase_orders:create": true}),
		"/core.CoreService/CreateSalesOrder",
	)

	// Customer note-fold lookup (no note → nothing folded).
	suite.customerRepo.EXPECT().Get(gomock.Any(), "ac_target", "ac_customer", gomock.Any()).
		Return(&domain.Customer{}, nil).Times(1)

	// Plan limit exceeded → rejected AFTER authorization, on the target account.
	planID := "plan_basic"
	periodEnd := time.Now().Add(15 * 24 * time.Hour)
	max := int32(10)
	suite.accountRepo.EXPECT().GetAccountContext(gomock.Any(), "ac_target").
		Return(&domain.AccountContext{IsSandbox: false}, nil).Times(1)
	suite.accountRepo.EXPECT().GetPlanIDAndPeriodEnd(gomock.Any(), "ac_target").
		Return(&planID, &periodEnd, nil).Times(1)
	suite.accountRepo.EXPECT().ListPlanLimits(gomock.Any(), planID).
		Return(map[string]*int32{"invoices_maximum": &max}, nil).Times(1)
	suite.invoiceRepo.EXPECT().CountSince(gomock.Any(), "ac_target", gomock.Any()).
		Return(int64(10), nil).Times(1)

	params := baseCreateOrderParams()
	params.BuyerAccountID = "ac_customer"

	_, apiErr := suite.svc.CreateSalesOrder(ctx, params)
	suite.Require().NotNil(apiErr)
	// Not an auth rejection — we reached and failed the plan-limit check.
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_PlanLimitExceeded_NonSandbox() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	// Non-sandbox + subscribed account with a configured invoice limit; usage is at the cap.
	planID := "plan_basic"
	periodEnd := time.Now().Add(15 * 24 * time.Hour)
	max := int32(10)

	suite.accountRepo.EXPECT().GetAccountContext(gomock.Any(), "ac_test").
		Return(&domain.AccountContext{IsSandbox: false}, nil).Times(1)
	suite.accountRepo.EXPECT().GetPlanIDAndPeriodEnd(gomock.Any(), "ac_test").
		Return(&planID, &periodEnd, nil).Times(1)
	suite.accountRepo.EXPECT().ListPlanLimits(gomock.Any(), planID).
		Return(map[string]*int32{"invoices_maximum": &max}, nil).Times(1)
	suite.invoiceRepo.EXPECT().CountSince(gomock.Any(), "ac_test", gomock.Any()).
		Return(int64(10), nil).Times(1) // at the cap

	_, apiErr := suite.svc.CreateSalesOrder(ctx, baseCreateOrderParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_DuplicateOrderNumber() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()

	// The order number is allocated inside the write transaction (after the read-only resolution above), so a duplicate auto-generated number is detected there and rolls the transaction back instead of consuming the number.
	suite.expectCreateOrderResolutionChain("ac_test")
	suite.expectCreateOrderReferenceValidationMocks("ac_test")
	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), "ac_test").Return("1001", nil).Times(1)
	suite.orderRepo.EXPECT().IsDuplicateOrderNumber(gomock.Any(), "ac_test", "1001", (*string)(nil)).
		Return(true, nil).Times(1)

	suite.expectCacheError()

	_, apiErr := suite.svc.CreateSalesOrder(ctx, baseCreateOrderParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
	suite.Equal("number", apiErr.Param)
}

// TestCreateSalesOrder_PreTransactionFailureDoesNotConsumeOrderNumber is the regression
// guard for the order-number gap bug: allocation must happen inside the write transaction,
// so any failure before it — here an address-validation error, but in the production incident
// it was the external Shippo rate lookup returning 401 — never allocates (and never burns) a
// number. GetNextOrderNumber is asserted to never be called on this path.
func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_PreTransactionFailureDoesNotConsumeOrderNumber() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()

	// Bill-to address fails validation → the create aborts before the write transaction.
	suite.addressRepo.EXPECT().IsInAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()

	// The order number must never be allocated when the create fails before the transaction.
	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), gomock.Any()).Times(0)

	suite.expectCacheError()

	_, apiErr := suite.svc.CreateSalesOrder(ctx, baseCreateOrderParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_DuplicateCustomerPONumber() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()

	// The customer-PO duplicate check runs up front (before the order number is allocated inside the transaction), so it fails fast without ever reaching number allocation or the resolution/Shippo work.
	suite.orderRepo.EXPECT().
		IsDuplicateCustomerPO(gomock.Any(), "ac_test", "ac_buyer", "PO-123", (*string)(nil)).
		Return(true, nil).Times(1)

	suite.expectCacheError()

	params := baseCreateOrderParams()
	po := "PO-123"
	params.CustomerPONumber = &po

	_, apiErr := suite.svc.CreateSalesOrder(ctx, params)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
	suite.Equal("customer_po_number", apiErr.Param)
}

func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_SalesRepResolvedFromCustomerDefault() {
	// When the customer record has a DefaultSalesRepID, service uses it and must NOT
	// fall through to zipcode / state lookups. This locks in the short-circuit.
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()

	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), "ac_test").Return("1001", nil).Times(1)
	suite.orderRepo.EXPECT().IsDuplicateOrderNumber(gomock.Any(), "ac_test", "1001", (*string)(nil)).Return(false, nil).Times(1)
	suite.addressRepo.EXPECT().IsInAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	suite.addressRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.Address{Geolocation: &domain.Geolocation{State: new("CA"), PostalCode: new("90001")}}, nil).AnyTimes()

	repID := "au_rep"
	// Not commission-exempt → resolution proceeds to the customer default.
	suite.customerRepo.EXPECT().IsCommissionExempt(gomock.Any(), "ac_test", "ac_buyer").
		Return(false, nil).Times(1)
	suite.orderRepo.EXPECT().AreAllLineProductLinesCommissionExempt(gomock.Any(), gomock.Any()).
		Return(false, nil).Times(1)
	// Get is called by both sales-rep resolution and the shipping-rate cascade. The request supplies
	// carrier / service level / shipping term / payment term (via baseCreateOrderParams), so the customer
	// only needs the default sales rep here.
	suite.customerRepo.EXPECT().Get(gomock.Any(), "ac_test", "ac_buyer", gomock.Any()).
		Return(&domain.Customer{DefaultSalesRepID: &repID}, nil).AnyTimes()
	// Line resolution (pricing + cost + unit-group validation) loads the bundle.
	suite.pricingRepo.EXPECT().LoadPricingBundle(gomock.Any(), gomock.Any()).
		Return(basePricingBundle(), nil).AnyTimes()

	// Shipping-rate cascade inputs (no carrier → rate 0).
	suite.orderRepo.EXPECT().GetProductTypesAndLines(gomock.Any(), gomock.Any()).
		Return([]domain.ProductTypeLine{}, nil).AnyTimes()
	suite.orderRepo.EXPECT().GetAccountOriginAddress(gomock.Any(), "ac_test").
		Return(nil, nil).AnyTimes()
	suite.unitRepo.EXPECT().GetByIDs(gomock.Any(), "ac_test", gomock.Any()).
		Return(nil, nil).AnyTimes()
	suite.expectCreateOrderReferenceValidationMocks("ac_test")
	// Zipcode / state lookups must NOT happen.

	suite.orderRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
			suite.Require().NotNil(p.SalesRepID)
			suite.Equal("au_rep", *p.SalesRepID)
			return &domain.SalesOrder{ID: id, Number: p.Number}, nil
		}).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{}, nil).Times(1)
	suite.productRepo.EXPECT().GetSystemProduct(gomock.Any(), "ac_test", "shipping").
		Return(&domain.SystemProductInfo{ProductID: "prod_ship", ProductSKU: "SHIP", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return("un_usd", nil).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{}, nil).Times(1)
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", gomock.Any()).
		Return(&domain.SalesOrder{ID: "or_new", Number: "1001"}, nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.CreateSalesOrder(ctx, baseCreateOrderParams())
	suite.Nil(apiErr)
}

// When the request omits carrier / service level / shipping term / payment term, the
// create path fills them from the buyer's customer-relation defaults, so the persisted
// order carries the references the read adapter requires.
func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_AppliesCustomerDefaultsWhenOmitted() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()
	suite.expectCreateOrderResolutionChain("ac_test")
	suite.expectCreateOrderReferenceValidationMocks("ac_test")
	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), "ac_test").Return("1001", nil).Times(1)
	suite.orderRepo.EXPECT().IsDuplicateOrderNumber(gomock.Any(), "ac_test", "1001", (*string)(nil)).Return(false, nil).Times(1)

	suite.orderRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
			suite.Require().NotNil(p.CarrierID)
			suite.Equal("cr_cust_default", *p.CarrierID)
			suite.Require().NotNil(p.ServiceLevelID)
			suite.Equal("svcl_default", *p.ServiceLevelID)
			suite.Require().NotNil(p.ShippingTermID)
			suite.Equal("shtm_cust_default", *p.ShippingTermID)
			suite.Require().NotNil(p.PaymentTermID)
			suite.Equal("pmtm_cust_default", *p.PaymentTermID)
			return &domain.SalesOrder{ID: id, Number: p.Number}, nil
		}).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{}, nil).Times(1)
	suite.productRepo.EXPECT().GetSystemProduct(gomock.Any(), "ac_test", "shipping").
		Return(&domain.SystemProductInfo{ProductID: "prod_ship", ProductSKU: "SHIP", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return("un_usd", nil).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{}, nil).Times(1)
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", gomock.Any()).
		Return(&domain.SalesOrder{ID: "or_new", Number: "1001"}, nil).Times(1)
	suite.expectCacheSuccess()

	// Omit every field that has a customer default so the create path must fall back to the customer relation.
	params := baseCreateOrderParams()
	params.CarrierID = nil
	params.ServiceLevelID = nil
	params.ShippingTermID = nil
	params.PaymentTermID = nil

	_, apiErr := suite.svc.CreateSalesOrder(ctx, params)
	suite.Nil(apiErr)
}

// A caller-supplied carrier / shipping term / payment term must win over the customer
// default (the default is only a fallback, never an override).
func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_RequestValuesOverrideCustomerDefaults() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()
	suite.expectCreateOrderResolutionChain("ac_test")
	suite.expectCreateOrderReferenceValidationMocks("ac_test")
	suite.serviceLevelRepo.EXPECT().Get(gomock.Any(), "ac_test", "svcl_req").Return(&domain.ServiceLevel{}, nil).AnyTimes()
	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), "ac_test").Return("1001", nil).Times(1)
	suite.orderRepo.EXPECT().IsDuplicateOrderNumber(gomock.Any(), "ac_test", "1001", (*string)(nil)).Return(false, nil).Times(1)

	suite.orderRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
			suite.Equal("cr_req", *p.CarrierID)
			suite.Equal("svcl_req", *p.ServiceLevelID)
			suite.Equal("shpt_req", *p.ShippingTermID)
			suite.Equal("pyt_req", *p.PaymentTermID)
			return &domain.SalesOrder{ID: id, Number: p.Number}, nil
		}).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{}, nil).Times(1)
	suite.productRepo.EXPECT().GetSystemProduct(gomock.Any(), "ac_test", "shipping").
		Return(&domain.SystemProductInfo{ProductID: "prod_ship", ProductSKU: "SHIP", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return("un_usd", nil).Times(1)
	suite.lineRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{}, nil).Times(1)
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", gomock.Any()).
		Return(&domain.SalesOrder{ID: "or_new", Number: "1001"}, nil).Times(1)
	suite.expectCacheSuccess()

	params := baseCreateOrderParams()
	carrier, serviceLevel, shippingTerm, paymentTerm := "cr_req", "svcl_req", "shpt_req", "pyt_req"
	params.CarrierID = &carrier
	params.ServiceLevelID = &serviceLevel
	params.ShippingTermID = &shippingTerm
	params.PaymentTermID = &paymentTerm

	_, apiErr := suite.svc.CreateSalesOrder(ctx, params)
	suite.Nil(apiErr)
}

// When neither the request nor the customer supplies a carrier, the create is rejected
// with a field-scoped 400 (before allocating an order number) rather than persisting an
// order that 500s on read.
func (suite *SalesOrderSvcTestSuite) TestCreateSalesOrder_MissingCarrierWithoutCustomerDefault_Rejected() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrder")

	suite.expectPlanLimitAllows()
	suite.expectIdempotencyStarted()

	suite.addressRepo.EXPECT().IsInAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	suite.addressRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(&domain.Address{Geolocation: &domain.Geolocation{State: new("CA"), PostalCode: new("90001")}}, nil).AnyTimes()
	// Customer relation has no defaults, so the omitted carrier cannot be filled.
	suite.customerRepo.EXPECT().Get(gomock.Any(), "ac_test", "ac_buyer", gomock.Any()).
		Return(&domain.Customer{}, nil).AnyTimes()

	// The order number must never be allocated when the create fails validation.
	suite.orderRepo.EXPECT().GetNextOrderNumber(gomock.Any(), gomock.Any()).Times(0)
	suite.expectCacheError()

	params := baseCreateOrderParams()
	params.CarrierID = nil // and the customer has no default carrier

	_, apiErr := suite.svc.CreateSalesOrder(ctx, params)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	suite.Equal("carrier_id", apiErr.Param)
}

// --- UpdateSalesOrder ---

func (suite *SalesOrderSvcTestSuite) TestUpdateSalesOrder_PreservesNullableFieldsWhenOmitted() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/UpdateSalesOrder")

	suite.expectIdempotencyStarted()

	existingCarrier := "cr_existing"
	existingDiscount := "odsc_existing"
	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{
			ID:              "or_1",
			BuyerAccountID:  "ac_buyer",
			CarrierID:       &existingCarrier,
			OrderDiscountID: &existingDiscount,
		}, nil).Times(1)

	// Service must backfill omitted-nullable fields with the existing values
	// BEFORE calling the repo. Assert that behavior here.
	suite.orderRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.UpdateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
			suite.Require().NotNil(p.CarrierID)
			suite.Equal("cr_existing", *p.CarrierID)
			suite.Require().NotNil(p.OrderDiscountID)
			suite.Equal("odsc_existing", *p.OrderDiscountID)
			return &domain.SalesOrder{ID: "or_1"}, nil
		}).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrder(ctx, domain.UpdateSalesOrderParams{
		SalesOrderID: "or_1",
		Note:         new("updated note"),
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestUpdateSalesOrder_DuplicateOrderNumberRejected() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/UpdateSalesOrder")

	suite.expectIdempotencyStarted()

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", Number: "1000", BuyerAccountID: "ac_buyer"}, nil).Times(1)
	suite.orderRepo.EXPECT().
		IsDuplicateOrderNumber(gomock.Any(), "ac_test", "1001", new("or_1")).
		Return(true, nil).Times(1)

	suite.expectCacheError()

	newNum := "1001"
	_, apiErr := suite.svc.UpdateSalesOrder(ctx, domain.UpdateSalesOrderParams{
		SalesOrderID: "or_1",
		Number:       &newNum,
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
	suite.Equal("number", apiErr.Param)
}

func (suite *SalesOrderSvcTestSuite) TestUpdateSalesOrder_ShippingAddressRepointsOrder() {
	// Supplying a shipping_address_id re-points the order to an existing address
	// via the order update. The service must NOT mutate any address or
	// geolocation records — editing an address is a separate concern handled by
	// the update-address endpoint.
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/UpdateSalesOrder")

	suite.expectIdempotencyStarted()

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", ShippingAddressID: "addr_ship", BuyerAccountID: "ac_buyer"}, nil).Times(1)

	suite.orderRepo.EXPECT().
		Update(gomock.Any(), gomock.Cond(func(p domain.UpdateSalesOrderParams) bool {
			return p.ShippingAddressID != nil && *p.ShippingAddressID == "addr_new"
		})).
		Return(&domain.SalesOrder{ID: "or_1"}, nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrder(ctx, domain.UpdateSalesOrderParams{
		SalesOrderID:      "or_1",
		ShippingAddressID: new("addr_new"),
	})
	suite.Nil(apiErr)
}

// --- DeleteSalesOrder ---

func (suite *SalesOrderSvcTestSuite) TestDeleteSalesOrder_Success() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).Times(1)
	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrder, "or_1", gomock.Any()).
		Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeleteCascade(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)

	apiErr := suite.svc.DeleteSalesOrder(ctx, domain.DeleteSalesOrderParams{SalesOrderID: "or_1"})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestDeleteSalesOrder_BlockedWhenFulfilled() {
	ctx := salesOrderInternalCtx("ac_test")

	now := time.Now()
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", CompletedAt: &now}, nil).Times(1)

	apiErr := suite.svc.DeleteSalesOrder(ctx, domain.DeleteSalesOrderParams{SalesOrderID: "or_1"})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestDeleteSalesOrder_AlreadyDeletedReturnsSemanticError() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(nil, apierror.NewResourceNotFoundError("Sales order not found.")).Times(1)
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrder, "or_1").
		Return(true, nil).Times(1)

	apiErr := suite.svc.DeleteSalesOrder(ctx, domain.DeleteSalesOrderParams{SalesOrderID: "or_1"})
	suite.NotNil(apiErr)
	suite.Contains(apiErr.PublicMessage, "already been deleted")
}

// --- BulkDeleteSalesOrders ---

func (suite *SalesOrderSvcTestSuite) TestBulkDeleteSalesOrders_RejectsIfAnyFulfilled() {
	// Bulk delete is atomic: a single fulfilled order must abort the whole batch
	// (no partial deletes).
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "BulkDeleteSalesOrders")

	suite.expectIdempotencyStarted()
	suite.expectCacheError()

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_ok").
		Return(&domain.SalesOrder{ID: "or_ok"}, nil).Times(1)
	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrder, "or_ok", gomock.Any()).
		Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeleteCascade(gomock.Any(), "ac_test", "or_ok").Return(nil).Times(1)

	// Second order is fulfilled → batch aborts.
	now := time.Now()
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_fulfilled").
		Return(&domain.SalesOrder{ID: "or_fulfilled", CompletedAt: &now}, nil).Times(1)

	apiErr := suite.svc.BulkDeleteSalesOrders(ctx, domain.BulkDeleteSalesOrdersParams{
		SalesOrderIDs: []string{"or_ok", "or_fulfilled"},
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// --- ChangeSalesOrderStatus ---

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_IssueFromEstimate_CreatesPickAndReservation() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", Number: "1001", SalesOrderStatusCode: "estimate"}, nil).Times(1)

	// Transactional block.
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "issued", gomock.Any(), (*time.Time)(nil)).
		Return(nil).Times(1)
	suite.orderRepo.EXPECT().CreatePick(gomock.Any(), gomock.Any(), "1001", "or_1", "ac_test").Return(nil).Times(1)

	// Only sale lines (shipping / discount / credit are excluded by the repo).
	// One item-bearing line → expect both pick line AND reserved inventory issue.
	itemID := "itm_1"
	suite.orderRepo.EXPECT().GetSaleLinesForIssue(gomock.Any(), "or_1").
		Return([]domain.SalesOrderSaleLineForIssue{
			{ID: "orl_1", ItemID: &itemID, QuantityValue: "5", QuantityUnitID: "un_ea"},
		}, nil).Times(1)
	suite.orderRepo.EXPECT().DeleteReservedInventoryIssues(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)
	suite.lineRepo.EXPECT().CreateQuantity(gomock.Any(), gomock.Any(), "5", "un_ea").Return(nil).Times(1)
	suite.orderRepo.EXPECT().CreatePickLine(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "orl_1").Return(nil).Times(1)
	suite.lineRepo.EXPECT().CreateQuantity(gomock.Any(), gomock.Any(), "5", "un_ea").Return(nil).Times(1)
	suite.orderRepo.EXPECT().
		CreateReservedInventoryIssue(gomock.Any(), gomock.Any(), "ac_test", "itm_1", gomock.Any(), "or_1").
		Return(nil).Times(1)

	// Final re-fetch.
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "issued"}, nil).Times(1)

	result, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "issue",
	})
	suite.Nil(apiErr)
	suite.Equal("issued", result.SalesOrderStatusCode)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_IssueWithoutItemID_SkipsInventoryReservation() {
	// Sale lines without an ItemID should get a pick line but NOT a reserved
	// inventory issue. Locks in the item-ID guard in the inventory-reservation path.
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", Number: "1001", SalesOrderStatusCode: "estimate"}, nil).Times(1)

	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "issued", gomock.Any(), (*time.Time)(nil)).
		Return(nil).Times(1)
	suite.orderRepo.EXPECT().CreatePick(gomock.Any(), gomock.Any(), "1001", "or_1", "ac_test").Return(nil).Times(1)

	suite.orderRepo.EXPECT().GetSaleLinesForIssue(gomock.Any(), "or_1").
		Return([]domain.SalesOrderSaleLineForIssue{
			{ID: "orl_1", ItemID: nil, QuantityValue: "1", QuantityUnitID: "un_ea"},
		}, nil).Times(1)
	suite.orderRepo.EXPECT().DeleteReservedInventoryIssues(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)

	// Pick line is created for the non-item line, but NO CreateReservedInventoryIssue.
	suite.lineRepo.EXPECT().CreateQuantity(gomock.Any(), gomock.Any(), "1", "un_ea").Return(nil).Times(1)
	suite.orderRepo.EXPECT().CreatePickLine(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "orl_1").Return(nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "issued"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "issue",
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_IssueRejectsNonEstimate() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "issued"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "issue",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_Unissue_ReleasesInventory() {
	ctx := salesOrderInternalCtx("ac_test")

	issuedAt := time.Now()
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "issued", IssuedAt: &issuedAt}, nil).Times(1)

	// Full release chain: pick quantities → pick lines → pick → allocations → reserved issues.
	suite.orderRepo.EXPECT().DeleteQuantitiesByPickLines(gomock.Any(), "or_1").Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeletePickLinesBySalesOrder(gomock.Any(), "or_1").Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeletePickBySalesOrder(gomock.Any(), "or_1").Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeleteInventoryAllocationsByReservedIssues(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeleteReservedInventoryIssues(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)

	// Status back to estimate; issuedAt cleared.
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "estimate", (*time.Time)(nil), (*time.Time)(nil)).
		Return(nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "estimate"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "unissue",
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_Close_MarksPickPacked() {
	ctx := salesOrderInternalCtx("ac_test")

	issuedAt := time.Now().Add(-time.Hour)
	pickID := "pk_1"
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{
			ID: "or_1", SalesOrderStatusCode: "issued", IssuedAt: &issuedAt, PickID: &pickID,
		}, nil).Times(1)

	// Status transitions to fulfilled; issuedAt preserved, completedAt set.
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "fulfilled", &issuedAt, gomock.Any()).
		Return(nil).Times(1)

	// Pick packed.
	suite.pickRepo.EXPECT().
		UpdateFinishedAt(gomock.Any(), "ac_test", "pk_1", gomock.Any()).
		Return(nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "fulfilled"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "close",
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_Open_ReopensFulfilled() {
	ctx := salesOrderInternalCtx("ac_test")

	issuedAt := time.Now().Add(-time.Hour)
	pickID := "pk_1"
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{
			ID: "or_1", SalesOrderStatusCode: "fulfilled", IssuedAt: &issuedAt, PickID: &pickID,
		}, nil).Times(1)

	// Status back to issued; completedAt cleared; issuedAt preserved.
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "issued", &issuedAt, (*time.Time)(nil)).
		Return(nil).Times(1)

	suite.pickRepo.EXPECT().ClearFinishedAt(gomock.Any(), "ac_test", "pk_1").Return(nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "issued"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "open",
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_InvalidAction() {
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "estimate"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "BOGUS_ACTION",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_IssueWithSendEmailFiresNotification() {
	ctx := salesOrderInternalCtx("ac_test")

	// Minimum setup to reach the post-issue email branch.
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", Number: "1001", SalesOrderStatusCode: "estimate"}, nil).Times(1)
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "issued", gomock.Any(), (*time.Time)(nil)).
		Return(nil).Times(1)
	suite.orderRepo.EXPECT().CreatePick(gomock.Any(), gomock.Any(), "1001", "or_1", "ac_test").Return(nil).Times(1)
	suite.orderRepo.EXPECT().GetSaleLinesForIssue(gomock.Any(), "or_1").Return(nil, nil).Times(1)
	suite.orderRepo.EXPECT().DeleteReservedInventoryIssues(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)

	// Email branch: recipients fetched, account name resolved, email published, ack marked sent.
	suite.orderRepo.EXPECT().GetAcknowledgementRecipients(gomock.Any(), "or_1").
		Return([]string{"buyer@example.com"}, nil).Times(1)
	suite.accountRepo.EXPECT().GetName(gomock.Any(), "ac_test").Return("Test Seller", nil).Times(1)
	suite.notifier.EXPECT().PublishSendEmail(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	suite.orderRepo.EXPECT().MarkAcknowledgementSent(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", SalesOrderStatusCode: "issued"}, nil).Times(1)

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "issue",
		SendEmail:    true,
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderSvcTestSuite) TestChangeStatus_IssueWithoutSendEmailDoesNotNotify() {
	// Inverse of the previous: when SendEmail is false the notifier must never be called.
	ctx := salesOrderInternalCtx("ac_test")

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", Number: "1001", SalesOrderStatusCode: "estimate"}, nil).Times(1)
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_1", "issued", gomock.Any(), (*time.Time)(nil)).
		Return(nil).Times(1)
	suite.orderRepo.EXPECT().CreatePick(gomock.Any(), gomock.Any(), "1001", "or_1", "ac_test").Return(nil).Times(1)
	suite.orderRepo.EXPECT().GetSaleLinesForIssue(gomock.Any(), "or_1").Return(nil, nil).Times(1)
	suite.orderRepo.EXPECT().DeleteReservedInventoryIssues(gomock.Any(), "ac_test", "or_1").Return(nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).Times(1)

	// No notifier expectations → gomock.Finish() fails if anything is published.

	_, apiErr := suite.svc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: "or_1",
		StatusChange: "issue",
		SendEmail:    false,
	})
	suite.Nil(apiErr)
}

// --- CheckoutSalesOrder ---

func (suite *SalesOrderSvcTestSuite) TestCheckoutSalesOrder_Success() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CheckoutSalesOrder")

	suite.expectIdempotencyStarted()

	// Encrypt realistic Stripe creds so the service can round-trip decrypt them. The account ID is the additional authenticated data, matching how integration credentials are sealed by both this service and the legacy dashboard API.
	credsJSON, _ := json.Marshal(domain.StripeCredentials{PrivateKey: "sk_test_xxx"})
	encrypted, err := crypto.EncryptAESGCM(credsJSON, suite.encryptionKey, []byte("ac_test"), "k1")
	suite.Require().NoError(err)

	suite.accountIntegrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), "ac_test", constants.IntegrationCodeStripe).
		Return(encrypted, true, nil).Times(1)
	suite.orderRepo.EXPECT().CheckPaymentStatus(gomock.Any(), "or_1").Return(false, nil).Times(1)
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", Number: "1001", BuyerAccountID: "ac_buyer"}, nil).Times(1)
	suite.orderRepo.EXPECT().GetLines(gomock.Any(), "or_1").
		Return([]*domain.SalesOrderLine{
			{ProductSKU: "SKU-1", QuantityValue: "2", UnitPriceValue: "19.99"},
		}, nil).Times(1)

	suite.checkoutFactory.EXPECT().Build("sk_test_xxx").Return(suite.checkoutClient).Times(1)
	suite.checkoutClient.EXPECT().
		CreateOneTimeCheckoutSession(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateCheckoutSessionParams) (*domain.StripeCheckoutSession, *apierror.APIError) {
			suite.Equal("buyer@example.com", params.CustomerEmail)
			suite.Require().Len(params.LineItems, 1)
			// 19.99 → 1999 cents, qty 2.
			suite.Equal(int64(1999), params.LineItems[0].AmountCents)
			suite.Equal(int64(2), params.LineItems[0].Quantity)
			// Metadata must carry orderID + customerID for webhook correlation.
			suite.Equal("or_1", params.PaymentIntentMetadata["orderID"])
			suite.Equal("ac_buyer", params.PaymentIntentMetadata["customerID"])
			return &domain.StripeCheckoutSession{URL: "https://checkout.stripe.com/test"}, nil
		}).Times(1)

	suite.notifier.EXPECT().PublishSendEmail(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	suite.expectCacheSuccess()

	result, apiErr := suite.svc.CheckoutSalesOrder(ctx, domain.CheckoutSalesOrderParams{
		SalesOrderID: "or_1",
		Email:        "buyer@example.com",
	})
	suite.Nil(apiErr)
	suite.Equal("https://checkout.stripe.com/test", result.CheckoutURL)
}

func (suite *SalesOrderSvcTestSuite) TestCheckoutSalesOrder_StripeIntegrationInactive() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CheckoutSalesOrder")

	suite.expectIdempotencyStarted()
	suite.accountIntegrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), "ac_test", constants.IntegrationCodeStripe).
		Return("", false, nil).Times(1)

	_, apiErr := suite.svc.CheckoutSalesOrder(ctx, domain.CheckoutSalesOrderParams{
		SalesOrderID: "or_1",
		Email:        "buyer@example.com",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestCheckoutSalesOrder_AlreadyPaidRejected() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CheckoutSalesOrder")

	suite.expectIdempotencyStarted()

	credsJSON, _ := json.Marshal(domain.StripeCredentials{PrivateKey: "sk_test_xxx"})
	encrypted, err := crypto.EncryptAESGCM(credsJSON, suite.encryptionKey, []byte("ac_test"), "k1")
	suite.Require().NoError(err)

	suite.accountIntegrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), "ac_test", constants.IntegrationCodeStripe).
		Return(encrypted, true, nil).Times(1)
	suite.orderRepo.EXPECT().CheckPaymentStatus(gomock.Any(), "or_1").Return(true, nil).Times(1)

	_, apiErr := suite.svc.CheckoutSalesOrder(ctx, domain.CheckoutSalesOrderParams{
		SalesOrderID: "or_1",
		Email:        "buyer@example.com",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

// --- CreateSalesOrderProductionRun ---

func (suite *SalesOrderSvcTestSuite) TestCreateProductionRun_RejectsIfOrderAlreadyHasRun() {
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrderProductionRun")

	suite.expectIdempotencyStarted()

	suite.accountUserRepo.EXPECT().
		FindByAccountAndUserID(gomock.Any(), "usr_test123", "ac_test").
		Return(&domain.AccountUser{ID: "au_1"}, nil).Times(1)

	existingRun := "pr_existing"
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", ProductionRunID: &existingRun}, nil).Times(1)

	_, apiErr := suite.svc.CreateSalesOrderProductionRun(ctx, domain.CreateSalesOrderProductionRunParams{
		SalesOrderID: "or_1",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
	suite.Equal("id", apiErr.Param)
}

func (suite *SalesOrderSvcTestSuite) TestCreateProductionRun_SuccessUsesBOMLinesOnly() {
	// Verify GetLinesForBOM (not GetLines) is the method called — this is how
	// the service excludes synthetic shipping / discount / credit lines from
	// the production plan.
	ctx := salesOrderIdempotencyCtx(salesOrderInternalCtx("ac_test"), "/core.CoreService/CreateSalesOrderProductionRun")

	suite.expectIdempotencyStarted()

	suite.accountUserRepo.EXPECT().
		FindByAccountAndUserID(gomock.Any(), "usr_test123", "ac_test").
		Return(&domain.AccountUser{ID: "au_1"}, nil).Times(1)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).Times(1)

	// GetLinesForBOM (NOT GetLines) is the critical assertion here.
	suite.orderRepo.EXPECT().GetLinesForBOM(gomock.Any(), "or_1").
		Return([]domain.SalesOrderLineForBOM{{ID: "orl_1", ItemID: "itm_1", QuantityUnitID: "un_ea"}}, nil).Times(1)

	suite.productionRunQueryRepo.EXPECT().GetNextNumber(gomock.Any(), "ac_test").Return("PR-001", nil).Times(1)
	suite.materialDemandRepo.EXPECT().
		GetMaterialDemand(gomock.Any(), "ac_test", "itm_1", gomock.Any(), "un_ea").
		Return([]domain.MaterialDemandItem{}, nil).Times(1)

	// Inside the transaction: create run + batch + link.
	suite.productionRunQueryRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), "au_1", "PR-001", "ac_test").
		Return(nil).Times(1)
	suite.batchRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.BaseBatch{}, nil).Times(1)
	suite.orderRepo.EXPECT().SetProductionRunID(gomock.Any(), "ac_test", "or_1", gomock.Any()).Return(nil).Times(1)
	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).Times(1)

	suite.expectCacheSuccess()

	result, apiErr := suite.svc.CreateSalesOrderProductionRun(ctx, domain.CreateSalesOrderProductionRunParams{
		SalesOrderID: "or_1",
	})
	suite.Nil(apiErr)
	suite.NotEmpty(result.ProductionRunID)
}
