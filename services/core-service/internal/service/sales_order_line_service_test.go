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
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewSalesOrderRepo().Return(suite.orderRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewSalesOrderLineRepo().Return(suite.lineRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPickLineRepo().Return(suite.pickLineRepo).AnyTimes()
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

func (suite *SalesOrderLineSvcTestSuite) TestCreateSalesOrderLine_RoundsUnitPriceAndCostToCent() {
	ctx := salesOrderLineIdempotencyCtx(
		salesOrderLineCtx("ac_test"),
		"/core.CoreService/CreateSalesOrderLine",
	)

	suite.expectIdempotencyStarted()

	params := baseCreateLineParams()
	params.UnitPriceValue = "1.234" // rounds to 1.23
	cost := "2.1899999"             // rounds to 2.19
	params.UnitCostValue = &cost
	numID := "un_usd"
	denID := "un_ea"
	params.UnitCostNumeratorUnitID = &numID
	params.UnitCostDenominatorUnitID = &denID

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test"}, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			suite.Equal("1.23", p.UnitPriceValue)
			suite.NotNil(p.UnitCostValue)
			suite.Equal("2.19", *p.UnitCostValue)
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

	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, _ domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			return &domain.SalesOrderLine{ID: id, SalesOrderID: "or_test"}, nil
		}).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)

	// Active pick → remaining quantity pulled from the repo.
	suite.pickLineRepo.EXPECT().
		CalculateRemainingForOrderLine(gomock.Any(), gomock.Any()).
		Return("10", "un_ea", nil).
		Times(1)

	// No unpacked pick line exists yet, so a new one must be created.
	suite.pickLineRepo.EXPECT().
		HasUnpackedPickLineForOrderLine(gomock.Any(), gomock.Any()).
		Return(false, nil).
		Times(1)

	suite.lineRepo.EXPECT().
		CreateQuantity(gomock.Any(), gomock.Any(), "10", "un_ea").
		Return(nil).
		Times(1)

	suite.pickLineRepo.EXPECT().
		CreateForRemaining(gomock.Any(), gomock.Any(), gomock.Any(), pickID, gomock.Any()).
		Return(nil).
		Times(1)

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

	suite.lineRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, _ domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
			return &domain.SalesOrderLine{ID: id}, nil
		}).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	suite.pickLineRepo.EXPECT().
		CalculateRemainingForOrderLine(gomock.Any(), gomock.Any()).
		Return("5", "un_ea", nil).
		Times(1)
	// An unpacked pick line is already present → skip creating another.
	suite.pickLineRepo.EXPECT().
		HasUnpackedPickLineForOrderLine(gomock.Any(), gomock.Any()).
		Return(true, nil).
		Times(1)

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
			AccountID:    ptr("ac_test"),
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

	suite.lineRepo.EXPECT().IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").Return(true, nil).Times(1)
	suite.lineRepo.EXPECT().Get(gomock.Any(), "orl_test").Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).Times(1)
	suite.lineRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.SalesOrderLine{ID: "orl_test"}, nil).
		Times(1)

	pickID := "pk_test"
	suite.orderRepo.EXPECT().GetPickID(gomock.Any(), "or_test").Return(&pickID, nil).Times(1)
	suite.pickLineRepo.EXPECT().
		CalculateRemainingForOrderLine(gomock.Any(), "orl_test").
		Return("3", "un_ea", nil).
		Times(1)
	suite.pickLineRepo.EXPECT().
		HasUnpackedPickLineForOrderLine(gomock.Any(), "orl_test").
		Return(false, nil).
		Times(1)
	suite.lineRepo.EXPECT().CreateQuantity(gomock.Any(), gomock.Any(), "3", "un_ea").Return(nil).Times(1)
	suite.pickLineRepo.EXPECT().
		CreateForRemaining(gomock.Any(), gomock.Any(), gomock.Any(), pickID, "orl_test").
		Return(nil).
		Times(1)

	suite.expectCacheSuccess()

	_, apiErr := suite.svc.UpdateSalesOrderLine(ctx, baseUpdateLineParams())
	suite.Nil(apiErr)
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
		HasShippedAgainstOrderLine(gomock.Any(), "orl_test").
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

	apiErr := suite.svc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: "orl_test",
		SalesOrderID:     "or_test",
	})
	suite.Nil(apiErr)
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

func (suite *SalesOrderLineSvcTestSuite) TestDeleteSalesOrderLine_BlockedWhenShippedAgainst() {
	ctx := salesOrderLineCtx("ac_test")

	suite.lineRepo.EXPECT().
		IsInOrder(gomock.Any(), "orl_test", "or_test", "ac_test").
		Return(true, nil).
		Times(1)

	suite.orderRepo.EXPECT().
		Get(gomock.Any(), "ac_test", "or_test").
		Return(&domain.SalesOrder{ID: "or_test", SalesOrderStatusCode: string(constants.SalesOrderStatusCodeIssued)}, nil).
		Times(1)

	// A shipment has already gone out against this line → deletion must be blocked.
	suite.lineRepo.EXPECT().
		HasShippedAgainstOrderLine(gomock.Any(), "orl_test").
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
		HasShippedAgainstOrderLine(gomock.Any(), "orl_test").
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

// ptr is a tiny helper so tests can inline pointer literals for nullable params.
func ptr[T any](v T) *T { return &v }
