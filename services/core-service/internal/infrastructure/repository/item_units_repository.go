package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
)

// Items used to take their units from unit groups: the stocking unit was the category group's base unit, and the
// units an item could be ordered in were its product line's group (or else its category's), each member carrying a
// portal visibility and a discount. Each item now stores its own (migration 00099). Until ItemUnitsBackfill or a
// write stores an item's units, they are derived here from its groups exactly as pricing and the order checks read
// them, so nothing changes for the item when they are stored.

type itemUnitsRepo struct {
	queries *sqlc.Queries
}

// NewItemUnitsRepo reads and writes items' units.
func NewItemUnitsRepo(queries *sqlc.Queries) domain.ItemUnitsRepo {
	return &itemUnitsRepo{queries: queries}
}

func (r *itemUnitsRepo) Get(ctx context.Context, itemIDs []string) (map[string]*domain.ItemUnits, *apierror.APIError) {
	itemIDs = uniqueNonEmpty(itemIDs)
	out := make(map[string]*domain.ItemUnits, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	sources, err := r.queries.GetItemUnitSources(ctx, itemIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	var storedIDs []string
	var unstored []sqlc.GetItemUnitSourcesRow
	for _, s := range sources {
		if s.DefaultOrderUnitID.Valid {
			storedIDs = append(storedIDs, s.ID)
			out[s.ID] = &domain.ItemUnits{
				StockingUnitID:     s.StockingUnitID.String,
				DefaultOrderUnitID: s.DefaultOrderUnitID.String,
				OrderUnits:         []domain.ItemOrderUnit{},
				Stored:             true,
			}
		} else {
			unstored = append(unstored, s)
		}
	}
	if len(storedIDs) > 0 {
		rows, err := r.queries.GetItemOrderUnitsByItemIDs(ctx, storedIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, apiErr
		}
		for _, row := range rows {
			if u := out[row.ItemID]; u != nil {
				u.OrderUnits = append(u.OrderUnits, orderUnitFromRow(row))
			}
		}
	}
	if len(unstored) > 0 {
		derived, apiErr := r.derive(ctx, unstored)
		if apiErr != nil {
			return nil, apiErr
		}
		for id, u := range derived {
			out[id] = u
		}
	}
	return out, nil
}

func orderUnitFromRow(row sqlc.GetItemOrderUnitsByItemIDsRow) domain.ItemOrderUnit {
	ou := domain.ItemOrderUnit{
		UnitID:           row.UnitID,
		PortalVisibility: constants.CustomerPortalVisibility(row.PortalVisibility),
	}
	if row.StockingQuantity.Valid {
		q := trimDecimal(row.StockingQuantity.String)
		ou.StockingQuantity = &q
	}
	switch constants.DiscountType(row.DiscountType.String) {
	case constants.DiscountTypePercentOff:
		pct := trimDecimal(row.DiscountPercentOff.String)
		ou.Discount = &domain.Discount{Type: constants.DiscountTypePercentOff, PercentOff: &pct}
	case constants.DiscountTypeAmountOff:
		ou.Discount = &domain.Discount{Type: constants.DiscountTypeAmountOff, AmountOff: &domain.Money{
			Amount:   trimDecimal(row.DiscountAmountOffAmount.String),
			Currency: constants.Currency(row.DiscountAmountOffCurrency.String),
		}}
	}
	return ou
}

// trimDecimal drops a stored decimal's fixed-point padding.
func trimDecimal(s string) string {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return s
	}
	return d.String()
}

// derive computes the units of items not stored yet from their unit groups. A physical item is stocked in its
// category group's base unit. Its order units are the members of its product line's group when it has one, else of
// its category's, other than the stocking unit and any of another dimension. A service is never stocked: every member
// is an order unit and the group's base unit is the default. A member's discount was taken off the base price before
// converting it to the member's unit; it becomes the same discount after converting: a percentage as percent_off, a
// fixed amount as amount_off (the amount scaled to the unit), and both together as the amount they take off the
// current price.
func (r *itemUnitsRepo) derive(ctx context.Context, items []sqlc.GetItemUnitSourcesRow) (map[string]*domain.ItemUnits, *apierror.APIError) {
	groupIDs := make([]string, 0, 2*len(items))
	unitIDs := make([]string, 0, 3*len(items))
	for _, it := range items {
		groupIDs = append(groupIDs, orderGroupID(it))
		unitIDs = append(unitIDs, it.CategoryBaseUnitID, it.UnitValuePerUnitID, it.ProductLineBaseUnitID.String)
	}
	members, err := r.queries.GetUnitGroupMembersForItemUnits(ctx, uniqueNonEmpty(groupIDs))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	byGroup := make(map[string][]sqlc.GetUnitGroupMembersForItemUnitsRow)
	for _, m := range members {
		byGroup[m.UnitGroupID] = append(byGroup[m.UnitGroupID], m)
		unitIDs = append(unitIDs, m.UnitID)
	}
	units, err := r.queries.GetUnitFactorsForItemUnits(ctx, uniqueNonEmpty(unitIDs))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	factors := make(map[string]sqlc.GetUnitFactorsForItemUnitsRow, len(units))
	for _, u := range units {
		factors[u.ID] = u
	}

	out := make(map[string]*domain.ItemUnits, len(items))
	for _, it := range items {
		groupID := orderGroupID(it)
		baseUnitID := it.CategoryBaseUnitID
		if it.ProductLineUnitGroupID.Valid && it.ProductLineUnitGroupID.String != "" && it.ProductLineBaseUnitID.Valid {
			if constants.ItemTypeCode(it.ItemTypeCode) == constants.ItemTypeCodeService {
				baseUnitID = it.ProductLineBaseUnitID.String
			}
		}
		u := &domain.ItemUnits{OrderUnits: []domain.ItemOrderUnit{}}
		service := constants.ItemTypeCode(it.ItemTypeCode) == constants.ItemTypeCodeService
		if !service {
			u.StockingUnitID = it.CategoryBaseUnitID
		}
		u.DefaultOrderUnitID = baseUnitID
		dimension := factors[baseUnitID].UnitDimensionCode

		price, _ := decimal.NewFromString(it.UnitValue)
		valueFactor := itemUnitRatio(factors[it.UnitValuePerUnitID])
		seenBase := false
		for _, m := range byGroup[groupID] {
			if m.UnitID == u.StockingUnitID {
				continue
			}
			f, ok := factors[m.UnitID]
			if !ok || (dimension != "" && f.UnitDimensionCode != dimension) {
				continue
			}
			if m.UnitID == baseUnitID {
				seenBase = true
			}
			ou := domain.ItemOrderUnit{UnitID: m.UnitID, PortalVisibility: constants.CustomerPortalVisibilityVisible}
			if !m.IsVisible {
				ou.PortalVisibility = constants.CustomerPortalVisibilityHidden
			}
			ou.Discount = derivedDiscount(m.DiscountPercentage, m.DiscountFixed, price, itemUnitRatio(f), valueFactor)
			u.OrderUnits = append(u.OrderUnits, ou)
		}
		if service && !seenBase && baseUnitID != "" {
			u.OrderUnits = append([]domain.ItemOrderUnit{{UnitID: baseUnitID, PortalVisibility: constants.CustomerPortalVisibilityVisible}}, u.OrderUnits...)
		}
		out[it.ID] = u
	}
	return out, nil
}

// orderGroupID is the group an item's order units came from: its product line's when it has one, else its category's.
func orderGroupID(it sqlc.GetItemUnitSourcesRow) string {
	if it.ProductLineUnitGroupID.Valid && it.ProductLineUnitGroupID.String != "" {
		return it.ProductLineUnitGroupID.String
	}
	return it.CategoryUnitGroupID
}

// itemUnitRatio is how many of its dimension's base unit one of the unit holds; zero when unknown.
func itemUnitRatio(f sqlc.GetUnitFactorsForItemUnitsRow) decimal.Decimal {
	num, err1 := decimal.NewFromString(f.RatioNumerator)
	den, err2 := decimal.NewFromString(f.RatioDenominator)
	if err1 != nil || err2 != nil || den.IsZero() {
		return decimal.Zero
	}
	return num.Div(den)
}

// derivedDiscount restates a group member's discount, (price − fixed) × (1 − percentage) before converting price
// from its per unit to the member's unit, as a discount taken after converting.
func derivedDiscount(percentage, fixed string, price, memberRatio, valueRatio decimal.Decimal) *domain.Discount {
	pct, _ := decimal.NewFromString(percentage)
	amt, _ := decimal.NewFromString(fixed)
	if pct.IsZero() && amt.IsZero() {
		return nil
	}
	if amt.IsZero() {
		percentOff := pct.Mul(decimal.NewFromInt(100)).String()
		return &domain.Discount{Type: constants.DiscountTypePercentOff, PercentOff: &percentOff}
	}
	k := decimal.NewFromInt(1)
	if !valueRatio.IsZero() && !memberRatio.IsZero() {
		k = memberRatio.Div(valueRatio)
	}
	// k × (fixed + percentage × (price − fixed)) is what the old order took off the converted price.
	off := k.Mul(amt.Add(pct.Mul(price.Sub(amt))))
	return &domain.Discount{Type: constants.DiscountTypeAmountOff, AmountOff: &domain.Money{
		Amount: off.Round(6).String(), Currency: constants.CurrencyUSD,
	}}
}

func (r *itemUnitsRepo) Materialize(ctx context.Context, itemIDs []string) *apierror.APIError {
	itemIDs = uniqueNonEmpty(itemIDs)
	if len(itemIDs) == 0 {
		return nil
	}
	sources, err := r.queries.GetItemUnitSources(ctx, itemIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	var unstored []sqlc.GetItemUnitSourcesRow
	accounts := make(map[string]string, len(sources))
	for _, s := range sources {
		if !s.DefaultOrderUnitID.Valid {
			unstored = append(unstored, s)
			accounts[s.ID] = s.AccountID
		}
	}
	if len(unstored) == 0 {
		return nil
	}
	derived, apiErr := r.derive(ctx, unstored)
	if apiErr != nil {
		return apiErr
	}
	for _, s := range unstored {
		if apiErr := r.storeIfUnstored(ctx, accounts[s.ID], s.ID, derived[s.ID]); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// storeIfUnstored writes derived units onto an item that still has none stored; a writer that stored them first wins.
func (r *itemUnitsRepo) storeIfUnstored(ctx context.Context, accountID, itemID string, u *domain.ItemUnits) *apierror.APIError {
	if u == nil || u.DefaultOrderUnitID == "" {
		return nil
	}
	n, err := r.queries.SetItemUnitsIfUnstored(ctx, sqlc.SetItemUnitsIfUnstoredParams{
		StockingUnitID:     sql.NullString{String: u.StockingUnitID, Valid: u.StockingUnitID != ""},
		DefaultOrderUnitID: sql.NullString{String: u.DefaultOrderUnitID, Valid: true},
		ID:                 itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	if n == 0 {
		return nil
	}
	if err := r.queries.DeleteItemOrderUnits(ctx, itemID); err != nil {
		return db.MapSQLError(err)
	}
	return r.insertOrderUnits(ctx, accountID, itemID, u.OrderUnits)
}

func (r *itemUnitsRepo) insertOrderUnits(ctx context.Context, accountID, itemID string, orderUnits []domain.ItemOrderUnit) *apierror.APIError {
	for i, ou := range orderUnits {
		params := sqlc.InsertItemOrderUnitParams{
			ItemID:           itemID,
			UnitID:           ou.UnitID,
			AccountID:        accountID,
			Position:         int32(i), //nolint:gosec // bounded by the request's list length
			PortalVisibility: string(ou.PortalVisibility),
		}
		if params.PortalVisibility == "" {
			params.PortalVisibility = string(constants.CustomerPortalVisibilityVisible)
		}
		if ou.StockingQuantity != nil {
			params.StockingQuantity = sql.NullString{String: *ou.StockingQuantity, Valid: true}
		}
		if d := ou.Discount; d != nil {
			params.DiscountType = sql.NullString{String: string(d.Type), Valid: true}
			if d.PercentOff != nil {
				params.DiscountPercentOff = sql.NullString{String: *d.PercentOff, Valid: true}
			}
			if d.AmountOff != nil {
				params.DiscountAmountOffAmount = sql.NullString{String: d.AmountOff.Amount, Valid: true}
				params.DiscountAmountOffCurrency = sql.NullString{String: string(d.AmountOff.Currency), Valid: true}
			}
		}
		if err := r.queries.InsertItemOrderUnit(ctx, params); err != nil {
			return db.MapSQLError(err)
		}
	}
	return nil
}

func (r *itemUnitsRepo) SetStockingUnit(ctx context.Context, itemID, unitID string) *apierror.APIError {
	return db.MapSQLError(r.queries.SetItemStockingUnit(ctx, sqlc.SetItemStockingUnitParams{
		StockingUnitID: sql.NullString{String: unitID, Valid: unitID != ""},
		ID:             itemID,
	}))
}

func (r *itemUnitsRepo) Replace(ctx context.Context, accountID, itemID, defaultOrderUnitID string, orderUnits []domain.ItemOrderUnit) *apierror.APIError {
	if err := r.queries.SetItemDefaultOrderUnit(ctx, sqlc.SetItemDefaultOrderUnitParams{
		DefaultOrderUnitID: sql.NullString{String: defaultOrderUnitID, Valid: defaultOrderUnitID != ""},
		ID:                 itemID,
	}); err != nil {
		return db.MapSQLError(err)
	}
	if err := r.queries.DeleteItemOrderUnits(ctx, itemID); err != nil {
		return db.MapSQLError(err)
	}
	return r.insertOrderUnits(ctx, accountID, itemID, orderUnits)
}

func (r *itemUnitsRepo) RemoveOrderUnits(ctx context.Context, itemID string, unitIDs []string) *apierror.APIError {
	if len(unitIDs) == 0 {
		return nil
	}
	return db.MapSQLError(r.queries.DeleteItemOrderUnitsByUnitIDs(ctx, sqlc.DeleteItemOrderUnitsByUnitIDsParams{
		ItemID: itemID, UnitIds: unitIDs,
	}))
}

// uniqueNonEmpty drops blanks and repeats, keeping order.
func uniqueNonEmpty(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (r *itemUnitsRepo) MatchItems(ctx context.Context, accountID string, f domain.SetOrderUnitsFilter, limit int) ([]domain.ItemTypeRef, *apierror.APIError) {
	where := []string{"i.account_id = ?", "i.deleted_at IS NULL"}
	args := []any{accountID}
	if len(f.ItemIDs) > 0 {
		where = append(where, "i.id IN ("+placeholders(len(f.ItemIDs))+")")
		args = append(args, stringsToAny(f.ItemIDs)...)
	}
	if len(f.CategoryIDs) > 0 {
		where = append(where, "i.item_category_id IN ("+placeholders(len(f.CategoryIDs))+")")
		args = append(args, stringsToAny(f.CategoryIDs)...)
	}
	if len(f.ProductLineIDs) > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id AND p.product_line_id IN ("+placeholders(len(f.ProductLineIDs))+"))")
		args = append(args, stringsToAny(f.ProductLineIDs)...)
	}
	if len(f.AttributeIDs) > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM `_item_attributes` ia WHERE ia.B = i.id AND ia.A IN ("+placeholders(len(f.AttributeIDs))+"))")
		args = append(args, stringsToAny(f.AttributeIDs)...)
	}
	args = append(args, limit)
	rows, err := r.queries.DB().QueryContext(ctx, "SELECT i.id, i.item_type_code FROM item i WHERE "+strings.Join(where, " AND ")+" ORDER BY i.id LIMIT ?", args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = rows.Close() }()
	var out []domain.ItemTypeRef
	for rows.Next() {
		var ref domain.ItemTypeRef
		if err := rows.Scan(&ref.ID, &ref.Type); err != nil {
			return nil, db.MapSQLError(err)
		}
		out = append(out, ref)
	}
	return out, db.MapSQLError(rows.Err())
}
