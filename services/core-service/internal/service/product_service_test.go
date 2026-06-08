package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/pagination"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// --- Local test stubs (matches pattern in unit_service_test.go) ---

type productStubOutboxRepo struct{}

func (s *productStubOutboxRepo) Create(_ context.Context, _ messaging.OutboxMessageInput) (int64, error) {
	return 0, nil
}

type productStubTxManager struct {
	factory domain.RepoFactory
}

func (m *productStubTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.factory)
}

// --- Suite ---

type ProductSvcTestSuite struct {
	suite.Suite
	ctrl                *gomock.Controller
	productSvc          domain.ProductSvc
	productRepo         *repositorymock.MockProductRepo
	itemRepo            *repositorymock.MockItemRepo
	inventoryMutRepo    *repositorymock.MockInventoryMutationRepo
	productLineRepo     *repositorymock.MockProductLineRepo
	deletedRecordRepo   *repositorymock.MockDeletedRecordRepo
	accountRelationRepo *repositorymock.MockAccountRelationRepo
	unitRepo            *repositorymock.MockUnitRepo
	repoFactory         *factorymock.MockRepoFactory
	mediatorFactory     *factorymock.MockMediatorFactory
	idempotencyMed      *mediatormock.MockIdempotencyMed
	readAccessMed       *mediatormock.MockReadAccessMed
}

func (s *ProductSvcTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.productRepo = repositorymock.NewMockProductRepo(s.ctrl)
	s.itemRepo = repositorymock.NewMockItemRepo(s.ctrl)
	s.inventoryMutRepo = repositorymock.NewMockInventoryMutationRepo(s.ctrl)
	s.productLineRepo = repositorymock.NewMockProductLineRepo(s.ctrl)
	s.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(s.ctrl)
	s.accountRelationRepo = repositorymock.NewMockAccountRelationRepo(s.ctrl)
	s.unitRepo = repositorymock.NewMockUnitRepo(s.ctrl)

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewProductRepo().Return(s.productRepo).AnyTimes()
	s.repoFactory.EXPECT().NewItemRepo().Return(s.itemRepo).AnyTimes()
	s.repoFactory.EXPECT().NewInventoryMutationRepo().Return(s.inventoryMutRepo).AnyTimes()
	s.repoFactory.EXPECT().NewProductLineRepo().Return(s.productLineRepo).AnyTimes()
	s.repoFactory.EXPECT().NewDeletedRecordRepo().Return(s.deletedRecordRepo).AnyTimes()
	s.repoFactory.EXPECT().NewAccountRelationRepo().Return(s.accountRelationRepo).AnyTimes()
	s.repoFactory.EXPECT().NewUnitRepo().Return(s.unitRepo).AnyTimes()
	s.repoFactory.EXPECT().NewOutboxRepo().Return(&productStubOutboxRepo{}).AnyTimes()

	s.idempotencyMed = mediatormock.NewMockIdempotencyMed(s.ctrl)
	s.readAccessMed = mediatormock.NewMockReadAccessMed(s.ctrl)
	s.mediatorFactory = factorymock.NewMockMediatorFactory(s.ctrl)
	s.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: s.idempotencyMed,
		ReadAccess:  s.readAccessMed,
	}).AnyTimes()

	s.productSvc = NewProductSvc(&ProductSvcConfig{
		Repos:           s.repoFactory,
		MediatorFactory: s.mediatorFactory,
		TxManager:       &productStubTxManager{factory: s.repoFactory},
	})
}

func (s *ProductSvcTestSuite) TearDownSuite() {
	s.ctrl.Finish()
}

func TestProductSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProductSvcTestSuite))
}

// --- Identity context builders ---

// internalProductIdentityCtx returns a ctx with an internal admin actor
// targeting the given account. Admins bypass per-permission checks, which
// keeps non-permission tests focused on business logic.
func internalProductIdentityCtx(accountID string) context.Context {
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
				"items:read":     true,
				"items:create":   true,
				"items:update":   true,
				"items:delete":   true,
				"customers:read": true,
				"suppliers:read": true,
			},
		},
	})
}

// readOnlyProductIdentityCtx returns an internal non-admin actor with only
// items:read (no create/update/delete). Used to assert permission gating.
func readOnlyProductIdentityCtx(accountID string) context.Context {
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &accountID,
			RoleType:     &customCode,
			Permissions: map[string]bool{
				"items:read": true,
			},
		},
	})
}

// customerProductIdentityCtx returns a customer actor whose account differs
// from the target (external customer viewing an owner's catalog).
func customerProductIdentityCtx(customerAccountID, targetAccountID string) context.Context {
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_cust123",
			AccountID:    &customerAccountID,
			RoleType:     &customCode,
			Permissions:  map[string]bool{},
		},
	})
}

func productIdempotencyCtx(ctx context.Context) context.Context {
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithHandler(ctx, "/core.CoreService/TestHandler")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

// --- Idempotency helpers ---

func (s *ProductSvcTestSuite) expectIdempotencyStarted() {
	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

func (s *ProductSvcTestSuite) expectIdempotencyFinishedWithProduct(product *domain.ProductFull) {
	body, err := json.Marshal(product)
	s.Require().NoError(err)
	code := 200
	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  body,
		}, nil).
		Times(1)
}

func (s *ProductSvcTestSuite) expectCacheSuccess() {
	s.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).
		Return(nil).
		Times(1)
}

func (s *ProductSvcTestSuite) expectCacheError() {
	s.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

func (s *ProductSvcTestSuite) expectAttachAttributes(itemID string) {
	s.itemRepo.EXPECT().
		Get(gomock.Any(), gomock.AssignableToTypeOf(domain.GetItemParams{})).
		DoAndReturn(func(_ context.Context, params domain.GetItemParams) (*domain.Item, *apierror.APIError) {
			s.Equal("ac_test123", params.AccountID)
			s.Equal(itemID, params.ItemID)
			s.Equal([]string{"attributes"}, params.Includes)
			return &domain.Item{ID: itemID}, nil
		}).
		Times(1)
}

func (s *ProductSvcTestSuite) expectAttachAttributesForCreatedItem() {
	s.itemRepo.EXPECT().
		Get(gomock.Any(), gomock.AssignableToTypeOf(domain.GetItemParams{})).
		DoAndReturn(func(_ context.Context, params domain.GetItemParams) (*domain.Item, *apierror.APIError) {
			s.Equal("ac_test123", params.AccountID)
			s.NotEmpty(params.ItemID)
			s.Equal([]string{"attributes"}, params.Includes)
			return &domain.Item{ID: params.ItemID}, nil
		}).
		Times(1)
}

// =============================================================================
// CreateProduct
// =============================================================================

func (s *ProductSvcTestSuite) createdProduct(itemID string) *domain.ProductFull {
	return &domain.ProductFull{
		ID:              "prod_created",
		ItemID:          itemID,
		ProductTypeCode: "sale",
		IsPortalReady:   true,
		Item:            &domain.Item{ID: itemID, SKU: "SKU-NEW"},
		ProductType:     &domain.ProductType{Code: "sale", Name: "Sale"},
	}
}

func (s *ProductSvcTestSuite) TestCreateProduct_Success() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().
		CheckSKUExists(gomock.Any(), "ac_test123", "SKU-NEW", "").
		Return(false, nil).
		Times(1)
	s.itemRepo.EXPECT().
		GetCategoryBaseUnitID(gomock.Any(), "cat_123").
		Return("un_base", nil).
		Times(1)

	// unit_price and unit_cost validate their numerator/denominator dimensions.
	s.unitRepo.EXPECT().
		GetDimensionCodes(gomock.Any(), gomock.Any()).
		Return(map[string]string{"un_usd": "currency", "un_each": "discrete"}, nil).
		Times(2)

	// Caller-supplied unit_price/unit_cost flow through to InsertRate verbatim.
	// Burn rate is always initialized to "0" per day and recomputed later.
	s.productRepo.EXPECT().
		InsertRate(gomock.Any(), gomock.Any(), "1.50", "un_usd", "un_each").
		Return(nil).
		Times(1)
	s.productRepo.EXPECT().
		InsertRate(gomock.Any(), gomock.Any(), "0.75", "un_usd", "un_each").
		Return(nil).
		Times(1)
	s.productRepo.EXPECT().
		InsertRate(gomock.Any(), gomock.Any(), "0", "un_base", "day").
		Return(nil).
		Times(1)

	s.productRepo.EXPECT().
		InsertItem(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, itemID string, params domain.CreateProductParams) *apierror.APIError {
			s.Equal("ac_test123", params.AccountID)
			s.Equal("SKU-NEW", params.SKU)
			s.Equal("cat_123", params.CategoryID)
			s.NotEmpty(itemID)
			return nil
		}).
		Times(1)

	s.productRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, productID, itemID string, params domain.CreateProductParams) (*domain.ProductFull, *apierror.APIError) {
			return s.createdProduct(itemID), nil
		}).
		Times(1)

	// Two attributes supplied → two AddAttribute calls, in order.
	s.itemRepo.EXPECT().
		AddAttribute(gomock.Any(), gomock.AssignableToTypeOf(domain.AddItemAttributeParams{})).
		DoAndReturn(func(_ context.Context, params domain.AddItemAttributeParams) *apierror.APIError {
			s.Equal("ac_test123", params.AccountID)
			s.Equal("attr_red", params.AttributeID)
			return nil
		}).
		Times(1)
	s.itemRepo.EXPECT().
		AddAttribute(gomock.Any(), gomock.AssignableToTypeOf(domain.AddItemAttributeParams{})).
		DoAndReturn(func(_ context.Context, params domain.AddItemAttributeParams) *apierror.APIError {
			s.Equal("attr_large", params.AttributeID)
			return nil
		}).
		Times(1)

	s.inventoryMutRepo.EXPECT().
		CreateInventoryLog(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateInventoryLogParams{})).
		DoAndReturn(func(_ context.Context, params domain.CreateInventoryLogParams) *apierror.APIError {
			s.Equal("ac_test123", params.AccountID)
			s.Equal("un_base", params.UnitID)
			s.True(params.Measure.IsZero())
			return nil
		}).
		Times(1)
	s.inventoryMutRepo.EXPECT().
		CreateInventoryChangeLog(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateInventoryChangeLogParams{})).
		DoAndReturn(func(_ context.Context, params domain.CreateInventoryChangeLogParams) *apierror.APIError {
			s.Equal("user_action", params.ActionType)
			s.True(params.Measure.IsZero())
			return nil
		}).
		Times(1)

	s.expectAttachAttributesForCreatedItem()
	s.expectCacheSuccess()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-NEW",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
		IsPortalReady:   true,
		UnitPrice:       &domain.CreateRateParams{Value: "1.50", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"},
		UnitCost:        &domain.CreateRateParams{Value: "0.75", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"},
		AttributeIDs:    []string{"attr_red", "attr_large"},
	})

	s.Nil(err)
	s.NotNil(result)
	s.Equal("SKU-NEW", result.Item.SKU)
}

func (s *ProductSvcTestSuite) TestCreateProduct_DefaultsRatesToZero() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().CheckSKUExists(gomock.Any(), "ac_test123", "SKU-X", "").Return(false, nil).Times(1)
	s.itemRepo.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "cat_123").Return("un_base", nil).Times(1)

	// unit_value and unit_cost default to zero; burn_rate defaults to zero per day.
	s.productRepo.EXPECT().InsertRate(gomock.Any(), gomock.Any(), "0", "un_base", "un_base").Return(nil).Times(2)
	s.productRepo.EXPECT().InsertRate(gomock.Any(), gomock.Any(), "0", "un_base", "day").Return(nil).Times(1)

	s.productRepo.EXPECT().InsertItem(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.productRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, itemID string, _ domain.CreateProductParams) (*domain.ProductFull, *apierror.APIError) {
			return s.createdProduct(itemID), nil
		}).
		Times(1)
	s.inventoryMutRepo.EXPECT().CreateInventoryLog(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.inventoryMutRepo.EXPECT().CreateInventoryChangeLog(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.expectAttachAttributesForCreatedItem()
	s.expectCacheSuccess()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-X",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestCreateProduct_DuplicateSKU_Conflict() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().
		CheckSKUExists(gomock.Any(), "ac_test123", "SKU-DUPE", "").
		Return(true, nil).
		Times(1)
	// No GetCategoryBaseUnitID / InsertRate / InsertItem / Create / AddAttribute /
	// CreateInventoryLog / CreateInventoryChangeLog expected — tx must short-circuit.
	s.expectCacheError()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-DUPE",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	s.Equal("sku", err.Param)
}

func (s *ProductSvcTestSuite) TestCreateProduct_SkipsBlankAttributeIDs() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().CheckSKUExists(gomock.Any(), "ac_test123", "SKU-A", "").Return(false, nil).Times(1)
	s.itemRepo.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "cat_123").Return("un_base", nil).Times(1)
	s.productRepo.EXPECT().InsertRate(gomock.Any(), gomock.Any(), gomock.Any(), "un_base", "un_base").Return(nil).Times(2)
	s.productRepo.EXPECT().InsertRate(gomock.Any(), gomock.Any(), gomock.Any(), "un_base", "day").Return(nil).Times(1)
	s.productRepo.EXPECT().InsertItem(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.productRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, itemID string, _ domain.CreateProductParams) (*domain.ProductFull, *apierror.APIError) {
			return s.createdProduct(itemID), nil
		}).
		Times(1)

	// Only the single non-blank attribute should be linked.
	s.itemRepo.EXPECT().
		AddAttribute(gomock.Any(), gomock.AssignableToTypeOf(domain.AddItemAttributeParams{})).
		DoAndReturn(func(_ context.Context, params domain.AddItemAttributeParams) *apierror.APIError {
			s.Equal("attr_only", params.AttributeID)
			return nil
		}).
		Times(1)

	s.inventoryMutRepo.EXPECT().CreateInventoryLog(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.inventoryMutRepo.EXPECT().CreateInventoryChangeLog(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.expectAttachAttributesForCreatedItem()
	s.expectCacheSuccess()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-A",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
		AttributeIDs:    []string{"", "attr_only", ""},
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestCreateProduct_RejectsNonCurrencyNumeratorOnUnitCost() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().CheckSKUExists(gomock.Any(), "ac_test123", "SKU-X", "").Return(false, nil).Times(1)
	s.itemRepo.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "cat_123").Return("un_base", nil).Times(1)

	// unit_price validates first and passes.
	s.unitRepo.EXPECT().
		GetDimensionCodes(gomock.Any(), gomock.Any()).
		Return(map[string]string{"un_usd": "currency", "un_each": "discrete"}, nil).
		Times(1)
	// unit_price insert succeeds, then unit_cost validation rejects the bad numerator.
	s.productRepo.EXPECT().
		InsertRate(gomock.Any(), gomock.Any(), "1.00", "un_usd", "un_each").
		Return(nil).
		Times(1)
	// unit_cost validation: numerator un_each is non-currency → reject.
	s.unitRepo.EXPECT().
		GetDimensionCodes(gomock.Any(), gomock.Any()).
		Return(map[string]string{"un_each": "discrete"}, nil).
		Times(1)
	// No further inserts after validation fails.
	s.expectCacheError()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-X",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
		UnitPrice:       &domain.CreateRateParams{Value: "1.00", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"},
		UnitCost:        &domain.CreateRateParams{Value: "0.50", NumeratorUnitID: "un_each", DenominatorUnitID: "un_each"},
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	s.Equal("unit_cost.numerator_unit_id", err.Param)
}

func (s *ProductSvcTestSuite) TestCreateProduct_RejectsCurrencyDenominatorOnUnitPrice() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().CheckSKUExists(gomock.Any(), "ac_test123", "SKU-Y", "").Return(false, nil).Times(1)
	s.itemRepo.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "cat_123").Return("un_base", nil).Times(1)

	// unit_price validation: denominator un_usd is currency → reject. No InsertRate called.
	s.unitRepo.EXPECT().
		GetDimensionCodes(gomock.Any(), gomock.Any()).
		Return(map[string]string{"un_usd": "currency"}, nil).
		Times(1)
	s.expectCacheError()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-Y",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
		UnitPrice:       &domain.CreateRateParams{Value: "1.00", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_usd"},
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	s.Equal("unit_price.denominator_unit_id", err.Param)
}

func (s *ProductSvcTestSuite) TestCreateProduct_MissingIdentity() {
	result, err := s.productSvc.CreateProduct(context.Background(), domain.CreateProductParams{})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (s *ProductSvcTestSuite) TestCreateProduct_NotInternalActor_Forbidden() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-NEW",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestCreateProduct_MissingItemsCreatePermission_Forbidden() {
	ctx := readOnlyProductIdentityCtx("ac_test123")

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-NEW",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestCreateProduct_IdempotencyReplay_ReturnsCached() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	cached := &domain.ProductFull{ID: "prod_cached", ItemID: "it_cached"}
	s.expectIdempotencyFinishedWithProduct(cached)

	// No repo calls expected — replay returns cached body.
	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-WHATEVER",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
	})

	s.Nil(err)
	s.NotNil(result)
	s.Equal("prod_cached", result.ID)
}

func (s *ProductSvcTestSuite) TestCreateProduct_ErrorPath_CachesError() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.itemRepo.EXPECT().
		CheckSKUExists(gomock.Any(), "ac_test123", "SKU-E", "").
		Return(false, apierror.NewInternalError(nil, "db down")).
		Times(1)
	s.expectCacheError()

	result, err := s.productSvc.CreateProduct(ctx, domain.CreateProductParams{
		SKU:             "SKU-E",
		ProductTypeCode: "sale",
		CategoryID:      "cat_123",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// =============================================================================
// UpdateProduct
// =============================================================================

func (s *ProductSvcTestSuite) existingProduct(itemID, sku string) *domain.ProductFull {
	return &domain.ProductFull{
		ID:              "prod_existing",
		ItemID:          itemID,
		ProductTypeCode: "sale",
		IsPortalReady:   true,
		Item:            &domain.Item{ID: itemID, SKU: sku},
	}
}

func (s *ProductSvcTestSuite) existingProductWithRateIDs(itemID, sku string) *domain.ProductFull {
	p := s.existingProduct(itemID, sku)
	p.Item.UnitValueID = "rate_uv"
	p.Item.BurnRateID = "rate_br"
	return p
}

func (s *ProductSvcTestSuite) TestUpdateProduct_PartialUpdate_OnlyTouchesProvidedFields() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), domain.GetProductFullParams{AccountID: "ac_test123", ProductID: "it_1"}).
		Return(s.existingProduct("it_1", "OLD-SKU"), nil).
		Times(1)

	// SKU not changing → no CheckSKUExists call expected.
	s.productRepo.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(domain.UpdateProductParams{})).
		DoAndReturn(func(_ context.Context, params domain.UpdateProductParams) (*domain.ProductFull, *apierror.APIError) {
			s.Nil(params.SKU)
			s.True(params.Description.IsUnset())
			s.True(params.Notes.IsUnset())
			s.NotNil(params.IsPortalReady)
			s.False(*params.IsPortalReady)
			return s.existingProduct("it_1", "OLD-SKU"), nil
		}).
		Times(1)
	s.expectAttachAttributes("it_1")
	s.expectCacheSuccess()

	falseVal := false
	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID:     "it_1",
		IsPortalReady: &falseVal,
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_SKUChange_ChecksUniquenessExcludingSelf() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(s.existingProduct("it_1", "OLD"), nil).
		Times(1)
	// Uniqueness check must exclude the current itemID.
	s.itemRepo.EXPECT().
		CheckSKUExists(gomock.Any(), "ac_test123", "NEW-SKU", "it_1").
		Return(false, nil).
		Times(1)
	s.productRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(s.existingProduct("it_1", "NEW-SKU"), nil).
		Times(1)
	s.expectAttachAttributes("it_1")
	s.expectCacheSuccess()

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID: "it_1",
		SKU:       new("NEW-SKU"),
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_DuplicateSKU_Conflict() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(s.existingProduct("it_1", "OLD"), nil).
		Times(1)
	s.itemRepo.EXPECT().
		CheckSKUExists(gomock.Any(), "ac_test123", "TAKEN", "it_1").
		Return(true, nil).
		Times(1)
	// No Update expected — conflict must short-circuit.
	s.expectCacheError()

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID: "it_1",
		SKU:       new("TAKEN"),
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	s.Equal("sku", err.Param)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_UpdateDescriptionFlagSemantics_ExplicitNull() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(s.existingProduct("it_1", "OLD"), nil).
		Times(1)
	s.productRepo.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(domain.UpdateProductParams{})).
		DoAndReturn(func(_ context.Context, params domain.UpdateProductParams) (*domain.ProductFull, *apierror.APIError) {
			s.True(params.Description.IsClear())
			s.True(params.Notes.IsUnset())
			return s.existingProduct("it_1", "OLD"), nil
		}).
		Times(1)
	s.expectAttachAttributes("it_1")
	s.expectCacheSuccess()

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID:   "it_1",
		Description: field.Clear[string](),
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_MissingIdentity() {
	result, err := s.productSvc.UpdateProduct(context.Background(), domain.UpdateProductParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_NotInternalActor_Forbidden() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_MissingItemsUpdatePermission_Forbidden() {
	ctx := readOnlyProductIdentityCtx("ac_test123")

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_NotFound() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Product not found.")).
		Times(1)
	s.expectCacheError()

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID: "it_missing",
		SKU:       new("X"),
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_IdempotencyReplay_ReturnsCached() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	cached := &domain.ProductFull{ID: "prod_cached", ItemID: "it_cached"}
	s.expectIdempotencyFinishedWithProduct(cached)

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID: "it_cached",
		SKU:       new("NEVER"),
	})

	s.Nil(err)
	s.NotNil(result)
	s.Equal("prod_cached", result.ID)
}

func (s *ProductSvcTestSuite) TestUpdateProduct_UnitPrice_UpdatesRate() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(s.existingProductWithRateIDs("it_1", "SKU"), nil).
		Times(1)

	s.unitRepo.EXPECT().
		GetDimensionCodes(gomock.Any(), gomock.Any()).
		Return(map[string]string{"un_usd": "currency", "un_each": "discrete"}, nil).
		Times(1)

	rateParams := domain.CreateRateParams{Value: "9.99", NumeratorUnitID: "un_usd", DenominatorUnitID: "un_each"}
	s.itemRepo.EXPECT().
		UpdateRate(gomock.Any(), "rate_uv", rateParams).
		Return(nil).
		Times(1)

	s.productRepo.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(domain.UpdateProductParams{})).
		Return(s.existingProductWithRateIDs("it_1", "SKU"), nil).
		Times(1)
	s.expectAttachAttributes("it_1")
	s.expectCacheSuccess()

	result, err := s.productSvc.UpdateProduct(ctx, domain.UpdateProductParams{
		ProductID: "it_1",
		UnitPrice: &rateParams,
	})

	s.Nil(err)
	s.NotNil(result)
}

// =============================================================================
// DeleteProduct
// =============================================================================

func (s *ProductSvcTestSuite) TestDeleteProduct_Success_SoftDeletes() {
	ctx := internalProductIdentityCtx("ac_test123")

	existing := s.existingProduct("it_1", "SKU-DEL")
	s.productRepo.EXPECT().
		Get(gomock.Any(), domain.GetProductFullParams{AccountID: "ac_test123", ProductID: "it_1"}).
		Return(existing, nil).
		Times(1)

	// Order matters: deleted_record snapshot before soft-delete so the snapshot
	// captures pre-delete state.
	gomock.InOrder(
		s.deletedRecordRepo.EXPECT().
			Create(gomock.Any(), constants.DeletedRecordResourceTypeProduct, "it_1", existing).
			Return(nil).
			Times(1),
		s.productRepo.EXPECT().
			SoftDelete(gomock.Any(), domain.DeleteProductParams{AccountID: "ac_test123", ProductID: "it_1"}).
			Return(nil).
			Times(1),
	)

	result, err := s.productSvc.DeleteProduct(ctx, domain.DeleteProductParams{ProductID: "it_1"})

	s.Nil(err)
	s.NotNil(result)
	s.Equal("it_1", result.ItemID)
}

func (s *ProductSvcTestSuite) TestDeleteProduct_AlreadyDeleted_Returns410() {
	ctx := internalProductIdentityCtx("ac_test123")

	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Product not found.")).
		Times(1)
	s.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeProduct, "it_gone").
		Return(true, nil).
		Times(1)
		// No SoftDelete expected.

	result, err := s.productSvc.DeleteProduct(ctx, domain.DeleteProductParams{ProductID: "it_gone"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceGone, err.Code)
}

func (s *ProductSvcTestSuite) TestDeleteProduct_GenuinelyNotFound_Returns404() {
	ctx := internalProductIdentityCtx("ac_test123")

	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Product not found.")).
		Times(1)
	s.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeProduct, "it_missing").
		Return(false, nil).
		Times(1)

	result, err := s.productSvc.DeleteProduct(ctx, domain.DeleteProductParams{ProductID: "it_missing"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (s *ProductSvcTestSuite) TestDeleteProduct_NotInternalActor_Forbidden() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	result, err := s.productSvc.DeleteProduct(ctx, domain.DeleteProductParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestDeleteProduct_MissingItemsDeletePermission_Forbidden() {
	ctx := readOnlyProductIdentityCtx("ac_test123")

	result, err := s.productSvc.DeleteProduct(ctx, domain.DeleteProductParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestDeleteProduct_DeletedRecordCreateFails_RollsBack() {
	ctx := internalProductIdentityCtx("ac_test123")

	existing := s.existingProduct("it_1", "SKU-DEL")
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(existing, nil).
		Times(1)
	s.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeProduct, "it_1", gomock.Any()).
		Return(apierror.NewInternalError(nil, "db down")).
		Times(1)
		// SoftDelete must not be called when the snapshot insert fails.

	result, err := s.productSvc.DeleteProduct(ctx, domain.DeleteProductParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// =============================================================================
// ChangeProductProductLine
// =============================================================================

func (s *ProductSvcTestSuite) TestChangeProductLine_Success() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	old := s.existingProduct("it_1", "SKU-1")
	oldLineID := "pl_old"
	old.ProductLineID = &oldLineID

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), domain.GetProductFullParams{AccountID: "ac_test123", ProductID: "it_1"}).
		Return(old, nil).
		Times(1)
	s.productRepo.EXPECT().
		ChangeProductLine(gomock.Any(), gomock.AssignableToTypeOf(domain.ChangeProductProductLineParams{})).
		DoAndReturn(func(_ context.Context, params domain.ChangeProductProductLineParams) (*domain.ProductFull, *apierror.APIError) {
			s.Equal("ac_test123", params.AccountID)
			s.Equal("it_1", params.ProductID)
			s.Equal("pl_new", params.ProductLineID)
			newLineID := "pl_new"
			updated := s.existingProduct("it_1", "SKU-1")
			updated.ProductLineID = &newLineID
			return updated, nil
		}).
		Times(1)
	s.expectAttachAttributes("it_1")
	s.expectCacheSuccess()

	result, err := s.productSvc.ChangeProductProductLine(ctx, domain.ChangeProductProductLineParams{
		ProductID:     "it_1",
		ProductLineID: "pl_new",
	})

	s.Nil(err)
	s.NotNil(result)
	s.Equal("pl_new", *result.ProductLineID)
}

func (s *ProductSvcTestSuite) TestChangeProductLine_NotFound() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Product not found.")).
		Times(1)
	// ChangeProductLine must not be called.

	result, err := s.productSvc.ChangeProductProductLine(ctx, domain.ChangeProductProductLineParams{
		ProductID:     "it_missing",
		ProductLineID: "pl_new",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (s *ProductSvcTestSuite) TestChangeProductLine_NotInternalActor_Forbidden() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	result, err := s.productSvc.ChangeProductProductLine(ctx, domain.ChangeProductProductLineParams{
		ProductID:     "it_1",
		ProductLineID: "pl_new",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestChangeProductLine_MissingPermission_Forbidden() {
	ctx := readOnlyProductIdentityCtx("ac_test123")

	result, err := s.productSvc.ChangeProductProductLine(ctx, domain.ChangeProductProductLineParams{
		ProductID:     "it_1",
		ProductLineID: "pl_new",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestChangeProductLine_IdempotencyReplay_ReturnsCached() {
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	cached := &domain.ProductFull{ID: "prod_cached", ItemID: "it_cached"}
	s.expectIdempotencyFinishedWithProduct(cached)

	result, err := s.productSvc.ChangeProductProductLine(ctx, domain.ChangeProductProductLineParams{
		ProductID:     "it_cached",
		ProductLineID: "pl_new",
	})

	s.Nil(err)
	s.NotNil(result)
	s.Equal("prod_cached", result.ID)
}

func (s *ProductSvcTestSuite) TestChangeProductLine_ProductLineCrossAccount_Rejected() {
	// Express allowed binding a product to any productLineID without account
	// scoping — a security gap. Go's repo scopes the UPDATE by account_id, so
	// a cross-account target line surfaces as a repo-level not-found. This test
	// locks that behavior in at the service boundary.
	ctx := productIdempotencyCtx(internalProductIdentityCtx("ac_test123"))

	s.expectIdempotencyStarted()
	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(s.existingProduct("it_1", "SKU-1"), nil).
		Times(1)
	s.productRepo.EXPECT().
		ChangeProductLine(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Product not found.")).
		Times(1)
	s.expectCacheError()

	result, err := s.productSvc.ChangeProductProductLine(ctx, domain.ChangeProductProductLineParams{
		ProductID:     "it_1",
		ProductLineID: "pl_from_other_account",
	})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// =============================================================================
// ListProductsFull
// =============================================================================

func (s *ProductSvcTestSuite) TestListProductsFull_InternalActor_PassesParamsThrough() {
	ctx := internalProductIdentityCtx("ac_test123")

	customerIDs := []string{"ac_a", "ac_b"}
	portalReady := false
	s.productRepo.EXPECT().
		List(gomock.Any(), gomock.AssignableToTypeOf(domain.ListProductsFullParams{})).
		DoAndReturn(func(_ context.Context, params domain.ListProductsFullParams) (*domain.ListProductsFullResult, *apierror.APIError) {
			s.Equal("ac_test123", params.AccountID)
			s.Equal(customerIDs, params.CustomerIDs)
			s.NotNil(params.IsPortalReady)
			s.False(*params.IsPortalReady)
			return &domain.ListProductsFullResult{
				Products: []*domain.ProductFull{},
				PageInfo: pagination.PageInfo{},
			}, nil
		}).
		Times(1)

	result, err := s.productSvc.ListProductsFull(ctx, domain.ListProductsFullParams{
		Limit:         10,
		CustomerIDs:   customerIDs,
		IsPortalReady: &portalReady,
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestListProductsFull_CustomerActor_OverridesCustomerIDsAndPortalReady() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	// External target → counterparty read-access check must fire.
	s.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_owner").
		Return(nil).
		Times(1)

	s.productRepo.EXPECT().
		List(gomock.Any(), gomock.AssignableToTypeOf(domain.ListProductsFullParams{})).
		DoAndReturn(func(_ context.Context, params domain.ListProductsFullParams) (*domain.ListProductsFullResult, *apierror.APIError) {
			// Caller's attempt to pass other customer IDs / portal=false must be
			// overridden so a customer can't widen their own scope.
			s.Equal([]string{"ac_customer"}, params.CustomerIDs)
			s.NotNil(params.IsPortalReady)
			s.True(*params.IsPortalReady)
			s.Equal("ac_owner", params.AccountID)
			return &domain.ListProductsFullResult{
				Products: []*domain.ProductFull{},
				PageInfo: pagination.PageInfo{},
			}, nil
		}).
		Times(1)

	callerPortalReady := false
	result, err := s.productSvc.ListProductsFull(ctx, domain.ListProductsFullParams{
		Limit:         10,
		CustomerIDs:   []string{"ac_other_customer"}, // must be overridden
		IsPortalReady: &callerPortalReady,            // must be overridden to true
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestListProductsFull_MissingIdentity() {
	result, err := s.productSvc.ListProductsFull(context.Background(), domain.ListProductsFullParams{})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (s *ProductSvcTestSuite) TestListProductsFull_MissingTargetAccount_AuthError() {
	// Internal actor with no target account → AuthenticationError.
	adminCode := string(constants.RoleTypeAdmin)
	accountID := "ac_actor"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions:  map[string]bool{"items:read": true},
		},
	})

	result, err := s.productSvc.ListProductsFull(ctx, domain.ListProductsFullParams{})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (s *ProductSvcTestSuite) TestListProductsFull_ExternalTarget_ReadAccessDenied() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	s.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_owner").
		Return(apierror.NewAuthorizationError("no access")).
		Times(1)

	result, err := s.productSvc.ListProductsFull(ctx, domain.ListProductsFullParams{})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestListProductsFull_AttachesAttributesAndUnitGroup() {
	ctx := internalProductIdentityCtx("ac_test123")

	product := &domain.ProductFull{
		ID:     "prod_1",
		ItemID: "it_1",
		Item:   &domain.Item{ID: "it_1"},
		ProductLine: &domain.ProductLineFull{
			ID:          "pl_1",
			UnitGroupID: "ug_1",
		},
	}
	s.productRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&domain.ListProductsFullResult{Products: []*domain.ProductFull{product}}, nil).
		Times(1)

	attrs := []*domain.ItemAttribute{{ID: "attr_1", Value: "Red"}}
	s.itemRepo.EXPECT().
		Get(gomock.Any(), domain.GetItemParams{AccountID: "ac_test123", ItemID: "it_1", Includes: []string{"attributes"}}).
		Return(&domain.Item{ID: "it_1", Attributes: attrs}, nil).
		Times(1)
	s.productLineRepo.EXPECT().
		GetUnitGroup(gomock.Any(), "ug_1", gomock.Any()).
		Return(&domain.ProductLineUnitGroup{ID: "ug_1", Name: "Mass"}, nil).
		Times(1)

	result, err := s.productSvc.ListProductsFull(ctx, domain.ListProductsFullParams{Limit: 10})

	s.Nil(err)
	s.NotNil(result)
	s.Len(result.Products, 1)
	s.Equal(attrs, result.Products[0].Item.Attributes)
	s.NotNil(result.Products[0].ProductLine.UnitGroup)
	s.Equal("Mass", result.Products[0].ProductLine.UnitGroup.Name)
}

// =============================================================================
// ExportProducts
// =============================================================================

func (s *ProductSvcTestSuite) TestExportProducts_InternalActor_PassesParamsThrough() {
	ctx := internalProductIdentityCtx("ac_test123")

	customerIDs := []string{"ac_a", "ac_b"}
	s.productRepo.EXPECT().
		Export(gomock.Any(), gomock.AssignableToTypeOf(domain.ExportProductsParams{})).
		DoAndReturn(func(_ context.Context, params domain.ExportProductsParams) ([]*domain.ProductFull, *apierror.APIError) {
			s.Equal("ac_test123", params.AccountID)
			s.Equal(customerIDs, params.CustomerIDs)
			s.Nil(params.IsPortalReady)
			return []*domain.ProductFull{}, nil
		}).
		Times(1)

	result, err := s.productSvc.ExportProducts(ctx, domain.ExportProductsParams{
		CustomerIDs: customerIDs,
	})

	s.Nil(err)
	s.NotNil(result)
}

func (s *ProductSvcTestSuite) TestExportProducts_CustomerActor_OverridesCustomerIDsAndPortalReady() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	s.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_owner").
		Return(nil).
		Times(1)

	s.productRepo.EXPECT().
		Export(gomock.Any(), gomock.AssignableToTypeOf(domain.ExportProductsParams{})).
		DoAndReturn(func(_ context.Context, params domain.ExportProductsParams) ([]*domain.ProductFull, *apierror.APIError) {
			// Customer must not be able to widen scope or see non-portal products.
			s.Equal([]string{"ac_customer"}, params.CustomerIDs)
			s.NotNil(params.IsPortalReady)
			s.True(*params.IsPortalReady)
			s.Equal("ac_owner", params.AccountID)
			return []*domain.ProductFull{}, nil
		}).
		Times(1)

	result, err := s.productSvc.ExportProducts(ctx, domain.ExportProductsParams{
		CustomerIDs: []string{"ac_other_customer"}, // must be overridden
	})

	s.Nil(err)
	s.NotNil(result)
}

// =============================================================================
// GetProduct
// =============================================================================

func (s *ProductSvcTestSuite) TestGetProduct_Success_WithIncludes() {
	ctx := internalProductIdentityCtx("ac_test123")

	product := &domain.ProductFull{
		ID:     "prod_1",
		ItemID: "it_1",
		Item:   &domain.Item{ID: "it_1", SKU: "SKU-1"},
	}
	s.productRepo.EXPECT().
		Get(gomock.Any(), domain.GetProductFullParams{AccountID: "ac_test123", ProductID: "it_1"}).
		Return(product, nil).
		Times(1)
	s.itemRepo.EXPECT().
		Get(gomock.Any(), domain.GetItemParams{AccountID: "ac_test123", ItemID: "it_1", Includes: []string{"attributes"}}).
		Return(&domain.Item{ID: "it_1", Attributes: []*domain.ItemAttribute{{ID: "attr_1"}}}, nil).
		Times(1)
		// No ProductLine on product → GetUnitGroup must NOT be called.

	result, err := s.productSvc.GetProduct(ctx, domain.GetProductFullParams{ProductID: "it_1"})

	s.Nil(err)
	s.NotNil(result)
	s.Len(result.Item.Attributes, 1)
}

func (s *ProductSvcTestSuite) TestGetProduct_MissingTargetAccount_AuthError() {
	adminCode := string(constants.RoleTypeAdmin)
	accountID := "ac_actor"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions:  map[string]bool{"items:read": true},
		},
	})

	result, err := s.productSvc.GetProduct(ctx, domain.GetProductFullParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (s *ProductSvcTestSuite) TestGetProduct_ExternalTarget_ChecksCounterpartyReadAccess() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	s.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_owner").
		Return(apierror.NewAuthorizationError("no access")).
		Times(1)

	result, err := s.productSvc.GetProduct(ctx, domain.GetProductFullParams{ProductID: "it_1"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (s *ProductSvcTestSuite) TestGetProduct_NotFound() {
	ctx := internalProductIdentityCtx("ac_test123")

	s.productRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Product not found.")).
		Times(1)

	result, err := s.productSvc.GetProduct(ctx, domain.GetProductFullParams{ProductID: "it_missing"})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// =============================================================================
// ValidateProducts
// =============================================================================

func (s *ProductSvcTestSuite) TestValidateProducts_Success_PreservesKeys() {
	ctx := internalProductIdentityCtx("ac_test123")

	input := map[string]string{"rowA": "SKU-1", "rowB": "SKU-2"}
	s.productRepo.EXPECT().
		ValidateProducts(gomock.Any(), gomock.AssignableToTypeOf(domain.ValidateProductsParams{})).
		DoAndReturn(func(_ context.Context, params domain.ValidateProductsParams) (*domain.ValidateProductsResult, *apierror.APIError) {
			s.Equal("ac_test123", params.AccountID)
			s.Equal("SKU-1", params.ProductsMap["rowA"])
			s.Equal("SKU-2", params.ProductsMap["rowB"])
			s.Empty(params.Includes)
			return &domain.ValidateProductsResult{
				Products: map[string]*domain.ProductFull{
					"rowA": {ID: "prod_1", Item: &domain.Item{ID: "it_1", SKU: "SKU-1"}},
				},
			}, nil
		}).
		Times(1)

	s.itemRepo.EXPECT().
		Get(gomock.Any(), domain.GetItemParams{
			AccountID: "ac_test123",
			ItemID:    "it_1",
			Includes:  []string{"attributes"},
		}).
		Return(&domain.Item{ID: "it_1"}, nil).
		Times(1)

	result, err := s.productSvc.ValidateProducts(ctx, domain.ValidateProductsParams{ProductsMap: input})

	s.Nil(err)
	s.NotNil(result)
	s.Len(result.Products, 1)
	s.NotNil(result.Products["rowA"])
	s.Nil(result.Products["rowB"])
}

func (s *ProductSvcTestSuite) TestValidateProducts_MissingTargetAccount_AuthError() {
	adminCode := string(constants.RoleTypeAdmin)
	accountID := "ac_actor"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions:  map[string]bool{"items:read": true},
		},
	})

	result, err := s.productSvc.ValidateProducts(ctx, domain.ValidateProductsParams{})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (s *ProductSvcTestSuite) TestValidateProducts_ExternalTarget_ChecksCounterpartyReadAccess() {
	ctx := customerProductIdentityCtx("ac_customer", "ac_owner")

	s.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_owner").
		Return(apierror.NewAuthorizationError("no access")).
		Times(1)

	result, err := s.productSvc.ValidateProducts(ctx, domain.ValidateProductsParams{})

	s.Nil(result)
	s.NotNil(err)
	s.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}
