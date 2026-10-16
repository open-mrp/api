package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var priceListRepoTracer = tracing.GetTracer("core-service.price_list_repository")

type priceListRepoImpl struct {
	queries *sqlc.Queries
}

func NewPriceListRepo(queries *sqlc.Queries) domain.PriceListRepo {
	return &priceListRepoImpl{queries: queries}
}

const priceListColumns = `pl.id, pl.account_id, pl.name, pl.description, pl.is_active, pl.currency, pl.source_type, ` +
	`pl.source_id, pl.created_at, pl.updated_at`

func priceListFromSQLC(r sqlc.PriceList) *domain.PriceList {
	out := &domain.PriceList{
		ID:        r.ID,
		AccountID: r.AccountID,
		Name:      r.Name,
		Status:    constants.ArchiveStatusFromIsActive(r.IsActive),
		Currency:  constants.Currency(r.Currency),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	if r.Description.Valid {
		out.Description = &r.Description.String
	}
	if r.SourceType.Valid && r.SourceID.Valid {
		t := constants.PriceListSourceType(r.SourceType.String)
		out.SourceType, out.SourceID = &t, &r.SourceID.String
	}
	return out
}

func priceListCreatedAt(p *domain.PriceList) time.Time         { return p.CreatedAt }
func priceListID(p *domain.PriceList) string                   { return p.ID }
func priceListRuleCreatedAt(r *domain.PriceListRule) time.Time { return r.CreatedAt }
func priceListRuleID(r *domain.PriceListRule) string           { return r.ID }

// decodePriceListCursor reads a list cursor, or nil when there is none.
func decodePriceListCursor(raw *string) (*pagination.StringCursor, *pagination.Direction, pagination.Direction, *apierror.APIError) {
	dir := pagination.DirectionForward
	if raw == nil {
		return nil, nil, dir, nil
	}
	cur, err := pagination.DecodeStringCursor(*raw)
	if err != nil {
		return nil, nil, dir, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
	}
	return &cur, &cur.Direction, cur.Direction, nil
}

// keysetTail appends the cursor predicate and order for a created_at, id list on alias.
func keysetTail(alias string, where []string, args []any, cursor *pagination.StringCursor, dir pagination.Direction, limit int32) (string, []any) {
	if cursor != nil {
		cmp := "<"
		if dir == pagination.DirectionBackward {
			cmp = ">"
		}
		where = append(where, "("+alias+".created_at "+cmp+" ? OR ("+alias+".created_at = ? AND "+alias+".id "+cmp+" ?))")
		args = append(args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
	order := " ORDER BY " + alias + ".created_at DESC, " + alias + ".id DESC"
	if dir == pagination.DirectionBackward {
		order = " ORDER BY " + alias + ".created_at ASC, " + alias + ".id ASC"
	}
	args = append(args, limit)
	return " WHERE " + strings.Join(where, " AND ") + order + " LIMIT ?", args
}

func (r *priceListRepoImpl) List(ctx context.Context, params domain.ListPriceListsParams) (*domain.ListPriceListsResult, *apierror.APIError) {
	ctx, span := priceListRepoTracer.Start(ctx, "repository.price_list.list")
	defer span.End()

	cursor, cursorDir, dir, apiErr := decodePriceListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.IDs != nil && len(params.IDs) == 0 {
		return &domain.ListPriceListsResult{PriceLists: []*domain.PriceList{}}, nil
	}
	where := []string{"pl.account_id = ?"}
	args := []any{params.AccountID}
	if len(params.IDs) > 0 {
		where = append(where, "pl.id IN ("+placeholders(len(params.IDs))+")")
		args = append(args, stringArgs(params.IDs)...)
	}
	if active := constants.IsActiveFilter(params.Statuses); active != nil {
		where = append(where, "pl.is_active = ?")
		args = append(args, *active)
	}
	if params.Query != nil && strings.TrimSpace(*params.Query) != "" {
		where = append(where, "pl.name LIKE ?")
		args = append(args, "%"+db.EscapeLike(strings.TrimSpace(*params.Query))+"%")
	}
	tail, args := keysetTail("pl", where, args, cursor, dir, params.Limit+1)
	rows, err := r.queries.DB().QueryContext(ctx, "SELECT "+priceListColumns+" FROM price_list pl"+tail, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var lists []*domain.PriceList
	for rows.Next() {
		var p sqlc.PriceList
		if err := rows.Scan(&p.ID, &p.AccountID, &p.Name, &p.Description, &p.IsActive, &p.Currency, &p.SourceType,
			&p.SourceID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		lists = append(lists, priceListFromSQLC(p))
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(lists, params.Limit, cursorDir, priceListCreatedAt, priceListID)
	return &domain.ListPriceListsResult{PriceLists: result, PageInfo: pageInfo}, nil
}

func (r *priceListRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.PriceList, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetPriceListsByIDs(ctx, sqlc.GetPriceListsByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	out := make([]*domain.PriceList, len(rows))
	for i, row := range rows {
		out[i] = priceListFromSQLC(row)
	}
	return out, nil
}

func (r *priceListRepoImpl) GetBySource(ctx context.Context, accountID string, sourceType constants.PriceListSourceType, sourceID string) (*domain.PriceList, *apierror.APIError) {
	row, err := r.queries.GetPriceListBySource(ctx, sqlc.GetPriceListBySourceParams{
		AccountID:  accountID,
		SourceType: gosql.NullString{String: string(sourceType), Valid: true},
		SourceID:   gosql.NullString{String: sourceID, Valid: true},
	})
	if errors.Is(err, gosql.ErrNoRows) {
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return priceListFromSQLC(row), nil
}

func (r *priceListRepoImpl) ListBySourceType(ctx context.Context, accountID string, sourceType constants.PriceListSourceType) ([]*domain.PriceList, *apierror.APIError) {
	rows, err := r.queries.ListPriceListsBySourceType(ctx, sqlc.ListPriceListsBySourceTypeParams{
		AccountID:  accountID,
		SourceType: gosql.NullString{String: string(sourceType), Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	out := make([]*domain.PriceList, len(rows))
	for i, row := range rows {
		out[i] = priceListFromSQLC(row)
	}
	return out, nil
}

func nullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func (r *priceListRepoImpl) Insert(ctx context.Context, list *domain.PriceList) *apierror.APIError {
	params := sqlc.InsertPriceListParams{
		ID:          list.ID,
		AccountID:   list.AccountID,
		Name:        list.Name,
		Description: nullString(list.Description),
		IsActive:    list.Status.IsActive(),
		Currency:    string(list.Currency),
		SourceID:    nullString(list.SourceID),
	}
	if list.SourceType != nil {
		params.SourceType = gosql.NullString{String: string(*list.SourceType), Valid: true}
	}
	return db.MapSQLError(r.queries.InsertPriceList(ctx, params))
}

func (r *priceListRepoImpl) Update(ctx context.Context, list *domain.PriceList) *apierror.APIError {
	return db.MapSQLError(r.queries.UpdatePriceList(ctx, sqlc.UpdatePriceListParams{
		ID:          list.ID,
		AccountID:   list.AccountID,
		Name:        list.Name,
		Description: nullString(list.Description),
		IsActive:    list.Status.IsActive(),
	}))
}

// Delete removes the list with its rules and assignments.
func (r *priceListRepoImpl) Delete(ctx context.Context, accountID, id string) *apierror.APIError {
	ruleRows, err := r.queries.GetPriceListRuleIDsByListIDs(ctx, []string{id})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	ruleIDs := make([]string, len(ruleRows))
	for i, row := range ruleRows {
		ruleIDs[i] = row.ID
	}
	if apiErr := r.deleteRuleChildren(ctx, ruleIDs); apiErr != nil {
		return apiErr
	}
	if apiErr := db.MapSQLError(r.queries.DeletePriceListRulesByList(ctx, id)); apiErr != nil {
		return apiErr
	}
	if apiErr := db.MapSQLError(r.queries.DeletePriceListAssignmentsByList(ctx, id)); apiErr != nil {
		return apiErr
	}
	if apiErr := db.MapSQLError(r.queries.DeletePriceListGroupAssignmentsByList(ctx, id)); apiErr != nil {
		return apiErr
	}
	return db.MapSQLError(r.queries.DeletePriceList(ctx, sqlc.DeletePriceListParams{ID: id, AccountID: accountID}))
}

// --- Rules -------------------------------------------------------------------

const priceListRuleColumns = `r.id, r.price_list_id, r.account_id, r.price_type, r.unit_price_amount, r.unit_price_currency, ` +
	`r.unit_price_per_unit_id, r.percent_of_base, r.tier_unit_id, r.source_id, r.created_at, r.updated_at`

func scanPriceListRules(rows *gosql.Rows) ([]sqlc.PriceListRule, error) {
	var out []sqlc.PriceListRule
	for rows.Next() {
		var p sqlc.PriceListRule
		if err := rows.Scan(&p.ID, &p.PriceListID, &p.AccountID, &p.PriceType, &p.UnitPriceAmount, &p.UnitPriceCurrency,
			&p.UnitPricePerUnitID, &p.PercentOfBase, &p.TierUnitID, &p.SourceID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *priceListRepoImpl) FirstRuleIDs(ctx context.Context, listIDs []string) (map[string]domain.PriceListRuleIDs, *apierror.APIError) {
	out := make(map[string]domain.PriceListRuleIDs, len(listIDs))
	for _, listID := range listIDs {
		rows, err := r.queries.DB().QueryContext(ctx,
			"SELECT r.id, r.created_at FROM price_list_rule r WHERE r.price_list_id = ? ORDER BY r.created_at DESC, r.id DESC LIMIT ?",
			listID, domain.PriceListEmbeddedRules)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, apiErr
		}
		var ids []string
		var times []time.Time
		for rows.Next() {
			var id string
			var at time.Time
			if err := rows.Scan(&id, &at); err != nil {
				_ = rows.Close()
				return nil, db.MapSQLError(err)
			}
			ids = append(ids, id)
			times = append(times, at)
		}
		_ = rows.Close()
		entry := domain.PriceListRuleIDs{IDs: ids}
		if shown := domain.PriceListEmbeddedRules - 1; len(ids) > shown {
			entry.NextCursor = pagination.EncodeStringCursor(pagination.StringCursor{
				OccurredAt: times[shown-1], ID: ids[shown-1], Direction: pagination.DirectionForward,
			})
		}
		out[listID] = entry
	}
	return out, nil
}

func (r *priceListRepoImpl) ListRules(ctx context.Context, params domain.ListPriceListRulesParams) (*domain.ListPriceListRulesResult, *apierror.APIError) {
	ctx, span := priceListRepoTracer.Start(ctx, "repository.price_list.list_rules")
	defer span.End()

	cursor, cursorDir, dir, apiErr := decodePriceListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	where := []string{"r.price_list_id = ?", "r.account_id = ?"}
	args := []any{params.PriceListID, params.AccountID}
	if len(params.ItemIDs) > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM price_list_rule_item s WHERE s.price_list_rule_id = r.id AND s.item_id IN ("+placeholders(len(params.ItemIDs))+"))")
		args = append(args, stringArgs(params.ItemIDs)...)
	}
	if len(params.ProductLineIDs) > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM price_list_rule_product_line s WHERE s.price_list_rule_id = r.id AND s.product_line_id IN ("+placeholders(len(params.ProductLineIDs))+"))")
		args = append(args, stringArgs(params.ProductLineIDs)...)
	}
	tail, args := keysetTail("r", where, args, cursor, dir, params.Limit+1)
	rows, err := r.queries.DB().QueryContext(ctx, "SELECT "+priceListRuleColumns+" FROM price_list_rule r"+tail, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	raw, err := scanPriceListRules(rows)
	_ = rows.Close()
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rules, apiErr := r.hydrateRules(ctx, raw)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(rules, params.Limit, cursorDir, priceListRuleCreatedAt, priceListRuleID)
	return &domain.ListPriceListRulesResult{Rules: result, PageInfo: pageInfo}, nil
}

func (r *priceListRepoImpl) GetRulesByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.PriceListRule, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetPriceListRulesByIDs(ctx, sqlc.GetPriceListRulesByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return r.hydrateRules(ctx, rows)
}

func (r *priceListRepoImpl) GetRulesByListIDs(ctx context.Context, listIDs []string) ([]*domain.PriceListRule, *apierror.APIError) {
	if len(listIDs) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetPriceListRulesByListIDs(ctx, listIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return r.hydrateRules(ctx, rows)
}

func (r *priceListRepoImpl) GetRuleBySource(ctx context.Context, listID, sourceID string) (*domain.PriceListRule, *apierror.APIError) {
	row, err := r.queries.GetPriceListRuleBySource(ctx, sqlc.GetPriceListRuleBySourceParams{
		PriceListID: listID,
		SourceID:    gosql.NullString{String: sourceID, Valid: true},
	})
	if errors.Is(err, gosql.ErrNoRows) {
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	rules, apiErr := r.hydrateRules(ctx, []sqlc.PriceListRule{row})
	if apiErr != nil {
		return nil, apiErr
	}
	return rules[0], nil
}

func priceListUnitPrice(amount, currency, perUnitID gosql.NullString) *domain.UnitPrice {
	if !amount.Valid {
		return nil
	}
	c := constants.CurrencyUSD
	if currency.Valid && currency.String != "" {
		c = constants.Currency(currency.String)
	}
	return &domain.UnitPrice{Amount: amount.String, Currency: c, PerUnitID: perUnitID.String}
}

// hydrateRules reads the rules' tiers and scopes, keeping the rows' order.
func (r *priceListRepoImpl) hydrateRules(ctx context.Context, rows []sqlc.PriceListRule) ([]*domain.PriceListRule, *apierror.APIError) {
	if len(rows) == 0 {
		return []*domain.PriceListRule{}, nil
	}
	out := make([]*domain.PriceListRule, len(rows))
	byID := make(map[string]*domain.PriceListRule, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		rule := &domain.PriceListRule{
			ID:          row.ID,
			PriceListID: row.PriceListID,
			AccountID:   row.AccountID,
			Price: domain.PriceListPrice{
				Type:      constants.PriceListPriceType(row.PriceType),
				UnitPrice: priceListUnitPrice(row.UnitPriceAmount, row.UnitPriceCurrency, row.UnitPricePerUnitID),
				UnitID:    row.TierUnitID.String,
			},
			Scope:     domain.PriceListRuleScope{ItemIDs: []string{}, ProductLineIDs: []string{}, CategoryIDs: []string{}, AttributeIDs: []string{}},
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.PercentOfBase.Valid {
			v := trimDecimal(row.PercentOfBase.String)
			rule.Price.PercentOfBase = &v
		}
		if rule.Price.UnitPrice != nil {
			rule.Price.UnitPrice.Amount = trimDecimal(rule.Price.UnitPrice.Amount)
		}
		if row.SourceID.Valid {
			rule.SourceID = &row.SourceID.String
		}
		out[i], byID[row.ID], ids[i] = rule, rule, row.ID
	}

	tiers, err := r.queries.GetPriceListRuleTiers(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, t := range tiers {
		rule := byID[t.PriceListRuleID]
		tier := domain.PriceListTier{
			MinQuantity: trimDecimal(t.MinQuantity),
			UnitPrice:   priceListUnitPrice(t.UnitPriceAmount, t.UnitPriceCurrency, t.UnitPricePerUnitID),
		}
		if tier.UnitPrice != nil {
			tier.UnitPrice.Amount = trimDecimal(tier.UnitPrice.Amount)
		}
		if t.PercentOfBase.Valid {
			v := trimDecimal(t.PercentOfBase.String)
			tier.PercentOfBase = &v
		}
		rule.Price.Tiers = append(rule.Price.Tiers, tier)
	}

	items, err := r.queries.GetPriceListRuleItems(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, s := range items {
		byID[s.PriceListRuleID].Scope.ItemIDs = append(byID[s.PriceListRuleID].Scope.ItemIDs, s.ItemID)
	}
	lines, err := r.queries.GetPriceListRuleProductLines(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, s := range lines {
		byID[s.PriceListRuleID].Scope.ProductLineIDs = append(byID[s.PriceListRuleID].Scope.ProductLineIDs, s.ProductLineID)
	}
	cats, err := r.queries.GetPriceListRuleCategories(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, s := range cats {
		byID[s.PriceListRuleID].Scope.CategoryIDs = append(byID[s.PriceListRuleID].Scope.CategoryIDs, s.ItemCategoryID)
	}
	attrs, err := r.queries.GetPriceListRuleAttributes(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, s := range attrs {
		byID[s.PriceListRuleID].Scope.AttributeIDs = append(byID[s.PriceListRuleID].Scope.AttributeIDs, s.AttributeID)
	}
	return out, nil
}

func (r *priceListRepoImpl) GetRulesBySource(ctx context.Context, accountID, sourceID string) ([]*domain.PriceListRule, *apierror.APIError) {
	rows, err := r.queries.GetPriceListRulesBySource(ctx, sqlc.GetPriceListRulesBySourceParams{
		AccountID: accountID,
		SourceID:  gosql.NullString{String: sourceID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return r.hydrateRules(ctx, rows)
}

func unitPriceColumns(p *domain.UnitPrice) (gosql.NullString, gosql.NullString, gosql.NullString) {
	if p == nil {
		return gosql.NullString{}, gosql.NullString{}, gosql.NullString{}
	}
	return gosql.NullString{String: p.Amount, Valid: true},
		gosql.NullString{String: string(p.Currency), Valid: true},
		gosql.NullString{String: p.PerUnitID, Valid: p.PerUnitID != ""}
}

func rulePriceColumns(p domain.PriceListPrice) (amount, currency, perUnit, pct, tierUnit gosql.NullString) {
	amount, currency, perUnit = unitPriceColumns(p.UnitPrice)
	pct = nullString(p.PercentOfBase)
	tierUnit = gosql.NullString{String: p.UnitID, Valid: p.UnitID != ""}
	return
}

func (r *priceListRepoImpl) InsertRule(ctx context.Context, rule *domain.PriceListRule) *apierror.APIError {
	amount, currency, perUnit, pct, tierUnit := rulePriceColumns(rule.Price)
	if apiErr := db.MapSQLError(r.queries.InsertPriceListRule(ctx, sqlc.InsertPriceListRuleParams{
		ID:                 rule.ID,
		PriceListID:        rule.PriceListID,
		AccountID:          rule.AccountID,
		PriceType:          string(rule.Price.Type),
		UnitPriceAmount:    amount,
		UnitPriceCurrency:  currency,
		UnitPricePerUnitID: perUnit,
		PercentOfBase:      pct,
		TierUnitID:         tierUnit,
		SourceID:           nullString(rule.SourceID),
	})); apiErr != nil {
		return apiErr
	}
	return r.writeRuleChildren(ctx, rule)
}

func (r *priceListRepoImpl) UpdateRule(ctx context.Context, rule *domain.PriceListRule) *apierror.APIError {
	amount, currency, perUnit, pct, tierUnit := rulePriceColumns(rule.Price)
	if apiErr := db.MapSQLError(r.queries.UpdatePriceListRulePrice(ctx, sqlc.UpdatePriceListRulePriceParams{
		ID:                 rule.ID,
		PriceType:          string(rule.Price.Type),
		UnitPriceAmount:    amount,
		UnitPriceCurrency:  currency,
		UnitPricePerUnitID: perUnit,
		PercentOfBase:      pct,
		TierUnitID:         tierUnit,
	})); apiErr != nil {
		return apiErr
	}
	if apiErr := r.deleteRuleChildren(ctx, []string{rule.ID}); apiErr != nil {
		return apiErr
	}
	return r.writeRuleChildren(ctx, rule)
}

func (r *priceListRepoImpl) DeleteRule(ctx context.Context, id string) *apierror.APIError {
	if apiErr := r.deleteRuleChildren(ctx, []string{id}); apiErr != nil {
		return apiErr
	}
	return db.MapSQLError(r.queries.DeletePriceListRule(ctx, id))
}

func (r *priceListRepoImpl) writeRuleChildren(ctx context.Context, rule *domain.PriceListRule) *apierror.APIError {
	for i, t := range rule.Price.Tiers {
		amount, currency, perUnit := unitPriceColumns(t.UnitPrice)
		if apiErr := db.MapSQLError(r.queries.InsertPriceListRuleTier(ctx, sqlc.InsertPriceListRuleTierParams{
			PriceListRuleID:    rule.ID,
			Position:           int16(i), //nolint:gosec // tiers are capped at MaxPriceListRuleTiers
			MinQuantity:        t.MinQuantity,
			PercentOfBase:      nullString(t.PercentOfBase),
			UnitPriceAmount:    amount,
			UnitPriceCurrency:  currency,
			UnitPricePerUnitID: perUnit,
		})); apiErr != nil {
			return apiErr
		}
	}
	for _, id := range uniqueStrings(rule.Scope.ItemIDs) {
		if apiErr := db.MapSQLError(r.queries.InsertPriceListRuleItem(ctx, sqlc.InsertPriceListRuleItemParams{PriceListRuleID: rule.ID, ItemID: id})); apiErr != nil {
			return apiErr
		}
	}
	for _, id := range uniqueStrings(rule.Scope.ProductLineIDs) {
		if apiErr := db.MapSQLError(r.queries.InsertPriceListRuleProductLine(ctx, sqlc.InsertPriceListRuleProductLineParams{PriceListRuleID: rule.ID, ProductLineID: id})); apiErr != nil {
			return apiErr
		}
	}
	for _, id := range uniqueStrings(rule.Scope.CategoryIDs) {
		if apiErr := db.MapSQLError(r.queries.InsertPriceListRuleCategory(ctx, sqlc.InsertPriceListRuleCategoryParams{PriceListRuleID: rule.ID, ItemCategoryID: id})); apiErr != nil {
			return apiErr
		}
	}
	for _, id := range uniqueStrings(rule.Scope.AttributeIDs) {
		if apiErr := db.MapSQLError(r.queries.InsertPriceListRuleAttribute(ctx, sqlc.InsertPriceListRuleAttributeParams{PriceListRuleID: rule.ID, AttributeID: id})); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *priceListRepoImpl) deleteRuleChildren(ctx context.Context, ruleIDs []string) *apierror.APIError {
	if len(ruleIDs) == 0 {
		return nil
	}
	for _, del := range []func(context.Context, []string) error{
		r.queries.DeletePriceListRuleTiers,
		r.queries.DeletePriceListRuleItems,
		r.queries.DeletePriceListRuleProductLines,
		r.queries.DeletePriceListRuleCategories,
		r.queries.DeletePriceListRuleAttributes,
	} {
		if apiErr := db.MapSQLError(del(ctx, ruleIDs)); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok || v == "" {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// --- Assignments -------------------------------------------------------------

func (r *priceListRepoImpl) CustomerListIDs(ctx context.Context, relationIDs []string) (domain.PriceListAssignments, *apierror.APIError) {
	out := domain.PriceListAssignments{}
	if len(relationIDs) == 0 {
		return out, nil
	}
	rows, err := r.queries.GetPriceListCustomerAssignments(ctx, relationIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		out[row.AccountRelationID] = append(out[row.AccountRelationID], row.PriceListID)
	}
	return out, nil
}

func (r *priceListRepoImpl) CustomerListIDsByAccounts(ctx context.Context, ownerAccountID string, customerAccountIDs []string) (domain.PriceListAssignments, *apierror.APIError) {
	out := domain.PriceListAssignments{}
	if len(customerAccountIDs) == 0 {
		return out, nil
	}
	rows, err := r.queries.GetPriceListIDsByCustomerAccounts(ctx, sqlc.GetPriceListIDsByCustomerAccountsParams{
		OwnerAccountID:     ownerAccountID,
		CustomerAccountIds: customerAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		out[row.CustomerAccountID] = append(out[row.CustomerAccountID], row.PriceListID)
	}
	return out, nil
}

func (r *priceListRepoImpl) GroupListIDs(ctx context.Context, groupIDs []string) (domain.PriceListAssignments, *apierror.APIError) {
	out := domain.PriceListAssignments{}
	if len(groupIDs) == 0 {
		return out, nil
	}
	rows, err := r.queries.GetPriceListGroupAssignments(ctx, groupIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		out[row.AccountGroupID] = append(out[row.AccountGroupID], row.PriceListID)
	}
	return out, nil
}

func (r *priceListRepoImpl) SetCustomerLists(ctx context.Context, relationID string, listIDs []string) *apierror.APIError {
	if apiErr := db.MapSQLError(r.queries.DeletePriceListCustomersByRelations(ctx, []string{relationID})); apiErr != nil {
		return apiErr
	}
	for _, id := range uniqueStrings(listIDs) {
		if apiErr := r.AddCustomerList(ctx, relationID, id); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *priceListRepoImpl) SetGroupLists(ctx context.Context, groupID string, listIDs []string) *apierror.APIError {
	if apiErr := db.MapSQLError(r.queries.DeletePriceListAccountGroupsByGroup(ctx, groupID)); apiErr != nil {
		return apiErr
	}
	for _, id := range uniqueStrings(listIDs) {
		if apiErr := db.MapSQLError(r.queries.InsertPriceListAccountGroup(ctx, sqlc.InsertPriceListAccountGroupParams{AccountGroupID: groupID, PriceListID: id})); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *priceListRepoImpl) AddCustomerList(ctx context.Context, relationID, listID string) *apierror.APIError {
	return db.MapSQLError(r.queries.InsertPriceListCustomer(ctx, sqlc.InsertPriceListCustomerParams{AccountRelationID: relationID, PriceListID: listID}))
}

func (r *priceListRepoImpl) RemoveCustomerList(ctx context.Context, relationID, listID string) *apierror.APIError {
	return db.MapSQLError(r.queries.DeletePriceListCustomer(ctx, sqlc.DeletePriceListCustomerParams{AccountRelationID: relationID, PriceListID: listID}))
}

func (r *priceListRepoImpl) ListCustomerRelations(ctx context.Context, listID string) ([]string, *apierror.APIError) {
	rows, err := r.queries.ListPriceListCustomerRelations(ctx, listID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return rows, nil
}

func (r *priceListRepoImpl) DeleteCustomerAssignments(ctx context.Context, relationIDs []string) *apierror.APIError {
	if len(relationIDs) == 0 {
		return nil
	}
	return db.MapSQLError(r.queries.DeletePriceListCustomersByRelations(ctx, relationIDs))
}

func (r *priceListRepoImpl) DeleteGroupAssignments(ctx context.Context, groupID string) *apierror.APIError {
	return db.MapSQLError(r.queries.DeletePriceListAccountGroupsByGroup(ctx, groupID))
}

func (r *priceListRepoImpl) ApplicableLists(ctx context.Context, ownerAccountID string, customerAccountIDs []string) (map[string]map[string]domain.PriceListSource, *apierror.APIError) {
	out := make(map[string]map[string]domain.PriceListSource, len(customerAccountIDs))
	ids := uniqueStrings(customerAccountIDs)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.queries.GetApplicablePriceLists(ctx, sqlc.GetApplicablePriceListsParams{
		OwnerAccountID:     ownerAccountID,
		CustomerAccountIds: ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		lists := out[row.CustomerAccountID]
		if lists == nil {
			lists = map[string]domain.PriceListSource{}
			out[row.CustomerAccountID] = lists
		}
		source := domain.PriceListSource(row.SourceRank)
		if prev, ok := lists[row.PriceListID]; !ok || source < prev {
			lists[row.PriceListID] = source
		}
	}
	return out, nil
}

// scopeTargetSQL reads each scope kind's ID and whether it is active, for one account.
var scopeTargetSQL = map[domain.PriceListScopeKind]string{
	domain.PriceListScopeItem:        "SELECT id, is_active = 1 AND deleted_at IS NULL FROM item WHERE account_id = ? AND id IN (%s)",
	domain.PriceListScopeProductLine: "SELECT id, is_active = 1 FROM product_line WHERE (account_id = ? OR account_id IS NULL) AND id IN (%s)",
	domain.PriceListScopeCategory:    "SELECT id, TRUE FROM item_category WHERE (account_id = ? OR account_id IS NULL) AND id IN (%s)",
	domain.PriceListScopeAttribute:   "SELECT id, is_active = 1 FROM attribute WHERE account_id = ? AND id IN (%s)",
	domain.PriceListScopeUnit:        "SELECT id, TRUE FROM unit WHERE (account_id = ? OR account_id IS NULL) AND id IN (%s)",
}

func (r *priceListRepoImpl) ScopeTargets(ctx context.Context, kind domain.PriceListScopeKind, accountID string, ids []string) (map[string]bool, *apierror.APIError) {
	out := map[string]bool{}
	ids = uniqueStrings(ids)
	if len(ids) == 0 {
		return out, nil
	}
	query, ok := scopeTargetSQL[kind]
	if !ok {
		return nil, apierror.NewInvariantViolationError("Unknown price list scope kind.")
	}
	args := append([]any{accountID}, stringArgs(ids)...)
	rows, err := r.queries.DB().QueryContext(ctx, fmt.Sprintf(query, placeholders(len(ids))), args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var active bool
		if err := rows.Scan(&id, &active); err != nil {
			return nil, db.MapSQLError(err)
		}
		out[id] = active
	}
	return out, db.MapSQLError(rows.Err())
}

func (r *priceListRepoImpl) IsLegacySynced(ctx context.Context, accountID string) (bool, *apierror.APIError) {
	synced, err := r.queries.IsPriceListLegacySynced(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, apiErr
	}
	return synced, nil
}

func (r *priceListRepoImpl) MarkLegacySynced(ctx context.Context, accountID string) *apierror.APIError {
	return db.MapSQLError(r.queries.MarkPriceListLegacySynced(ctx, accountID))
}
