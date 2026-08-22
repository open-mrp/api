package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var volumeDiscountRepoTracer = tracing.GetTracer("core-service.volume_discount_repository")

type volumeDiscountRepoImpl struct {
	queries *sqlc.Queries
}

func NewVolumeDiscountRepo(queries *sqlc.Queries) domain.VolumeDiscountRepo {
	return &volumeDiscountRepoImpl{queries: queries}
}

func volumeDiscountCreatedAt(d *domain.VolumeDiscount) time.Time { return d.CreatedAt }
func volumeDiscountID(d *domain.VolumeDiscount) string           { return d.ID }

func buildVolumeDiscountSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func toNullStringForVD(s *string) gosql.NullString {
	if s == nil || *s == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func mapVolumeDiscountBase(id, name, accountID string, createdAt, updatedAt time.Time) *domain.VolumeDiscount {
	return &domain.VolumeDiscount{
		ID:        id,
		Name:      name,
		AccountID: accountID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func (r *volumeDiscountRepoImpl) enrichSingle(ctx context.Context, d *domain.VolumeDiscount, incs []string) *apierror.APIError {
	discountID := d.ID

	// Tiers are always fetched (not expandable).
	tierRows, err := r.queries.GetVolumeDiscountTiers(ctx, gosql.NullString{String: discountID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	d.Tiers = make([]*domain.VolumeDiscountTier, len(tierRows))
	for i, t := range tierRows {
		var parentTierID *string
		if t.ParentTierID.Valid {
			parentTierID = &t.ParentTierID.String
		}
		d.Tiers[i] = &domain.VolumeDiscountTier{
			ID:                 t.ID,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierID:       parentTierID,
			CreatedAt:          t.CreatedAt,
			UpdatedAt:          t.UpdatedAt,
		}
	}

	if slices.Contains(incs, "customer_groups") {
		cgRows, err := r.queries.GetVolumeDiscountCustomerGroups(ctx, discountID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		d.CustomerGroups = make([]*domain.VolumeDiscountCustomerGroup, len(cgRows))
		for i, cg := range cgRows {
			d.CustomerGroups[i] = &domain.VolumeDiscountCustomerGroup{
				ID:               cg.ID,
				AccountGroupID:   cg.AccountGroupID,
				Name:             cg.Name,
				CommissionPolicy: cg.CommissionStatusCode,
				FreightPolicy:    cg.FreightStatusCode,
				Type:             cg.AccountGroupTypeCode,
				CreatedAt:        cg.CreatedAt,
				UpdatedAt:        cg.UpdatedAt,
			}
		}
	}

	if slices.Contains(incs, "product_lines") {
		plRows, err := r.queries.GetVolumeDiscountProductLines(ctx, discountID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		d.ProductLines = make([]*domain.VolumeDiscountProductLine, len(plRows))
		for i, pl := range plRows {
			d.ProductLines[i] = &domain.VolumeDiscountProductLine{
				ID:                 pl.ID,
				Name:               pl.Name,
				IsCommissionExempt: pl.IsCommissionExempt,
				IsFreightExempt:    pl.IsFreightExempt,
				CreatedAt:          pl.CreatedAt,
				UpdatedAt:          pl.UpdatedAt,
			}
		}
	}

	if slices.Contains(incs, "categories") {
		catRows, err := r.queries.GetVolumeDiscountCategories(ctx, discountID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		d.Categories = make([]*domain.VolumeDiscountCategory, len(catRows))
		for i, cat := range catRows {
			d.Categories[i] = &domain.VolumeDiscountCategory{
				ID:        cat.ID,
				Name:      cat.Name,
				Type:      cat.ItemCategoryTypeCode,
				CreatedAt: cat.CreatedAt,
				UpdatedAt: cat.UpdatedAt,
			}
		}
	}

	if slices.Contains(incs, "attributes") {
		attrRows, err := r.queries.GetVolumeDiscountAttributes(ctx, discountID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		d.Attributes = make([]*domain.VolumeDiscountAttribute, len(attrRows))
		for i, attr := range attrRows {
			d.Attributes[i] = &domain.VolumeDiscountAttribute{
				ID:        attr.ID,
				Name:      attr.Name,
				ColorCode: attr.ColorCode,
				CreatedAt: attr.CreatedAt,
				UpdatedAt: attr.UpdatedAt,
			}
		}
	}

	if slices.Contains(incs, "acceptable_units") {
		unitRows, err := r.queries.GetVolumeDiscountUnits(ctx, discountID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		d.AcceptableUnits = make([]*domain.VolumeDiscountUnit, len(unitRows))
		for i, u := range unitRows {
			d.AcceptableUnits[i] = &domain.VolumeDiscountUnit{
				ID:                u.ID,
				Name:              u.Name,
				Abbreviation:      u.Abbreviation,
				Type:              u.Type,
				RatioNumerator:    u.RatioNumerator,
				RatioDenominator:  u.RatioDenominator,
				OffsetNumerator:   u.OffsetNumerator,
				OffsetDenominator: u.OffsetDenominator,
				CreatedAt:         u.CreatedAt,
				UpdatedAt:         u.UpdatedAt,
			}
		}
	}

	return nil
}

func (r *volumeDiscountRepoImpl) enrichBatch(ctx context.Context, discounts []*domain.VolumeDiscount, incs []string) *apierror.APIError {
	if len(discounts) == 0 {
		return nil
	}

	ids := make([]string, len(discounts))
	nullIDs := make([]gosql.NullString, len(discounts))
	for i, d := range discounts {
		ids[i] = d.ID
		nullIDs[i] = gosql.NullString{String: d.ID, Valid: true}
	}

	// Tiers
	tierRows, err := r.queries.GetVolumeDiscountTiersByDiscountIDs(ctx, nullIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	tierMap := make(map[string][]*domain.VolumeDiscountTier)
	for _, t := range tierRows {
		var parentTierID *string
		if t.ParentTierID.Valid {
			parentTierID = &t.ParentTierID.String
		}
		discountID := ""
		if t.QuantityDiscountID.Valid {
			discountID = t.QuantityDiscountID.String
		}
		tierMap[discountID] = append(tierMap[discountID], &domain.VolumeDiscountTier{
			ID:                 t.ID,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierID:       parentTierID,
			CreatedAt:          t.CreatedAt,
			UpdatedAt:          t.UpdatedAt,
		})
	}

	cgMap := make(map[string][]*domain.VolumeDiscountCustomerGroup)
	if slices.Contains(incs, "customer_groups") {
		cgRows, err := r.queries.GetVolumeDiscountCustomerGroupsByDiscountIDs(ctx, ids)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, cg := range cgRows {
			cgMap[cg.QuantityDiscountID] = append(cgMap[cg.QuantityDiscountID], &domain.VolumeDiscountCustomerGroup{
				ID:               cg.ID,
				AccountGroupID:   cg.AccountGroupID,
				Name:             cg.Name,
				CommissionPolicy: cg.CommissionStatusCode,
				FreightPolicy:    cg.FreightStatusCode,
				Type:             cg.AccountGroupTypeCode,
				CreatedAt:        cg.CreatedAt,
				UpdatedAt:        cg.UpdatedAt,
			})
		}
	}

	plMap := make(map[string][]*domain.VolumeDiscountProductLine)
	if slices.Contains(incs, "product_lines") {
		plRows, err := r.queries.GetVolumeDiscountProductLinesByDiscountIDs(ctx, ids)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, pl := range plRows {
			plMap[pl.QuantityDiscountID] = append(plMap[pl.QuantityDiscountID], &domain.VolumeDiscountProductLine{
				ID:                 pl.ID,
				Name:               pl.Name,
				IsCommissionExempt: pl.IsCommissionExempt,
				IsFreightExempt:    pl.IsFreightExempt,
				CreatedAt:          pl.CreatedAt,
				UpdatedAt:          pl.UpdatedAt,
			})
		}
	}

	catMap := make(map[string][]*domain.VolumeDiscountCategory)
	if slices.Contains(incs, "categories") {
		catRows, err := r.queries.GetVolumeDiscountCategoriesByDiscountIDs(ctx, ids)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, cat := range catRows {
			catMap[cat.QuantityDiscountID] = append(catMap[cat.QuantityDiscountID], &domain.VolumeDiscountCategory{
				ID:        cat.ID,
				Name:      cat.Name,
				Type:      cat.ItemCategoryTypeCode,
				CreatedAt: cat.CreatedAt,
				UpdatedAt: cat.UpdatedAt,
			})
		}
	}

	attrMap := make(map[string][]*domain.VolumeDiscountAttribute)
	if slices.Contains(incs, "attributes") {
		attrRows, err := r.queries.GetVolumeDiscountAttributesByDiscountIDs(ctx, ids)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, attr := range attrRows {
			attrMap[attr.QuantityDiscountID] = append(attrMap[attr.QuantityDiscountID], &domain.VolumeDiscountAttribute{
				ID:        attr.ID,
				Name:      attr.Name,
				ColorCode: attr.ColorCode,
				CreatedAt: attr.CreatedAt,
				UpdatedAt: attr.UpdatedAt,
			})
		}
	}

	unitMap := make(map[string][]*domain.VolumeDiscountUnit)
	if slices.Contains(incs, "acceptable_units") {
		unitRows, err := r.queries.GetVolumeDiscountUnitsByDiscountIDs(ctx, ids)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, u := range unitRows {
			unitMap[u.QuantityDiscountID] = append(unitMap[u.QuantityDiscountID], &domain.VolumeDiscountUnit{
				ID:                u.ID,
				Name:              u.Name,
				Abbreviation:      u.Abbreviation,
				Type:              u.Type,
				RatioNumerator:    u.RatioNumerator,
				RatioDenominator:  u.RatioDenominator,
				OffsetNumerator:   u.OffsetNumerator,
				OffsetDenominator: u.OffsetDenominator,
				CreatedAt:         u.CreatedAt,
				UpdatedAt:         u.UpdatedAt,
			})
		}
	}

	for _, d := range discounts {
		d.Tiers = tierMap[d.ID]
		if d.Tiers == nil {
			d.Tiers = []*domain.VolumeDiscountTier{}
		}
		if slices.Contains(incs, "customer_groups") {
			d.CustomerGroups = cgMap[d.ID]
		}
		if slices.Contains(incs, "product_lines") {
			d.ProductLines = plMap[d.ID]
		}
		if slices.Contains(incs, "categories") {
			d.Categories = catMap[d.ID]
		}
		if slices.Contains(incs, "attributes") {
			d.Attributes = attrMap[d.ID]
		}
		if slices.Contains(incs, "acceptable_units") {
			d.AcceptableUnits = unitMap[d.ID]
		}
	}

	return nil
}

func (r *volumeDiscountRepoImpl) List(ctx context.Context, params domain.ListVolumeDiscountsParams) (*domain.ListVolumeDiscountsResult, *apierror.APIError) {
	ctx, span := volumeDiscountRepoTracer.Start(ctx, "repository.volume_discount.list")
	defer span.End()

	searchQuery := buildVolumeDiscountSearchParams(params.Query)
	isCustomer := params.CustomerAccountID != nil

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			var discounts []*domain.VolumeDiscount
			if isCustomer {
				rows, err := r.queries.ListVolumeDiscountsForCustomerBackward(ctx, sqlc.ListVolumeDiscountsForCustomerBackwardParams{
					AccountID:         params.AccountID,
					SearchQuery:       searchQuery,
					CursorCreatedAt:   cur.OccurredAt,
					CursorID:          cur.ID,
					CustomerAccountID: *params.CustomerAccountID,
					Limit:             params.Limit + 1,
				})
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				discounts = make([]*domain.VolumeDiscount, len(rows))
				for i, row := range rows {
					discounts[i] = mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
				}
			} else {
				rows, err := r.queries.ListVolumeDiscountsBackward(ctx, sqlc.ListVolumeDiscountsBackwardParams{
					AccountID:       params.AccountID,
					SearchQuery:     searchQuery,
					CursorCreatedAt: cur.OccurredAt,
					CursorID:        cur.ID,
					Limit:           params.Limit + 1,
				})
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				discounts = make([]*domain.VolumeDiscount, len(rows))
				for i, row := range rows {
					discounts[i] = mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
				}
			}

			result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, volumeDiscountCreatedAt, volumeDiscountID)
			if apiErr := r.enrichBatch(ctx, result, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListVolumeDiscountsResult{VolumeDiscounts: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		var discounts []*domain.VolumeDiscount
		if isCustomer {
			rows, err := r.queries.ListVolumeDiscountsForCustomerForward(ctx, sqlc.ListVolumeDiscountsForCustomerForwardParams{
				AccountID:         params.AccountID,
				SearchQuery:       searchQuery,
				CursorCreatedAt:   gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:          gosql.NullString{String: cur.ID, Valid: true},
				CustomerAccountID: *params.CustomerAccountID,
				Limit:             params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			discounts = make([]*domain.VolumeDiscount, len(rows))
			for i, row := range rows {
				discounts[i] = mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
			}
		} else {
			rows, err := r.queries.ListVolumeDiscountsForward(ctx, sqlc.ListVolumeDiscountsForwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:        gosql.NullString{String: cur.ID, Valid: true},
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			discounts = make([]*domain.VolumeDiscount, len(rows))
			for i, row := range rows {
				discounts[i] = mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
			}
		}

		result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, volumeDiscountCreatedAt, volumeDiscountID)
		if apiErr := r.enrichBatch(ctx, result, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListVolumeDiscountsResult{VolumeDiscounts: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	var discounts []*domain.VolumeDiscount
	if isCustomer {
		rows, err := r.queries.ListVolumeDiscountsForCustomerForward(ctx, sqlc.ListVolumeDiscountsForCustomerForwardParams{
			AccountID:         params.AccountID,
			SearchQuery:       searchQuery,
			CustomerAccountID: *params.CustomerAccountID,
			Limit:             params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		discounts = make([]*domain.VolumeDiscount, len(rows))
		for i, row := range rows {
			discounts[i] = mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
		}
	} else {
		rows, err := r.queries.ListVolumeDiscountsForward(ctx, sqlc.ListVolumeDiscountsForwardParams{
			AccountID:   params.AccountID,
			SearchQuery: searchQuery,
			Limit:       params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		discounts = make([]*domain.VolumeDiscount, len(rows))
		for i, row := range rows {
			discounts[i] = mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
		}
	}

	result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, volumeDiscountCreatedAt, volumeDiscountID)
	if apiErr := r.enrichBatch(ctx, result, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListVolumeDiscountsResult{VolumeDiscounts: result, PageInfo: pageInfo}, nil
}

func (r *volumeDiscountRepoImpl) Get(ctx context.Context, params domain.GetVolumeDiscountParams) (*domain.VolumeDiscount, *apierror.APIError) {
	ctx, span := volumeDiscountRepoTracer.Start(ctx, "repository.volume_discount.get")
	defer span.End()

	row, err := r.queries.GetVolumeDiscount(ctx, sqlc.GetVolumeDiscountParams{
		ID:        params.VolumeDiscountID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	d := mapVolumeDiscountBase(row.ID, row.Name, row.AccountID, row.CreatedAt, row.UpdatedAt)
	if apiErr := r.enrichSingle(ctx, d, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return d, nil
}

func (r *volumeDiscountRepoImpl) Create(ctx context.Context, id string, params domain.CreateVolumeDiscountParams) (*domain.VolumeDiscount, *apierror.APIError) {
	ctx, span := volumeDiscountRepoTracer.Start(ctx, "repository.volume_discount.create")
	defer span.End()

	// Insert discount
	err := r.queries.InsertVolumeDiscount(ctx, sqlc.InsertVolumeDiscountParams{
		ID:        id,
		Name:      params.Name,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Insert tiers
	for _, tier := range params.Tiers {
		err := r.queries.InsertVolumeDiscountTier(ctx, sqlc.InsertVolumeDiscountTierParams{
			ID:                 tier.ID,
			Name:               tier.Name,
			DiscountPercentage: tier.DiscountPercentage,
			Threshold:          tier.Threshold,
			ParentTierID:       toNullStringForVD(tier.ParentTierID),
			QuantityDiscountID: gosql.NullString{String: id, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert customer groups
	for _, cg := range params.CustomerGroups {
		err := r.queries.InsertVolumeDiscountCustomerGroup(ctx, sqlc.InsertVolumeDiscountCustomerGroupParams{
			ID:                 cg.ID,
			AccountGroupID:     cg.AccountGroupID,
			QuantityDiscountID: id,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert product lines
	for _, plID := range params.ProductLineIDs {
		err := r.queries.InsertVolumeDiscountProductLine(ctx, sqlc.InsertVolumeDiscountProductLineParams{
			ProductLineID:      plID,
			QuantityDiscountID: id,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert categories
	for _, catID := range params.CategoryIDs {
		err := r.queries.InsertVolumeDiscountCategory(ctx, sqlc.InsertVolumeDiscountCategoryParams{
			ItemCategoryID:     catID,
			QuantityDiscountID: id,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert attributes
	for _, attrID := range params.AttributeIDs {
		err := r.queries.InsertVolumeDiscountAttribute(ctx, sqlc.InsertVolumeDiscountAttributeParams{
			AttributeID:        attrID,
			QuantityDiscountID: id,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert units
	for _, unitID := range params.UnitIDs {
		err := r.queries.InsertVolumeDiscountUnit(ctx, sqlc.InsertVolumeDiscountUnitParams{
			QuantityDiscountID: id,
			UnitID:             unitID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, domain.GetVolumeDiscountParams{AccountID: params.AccountID, VolumeDiscountID: id, Includes: params.Includes})
}

func (r *volumeDiscountRepoImpl) Update(ctx context.Context, params domain.UpdateVolumeDiscountParams) (*domain.VolumeDiscount, *apierror.APIError) {
	ctx, span := volumeDiscountRepoTracer.Start(ctx, "repository.volume_discount.update")
	defer span.End()

	// Update base record
	// Rows-affected is deliberately not read as an existence check. MySQL reports rows
	// *changed*, not matched, so an update that only touches tiers or scopes — leaving
	// name untouched and landing on the same second as the last write — reports zero and
	// would be misread as a missing discount. The caller establishes existence with a Get
	// in this same transaction.
	_, err := r.queries.UpdateVolumeDiscount(ctx, sqlc.UpdateVolumeDiscountParams{
		Name:      toNullStringForVD(params.Name),
		ID:        params.VolumeDiscountID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Upsert tiers if provided
	if params.HasTiers {
		keepIDs := make([]string, 0)
		for _, tier := range params.Tiers {
			if tier.ID != nil {
				// Update existing tier
				tierParams := sqlc.UpdateVolumeDiscountTierParams{
					ParentTierID:       toNullStringForVD(tier.ParentTierID),
					ID:                 *tier.ID,
					QuantityDiscountID: gosql.NullString{String: params.VolumeDiscountID, Valid: true},
				}
				if tier.Name != nil {
					tierParams.Name = *tier.Name
				}
				if tier.DiscountPercentage != nil {
					tierParams.DiscountPercentage = *tier.DiscountPercentage
				}
				if tier.Threshold != nil {
					tierParams.Threshold = *tier.Threshold
				}
				_, err := r.queries.UpdateVolumeDiscountTier(ctx, tierParams)
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				keepIDs = append(keepIDs, *tier.ID)
			} else {
				// Insert new tier
				insertParams := sqlc.InsertVolumeDiscountTierParams{
					ID:                 tier.GeneratedID,
					ParentTierID:       toNullStringForVD(tier.ParentTierID),
					QuantityDiscountID: gosql.NullString{String: params.VolumeDiscountID, Valid: true},
				}
				if tier.Name != nil {
					insertParams.Name = *tier.Name
				}
				if tier.DiscountPercentage != nil {
					insertParams.DiscountPercentage = *tier.DiscountPercentage
				}
				if tier.Threshold != nil {
					insertParams.Threshold = *tier.Threshold
				}
				err := r.queries.InsertVolumeDiscountTier(ctx, insertParams)
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				keepIDs = append(keepIDs, tier.GeneratedID)
			}
		}

		// Delete tiers not in the keep list
		if len(keepIDs) > 0 {
			err := r.queries.DeleteTiersNotInIDs(ctx, sqlc.DeleteTiersNotInIDsParams{
				QuantityDiscountID: gosql.NullString{String: params.VolumeDiscountID, Valid: true},
				KeepIds:            keepIDs,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		} else {
			err := r.queries.DeleteTiersByDiscountID(ctx, gosql.NullString{String: params.VolumeDiscountID, Valid: true})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Set-replace customer groups if provided
	if params.HasCustomerGroups {
		if err := r.queries.DeleteCustomerGroupsByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		for _, cg := range params.CustomerGroups {
			err := r.queries.InsertVolumeDiscountCustomerGroup(ctx, sqlc.InsertVolumeDiscountCustomerGroupParams{
				ID:                 cg.ID,
				AccountGroupID:     cg.AccountGroupID,
				QuantityDiscountID: params.VolumeDiscountID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Set-replace product lines if provided
	if params.HasProductLines {
		if err := r.queries.DeleteProductLinesByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		for _, plID := range params.ProductLineIDs {
			err := r.queries.InsertVolumeDiscountProductLine(ctx, sqlc.InsertVolumeDiscountProductLineParams{
				ProductLineID:      plID,
				QuantityDiscountID: params.VolumeDiscountID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Set-replace categories if provided
	if params.HasCategories {
		if err := r.queries.DeleteCategoriesByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		for _, catID := range params.CategoryIDs {
			err := r.queries.InsertVolumeDiscountCategory(ctx, sqlc.InsertVolumeDiscountCategoryParams{
				ItemCategoryID:     catID,
				QuantityDiscountID: params.VolumeDiscountID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Set-replace attributes if provided
	if params.HasAttributes {
		if err := r.queries.DeleteAttributesByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		for _, attrID := range params.AttributeIDs {
			err := r.queries.InsertVolumeDiscountAttribute(ctx, sqlc.InsertVolumeDiscountAttributeParams{
				AttributeID:        attrID,
				QuantityDiscountID: params.VolumeDiscountID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Set-replace units if provided
	if params.HasUnits {
		if err := r.queries.DeleteUnitsByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		for _, unitID := range params.UnitIDs {
			err := r.queries.InsertVolumeDiscountUnit(ctx, sqlc.InsertVolumeDiscountUnitParams{
				QuantityDiscountID: params.VolumeDiscountID,
				UnitID:             unitID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	return r.Get(ctx, domain.GetVolumeDiscountParams{AccountID: params.AccountID, VolumeDiscountID: params.VolumeDiscountID, Includes: params.Includes})
}

func (r *volumeDiscountRepoImpl) Delete(ctx context.Context, params domain.DeleteVolumeDiscountParams) *apierror.APIError {
	ctx, span := volumeDiscountRepoTracer.Start(ctx, "repository.volume_discount.delete")
	defer span.End()

	// Delete junction tables
	if err := r.queries.DeleteCustomerGroupsByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.DeleteProductLinesByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.DeleteCategoriesByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.DeleteAttributesByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.DeleteUnitsByDiscountID(ctx, params.VolumeDiscountID); db.MapSQLError(err) != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	// Delete tiers
	if err := r.queries.DeleteTiersByDiscountID(ctx, gosql.NullString{String: params.VolumeDiscountID, Valid: true}); db.MapSQLError(err) != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	// Delete discount
	result, err := r.queries.DeleteVolumeDiscount(ctx, sqlc.DeleteVolumeDiscountParams{
		ID:        params.VolumeDiscountID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Volume discount not found."))
	}

	return nil
}

func (r *volumeDiscountRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := volumeDiscountRepoTracer.Start(ctx, "repository.volume_discount.exists_by_name")
	defer span.End()

	count, err := r.queries.CountVolumeDiscountsByName(ctx, sqlc.CountVolumeDiscountsByNameParams{
		AccountID: accountID,
		Name:      name,
		ExcludeID: toNullStringForVD(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
