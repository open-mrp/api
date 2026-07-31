package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// SalesOrderLineSvcTestSuite exercises sales_order_line_service.go end-to-end
// with all repository and mediator collaborators mocked. Mirrors the style of
// role_service_test.go and unit_service_test.go.
type SalesOrderLineSvcTestSuite struct {
	suite.Suite
	svc               domain.SalesOrderLineSvc
	orderRepo         *repositorymock.MockSalesOrderRepo
	lineRepo          *repositorymock.MockSalesOrderLineRepo
	pickLineRepo      *repositorymock.MockPickLineRepo
	pickRepo          *repositorymock.MockPickRepo
	pricingRepo       *repositorymock.MockPricingRepo
	deletedRecordRepo *repositorymock.MockDeletedRecordRepo
	repoFactory       *factorymock.MockRepoFactory
	mediatorFactory   *factorymock.MockMediatorFactory
	idempotencyMed    *mediatormock.MockIdempotencyMed
	editAccessMed     *mediatormock.MockEditAccessMed
	ctrl              *gomock.Controller
}

func (suite *SalesOrderLineSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.orderRepo = repositorymock.NewMockSalesOrderRepo(suite.ctrl)
	suite.lineRepo = repositorymock.NewMockSalesOrderLineRepo(suite.ctrl)
	suite.pickLineRepo = repositorymock.NewMockPickLineRepo(suite.ctrl)
	suite.pickRepo = repositorymock.NewMockPickRepo(suite.ctrl)
	suite.pricingRepo = repositorymock.NewMockPricingRepo(suite.ctrl)
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewSalesOrderRepo().Return(suite.orderRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewSalesOrderLineRepo().Return(suite.lineRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPickLineRepo().Return(suite.pickLineRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPickRepo().Return(suite.pickRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPricingRepo().Return(suite.pricingRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(suite.deletedRecordRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.editAccessMed = mediatormock.NewMockEditAccessMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
		EditAccess:  suite.editAccessMed,
	}).AnyTimes()

	suite.svc = NewSalesOrderLineSvc(&SalesOrderLineSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *SalesOrderLineSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestSalesOrderLineSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SalesOrderLineSvcTestSuite))
}

// --- Helpers ---

// salesOrderLineCtx returns an authenticated internal-actor context scoped to
// accountID with all sales-orders permissions. Tests that need to vary the
// actor construct their own identity inline.
func salesOrderLineCtx(accountID string) context.Context {
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
				"sales_orders:read":   true,
				"sales_orders:create": true,
				"sales_orders:update": true,
				"sales_orders:delete": true,
				"customers:update":    true,
				"suppliers:update":    true,
			},
		},
	})
}

// salesOrderLineIdempotencyCtx adds an idempotency key to the given context so
// the service's idempotency mediator is exercised instead of short-circuiting.
func salesOrderLineIdempotencyCtx(ctx context.Context, handler string) context.Context {
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithHandler(ctx, handler)
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func (suite *SalesOrderLineSvcTestSuite) expectIdempotencyStarted() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

func (suite *SalesOrderLineSvcTestSuite) expectCacheSuccess() {
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).
		Return(nil).
		Times(1)
}

func (suite *SalesOrderLineSvcTestSuite) expectCacheError() {
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

// createLinePricingBundle is a minimal pricing bundle for the create-line product
// ("prod_test", priced $10/ea, cost $4/ea) so resolveSalesOrderCreateLines validates the
// unit and prices the line without a real pricing repo.
func createLinePricingBundle() *domain.PricingBundle {
	return &domain.PricingBundle{
		Products: map[string]*domain.PricingProduct{
			"prod_test": {
				ProductID:                  "prod_test",
				ItemID:                     "it_test",
				SKU:                        "SKU-PROD",
				UnitCost:                   "4",
				UnitCostNumeratorUnitID:    "un_usd",
				UnitCostDenominatorUnitID:  "un_ea",
				UnitValue:                  "10",
				UnitValueNumeratorUnitID:   "un_usd",
				UnitValueDenominatorUnitID: "un_ea",
				CategoryUnitGroupID:        "ug_test",
			},
		},
		Units: map[string]*domain.PricingUnit{"un_ea": {ID: "un_ea", IsBaseUnit: true}},
		UnitGroupUnits: map[string]map[string]*domain.PricingUnitGroupUnit{
			"ug_test": {"un_ea": {UnitGroupID: "ug_test", UnitID: "un_ea"}},
		},
	}
}

// expectResequence mocks the line re-sequence a delete performs. It returns a single line
// already at position 1, so no SetLineItemNumber calls are needed (re-sequence is a no-op).
func (suite *SalesOrderLineSvcTestSuite) expectResequence() {
	suite.lineRepo.EXPECT().
		GetLineOrder(gomock.Any(), "or_test").
		Return([]*domain.SalesOrderLinePosition{{ID: "orl_rem", LineItemNumber: 1, IsSystem: true}}, nil).
		Times(1)
}

// expectCreateLinePricing mocks the one pricing-bundle load a create performs to resolve
// the line's price/cost from the product.
func (suite *SalesOrderLineSvcTestSuite) expectCreateLinePricing() {
	suite.pricingRepo.EXPECT().
		LoadPricingBundle(gomock.Any(), gomock.Any()).
		Return(createLinePricingBundle(), nil).
		Times(1)
}

// baseCreateLineParams returns a valid set of create-line params that every
// CreateSalesOrderLine test can start from and mutate.
func baseCreateLineParams() domain.CreateSalesOrderLineParams {
	return domain.CreateSalesOrderLineParams{
		SalesOrderID:               "or_test",
		ProductID:                  "prod_test",
		ProductSKU:                 "SKU-1",
		QuantityValue:              "10",
		QuantityUnitID:             "un_ea",
		UnitPriceValue:             "9.996", // rounds to 10.00 — verifies service rounds before persisting
		UnitPriceNumeratorUnitID:   "un_usd",
		UnitPriceDenominatorUnitID: "un_ea",
	}
}

// --- CreateSalesOrderLine ---

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_Success_NoPick() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	// Service validates the order exists via Get.
	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)

	// The line is priced from the product before being persisted.
	suite.expectCreateLinePricing()

	// Create returns the persisted line. Assert price rounding happened before reaching the repo.
	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, params domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			suite.Equal("10", params.UnitPriceValue, "unit price should be rounded before reaching repo — input was 9.995")
			// Actually 9.995 rounds to 10.00 (bankers/half-away), which formats as "10".
			return &domain.SalesOrderLine{ID: id, SalesOrderID: "or_test", ProductSKU: "SKU-1"}, nil
		}).
		Times(1)

	// No active pick.
	suite.orderRepo.EXPECT().
		GetPickID(gomock.Any(), "or_test").
		Return(nil, nil).
		Times(1)

	suite.expectCacheSuccess()

	result, apiErr := suite.svc.CreateSalesOrderLine(ctx, baseCreateLineParams())

	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal("or_test", result.SalesOrderID)
}

// When the caller omits the unit price, the line is priced server-side from the product
// (list price $10/ea here) and the unit cost is derived from the product — no caller price.
func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_CalculatesPriceWhenOmitted() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	params := baseCreateLineParams()
	params.UnitPriceValue = "" // omitted → server-calculated
	params.UnitPriceNumeratorUnitID = ""
	params.UnitPriceDenominatorUnitID = ""

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", BuyerAccountID: "ac_buyer"}, nil).
		Times(1)

	suite.expectCreateLinePricing()

	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			suite.Equal("10", p.UnitPriceValue, "price calculated from the product list price")
			suite.Equal("un_usd", p.UnitPriceNumeratorUnitID)
			suite.Equal("un_ea", p.UnitPriceDenominatorUnitID)
			suite.NotNil(p.UnitCostValue)
			suite.Equal("4", *p.UnitCostValue, "cost derived from the product")
			suite.NotNil(p.ItemID)
			suite.Equal("it_test", *p.ItemID, "item derived from the product")
			return &domain.SalesOrderLine{ID: id}, nil
		}).
		Times(1)

	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(nil, nil).Times(1)
	suite.expectCacheSuccess()

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, params)
	suite.Nil(apiErr)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_RoundsUnitPriceAndCostToCent() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	params := baseCreateLineParams()
	params.UnitPriceValue = "1.234" // internal override, rounds to 1.23

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)

	suite.expectCreateLinePricing()

	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			// The internal override is rounded to the cent and honored as the unit price.
			suite.Equal("1.23", p.UnitPriceValue)
			// The unit cost is always derived from the product (never the caller's input).
			suite.NotNil(p.UnitCostValue)
			suite.Equal("4", *p.UnitCostValue)
			return &domain.SalesOrderLine{ID: id}, nil
		}).
		Times(1)

	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(nil, nil).Times(1)
	suite.expectCacheSuccess()

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, params)
	suite.Nil(apiErr)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_CreatesPickLineWhenActivePickExists() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)

	suite.expectCreateLinePricing()

	saleType := string(constants.ProductTypeCodeSale)
	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, _ domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			return &domain.SalesOrderLine{ID: id, SalesOrderID: "or_test", ProductTypeCode: &saleType}, nil
		}).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)

	// Outstanding (ordered 10 - packed 0) > 0 → a new placeholder pick line is created.
	suite.pickLineRepo.EXPECT().
		GetOrderLinePackProgress(gomock.Any(), gomock.Any()).
		Return("10", "0", "un_ea", nil).
		Times(1)

	// No unpacked pick line exists yet, so a new one must be created.
	suite.pickLineRepo.EXPECT().
		HasUnpackedPickLineForOrderLine(gomock.Any(), gomock.Any()).
		Return(false, nil).
		Times(1)

	// The placeholder pick line is seeded at 0 picked (outstanding is only the >0 guard).
	suite.lineRepo.EXPECT().
		CreateQuantity(gomock.Any(), gomock.Any(), "0", "un_ea").
		Return(nil).
		Times(1)

	suite.pickLineRepo.EXPECT().
		CreateForRemaining(gomock.Any(), gomock.Any(), gomock.Any(), pickID, gomock.Any()).
		Return(nil).
		Times(1)

	// Outstanding work means the pick is reopened.
	suite.pickRepo.EXPECT().ClearFinishedAt(gomock.Any(), "ac_test", pickID).Return(nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, baseCreateLineParams())
	suite.Nil(apiErr)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_SkipsPickLineWhenAlreadyUnpacked() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)

	suite.expectCreateLinePricing()

	saleType := string(constants.ProductTypeCodeSale)
	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, _ domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			return &domain.SalesOrderLine{ID: id, ProductTypeCode: &saleType}, nil
		}).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	suite.pickLineRepo.EXPECT().
		GetOrderLinePackProgress(gomock.Any(), gomock.Any()).
		Return("5", "0", "un_ea", nil).
		Times(1)
	// An unpacked pick line is already present → skip creating another.
	suite.pickLineRepo.EXPECT().
		HasUnpackedPickLineForOrderLine(gomock.Any(), gomock.Any()).
		Return(true, nil).
		Times(1)
	// Outstanding work still means the pick is reopened.
	suite.pickRepo.EXPECT().ClearFinishedAt(gomock.Any(), "ac_test", pickID).Return(nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, baseCreateLineParams())
	suite.Nil(apiErr)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_MissingIdentity() {
	_, apiErr := suite.svc.CreateSalesOrderLine(context.Background(), baseCreateLineParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_test"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_ro",
			AccountID:    new("ac_test"),
			RoleType:     &customCode,
			Permissions:  map[string]bool{"sales_orders:read": true},
		},
	})

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, baseCreateLineParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_ExternalTargetChecksEditAccess() {
	// External target: actor belongs to a *different* account than the target.
	// The service should route through the EditAccess mediator before touching the repos.
	adminCode := string(constants.RoleTypeAdmin)
	actorAcct := "ac_actor"
	targetAcct := "ac_target"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAcct},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_ext",
			AccountID:    &actorAcct,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"sales_orders:update": true,
				"customers:update":    true,
				"suppliers:update":    true,
			},
		},
	})
	ctx = salesOrderLineIdempotencyCtx(ctx, "/core.CoreService/CreateSalesOrderLine")

	// EditAccess rejects → service must NOT touch the idempotency mediator or any repo.
	suite.editAccessMed.EXPECT().
		CheckEditAccess(gomock.Any(), actorAcct, targetAcct).
		Return(apierror.NewAuthorizationError("No edit access.")).
		Times(1)

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, baseCreateLineParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_OrderNotFound() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(nil, apierror.NewResourceNotFoundError("Sales order not found.")).
		Times(1)

	suite.expectCacheError()

	_, apiErr := suite.svc.CreateSalesOrderLine(ctx, baseCreateLineParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

// --- UpdateSalesOrderLine ---

// baseUpdateLineParams returns a valid set of update-line params.
func baseUpdateLineParams() domain.UpdateSalesOrderLineParams {
	price := "5.005" // rounds to 5.01
	return domain.UpdateSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
		UnitPriceValue:   &price,
	}
}

func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_Success_NoPick() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		Get(gomock.Any(), "orl_test").
		Return(&domain.SalesOrderLine{ID: "orl_test", UnitPriceValue: "5.00"}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.UpdateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			suite.NotNil(p.UnitPriceValue)
			// 5.005 → 5.01 when rounded half-away; strconv.FormatFloat without trailing zero = "5.01".
			suite.Equal("5.01", *p.UnitPriceValue)
			return &domain.SalesOrderLine{ID: "orl_test", UnitPriceValue: "5.01"}, nil
		}).
		Times(1)

	suite.orderRepo.EXPECT().
		GetPickID(gomock.Any(), "or_test").
		Return(nil, nil).
		Times(1)

	suite.expectCacheSuccess()

	result, apiErr := suite.svc.UpdateSalesOrderLine(ctx, baseUpdateLineParams())
	suite.Nil(apiErr)
	suite.NotNil(result)
}

func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_LineNotInOrder() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(false, nil).
		Times(1)

	suite.expectCacheError()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, baseUpdateLineParams())
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_RecomputesPickLineWhenPickExists() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	saleType := string(constants.ProductTypeCodeSale)
	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", ProductTypeCode: &saleType}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	// Outstanding (ordered 13 - packed 10) > 0 → open a placeholder pick line + reopen.
	suite.pickLineRepo.EXPECT().
		GetOrderLinePackProgress(gomock.Any(), "orl_test").
		Return("13", "10", "un_ea", nil).
		Times(1)
	suite.pickLineRepo.EXPECT().
		HasUnpackedPickLineForOrderLine(gomock.Any(), "orl_test").
		Return(false, nil).
		Times(1)
	// Placeholder pick line seeded at 0 picked (outstanding is only the >0 guard).
	suite.lineRepo.EXPECT().CreateQuantity(gomock.Any(), gomock.Any(), "0", "un_ea").Return(nil).Times(1)
	suite.pickLineRepo.EXPECT().
		CreateForRemaining(gomock.Any(), gomock.Any(), gomock.Any(), pickID, "orl_test").
		Return(nil).
		Times(1)
	suite.pickRepo.EXPECT().ClearFinishedAt(gomock.Any(), "ac_test", pickID).Return(nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, baseUpdateLineParams())
	suite.Nil(apiErr)
}

// A decrease that drops the order line back to (or below) the already-packed quantity
// leaves no outstanding work: the open remainder pick line is deleted and the pick is
// finished when everything left is packed.
func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_DeletesOpenPickLineOnDecrease() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	saleType := string(constants.ProductTypeCodeSale)
	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", ProductTypeCode: &saleType}, nil).Times(1)
	suite.lineRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	// Outstanding (ordered 10 - packed 10) <= 0 → the open remainder line is surplus.
	suite.pickLineRepo.EXPECT().
		GetOrderLinePackProgress(gomock.Any(), "orl_test").
		Return("10", "10", "un_ea", nil).
		Times(1)
	suite.pickLineRepo.EXPECT().DeleteUnpackedForOrderLine(gomock.Any(), "orl_test").Return(nil).Times(1)
	suite.pickRepo.EXPECT().MarkFinishedIfAllPacked(gomock.Any(), pickID).Return(nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, baseUpdateLineParams())
	suite.Nil(apiErr)
}

// A freight/credit (system) line is never picked, so updating one on a picked order
// must NOT seed a placeholder pick line — matching legacy, which gates pick-line
// creation on product.productType === 'sale'.
func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_SkipsPickLineForSystemLine() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	shippingType := string(constants.ProductTypeCodeShipping)
	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", ProductTypeCode: &shippingType}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	// No CalculateRemainingForOrderLine / CreateForRemaining expectations: gomock.Finish
	// fails if pick-line creation is attempted for this system line.

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, baseUpdateLineParams())
	suite.Nil(apiErr)
}

// Editing an order line's quantity must push the new value + unit into the invoice
// and shipment lines referencing it, and relabel pick line units — all three snapshot
// quantity at creation and would otherwise go stale.
func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_SyncsInvoiceLineQuantities() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	qty := "12"
	unit := "un_cs"
	params := domain.UpdateSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
		QuantityValue:    &qty,
		QuantityUnitID:   &unit,
	}

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "10", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "12", QuantityUnitID: "un_cs"}, nil).
		Times(1)

	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(nil, nil).Times(1)

	// The syncs must carry the pre-update value (the mirror guard) and the
	// *updated* line's quantity value and unit; the unit change also relabels
	// pick line quantities (value handled by pick reconciliation, not sync).
	suite.lineRepo.EXPECT().
		SyncPickLineQuantityUnits(gomock.Any(), "orl_test", "un_cs").
		Return(nil).
		Times(1)
	suite.lineRepo.EXPECT().
		SyncInvoiceLineQuantities(gomock.Any(), "orl_test", "10", "12", "un_cs").
		Return(nil).
		Times(1)
	suite.lineRepo.EXPECT().
		SyncShipmentLineQuantities(gomock.Any(), "orl_test", "10", "12", "un_cs").
		Return(nil).
		Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, params)
	suite.Nil(apiErr)
}

// A unit-only change (no value change) must still sync — this was the reported bug:
// changing the unit on an order line left the invoice line showing the old unit.
func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_SyncsInvoiceLineOnUnitOnlyChange() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	unit := "un_cs"
	params := domain.UpdateSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
		QuantityUnitID:   &unit,
	}

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "10", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "10", QuantityUnitID: "un_cs"}, nil).
		Times(1)

	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(nil, nil).Times(1)

	suite.lineRepo.EXPECT().
		SyncPickLineQuantityUnits(gomock.Any(), "orl_test", "un_cs").
		Return(nil).
		Times(1)
	suite.lineRepo.EXPECT().
		SyncInvoiceLineQuantities(gomock.Any(), "orl_test", "10", "10", "un_cs").
		Return(nil).
		Times(1)
	suite.lineRepo.EXPECT().
		SyncShipmentLineQuantities(gomock.Any(), "orl_test", "10", "10", "un_cs").
		Return(nil).
		Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, params)
	suite.Nil(apiErr)
}

// A value-only change must not relabel pick line units — the pick relabel only runs
// when the unit actually changed; value deltas are the pick reconciliation's job.
// gomock's strict mode fails the test if SyncPickLineQuantityUnits is called.
func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_ValueOnlyChangeSkipsPickUnitRelabel() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	qty := "12"
	params := domain.UpdateSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
		QuantityValue:    &qty,
	}

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "10", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "12", QuantityUnitID: "un_ea"}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		SyncInvoiceLineQuantities(gomock.Any(), "orl_test", "10", "12", "un_ea").
		Return(nil).
		Times(1)
	suite.lineRepo.EXPECT().
		SyncShipmentLineQuantities(gomock.Any(), "orl_test", "10", "12", "un_ea").
		Return(nil).
		Times(1)

	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(nil, nil).Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, params)
	suite.Nil(apiErr)
}

// A sync failure must fail (and roll back) the whole update — a half-applied edit
// where the order changed but the invoice didn't is exactly the drift this prevents.
func (suite *SalesOrderLineSvcTestSuite) TestUpdateSalesOrderLine_FailsWhenInvoiceSyncFails() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/UpdateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	qty := "12"
	params := domain.UpdateSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
		QuantityValue:    &qty,
	}

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "10", QuantityUnitID: "un_ea"}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test", QuantityValue: "12", QuantityUnitID: "un_ea"}, nil).
		Times(1)

	// The sync runs before pick reconciliation, so a failure short-circuits the tx:
	// no GetPickID / SyncShipmentLineQuantities expectations — gomock fails if called.
	suite.lineRepo.EXPECT().
		SyncInvoiceLineQuantities(gomock.Any(), "orl_test", "10", "12", "un_ea").
		Return(apierror.NewInternalError(nil, "boom")).
		Times(1)

	suite.expectCacheError()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, params)
	suite.NotNil(apiErr)
}

// --- DeleteSalesOrderLine ---

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_Success() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	// Not fulfilled, no CompletedAt.
	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeEstimate)}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		HasShipmentAgainstOrderLine(gomock.Any(), "orl_test").
		Return(false, nil).
		Times(1)

	// No shipped shipments — no admin gate needed.
	suite.orderRepo.EXPECT().
		HasShippedShipment(gomock.Any(), "or_test").
		Return(false, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		Get(gomock.Any(), "orl_test").
		Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).
		Times(1)

	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrderLine, "orl_test", gomock.Any()).
		Return(nil).
		Times(1)

	suite.lineRepo.EXPECT().
		DeleteCascade(gomock.Any(), "orl_test").
		Return(nil).
		Times(1)

	suite.expectResequence()

	// An estimate order has no pick, so the pick reconciliation is a no-op.
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(nil, nil).Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.Nil(apiErr)
}

// Deleting a line from an issued order whose pick still has other lines finishes the
// pick when everything that remains is packed.
func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_FinishesPickWhenLinesRemain() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeIssued)}, nil).
		Times(1)
	suite.lineRepo.EXPECT().HasShipmentAgainstOrderLine(gomock.Any(), "orl_test").Return(false, nil).Times(1)
	suite.orderRepo.EXPECT().HasShippedShipment(gomock.Any(), "or_test").Return(false, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).Times(1)
	suite.deletedRecordRepo.EXPECT().Create(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrderLine, "orl_test", gomock.Any()).Return(nil).Times(1)
	suite.lineRepo.EXPECT().DeleteCascade(gomock.Any(), "orl_test").Return(nil).Times(1)

	suite.expectResequence()

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	// Two lines remain in the pick → finish it if everything left is packed.
	suite.pickRepo.EXPECT().CountLines(gomock.Any(), pickID).Return(int64(2), nil).Times(1)
	suite.pickRepo.EXPECT().MarkFinishedIfAllPacked(gomock.Any(), pickID).Return(nil).Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.Nil(apiErr)
}

// Deleting the last line that leaves the pick empty deletes the pick and reverts the
// order to estimate, releasing reserved inventory.
func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_UnissuesWhenPickEmptied() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", Number: "SO-1", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeIssued)}, nil).
		Times(1)
	suite.lineRepo.EXPECT().HasShipmentAgainstOrderLine(gomock.Any(), "orl_test").Return(false, nil).Times(1)
	suite.orderRepo.EXPECT().HasShippedShipment(gomock.Any(), "or_test").Return(false, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).Times(1)
	suite.deletedRecordRepo.EXPECT().Create(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrderLine, "orl_test", gomock.Any()).Return(nil).Times(1)
	suite.lineRepo.EXPECT().DeleteCascade(gomock.Any(), "orl_test").Return(nil).Times(1)

	suite.expectResequence()

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	// No lines remain → tear down the pick and unissue the order.
	suite.pickRepo.EXPECT().CountLines(gomock.Any(), pickID).Return(int64(0), nil).Times(1)
	suite.orderRepo.EXPECT().DeletePickBySalesOrder(gomock.Any(), "or_test").Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeleteInventoryAllocationsByReservedIssues(gomock.Any(), "ac_test", "or_test").Return(nil).Times(1)
	suite.orderRepo.EXPECT().DeleteReservedInventoryIssues(gomock.Any(), "ac_test", "or_test").Return(nil).Times(1)
	suite.orderRepo.EXPECT().
		UpdateStatus(gomock.Any(), "ac_test", "or_test", string(constants.SalesOrderStatusCodeEstimate), nil, nil).
		Return(nil).
		Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.Nil(apiErr)
}

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_BlockedWhenHasShippedShipment_NonAdmin() {
	accountID := "ac_test"
	operatorRole := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &accountID,
			RoleType:     &operatorRole,
			Permissions: map[string]bool{
				"sales_orders:read":   true,
				"sales_orders:create": true,
				"sales_orders:update": true,
				"sales_orders:delete": true,
			},
		},
	})

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeIssued)}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		HasShipmentAgainstOrderLine(gomock.Any(), "orl_test").
		Return(false, nil).
		Times(1)

	// There is a shipped shipment — admin check will fire and fail for a non-admin.
	suite.orderRepo.EXPECT().
		HasShippedShipment(gomock.Any(), "or_test").
		Return(true, nil).
		Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_BlockedWhenOrderFulfilled() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeFulfilled)}, nil).
		Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_BlockedWhenShipmentAgainst() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeIssued)}, nil).
		Times(1)

	// The line is part of a shipment (packed or shipped) → deletion is blocked, and the
	// line's pick lines are NOT re-sequenced or reconciled (the guard returns first).
	suite.lineRepo.EXPECT().
		HasShipmentAgainstOrderLine(gomock.Any(), "orl_test").
		Return(true, nil).
		Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_LineNotInOrder() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(false, nil).
		Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_AlreadyDeleted() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeEstimate)}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		HasShipmentAgainstOrderLine(gomock.Any(), "orl_test").
		Return(false, nil).
		Times(1)

	// No shipped shipments — no admin gate needed.
	suite.orderRepo.EXPECT().
		HasShippedShipment(gomock.Any(), "or_test").
		Return(false, nil).
		Times(1)

	// Get on the line returns not-found → service checks the tombstone.
	suite.lineRepo.EXPECT().
		Get(gomock.Any(), "orl_test").
		Return(nil, apierror.NewResourceNotFoundError("Sales order line not found.")).
		Times(1)

	// Tombstone present → service surfaces the "already deleted" semantic error.
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeSalesOrderLine, "orl_test").
		Return(true, nil).
		Times(1)

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.NotNil(apiErr)
	// `AlreadyDeletedError` is a specific construction; the stable assertion is the error message,
	// which the endpoint layer relies on to render the 409 response.
	suite.Contains(apiErr.PublicMessage, "already been deleted")
}

// --- ReorderSalesOrderLines ---

// reorderPositions is the standard fixture: two product lines (A, B) followed by
// a freight (system) line (F), in their current positions.
func reorderPositions() []*domain.SalesOrderLinePosition {
	return []*domain.SalesOrderLinePosition{
		{ID: "orln_a", LineItemNumber: 1, IsSystem: false},
		{ID: "orln_b", LineItemNumber: 2, IsSystem: false},
		{ID: "orln_f", LineItemNumber: 3, IsSystem: true},
	}
}

func (suite *SalesOrderLineSvcTestSuite) TestReorderSalesOrderLines_Success_KeepsSystemLinesAtBottom() {
	ctx := salesOrderLineCtx("ac_test")

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		GetLineOrder(gomock.Any(), "or_test").
		Return(reorderPositions(), nil).
		Times(1)

	// Swap A and B: B moves to 1, A moves to 2. The freight line stays at 3 and is
	// never touched because its number does not change.
	suite.lineRepo.EXPECT().SetLineItemNumber(gomock.Any(), "orln_b", int32(1)).Return(nil).Times(1)
	suite.lineRepo.EXPECT().SetLineItemNumber(gomock.Any(), "orln_a", int32(2)).Return(nil).Times(1)

	suite.lineRepo.EXPECT().
		List(gomock.Any(), "or_test").
		Return([]*domain.SalesOrderLine{{ID: "orln_b"}, {ID: "orln_a"}, {ID: "orln_f"}}, nil).
		Times(1)

	result, apiErr := suite.svc.ReorderSalesOrderLines(ctx, domain.ReorderSalesOrderLinesParams{
		SalesOrderID: "or_test",
		LineIDs:      []string{"orln_b", "orln_a"},
	})

	suite.Nil(apiErr)
	suite.Require().Len(result, 3)
	suite.Equal("orln_b", result[0].ID)
	suite.Equal("orln_f", result[2].ID)
}

func (suite *SalesOrderLineSvcTestSuite) TestReorderSalesOrderLines_RejectsIncludingSystemLine() {
	ctx := salesOrderLineCtx("ac_test")

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)
	suite.lineRepo.EXPECT().GetLineOrder(gomock.Any(), "or_test").Return(reorderPositions(), nil).Times(1)

	// Submitting the freight line among the product ordering is rejected — count matches
	// the product-line count but the freight ID is not a product line.
	_, apiErr := suite.svc.ReorderSalesOrderLines(ctx, domain.ReorderSalesOrderLinesParams{
		SalesOrderID: "or_test",
		LineIDs:      []string{"orln_a", "orln_f"},
	})

	suite.NotNil(apiErr)
	suite.Contains(apiErr.PublicMessage, "does not belong")
}

func (suite *SalesOrderLineSvcTestSuite) TestReorderSalesOrderLines_RejectsWrongCount() {
	ctx := salesOrderLineCtx("ac_test")

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)
	suite.lineRepo.EXPECT().GetLineOrder(gomock.Any(), "or_test").Return(reorderPositions(), nil).Times(1)

	// Only one of the two product lines is supplied.
	_, apiErr := suite.svc.ReorderSalesOrderLines(ctx, domain.ReorderSalesOrderLinesParams{
		SalesOrderID: "or_test",
		LineIDs:      []string{"orln_a"},
	})

	suite.NotNil(apiErr)
	suite.Contains(apiErr.PublicMessage, "exactly once")
}

func (suite *SalesOrderLineSvcTestSuite) TestReorderSalesOrderLines_RejectsDuplicate() {
	ctx := salesOrderLineCtx("ac_test")

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)
	suite.lineRepo.EXPECT().GetLineOrder(gomock.Any(), "or_test").Return(reorderPositions(), nil).Times(1)

	_, apiErr := suite.svc.ReorderSalesOrderLines(ctx, domain.ReorderSalesOrderLinesParams{
		SalesOrderID: "or_test",
		LineIDs:      []string{"orln_a", "orln_a"},
	})

	suite.NotNil(apiErr)
	suite.Contains(apiErr.PublicMessage, "Duplicate")
}
