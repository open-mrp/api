package service

import (
	"context"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	apierror "github.com/open-mrp/apikit/apierror"
)

// The legacy pricing records (account prices, volume discounts, price groups) are mirrored into price lists while
// 1.0.forge-preview.9 callers still write them. Each mirrored list and rule carries the legacy record it comes from,
// so a write to one record re-mirrors just that record and keeps every ID stable. Until an account's records are first
// mirrored (price_list_legacy_sync), its orders are priced from the legacy records.

// accountListName names the list mirroring the account's volume discounts that name no customer group.
const accountListName = "Volume discounts"

// ensurePriceListsSynced mirrors the account's legacy pricing once, before anything reads or writes its price lists.
func ensurePriceListsSynced(ctx context.Context, repos domain.RepoFactory, accountID string) *apierror.APIError {
	repo := repos.NewPriceListRepo()
	synced, apiErr := repo.IsLegacySynced(ctx, accountID)
	if apiErr != nil || synced {
		return apiErr
	}
	if apiErr := syncLegacyPricing(ctx, repos, accountID); apiErr != nil {
		return apiErr
	}
	return repo.MarkLegacySynced(ctx, accountID)
}

// syncLegacyPricing mirrors every legacy pricing record of the account and removes mirrors whose record is gone.
func syncLegacyPricing(ctx context.Context, repos domain.RepoFactory, accountID string) *apierror.APIError {
	legacy := repos.NewLegacyPricingRepo()
	groups, apiErr := legacy.PricingGroups(ctx, accountID, nil)
	if apiErr != nil {
		return apiErr
	}
	groupIDs := make([]string, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
		if _, apiErr := mirrorGroup(ctx, repos, accountID, g); apiErr != nil {
			return apiErr
		}
	}
	if apiErr := dropMirroredGroupsExcept(ctx, repos, accountID, groupIDs); apiErr != nil {
		return apiErr
	}
	discounts, apiErr := legacy.VolumeDiscounts(ctx, accountID, nil)
	if apiErr != nil {
		return apiErr
	}
	for _, vd := range discounts {
		if apiErr := mirrorVolumeDiscount(ctx, repos, accountID, vd.ID, vd); apiErr != nil {
			return apiErr
		}
	}
	prices, apiErr := legacy.AccountPrices(ctx, accountID, nil)
	if apiErr != nil {
		return apiErr
	}
	for _, ap := range prices {
		if apiErr := mirrorAccountPrice(ctx, repos, accountID, ap.ID, ap); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// syncLegacyAccountPrice re-mirrors one account price after a legacy write (or its delete).
func syncLegacyAccountPrice(ctx context.Context, repos domain.RepoFactory, accountID, accountPriceID string) *apierror.APIError {
	if done, apiErr := firstSync(ctx, repos, accountID); done || apiErr != nil {
		return apiErr
	}
	prices, apiErr := repos.NewLegacyPricingRepo().AccountPrices(ctx, accountID, []string{accountPriceID})
	if apiErr != nil {
		return apiErr
	}
	var ap *domain.LegacyAccountPrice
	if len(prices) > 0 {
		ap = prices[0]
	}
	return mirrorAccountPrice(ctx, repos, accountID, accountPriceID, ap)
}

// syncLegacyVolumeDiscount re-mirrors one volume discount after a legacy write (or its delete).
func syncLegacyVolumeDiscount(ctx context.Context, repos domain.RepoFactory, accountID, discountID string) *apierror.APIError {
	if done, apiErr := firstSync(ctx, repos, accountID); done || apiErr != nil {
		return apiErr
	}
	discounts, apiErr := repos.NewLegacyPricingRepo().VolumeDiscounts(ctx, accountID, []string{discountID})
	if apiErr != nil {
		return apiErr
	}
	var vd *domain.LegacyVolumeDiscount
	if len(discounts) > 0 {
		vd = discounts[0]
	}
	if apiErr := mirrorVolumeDiscount(ctx, repos, accountID, discountID, vd); apiErr != nil {
		return apiErr
	}
	// A group the discount no longer names may carry no pricing now.
	groups, apiErr := repos.NewLegacyPricingRepo().PricingGroups(ctx, accountID, nil)
	if apiErr != nil {
		return apiErr
	}
	keepIDs := make([]string, len(groups))
	for i, g := range groups {
		keepIDs[i] = g.ID
	}
	return dropMirroredGroupsExcept(ctx, repos, accountID, keepIDs)
}

// syncLegacyGroup re-mirrors one account group's list after the group, its discounts or its price-group members
// changed. A group that no longer carries pricing loses its mirrored list.
func syncLegacyGroup(ctx context.Context, repos domain.RepoFactory, accountID, groupID string) *apierror.APIError {
	if done, apiErr := firstSync(ctx, repos, accountID); done || apiErr != nil {
		return apiErr
	}
	groups, apiErr := repos.NewLegacyPricingRepo().PricingGroups(ctx, accountID, []string{groupID})
	if apiErr != nil {
		return apiErr
	}
	if len(groups) == 0 {
		return dropMirroredGroup(ctx, repos, accountID, groupID)
	}
	_, apiErr = mirrorGroup(ctx, repos, accountID, groups[0])
	return apiErr
}

// syncLegacyCustomerPriceGroups assigns a customer the mirrored lists of exactly the price groups it lists.
func syncLegacyCustomerPriceGroups(ctx context.Context, repos domain.RepoFactory, accountID, relationID string) *apierror.APIError {
	if done, apiErr := firstSync(ctx, repos, accountID); done || apiErr != nil {
		return apiErr
	}
	groupIDs, apiErr := repos.NewLegacyPricingRepo().RelationPriceGroupIDs(ctx, relationID)
	if apiErr != nil {
		return apiErr
	}
	repo := repos.NewPriceListRepo()
	lists, apiErr := repo.ListBySourceType(ctx, accountID, constants.PriceListSourceAccountGroup)
	if apiErr != nil {
		return apiErr
	}
	for _, list := range lists {
		if slices.Contains(groupIDs, *list.SourceID) {
			apiErr = repo.AddCustomerList(ctx, relationID, list.ID)
		} else {
			apiErr = repo.RemoveCustomerList(ctx, relationID, list.ID)
		}
		if apiErr != nil {
			return apiErr
		}
	}
	// A price group added since the group was last mirrored may have no list yet.
	for _, gid := range groupIDs {
		if apiErr := syncLegacyGroup(ctx, repos, accountID, gid); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// firstSync runs the account's first full mirror when it has none yet; done reports that it ran, which covers any
// single record too.
func firstSync(ctx context.Context, repos domain.RepoFactory, accountID string) (bool, *apierror.APIError) {
	synced, apiErr := repos.NewPriceListRepo().IsLegacySynced(ctx, accountID)
	if apiErr != nil || synced {
		return false, apiErr
	}
	return true, ensurePriceListsSynced(ctx, repos, accountID)
}

// mirroredList returns the list mirroring a legacy source, creating it when missing and renaming it when the source
// was renamed.
func mirroredList(ctx context.Context, repos domain.RepoFactory, accountID string, sourceType constants.PriceListSourceType,
	sourceID, name string, description *string) (*domain.PriceList, *apierror.APIError) {
	repo := repos.NewPriceListRepo()
	list, apiErr := repo.GetBySource(ctx, accountID, sourceType, sourceID)
	if apiErr != nil {
		return nil, apiErr
	}
	if list == nil {
		listID, apiErr := id.GenID(id.PriceListIDPrefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		st, sid := sourceType, sourceID
		list = &domain.PriceList{
			ID: listID, AccountID: accountID, Name: name, Description: description,
			Status: constants.ArchiveStatusActive, Currency: constants.CurrencyUSD, SourceType: &st, SourceID: &sid,
		}
		return list, repo.Insert(ctx, list)
	}
	if list.Name != name || !equalStringPtr(list.Description, description) {
		list.Name, list.Description = name, description
		if apiErr := repo.Update(ctx, list); apiErr != nil {
			return nil, apiErr
		}
	}
	return list, nil
}

// mirrorGroup mirrors an account group's list: assigned to the group and to every customer listing it as a price group.
func mirrorGroup(ctx context.Context, repos domain.RepoFactory, accountID string, g *domain.LegacyPricingGroup) (*domain.PriceList, *apierror.APIError) {
	list, apiErr := mirroredList(ctx, repos, accountID, constants.PriceListSourceAccountGroup, g.ID, g.Name, g.Description)
	if apiErr != nil {
		return nil, apiErr
	}
	repo := repos.NewPriceListRepo()
	assigned, apiErr := repo.GroupListIDs(ctx, []string{g.ID})
	if apiErr != nil {
		return nil, apiErr
	}
	if !slices.Contains(assigned[g.ID], list.ID) {
		if apiErr := repo.SetGroupLists(ctx, g.ID, append(assigned[g.ID], list.ID)); apiErr != nil {
			return nil, apiErr
		}
	}
	current, apiErr := repo.ListCustomerRelations(ctx, list.ID)
	if apiErr != nil {
		return nil, apiErr
	}
	for _, rel := range g.MemberRelationIDs {
		if !slices.Contains(current, rel) {
			if apiErr := repo.AddCustomerList(ctx, rel, list.ID); apiErr != nil {
				return nil, apiErr
			}
		}
	}
	for _, rel := range current {
		if !slices.Contains(g.MemberRelationIDs, rel) {
			if apiErr := repo.RemoveCustomerList(ctx, rel, list.ID); apiErr != nil {
				return nil, apiErr
			}
		}
	}
	return list, nil
}

func dropMirroredGroup(ctx context.Context, repos domain.RepoFactory, accountID, groupID string) *apierror.APIError {
	repo := repos.NewPriceListRepo()
	list, apiErr := repo.GetBySource(ctx, accountID, constants.PriceListSourceAccountGroup, groupID)
	if apiErr != nil || list == nil {
		return apiErr
	}
	return repo.Delete(ctx, accountID, list.ID)
}

func dropMirroredGroupsExcept(ctx context.Context, repos domain.RepoFactory, accountID string, keepIDs []string) *apierror.APIError {
	lists, apiErr := repos.NewPriceListRepo().ListBySourceType(ctx, accountID, constants.PriceListSourceAccountGroup)
	if apiErr != nil {
		return apiErr
	}
	for _, list := range lists {
		if !slices.Contains(keepIDs, *list.SourceID) {
			if apiErr := repos.NewPriceListRepo().Delete(ctx, accountID, list.ID); apiErr != nil {
				return apiErr
			}
		}
	}
	return nil
}

// mirrorAccountPrice keeps one fixed rule for an account price in its customer's own list, or removes it when the
// price is gone (ap nil).
func mirrorAccountPrice(ctx context.Context, repos domain.RepoFactory, accountID, sourceID string, ap *domain.LegacyAccountPrice) *apierror.APIError {
	var targets map[string]domain.PriceListPrice
	var scope domain.PriceListRuleScope
	if ap != nil {
		list, apiErr := mirroredList(ctx, repos, accountID, constants.PriceListSourceCustomer, ap.RecipientRelationID, ap.RecipientName, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repos.NewPriceListRepo().AddCustomerList(ctx, ap.RecipientRelationID, list.ID); apiErr != nil {
			return apiErr
		}
		targets = map[string]domain.PriceListPrice{list.ID: {
			Type:      constants.PriceListPriceTypeFixed,
			UnitPrice: &domain.UnitPrice{Amount: roundSix(ap.Value), Currency: constants.CurrencyUSD, PerUnitID: ap.DenominatorUnitID},
		}}
		scope = domain.PriceListRuleScope{
			ProductLineIDs: []string{ap.ProductLineID},
			CategoryIDs:    nonNil(ap.CategoryIDs),
			AttributeIDs:   nonNil(ap.AttributeIDs),
			ItemIDs:        []string{},
		}
	}
	return mirrorRules(ctx, repos, accountID, sourceID, scope, targets)
}

// mirrorVolumeDiscount keeps one rule for a volume discount in the list of each customer group it names (or the
// account's list when it names none), or removes them when the discount is gone (vd nil).
func mirrorVolumeDiscount(ctx context.Context, repos domain.RepoFactory, accountID, sourceID string, vd *domain.LegacyVolumeDiscount) *apierror.APIError {
	targets := map[string]domain.PriceListPrice{}
	var scope domain.PriceListRuleScope
	if vd != nil {
		price := volumeDiscountPrice(vd)
		if price != nil {
			var lists []*domain.PriceList
			if len(vd.GroupIDs) == 0 {
				list, apiErr := mirroredList(ctx, repos, accountID, constants.PriceListSourceAccount, accountID, accountListName, nil)
				if apiErr != nil {
					return apiErr
				}
				lists = append(lists, list)
			}
			if len(vd.GroupIDs) > 0 {
				groups, apiErr := repos.NewLegacyPricingRepo().PricingGroups(ctx, accountID, vd.GroupIDs)
				if apiErr != nil {
					return apiErr
				}
				for _, g := range groups {
					list, apiErr := mirrorGroup(ctx, repos, accountID, g)
					if apiErr != nil {
						return apiErr
					}
					lists = append(lists, list)
				}
			}
			for _, list := range lists {
				targets[list.ID] = *price
			}
		}
		scope = domain.PriceListRuleScope{
			ItemIDs:        []string{},
			ProductLineIDs: nonNil(vd.ProductLineIDs),
			CategoryIDs:    nonNil(vd.CategoryIDs),
			AttributeIDs:   nonNil(vd.AttributeIDs),
		}
	}
	return mirrorRules(ctx, repos, accountID, sourceID, scope, targets)
}

// mirrorRules makes the rules mirrored from sourceID exactly one per target list with the given scope and price.
func mirrorRules(ctx context.Context, repos domain.RepoFactory, accountID, sourceID string, scope domain.PriceListRuleScope,
	targets map[string]domain.PriceListPrice) *apierror.APIError {
	repo := repos.NewPriceListRepo()
	existing, apiErr := repo.GetRulesBySource(ctx, accountID, sourceID)
	if apiErr != nil {
		return apiErr
	}
	for _, rule := range existing {
		price, ok := targets[rule.PriceListID]
		if !ok {
			if apiErr := repo.DeleteRule(ctx, rule.ID); apiErr != nil {
				return apiErr
			}
			continue
		}
		delete(targets, rule.PriceListID)
		rule.Scope, rule.Price = scope, price
		if apiErr := repo.UpdateRule(ctx, rule); apiErr != nil {
			return apiErr
		}
	}
	listIDs := make([]string, 0, len(targets))
	for listID := range targets {
		listIDs = append(listIDs, listID)
	}
	slices.Sort(listIDs)
	for _, listID := range listIDs {
		ruleID, apiErr := id.GenID(id.PriceListRuleIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		sid := sourceID
		if apiErr := repo.InsertRule(ctx, &domain.PriceListRule{
			ID: ruleID, PriceListID: listID, AccountID: accountID, Scope: scope, Price: targets[listID], SourceID: &sid,
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// volumeDiscountPrice restates a volume discount as a rule price. Each legacy tier reached multiplies the price by
// (1 - its fraction), so a tier's percent_of_base is 100 times the product over it and every lower tier. Without a
// counting unit only tiers from zero are ever reached; a discount that reaches nothing has no rule.
func volumeDiscountPrice(vd *domain.LegacyVolumeDiscount) *domain.PriceListPrice {
	tiers := slices.Clone(vd.Tiers)
	slices.SortStableFunc(tiers, func(a, b domain.LegacyVolumeTier) int {
		return parseDecimal(a.Threshold).Cmp(parseDecimal(b.Threshold))
	})
	type step struct {
		min     decimal.Decimal
		percent decimal.Decimal
	}
	var steps []step
	multiplier := decimal.NewFromInt(1)
	for _, t := range tiers {
		multiplier = multiplier.Mul(decimal.NewFromInt(1).Sub(parseDecimal(t.Percentage)))
		minQty := parseDecimal(t.Threshold)
		if minQty.IsNegative() {
			minQty = decimal.Zero
		}
		pct := multiplier.Mul(hundred)
		if n := len(steps); n > 0 && steps[n-1].min.Equal(minQty) {
			steps[n-1].percent = pct
			continue
		}
		steps = append(steps, step{min: minQty, percent: pct})
	}
	if len(vd.UnitIDs) == 0 {
		// Nothing is ever counted, so only a tier from zero applies.
		if len(steps) == 0 || !steps[0].min.IsZero() {
			return nil
		}
		steps = steps[:1]
	}
	if len(steps) == 0 {
		return nil
	}
	if len(steps) == 1 && steps[0].min.IsZero() {
		pct := roundSix(steps[0].percent.String())
		return &domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: &pct}
	}
	price := &domain.PriceListPrice{Type: constants.PriceListPriceTypeTiered, UnitID: vd.UnitIDs[0]}
	for _, s := range steps {
		pct := roundSix(s.percent.String())
		price.Tiers = append(price.Tiers, domain.PriceListTier{MinQuantity: roundSix(s.min.String()), PercentOfBase: &pct})
	}
	return price
}

// roundSix rounds a decimal string to the six places prices and percents are stored at, without padding.
func roundSix(s string) string {
	return parseDecimal(s).Round(6).String()
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
