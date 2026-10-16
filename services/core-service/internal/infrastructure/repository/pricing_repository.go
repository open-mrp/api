package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

var pricingRepoTracer = tracing.GetTracer("core-service.pricing_repository")

type pricingRepoImpl struct {
	queries *sqlc.Queries
}

func NewPricingRepo(queries *sqlc.Queries) domain.PricingRepo {
	return &pricingRepoImpl{queries: queries}
}

func (r *pricingRepoImpl) LoadPricingBundle(ctx context.Context, params domain.LoadPricingBundleParams) (*domain.PricingBundle, *apierror.APIError) {
	ctx, span := pricingRepoTracer.Start(ctx, "repository.pricing.load_bundle")
	defer span.End()

	bundle := &domain.PricingBundle{
		Products:  make(map[string]*domain.PricingProduct),
		Units:     make(map[string]*domain.PricingUnit),
		ItemUnits: make(map[string]*domain.ItemUnits),
	}

	productIDs := dedupe(params.ProductIDs)

	// --- Products (list price + unit groups) --------------------------------
	if len(productIDs) > 0 {
		rows, err := r.queries.GetPricingProductsByIDs(ctx, sqlc.GetPricingProductsByIDsParams{
			ProductIds: productIDs,
			AccountID:  params.OwnerAccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range rows {
			p := &domain.PricingProduct{
				ProductID:                  row.ProductID,
				ItemID:                     row.ItemID,
				SKU:                        row.ItemSku,
				UnitCost:                   row.UnitCost,
				UnitCostNumeratorUnitID:    row.UnitCostNumeratorUnitID,
				UnitCostDenominatorUnitID:  row.UnitCostDenominatorUnitID,
				UnitValue:                  row.UnitValue,
				UnitValueNumeratorUnitID:   row.UnitValueNumeratorUnitID,
				UnitValueDenominatorUnitID: row.UnitValueDenominatorUnitID,
				CategoryUnitGroupID:        row.CategoryUnitGroupID,
				ItemCategoryID:             row.ItemCategoryID,
			}
			if row.ItemDescription.Valid {
				v := row.ItemDescription.String
				p.Description = &v
			}
			if row.ProductLineID.Valid {
				v := row.ProductLineID.String
				p.ProductLineID = &v
			}
			if row.ProductLineUnitGroupID.Valid {
				v := row.ProductLineUnitGroupID.String
				p.ProductLineUnitGroupID = &v
			}
			bundle.Products[row.ProductID] = p
		}

		// Product attribute ids.
		attrRows, err := r.queries.GetPricingProductAttributeIDs(ctx, productIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range attrRows {
			if p, ok := bundle.Products[row.ProductID]; ok {
				p.AttributeIDs = append(p.AttributeIDs, row.AttributeID)
			}
		}
	}

	// --- Item units (stocking unit, order units and their discounts) -------
	itemIDs := make([]string, 0, len(bundle.Products))
	for _, p := range bundle.Products {
		itemIDs = append(itemIDs, p.ItemID)
	}
	itemUnits, apiErr := NewItemUnitsRepo(r.queries).Get(ctx, itemIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	bundle.ItemUnits = itemUnits

	// --- Units (conversion data) --------------------------------------------
	// Collect every unit id we may need: each line's ordered unit, every product's base denominator unit and every
	// item's stocking unit.
	unitIDSet := make(map[string]struct{})
	for _, u := range itemUnits {
		if u.StockingUnitID != "" {
			unitIDSet[u.StockingUnitID] = struct{}{}
		}
	}
	for _, id := range params.OrderedUnitIDs {
		if id != "" {
			unitIDSet[id] = struct{}{}
		}
	}
	for _, p := range bundle.Products {
		if p.UnitValueDenominatorUnitID != "" {
			unitIDSet[p.UnitValueDenominatorUnitID] = struct{}{}
		}
	}
	unitIDs := keys(unitIDSet)
	if len(unitIDs) > 0 {
		rows, err := r.queries.GetPricingUnitsByIDs(ctx, unitIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range rows {
			bundle.Units[row.ID] = &domain.PricingUnit{
				ID:                row.ID,
				RatioNumerator:    row.RatioNumerator,
				RatioDenominator:  row.RatioDenominator,
				OffsetNumerator:   row.OffsetNumerator,
				OffsetDenominator: row.OffsetDenominator,
				IsBaseUnit:        row.IsBaseUnit,
			}
		}
	}

	// --- Account prices (buyer + parent account) ----------------------------
	recipientIDs := []string{params.BuyerAccountID}
	parentID, err := r.queries.GetBuyerParentAccountID(ctx, sqlc.GetBuyerParentAccountIDParams{
		OwnerAccountID:    params.OwnerAccountID,
		CustomerAccountID: params.BuyerAccountID,
	})
	switch {
	case err == nil:
		if parentID != "" && parentID != params.BuyerAccountID {
			recipientIDs = append(recipientIDs, parentID)
		}
	case err == gosql.ErrNoRows:
		// No parent account; only the buyer is a recipient.
	default:
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	apRows, err := r.queries.GetPricingAccountPrices(ctx, sqlc.GetPricingAccountPricesParams{
		OwnerAccountID:      params.OwnerAccountID,
		RecipientAccountIds: recipientIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	apByID := make(map[string]*domain.PricingAccountPrice, len(apRows))
	apIDs := make([]string, 0, len(apRows))
	for _, row := range apRows {
		ap := &domain.PricingAccountPrice{
			ID:                row.ID,
			ProductLineID:     row.ProductLineID,
			UnitValue:         row.UnitValue,
			NumeratorUnitID:   row.NumeratorUnitID,
			DenominatorUnitID: row.DenominatorUnitID,
			CreatedAt:         row.CreatedAt,
		}
		apByID[row.ID] = ap
		apIDs = append(apIDs, row.ID)
		bundle.AccountPrices = append(bundle.AccountPrices, ap)
	}
	if len(apIDs) > 0 {
		attrRows, err := r.queries.GetPricingAccountPriceAttributeIDs(ctx, apIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range attrRows {
			if ap, ok := apByID[row.AccountPriceID]; ok {
				ap.AttributeIDs = append(ap.AttributeIDs, row.AttributeID)
			}
		}
	}

	// --- Volume discounts ----------------------------------------------------
	vdRows, err := r.queries.GetPricingVolumeDiscountsForCustomer(ctx, sqlc.GetPricingVolumeDiscountsForCustomerParams{
		AccountID:         params.OwnerAccountID,
		CustomerAccountID: params.BuyerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	vdByID := make(map[string]*domain.PricingVolumeDiscount, len(vdRows))
	vdIDs := make([]string, 0, len(vdRows))
	nullVdIDs := make([]gosql.NullString, 0, len(vdRows))
	for _, row := range vdRows {
		vd := &domain.PricingVolumeDiscount{
			ID:                   row.ID,
			MatchesCustomerGroup: row.MatchesCustomerGroup,
		}
		vdByID[row.ID] = vd
		vdIDs = append(vdIDs, row.ID)
		nullVdIDs = append(nullVdIDs, gosql.NullString{String: row.ID, Valid: true})
		bundle.VolumeDiscounts = append(bundle.VolumeDiscounts, vd)
	}
	if len(vdIDs) > 0 {
		tierRows, err := r.queries.GetPricingVolumeDiscountTiers(ctx, nullVdIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range tierRows {
			if !row.QuantityDiscountID.Valid {
				continue
			}
			if vd, ok := vdByID[row.QuantityDiscountID.String]; ok {
				vd.Tiers = append(vd.Tiers, domain.PricingVolumeDiscountTier{
					Threshold:          row.Threshold,
					DiscountPercentage: row.DiscountPercentage,
				})
			}
		}

		unitRows, err := r.queries.GetPricingVolumeDiscountUnits(ctx, vdIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range unitRows {
			if vd, ok := vdByID[row.QuantityDiscountID]; ok {
				vd.AcceptableUnitIDs = append(vd.AcceptableUnitIDs, row.UnitID)
			}
		}

		// Scoping associations: product lines, item categories, attributes.
		plRows, err := r.queries.GetPricingVolumeDiscountProductLineIDs(ctx, vdIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range plRows {
			if vd, ok := vdByID[row.QuantityDiscountID]; ok {
				vd.ProductLineIDs = append(vd.ProductLineIDs, row.ProductLineID)
			}
		}

		catRows, err := r.queries.GetPricingVolumeDiscountCategoryIDs(ctx, vdIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range catRows {
			if vd, ok := vdByID[row.QuantityDiscountID]; ok {
				vd.CategoryIDs = append(vd.CategoryIDs, row.ItemCategoryID)
			}
		}

		vdAttrRows, err := r.queries.GetPricingVolumeDiscountAttributeIDs(ctx, vdIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range vdAttrRows {
			if vd, ok := vdByID[row.QuantityDiscountID]; ok {
				vd.AttributeIDs = append(vd.AttributeIDs, row.AttributeID)
			}
		}
	}

	return bundle, nil
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// ProductQuantityUnits maps each product to the units its item can be ordered in: its stocking unit and order units.
func (r *pricingRepoImpl) ProductQuantityUnits(ctx context.Context, accountID string, productIDs []string) (map[string]map[string]struct{}, *apierror.APIError) {
	ctx, span := pricingRepoTracer.Start(ctx, "repository.pricing.product_quantity_units")
	defer span.End()

	if len(productIDs) == 0 {
		return map[string]map[string]struct{}{}, nil
	}
	products, err := r.queries.GetPricingProductsByIDs(ctx, sqlc.GetPricingProductsByIDsParams{
		ProductIds: productIDs,
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	itemIDs := make([]string, len(products))
	for i, p := range products {
		itemIDs[i] = p.ItemID
	}
	units, apiErr := NewItemUnitsRepo(r.queries).Get(ctx, itemIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make(map[string]map[string]struct{}, len(products))
	for _, p := range products {
		out[p.ProductID] = allowedUnitSet(units[p.ItemID])
	}
	return out, nil
}

// ItemQuantityUnits maps each of the account's items to the units it can be ordered in.
func (r *pricingRepoImpl) ItemQuantityUnits(ctx context.Context, accountID string, itemIDs []string) (map[string]map[string]struct{}, *apierror.APIError) {
	ctx, span := pricingRepoTracer.Start(ctx, "repository.pricing.item_quantity_units")
	defer span.End()

	if len(itemIDs) == 0 {
		return map[string]map[string]struct{}{}, nil
	}
	items, err := r.queries.GetPurchaseOrderLineItemUnitGroups(ctx, sqlc.GetPurchaseOrderLineItemUnitGroupsParams{
		ItemIds:   itemIDs,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	owned := make([]string, len(items))
	for i, it := range items {
		owned[i] = it.ItemID
	}
	units, apiErr := NewItemUnitsRepo(r.queries).Get(ctx, owned)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make(map[string]map[string]struct{}, len(items))
	for _, id := range owned {
		out[id] = allowedUnitSet(units[id])
	}
	return out, nil
}

func allowedUnitSet(u *domain.ItemUnits) map[string]struct{} {
	out := make(map[string]struct{})
	for _, id := range u.AllowedUnitIDs() {
		out[id] = struct{}{}
	}
	return out
}
