package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/tracing"
)

// dayUnitID is the system unit burn rates are counted per.
const dayUnitID = "day"

type fieldQuantity = field.Clearable[domain.QuantityInput]

// CreateItem creates an item of any type, its sales section (always for products and services) and its material
// section (materials only).
func (s *itemSvcImpl) CreateItem(ctx context.Context, params domain.CreateItemParams) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.create")
	defer span.End()

	identity, apiErr := s.itemWriter(ctx, func(id *types.Identity) *apierror.APIError {
		return id.CheckHasPermission(types.PermissionDomainItems, types.ActionCreate)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID

	return runIdempotent(ctx, s, identity, func(txCtx context.Context, txSvc *itemSvcImpl) (*domain.Item, *apierror.APIError) {
		return createItemInTx(txCtx, txSvc.repos, params)
	})
}

// UpdateItem changes the fields sent. A material or part gains a sales section the first time one is sent.
func (s *itemSvcImpl) UpdateItem(ctx context.Context, params domain.UpdateItemParams) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.update")
	defer span.End()

	identity, apiErr := s.itemWriter(ctx, func(id *types.Identity) *apierror.APIError {
		return id.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID

	return runIdempotent(ctx, s, identity, func(txCtx context.Context, txSvc *itemSvcImpl) (*domain.Item, *apierror.APIError) {
		return updateItemInTx(txCtx, txSvc.repos, params)
	})
}

// DeleteItem hard-deletes an item that nothing references.
func (s *itemSvcImpl) DeleteItem(ctx context.Context, itemID string) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.delete")
	defer span.End()

	identity, apiErr := s.itemWriter(ctx, func(id *types.Identity) *apierror.APIError {
		return id.CheckHasPermission(types.PermissionDomainItems, types.ActionDelete)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	var deleted *domain.Item
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		item, apiErr := deleteItemInTx(txCtx, txSvc.repos, accountID, itemID)
		deleted = item
		return apiErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return deleted, nil
}

// itemWriter is the caller of an item write: an internal actor passing checkPermission.
func (s *itemSvcImpl) itemWriter(ctx context.Context, checkPermission func(*types.Identity) *apierror.APIError) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := checkPermission(identity); apiErr != nil {
		return nil, apiErr
	}
	if !identity.IsTargetAccountSet() {
		return nil, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required.")
	}
	return identity, nil
}

// runIdempotent runs an item write in one transaction under the request's idempotency key.
func runIdempotent(ctx context.Context, s *itemSvcImpl, identity *types.Identity, write func(context.Context, *itemSvcImpl) (*domain.Item, *apierror.APIError)) (*domain.Item, *apierror.APIError) {
	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Item](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Issue unmarshalling cached response.")
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Item
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			item, apiErr := write(txCtx, txSvc)
			if apiErr != nil {
				return apiErr
			}
			result = item
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, apierror.NewInvariantViolationError("Unexpected recovery point: " + idempotencyKey.RecoveryPoint)
	}
}

// readItem reads one of the account's items whole.
func readItem(ctx context.Context, repos domain.RepoFactory, accountID, itemID string) (*domain.Item, *apierror.APIError) {
	items, apiErr := repos.NewItemRepo().GetByIDsWithIncludes(ctx, accountID, []string{itemID}, domain.ItemReadIncludes)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(items) == 0 {
		return nil, apierror.NewNotFoundError("Item not found.")
	}
	return items[0], nil
}

// createItemInTx writes a new item and its sections inside the caller's transaction and returns it read back whole.
// It checks no permission: the caller authorizes the write.
func createItemInTx(ctx context.Context, repos domain.RepoFactory, p domain.CreateItemParams) (*domain.Item, *apierror.APIError) {
	if !p.Type.IsValid() {
		return nil, apierror.NewFieldError("type", apierror.CodeInvalidFormat, "type must be one of material, part, product or service.")
	}
	if p.Material != nil && p.Type != constants.ItemTypeCodeMaterial {
		return nil, apierror.NewFieldError("material", apierror.CodeInvalidFormat, "Only materials have a material section.")
	}
	if p.Status == "" {
		p.Status = constants.ArchiveStatusActive
	}
	if !p.Status.IsValid() {
		return nil, apierror.NewFieldError("status", apierror.CodeInvalidFormat, "status must be active or archived.")
	}

	itemRepo := repos.NewItemRepo()
	exists, apiErr := itemRepo.CheckSKUExists(ctx, p.AccountID, p.SKU, "")
	if apiErr != nil {
		return nil, apiErr
	}
	if exists {
		return nil, apierror.NewConflictErrorWithParam("sku", fmt.Sprintf("An item with the SKU '%s' already exists.", p.SKU))
	}

	baseUnitID, apiErr := itemCategoryBaseUnit(ctx, repos, p.AccountID, p.Type, p.CategoryID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := validateItemAttributes(ctx, repos, p.AccountID, p.CategoryID, p.AttributeIDs, "attribute_ids"); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := validateItemSales(ctx, repos, p.AccountID, p.Sales); apiErr != nil {
		return nil, apiErr
	}
	if p.UnitCost != nil {
		if apiErr := ValidateCostRateUnits(ctx, repos.NewUnitRepo(), p.UnitCost.NumeratorUnitID, p.UnitCost.DenominatorUnitID, "unit_cost"); apiErr != nil {
			return nil, apiErr
		}
	}

	ids, apiErr := genIDs(id.ItemIDPrefix, id.RateIDPrefix, id.RateIDPrefix, id.RateIDPrefix)
	if apiErr != nil {
		return nil, apiErr
	}
	itemID, unitValueID, unitCostID, burnRateID := ids[0], ids[1], ids[2], ids[3]

	zero := domain.CreateRateParams{Value: "0", NumeratorUnitID: baseUnitID, DenominatorUnitID: baseUnitID}
	unitValue, unitCost := zero, zero
	if p.Sales != nil && p.Sales.UnitPrice != nil {
		unitValue = *p.Sales.UnitPrice
	}
	if p.UnitCost != nil {
		unitCost = *p.UnitCost
	}
	burnRate := domain.CreateRateParams{Value: "0", NumeratorUnitID: baseUnitID, DenominatorUnitID: dayUnitID}
	for rateID, rate := range map[string]domain.CreateRateParams{unitValueID: unitValue, unitCostID: unitCost, burnRateID: burnRate} {
		if apiErr := itemRepo.InsertRate(ctx, rateID, rate); apiErr != nil {
			return nil, apiErr
		}
	}

	if apiErr := itemRepo.Insert(ctx, domain.InsertItemParams{
		ItemID:          itemID,
		AccountID:       p.AccountID,
		Type:            p.Type,
		SKU:             p.SKU,
		Description:     p.Description,
		CategoryID:      p.CategoryID,
		UnitValueRateID: unitValueID,
		UnitCostRateID:  unitCostID,
		BurnRateRateID:  burnRateID,
		Status:          p.Status,
		Supply:          p.Supply,
	}); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := itemRepo.ReplaceAttributes(ctx, itemID, uniqueIDs(p.AttributeIDs)); apiErr != nil {
		return nil, apiErr
	}

	sales := p.Sales
	if sales == nil && (p.Type == constants.ItemTypeCodeProduct || p.Type == constants.ItemTypeCodeService) {
		sales = &domain.ItemSalesInput{}
	}
	if sales != nil {
		if apiErr := insertItemSales(ctx, repos, itemID, sales); apiErr != nil {
			return nil, apiErr
		}
	}

	switch p.Type {
	case constants.ItemTypeCodeMaterial:
		if apiErr := insertItemMaterial(ctx, repos, itemID, baseUnitID, p.Material); apiErr != nil {
			return nil, apiErr
		}
	case constants.ItemTypeCodePart:
		partID, apiErr := id.GenID(id.PartIDPrefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		if apiErr := itemRepo.InsertPart(ctx, partID, itemID); apiErr != nil {
			return nil, apiErr
		}
	}

	if p.Type.IsPhysical() {
		if apiErr := openItemInventory(ctx, repos, p.AccountID, itemID, baseUnitID); apiErr != nil {
			return nil, apiErr
		}
	}

	created, apiErr := readItem(ctx, repos, p.AccountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionCreate,
		ResourceType: constants.ObjectTypeItem,
		ResourceID:   created.ID,
		Changes:      audit.ComputeChanges(nil, created),
	}); apiErr != nil {
		return nil, apiErr
	}
	return created, nil
}

// updateItemInTx applies an item update inside the caller's transaction and returns the item read back whole.
func updateItemInTx(ctx context.Context, repos domain.RepoFactory, p domain.UpdateItemParams) (*domain.Item, *apierror.APIError) {
	itemRepo := repos.NewItemRepo()
	before, apiErr := readItem(ctx, repos, p.AccountID, p.ItemID)
	if apiErr != nil {
		return nil, apiErr
	}
	itemType := constants.ItemTypeCode(before.ItemTypeCode)

	if p.Material != nil && before.MaterialSection == nil {
		return nil, apierror.NewFieldError("material", apierror.CodeInvalidFormat, "Only materials have a material section.")
	}
	if p.Status != nil && !p.Status.IsValid() {
		return nil, apierror.NewFieldError("status", apierror.CodeInvalidFormat, "status must be active or archived.")
	}
	if p.SKU != nil {
		exists, apiErr := itemRepo.CheckSKUExists(ctx, p.AccountID, *p.SKU, p.ItemID)
		if apiErr != nil {
			return nil, apiErr
		}
		if exists {
			return nil, apierror.NewConflictErrorWithParam("sku", fmt.Sprintf("An item with the SKU '%s' already exists.", *p.SKU))
		}
	}

	categoryID := before.ItemCategoryID
	if p.CategoryID != nil && *p.CategoryID != before.ItemCategoryID {
		if _, apiErr := itemCategoryBaseUnit(ctx, repos, p.AccountID, itemType, *p.CategoryID); apiErr != nil {
			return nil, apiErr
		}
		categoryID = *p.CategoryID
		if _, replacing := p.AttributeIDs.Value(); !replacing {
			if apiErr := validateCategoryCarriesItemAttributes(ctx, repos, before, categoryID, "category_id"); apiErr != nil {
				return nil, apiErr
			}
		}
	}
	attributeIDs, replaceAttributes := p.AttributeIDs.Value()
	if replaceAttributes {
		if apiErr := validateItemAttributes(ctx, repos, p.AccountID, categoryID, attributeIDs, "attribute_ids"); apiErr != nil {
			return nil, apiErr
		}
	}
	if apiErr := validateItemSales(ctx, repos, p.AccountID, p.Sales); apiErr != nil {
		return nil, apiErr
	}

	set := domain.SetItemFieldsParams{
		AccountID:      p.AccountID,
		ItemID:         p.ItemID,
		SKU:            p.SKU,
		SetDescription: p.UpdateDescription,
		Description:    p.Description,
		CategoryID:     p.CategoryID,
		Status:         p.Status,
		SetSupply:      !p.Supply.IsUnset(),
		Supply:         clearableValuePtr(p.Supply),
	}
	if apiErr := itemRepo.SetFields(ctx, set); apiErr != nil {
		return nil, apiErr
	}
	if replaceAttributes {
		if apiErr := itemRepo.ReplaceAttributes(ctx, p.ItemID, uniqueIDs(attributeIDs)); apiErr != nil {
			return nil, apiErr
		}
	}

	var unitPrice *domain.CreateRateParams
	if p.Sales != nil {
		unitPrice = p.Sales.UnitPrice
	}
	if unitPrice != nil || p.UnitCost != nil {
		if apiErr := applyItemRatesPublishingCostInTx(ctx, repos, p.AccountID, p.ItemID, before.UnitValueID, before.UnitCostID, unitPrice, p.UnitCost); apiErr != nil {
			return nil, apiErr
		}
		if p.UnitCost != nil {
			if apiErr := itemRepo.ClearItemDirtyFlag(ctx, p.AccountID, p.ItemID); apiErr != nil {
				return nil, apiErr
			}
		}
	}

	if p.Sales != nil {
		if before.Sales == nil {
			if apiErr := insertItemSales(ctx, repos, p.ItemID, p.Sales); apiErr != nil {
				return nil, apiErr
			}
		} else if apiErr := updateItemSales(ctx, repos, before, p.Sales); apiErr != nil {
			return nil, apiErr
		}
	}
	if p.Material != nil {
		if apiErr := updateItemMaterial(ctx, repos, before.MaterialSection, p.Material); apiErr != nil {
			return nil, apiErr
		}
	}

	after, apiErr := readItem(ctx, repos, p.AccountID, p.ItemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if changes := audit.ComputeChanges(before, after); len(changes) > 0 {
		if apiErr := audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeItem,
			ResourceID:   after.ID,
			Changes:      changes,
		}); apiErr != nil {
			return nil, apiErr
		}
	}
	return after, nil
}

// deleteItemInTx hard-deletes an item inside the caller's transaction, after checking nothing references it, and
// records the tombstone a replayed delete answers 410 from.
func deleteItemInTx(ctx context.Context, repos domain.RepoFactory, accountID, itemID string) (*domain.Item, *apierror.APIError) {
	item, apiErr := readItem(ctx, repos, accountID, itemID)
	if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
		deleted, existsErr := repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeItem, itemID, accountID)
		if existsErr != nil {
			return nil, existsErr
		}
		if deleted {
			return nil, apierror.NewGoneError("This item has already been deleted.")
		}
		return nil, apiErr
	}
	if apiErr != nil {
		return nil, apiErr
	}

	refs := repos.NewReferenceRepo()
	labels, apiErr := refs.InUseBy(ctx, item.ID, domain.ItemReferences.Probes)
	if apiErr != nil {
		return nil, apiErr
	}
	if item.MaterialSection != nil {
		supplierLabels, apiErr := refs.InUseBy(ctx, item.MaterialSection.MaterialID, domain.ItemMaterialReferences.Probes)
		if apiErr != nil {
			return nil, apiErr
		}
		labels = append(labels, supplierLabels...)
	}
	if len(labels) > 0 {
		return nil, domain.ItemReferences.InUseError(labels)
	}

	if apiErr := repos.NewDeletedRecordRepo().CreateInAccount(ctx, constants.DeletedRecordResourceTypeItem, item.ID, accountID, item); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := repos.NewItemRepo().HardDelete(ctx, item); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionDelete,
		ResourceType: constants.ObjectTypeItem,
		ResourceID:   item.ID,
		Changes:      audit.ComputeChanges(item, (*domain.Item)(nil)),
	}); apiErr != nil {
		return nil, apiErr
	}
	return item, nil
}

// itemCategoryBaseUnit checks the category exists and suits the item type, and returns its base unit.
func itemCategoryBaseUnit(ctx context.Context, repos domain.RepoFactory, accountID string, itemType constants.ItemTypeCode, categoryID string) (string, *apierror.APIError) {
	// The base unit lookup reads any category by id; only the account's own and the system categories may be used.
	visible, apiErr := repos.NewItemCategoryRepo().GetByIDs(ctx, accountID, []string{categoryID})
	if apiErr != nil {
		return "", apiErr
	}
	if len(visible) == 0 {
		return "", apierror.NewFieldError("category_id", apierror.CodeInvalidFormat, "The category does not exist.")
	}
	baseUnitID, categoryType, apiErr := repos.NewItemRepo().GetCategoryBaseUnitID(ctx, categoryID)
	if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
		return "", apierror.NewFieldError("category_id", apierror.CodeInvalidFormat, "The category does not exist.")
	}
	if apiErr != nil {
		return "", apiErr
	}
	if !categoryTypeMatchesItem(string(itemType), categoryType) {
		return "", apierror.NewFieldError("category_id", apierror.CodeInvalidFormat,
			"A material needs a material category; parts, products and services need a product category.")
	}
	return baseUnitID, nil
}

// validateItemAttributes checks attributes for an item in categoryID: each the account's, active, of a property the
// category carries, and at most one per property.
func validateItemAttributes(ctx context.Context, repos domain.RepoFactory, accountID, categoryID string, attributeIDs []string, param string) *apierror.APIError {
	attributeIDs = uniqueIDs(attributeIDs)
	if len(attributeIDs) == 0 {
		return nil
	}
	attributes, apiErr := repos.NewAttributeRepo().GetByIDs(ctx, accountID, attributeIDs)
	if apiErr != nil {
		return apiErr
	}
	if len(attributes) != len(attributeIDs) {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Every attribute must exist in your account.")
	}
	seen := make(map[string]bool, len(attributes))
	for _, attr := range attributes {
		if seen[attr.PropertyID] {
			return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "An item can have at most one attribute per property.")
		}
		seen[attr.PropertyID] = true
	}
	return validateAttributesForCategory(ctx, repos, accountID, categoryID, attributeIDs, param)
}

// validateItemSales checks a sales write: an active product line the account may assign, and a lead time in a time unit.
func validateItemSales(ctx context.Context, repos domain.RepoFactory, accountID string, sales *domain.ItemSalesInput) *apierror.APIError {
	if sales == nil {
		return nil
	}
	if sales.UnitPrice != nil {
		if apiErr := ValidateCostRateUnits(ctx, repos.NewUnitRepo(), sales.UnitPrice.NumeratorUnitID, sales.UnitPrice.DenominatorUnitID, "sales.unit_price"); apiErr != nil {
			return apiErr
		}
	}
	if lineID, ok := sales.ProductLineID.Value(); ok {
		line, apiErr := repos.NewProductLineRepo().Get(ctx, domain.GetProductLineParams{AccountID: accountID, ProductLineID: lineID})
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			return apierror.NewFieldError("sales.product_line_id", apierror.CodeInvalidFormat, "The product line does not exist.")
		}
		if apiErr != nil {
			return apiErr
		}
		if apiErr := domain.RequireAssignable("sales.product_line_id", "product line", line.Status); apiErr != nil {
			return apiErr
		}
	}
	if sales.PortalVisibility != nil && !sales.PortalVisibility.IsValid() {
		return apierror.NewFieldError("sales.portal_visibility", apierror.CodeInvalidFormat, "portal_visibility must be visible or hidden.")
	}
	if lt, ok := sales.LeadTime.Value(); ok {
		if apiErr := requireTimeUnit(ctx, repos, lt.UnitID, "sales.lead_time.unit_id"); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// requireTimeUnit rejects a unit that doesn't measure time.
func requireTimeUnit(ctx context.Context, repos domain.RepoFactory, unitID, param string) *apierror.APIError {
	dims, apiErr := repos.NewUnitRepo().GetDimensionCodes(ctx, []string{unitID})
	if apiErr != nil {
		return apiErr
	}
	dim, ok := dims[unitID]
	if !ok {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "The unit does not exist.")
	}
	if dim != string(constants.UnitDimensionTime) {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "A lead time is measured in a time unit, such as days.")
	}
	return nil
}

// insertItemSales writes the product row that holds an item's sales section, hidden and with no line unless sent.
func insertItemSales(ctx context.Context, repos domain.RepoFactory, itemID string, sales *domain.ItemSalesInput) *apierror.APIError {
	productID, apiErr := id.GenID(id.ProductIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	row := domain.ItemProductRow{ProductID: productID, ItemID: itemID}
	if lineID, ok := sales.ProductLineID.Value(); ok {
		row.ProductLineID = &lineID
	}
	if sales.PortalVisibility != nil {
		row.IsPortalReady = *sales.PortalVisibility == constants.CustomerPortalVisibilityVisible
	}
	if lt, ok := sales.LeadTime.Value(); ok {
		quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repos.NewItemRepo().InsertQuantity(ctx, quantityID, lt); apiErr != nil {
			return apiErr
		}
		row.LeadTimeID = &quantityID
	}
	return repos.NewItemRepo().InsertProduct(ctx, row)
}

// updateItemSales merges a sales write into an item's existing product row.
func updateItemSales(ctx context.Context, repos domain.RepoFactory, item *domain.Item, sales *domain.ItemSalesInput) *apierror.APIError {
	itemRepo := repos.NewItemRepo()
	set := domain.SetItemProductParams{ItemID: item.ID}
	if !sales.ProductLineID.IsUnset() {
		set.SetProductLine = true
		set.ProductLineID = clearableValuePtr(sales.ProductLineID)
	}
	if sales.PortalVisibility != nil {
		visible := *sales.PortalVisibility == constants.CustomerPortalVisibilityVisible
		set.IsPortalReady = &visible
	}
	switch lt, ok := sales.LeadTime.Value(); {
	case ok && item.Sales.LeadTimeID != nil:
		if apiErr := itemRepo.UpdateQuantity(ctx, *item.Sales.LeadTimeID, lt); apiErr != nil {
			return apiErr
		}
	case ok:
		quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := itemRepo.InsertQuantity(ctx, quantityID, lt); apiErr != nil {
			return apiErr
		}
		set.SetLeadTime, set.LeadTimeID = true, &quantityID
	case sales.LeadTime.IsClear() && item.Sales.LeadTimeID != nil:
		set.SetLeadTime = true
		if apiErr := itemRepo.SetProduct(ctx, set); apiErr != nil {
			return apiErr
		}
		return itemRepo.DeleteQuantities(ctx, []string{*item.Sales.LeadTimeID})
	}
	return itemRepo.SetProduct(ctx, set)
}

// insertItemMaterial writes a material's order point and lead time (zero when unsent) and its material row.
func insertItemMaterial(ctx context.Context, repos domain.RepoFactory, itemID, baseUnitID string, material *domain.ItemMaterialInput) *apierror.APIError {
	ids, apiErr := genIDs(id.MaterialIDPrefix, id.QuantityIDPrefix, id.QuantityIDPrefix)
	if apiErr != nil {
		return apiErr
	}
	materialID, orderPointID, leadTimeID := ids[0], ids[1], ids[2]
	orderPoint := domain.QuantityInput{Value: "0", UnitID: baseUnitID}
	leadTime := domain.QuantityInput{Value: "0", UnitID: dayUnitID}
	if material != nil {
		if v, ok := material.OrderPoint.Value(); ok {
			orderPoint = v
		}
		if v, ok := material.LeadTime.Value(); ok {
			if apiErr := requireTimeUnit(ctx, repos, v.UnitID, "material.lead_time.unit_id"); apiErr != nil {
				return apiErr
			}
			leadTime = v
		}
	}
	itemRepo := repos.NewItemRepo()
	if apiErr := itemRepo.InsertQuantity(ctx, orderPointID, orderPoint); apiErr != nil {
		return apiErr
	}
	if apiErr := itemRepo.InsertQuantity(ctx, leadTimeID, leadTime); apiErr != nil {
		return apiErr
	}
	return repos.NewMaterialRepo().Create(ctx, materialID, itemID, orderPointID, leadTimeID)
}

// updateItemMaterial merges a material write; a cleared quantity is stored as zero, which reads as none.
func updateItemMaterial(ctx context.Context, repos domain.RepoFactory, section *domain.ItemMaterialSection, material *domain.ItemMaterialInput) *apierror.APIError {
	itemRepo := repos.NewItemRepo()
	apply := func(f fieldQuantity, quantityID string, current *domain.QuantityValue) *apierror.APIError {
		if v, ok := f.Value(); ok {
			return itemRepo.UpdateQuantity(ctx, quantityID, v)
		}
		if f.IsClear() && current != nil {
			return itemRepo.UpdateQuantity(ctx, quantityID, domain.QuantityInput{Value: "0", UnitID: current.UnitID})
		}
		return nil
	}
	if v, ok := material.LeadTime.Value(); ok {
		if apiErr := requireTimeUnit(ctx, repos, v.UnitID, "material.lead_time.unit_id"); apiErr != nil {
			return apiErr
		}
	}
	if apiErr := apply(material.OrderPoint, section.OrderPointID, section.OrderPoint); apiErr != nil {
		return apiErr
	}
	return apply(material.LeadTime, section.LeadTimeID, section.LeadTime)
}

// openItemInventory logs the zero opening level a stocked item starts from.
func openItemInventory(ctx context.Context, repos domain.RepoFactory, accountID, itemID, baseUnitID string) *apierror.APIError {
	invRepo := repos.NewInventoryMutationRepo()
	if apiErr := invRepo.CreateInventoryLog(ctx, domain.CreateInventoryLogParams{
		AccountID: accountID,
		ItemID:    itemID,
		Measure:   decimal.Zero,
		UnitID:    baseUnitID,
	}); apiErr != nil {
		return apiErr
	}
	return invRepo.CreateInventoryChangeLog(ctx, domain.CreateInventoryChangeLogParams{
		AccountID:  accountID,
		ItemID:     itemID,
		Measure:    decimal.Zero,
		UnitID:     baseUnitID,
		ActionType: "user_action",
	})
}

// checkItemReadPermission is the permission an internal user needs to read items: items:read in its own account, or
// customers:read / suppliers:read when it targets one of its customers or suppliers.
func checkItemReadPermission(identity *types.Identity) *apierror.APIError {
	switch {
	case identity.IsTargetCustomerAccount():
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	case identity.IsTargetSupplierAccount():
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	default:
		return identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead)
	}
}

// clearableValuePtr is the value a clearable field sets, or nil when it is unset or cleared.
func clearableValuePtr[T any](f field.Clearable[T]) *T {
	if v, ok := f.Value(); ok {
		return &v
	}
	return nil
}

// genIDs generates one id per prefix, in order.
func genIDs(prefixes ...id.IDPrefix) ([]string, *apierror.APIError) {
	out := make([]string, len(prefixes))
	for i, prefix := range prefixes {
		v, apiErr := id.GenID(prefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		out[i] = v
	}
	return out, nil
}

// requireItemsAssignable rejects newly assigning any archived item among itemIDs with a 422 resource_archived on param.
func requireItemsAssignable(ctx context.Context, repos domain.RepoFactory, accountID string, itemIDs []string, param string) *apierror.APIError {
	states, apiErr := repos.NewItemRepo().GetAssignability(ctx, accountID, uniqueIDs(itemIDs))
	if apiErr != nil {
		return apiErr
	}
	for _, state := range states {
		if !state.IsAssignable {
			return domain.RequireAssignable(param, "item", constants.ArchiveStatusArchived)
		}
	}
	return nil
}
