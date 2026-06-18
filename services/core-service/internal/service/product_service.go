package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/mediator"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
	"github.com/shopspring/decimal"
)

var productSvcTracer = tracing.GetTracer("core-service.product_service")

type productSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ProductSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ProductSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("product service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("product service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("product service: tx manager is required")
	}
	return nil
}

func NewProductSvc(config *ProductSvcConfig) domain.ProductSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *productSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// SearchProducts finds products belonging to an account that match a SKU search query.
func (s *productSvcImpl) SearchProducts(ctx context.Context, accountID, query string) ([]domain.ProductInfo, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.search_products")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductRepo().SearchBySKU(ctx, accountID, query)
}

// ListProducts returns all products belonging to the specified account.
func (s *productSvcImpl) ListProducts(ctx context.Context, accountID string) ([]domain.ProductInfo, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.list_products")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductRepo().ListByAccount(ctx, accountID)
}

// GetCustomerByEmail looks up a customer account relation by email address within an owner account.
func (s *productSvcImpl) GetCustomerByEmail(ctx context.Context, ownerAccountID, email string) (*domain.CustomerByEmail, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.get_customer_by_email")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAccountRelationRepo().FindCustomerByEmail(ctx, ownerAccountID, email)
}

// ListProductsFull returns a paginated list of products for the caller's account. Supports both internal and customer actors. Customers only see portal-ready products from their accessible product lines.
func (s *productSvcImpl) ListProductsFull(ctx context.Context, params domain.ListProductsFullParams) (*domain.ListProductsFullResult, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.list_full")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkProductReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		if actorAccountID != nil {
			params.CustomerIDs = []string{*actorAccountID}
		}
		isPortalReady := true
		params.IsPortalReady = &isPortalReady
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewProductRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, p := range result.Products {
		if apiErr := s.attachProductIncludes(ctx, p, identity.Target.AccountID, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return result, nil
}

// ExportProducts returns all matching products for export (no pagination).
func (s *productSvcImpl) ExportProducts(ctx context.Context, params domain.ExportProductsParams) ([]*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.export")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkProductReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		if actorAccountID != nil {
			params.CustomerIDs = []string{*actorAccountID}
		}
		isPortalReady := true
		params.IsPortalReady = &isPortalReady
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewProductRepo().Export(ctx, params)
}

// GetProduct returns a single product by item ID.
func (s *productSvcImpl) GetProduct(ctx context.Context, params domain.GetProductFullParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkProductReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	product, apiErr := s.repos.NewProductRepo().Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := s.attachProductIncludes(ctx, product, identity.Target.AccountID, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if product != nil && product.Item != nil {
		mediator.RefreshItemBurnRateAfterGet(ctx, s.repos, s.mediators(), identity.Target.AccountID, product.Item, params.Includes)
	}

	return product, nil
}

// attachProductIncludes populates expandable sub-resources on a product that the product queries don't join. Specifically: item.attributes (loaded via item repo) and product_line.unit_group (loaded via product_line repo).
func (s *productSvcImpl) attachProductIncludes(ctx context.Context, product *domain.ProductFull, accountID string, includes []string) *apierror.APIError {
	if product == nil {
		return nil
	}

	if product.Item != nil {
		item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
			AccountID: accountID,
			ItemID:    product.Item.ID,
			Includes:  []string{"attributes"},
		})
		if apiErr != nil {
			// The item was concurrently soft-deleted between the list/get query and this enrichment call. Skip enrichment rather than surfacing a spurious 404 to the caller.
			if !apierror.IsNotFound(apiErr) {
				return apiErr
			}
		} else {
			product.Item.Attributes = item.Attributes
			if item.UnitValue != nil {
				product.Item.UnitValue = item.UnitValue
			}
			if item.UnitCost != nil {
				product.Item.UnitCost = item.UnitCost
			}
			if item.BurnRate != nil {
				product.Item.BurnRate = item.BurnRate
			}
		}
	}

	if product.ProductLine != nil && product.ProductLine.UnitGroupID != "" {
		unitGroup, apiErr := s.repos.NewProductLineRepo().GetUnitGroup(ctx, product.ProductLine.UnitGroupID, includes)
		if apiErr != nil {
			return apiErr
		}
		product.ProductLine.UnitGroup = unitGroup
	}

	return nil
}

// CreateProduct creates a new product with its associated item, rates, and product record.
func (s *productSvcImpl) CreateProduct(ctx context.Context, params domain.CreateProductParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productID, apiErr := id.GenID(id.ProductIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	itemID, apiErr := id.GenID(id.ItemIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitValueRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitCostRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	burnRateRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	params.ItemID = itemID
	params.UnitValueRateID = unitValueRateID
	params.UnitCostRateID = unitCostRateID
	params.BurnRateRateID = burnRateRateID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productSvcImpl) *apierror.APIError {
			txProductRepo := txSvc.repos.NewProductRepo()
			txItemRepo := txSvc.repos.NewItemRepo()

			// Check SKU uniqueness.
			exists, apiErr := txItemRepo.CheckSKUExists(txCtx, params.AccountID, params.SKU, "")
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam(fmt.Sprintf("An item with the SKU '%s' already exists.", params.SKU), "sku")
			}

			// Get base unit for rates from category.
			baseUnitID, apiErr := txItemRepo.GetCategoryBaseUnitID(txCtx, params.CategoryID)
			if apiErr != nil {
				return apiErr
			}

			// Insert rates for item (unit_value, unit_cost, burn_rate). Caller-supplied inputs override the defaults; unit_price and unit_cost additionally enforce the currency-numerator / non-currency-denominator rule.
			txUnitRepo := txSvc.repos.NewUnitRepo()

			unitPriceValue, unitPriceNum, unitPriceDen := "0", baseUnitID, baseUnitID
			if params.UnitPrice != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitPrice.NumeratorUnitID, params.UnitPrice.DenominatorUnitID, "unit_price"); apiErr != nil {
					return apiErr
				}
				unitPriceValue = params.UnitPrice.Value
				unitPriceNum = params.UnitPrice.NumeratorUnitID
				unitPriceDen = params.UnitPrice.DenominatorUnitID
			}
			if apiErr := txProductRepo.InsertRate(txCtx, unitValueRateID, unitPriceValue, unitPriceNum, unitPriceDen); apiErr != nil {
				return apiErr
			}

			unitCostValue, unitCostNum, unitCostDen := "0", baseUnitID, baseUnitID
			if params.UnitCost != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitCost.NumeratorUnitID, params.UnitCost.DenominatorUnitID, "unit_cost"); apiErr != nil {
					return apiErr
				}
				unitCostValue = params.UnitCost.Value
				unitCostNum = params.UnitCost.NumeratorUnitID
				unitCostDen = params.UnitCost.DenominatorUnitID
			}
			if apiErr := txProductRepo.InsertRate(txCtx, unitCostRateID, unitCostValue, unitCostNum, unitCostDen); apiErr != nil {
				return apiErr
			}

			// Burn rate is always initialized to "0" per day; it is recomputed from inventory history by the burn-rate mediator.
			if apiErr := txProductRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, "day"); apiErr != nil {
				return apiErr
			}

			// Insert item.
			if apiErr := txProductRepo.InsertItem(txCtx, itemID, params); apiErr != nil {
				return apiErr
			}

			// Insert product record.
			created, apiErr := txProductRepo.Create(txCtx, productID, itemID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			// Link caller-supplied attributes to the new item (matches Dashboard behavior).
			for _, attrID := range params.AttributeIDs {
				if attrID == "" {
					continue
				}
				if apiErr := txItemRepo.AddAttribute(txCtx, domain.AddItemAttributeParams{
					AccountID:   params.AccountID,
					ItemID:      itemID,
					AttributeID: attrID,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Initialize inventory tracking with zero-quantity log and change log.
			txInvMutRepo := txSvc.repos.NewInventoryMutationRepo()
			zeroMeasure := decimal.Zero

			if apiErr := txInvMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
				AccountID: params.AccountID,
				ItemID:    itemID,
				Measure:   zeroMeasure,
				UnitID:    baseUnitID,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := txInvMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
				AccountID:  params.AccountID,
				ItemID:     itemID,
				Measure:    zeroMeasure,
				UnitID:     baseUnitID,
				ActionType: "user_action",
			}); apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProduct,
				ResourceID:   created.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := txSvc.attachProductIncludes(txCtx, result, params.AccountID, params.Includes); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// UpdateProduct partially updates an existing product.
func (s *productSvcImpl) UpdateProduct(ctx context.Context, params domain.UpdateProductParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productSvcImpl) *apierror.APIError {
			txProductRepo := txSvc.repos.NewProductRepo()
			txItemRepo := txSvc.repos.NewItemRepo()

			// Fetch existing product before mutation for audit diff (same includes as the post-update fetch so include-only fields cannot produce false diffs).
			old, apiErr := txProductRepo.Get(txCtx, domain.GetProductFullParams{
				AccountID: params.AccountID,
				ProductID: params.ProductID,
				Includes:  params.Includes,
			})
			if apiErr != nil {
				return apiErr
			}

			if old.Item != nil {
				params.Description = params.Description.BackfillUnsetPtr(old.Item.Description)
				params.Notes = params.Notes.BackfillUnsetPtr(old.Item.Notes)
			}

			// Check SKU uniqueness if being updated.
			if params.SKU != nil {
				exists, apiErr := txItemRepo.CheckSKUExists(txCtx, params.AccountID, *params.SKU, old.ItemID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam(fmt.Sprintf("An item with the SKU '%s' already exists.", *params.SKU), "sku")
				}
			}

			txUnitRepo := txSvc.repos.NewUnitRepo()

			if params.UnitPrice != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitPrice.NumeratorUnitID, params.UnitPrice.DenominatorUnitID, "unit_price"); apiErr != nil {
					return apiErr
				}
				if old.Item == nil || old.Item.UnitValueID == "" {
					return apierror.NewInvariantViolationError("Product item or unit value rate is missing.")
				}
				if apiErr := txItemRepo.UpdateRate(txCtx, old.Item.UnitValueID, *params.UnitPrice); apiErr != nil {
					return apiErr
				}
			}

			// Update product + item fields.
			updated, apiErr := txProductRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)
			// Item-level fields (sku, description, notes) live on the joined item row; diff them the same way part updates do.
			changes = append(changes, audit.ComputeChanges(old.Item, updated.Item)...)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProduct,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := txSvc.attachProductIncludes(txCtx, result, params.AccountID, params.Includes); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// DeleteProduct soft-deletes a product by its product ID.
func (s *productSvcImpl) DeleteProduct(ctx context.Context, params domain.DeleteProductParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Fetch existing product before deletion.
	product, apiErr := s.repos.NewProductRepo().Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ProductID: params.ProductID})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProduct, params.ProductID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This product has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	// Soft-delete within a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProduct, params.ProductID, product); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewProductRepo().SoftDelete(txCtx, params); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(product, (*domain.ProductFull)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProduct,
			ResourceID:   product.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return product, nil
}

// ChangeProductProductLine changes the product line assigned to a product.
func (s *productSvcImpl) ChangeProductProductLine(ctx context.Context, params domain.ChangeProductProductLineParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.change_product_line")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		txProductRepo := s.repos.NewProductRepo()

		old, apiErr := txProductRepo.Get(ctx, domain.GetProductFullParams{
			AccountID: params.AccountID,
			ProductID: params.ProductID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.ProductFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productSvcImpl) *apierror.APIError {
			updated, apiErr := txSvc.repos.NewProductRepo().ChangeProductLine(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProduct,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := txSvc.attachProductIncludes(txCtx, result, params.AccountID, params.Includes); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// ValidateProducts validates a map of SKUs and returns matching products.
func (s *productSvcImpl) ValidateProducts(ctx context.Context, params domain.ValidateProductsParams) (*domain.ValidateProductsResult, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.validate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkProductReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewProductRepo().ValidateProducts(ctx, params)
	if apiErr != nil {
		return nil, apiErr
	}

	for _, p := range result.Products {
		if apiErr := s.attachProductIncludes(ctx, p, params.AccountID, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return result, nil
}

func (s *productSvcImpl) BatchGetProductsByIDs(ctx context.Context, ids []string) ([]*domain.ProductFull, *apierror.APIError) {
	ctx, span := productSvcTracer.Start(ctx, "service.product.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	meds := s.mediators()
	if apiErr := authorizeCatalogBatchRead(ctx, identity, span, meds, func() *apierror.APIError {
		return checkProductReadPermission(identity)
	}); apiErr != nil {
		return nil, apiErr
	}
	if len(ids) == 0 {
		return nil, nil
	}

	products, apiErr := s.repos.NewProductRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return products, nil
}

// checkProductReadPermission checks the appropriate read permission based on the identity context. Internal actors need items:read for their own account, or customers:read / suppliers:read for external accounts.
func checkProductReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead)
}
