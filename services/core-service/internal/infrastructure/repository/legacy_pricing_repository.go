package repository

import (
	"context"
	gosql "database/sql"
	"slices"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
)

type legacyPricingRepoImpl struct {
	queries *sqlc.Queries
}

// NewLegacyPricingRepo reads account prices, volume discounts and price groups for the mirror into price lists.
func NewLegacyPricingRepo(queries *sqlc.Queries) domain.LegacyPricingRepo {
	return &legacyPricingRepoImpl{queries: queries}
}

func keep(ids []string) func(string) bool {
	if ids == nil {
		return func(string) bool { return true }
	}
	return func(id string) bool { return slices.Contains(ids, id) }
}

func (r *legacyPricingRepoImpl) AccountPrices(ctx context.Context, ownerAccountID string, ids []string) ([]*domain.LegacyAccountPrice, *apierror.APIError) {
	rows, err := r.queries.GetLegacyAccountPrices(ctx, ownerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	wanted := keep(ids)
	byID := map[string]*domain.LegacyAccountPrice{}
	var out []*domain.LegacyAccountPrice
	var priceIDs []string
	for _, row := range rows {
		if !wanted(row.ID) {
			continue
		}
		ap := &domain.LegacyAccountPrice{
			ID:                  row.ID,
			RecipientRelationID: row.RelationID,
			RecipientAccountID:  row.RecipientAccountID,
			RecipientName:       row.RecipientName,
			ProductLineID:       row.ProductLineID,
			Value:               row.Value,
			DenominatorUnitID:   row.DenominatorUnitID,
			CreatedAt:           row.CreatedAt,
		}
		out = append(out, ap)
		byID[ap.ID] = ap
		priceIDs = append(priceIDs, ap.ID)
	}
	if len(priceIDs) == 0 {
		return out, nil
	}
	cats, err := r.queries.GetLegacyAccountPriceCategoryIDs(ctx, priceIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, c := range cats {
		byID[c.AccountPriceID].CategoryIDs = append(byID[c.AccountPriceID].CategoryIDs, c.ItemCategoryID)
	}
	attrs, err := r.queries.GetLegacyAccountPriceAttributeIDs(ctx, priceIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, a := range attrs {
		byID[a.AccountPriceID].AttributeIDs = append(byID[a.AccountPriceID].AttributeIDs, a.AttributeID)
	}
	return out, nil
}

func (r *legacyPricingRepoImpl) VolumeDiscounts(ctx context.Context, ownerAccountID string, ids []string) ([]*domain.LegacyVolumeDiscount, *apierror.APIError) {
	rows, err := r.queries.GetLegacyVolumeDiscounts(ctx, ownerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	wanted := keep(ids)
	byID := map[string]*domain.LegacyVolumeDiscount{}
	var out []*domain.LegacyVolumeDiscount
	var vdIDs []string
	var nullIDs []gosql.NullString
	for _, row := range rows {
		if !wanted(row.ID) {
			continue
		}
		vd := &domain.LegacyVolumeDiscount{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt}
		out = append(out, vd)
		byID[vd.ID] = vd
		vdIDs = append(vdIDs, vd.ID)
		nullIDs = append(nullIDs, gosql.NullString{String: vd.ID, Valid: true})
	}
	if len(vdIDs) == 0 {
		return out, nil
	}
	groups, err := r.queries.GetLegacyVolumeDiscountGroupIDs(ctx, vdIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, g := range groups {
		byID[g.QuantityDiscountID].GroupIDs = append(byID[g.QuantityDiscountID].GroupIDs, g.AccountGroupID)
	}
	tiers, err := r.queries.GetPricingVolumeDiscountTiers(ctx, nullIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, t := range tiers {
		if vd := byID[t.QuantityDiscountID.String]; vd != nil {
			vd.Tiers = append(vd.Tiers, domain.LegacyVolumeTier{Threshold: t.Threshold, Percentage: t.DiscountPercentage})
		}
	}
	units, err := r.queries.GetPricingVolumeDiscountUnits(ctx, vdIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, u := range units {
		byID[u.QuantityDiscountID].UnitIDs = append(byID[u.QuantityDiscountID].UnitIDs, u.UnitID)
	}
	lines, err := r.queries.GetPricingVolumeDiscountProductLineIDs(ctx, vdIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, l := range lines {
		byID[l.QuantityDiscountID].ProductLineIDs = append(byID[l.QuantityDiscountID].ProductLineIDs, l.ProductLineID)
	}
	cats, err := r.queries.GetPricingVolumeDiscountCategoryIDs(ctx, vdIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, c := range cats {
		byID[c.QuantityDiscountID].CategoryIDs = append(byID[c.QuantityDiscountID].CategoryIDs, c.ItemCategoryID)
	}
	attrs, err := r.queries.GetPricingVolumeDiscountAttributeIDs(ctx, vdIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, a := range attrs {
		byID[a.QuantityDiscountID].AttributeIDs = append(byID[a.QuantityDiscountID].AttributeIDs, a.AttributeID)
	}
	for _, vd := range out {
		// The first acceptable unit counts a tiered rule; keep the choice stable.
		slices.Sort(vd.UnitIDs)
	}
	return out, nil
}

func (r *legacyPricingRepoImpl) PricingGroups(ctx context.Context, ownerAccountID string, ids []string) ([]*domain.LegacyPricingGroup, *apierror.APIError) {
	rows, err := r.queries.GetLegacyPricingGroups(ctx, ownerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	wanted := keep(ids)
	byID := map[string]*domain.LegacyPricingGroup{}
	var out []*domain.LegacyPricingGroup
	var groupIDs []string
	for _, row := range rows {
		if !wanted(row.ID) {
			continue
		}
		g := &domain.LegacyPricingGroup{ID: row.ID, Name: row.Name}
		if row.Description.Valid {
			g.Description = &row.Description.String
		}
		out = append(out, g)
		byID[g.ID] = g
		groupIDs = append(groupIDs, g.ID)
	}
	if len(groupIDs) == 0 {
		return out, nil
	}
	members, err := r.queries.GetLegacyPriceGroupMembers(ctx, groupIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, m := range members {
		byID[m.AccountGroupID].MemberRelationIDs = append(byID[m.AccountGroupID].MemberRelationIDs, m.AccountRelationID)
	}
	return out, nil
}

func (r *legacyPricingRepoImpl) RelationPriceGroupIDs(ctx context.Context, relationID string) ([]string, *apierror.APIError) {
	rows, err := r.queries.GetLegacyRelationPriceGroupIDs(ctx, relationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return rows, nil
}

func (r *legacyPricingRepoImpl) AccountIDsAfter(ctx context.Context, cursor string, limit int32) ([]string, *apierror.APIError) {
	rows, err := r.queries.ListAccountIDsAfter(ctx, sqlc.ListAccountIDsAfterParams{Cursor: cursor, Limit: limit})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return rows, nil
}
