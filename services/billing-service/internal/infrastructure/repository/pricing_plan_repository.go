package repository

import (
	"context"
	gosql "database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	"github.com/open-mrp/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var pricingPlanRepoTracer = tracing.GetTracer("billing-service.pricing_plan_repository")

type pricingPlanRepoImpl struct {
	queries *sqlc.Queries
}

func NewPricingPlanRepo(queries *sqlc.Queries) domain.PricingPlanRepo {
	return &pricingPlanRepoImpl{queries: queries}
}

func planCreatedAt(p domain.PricingPlan) time.Time { return p.CreatedAt }
func planID(p domain.PricingPlan) int64            { return p.ID }

func mapPricingPlanForwardRow(row sqlc.ListPricingPlansForwardRow) domain.PricingPlan {
	pricePerSeat, _ := strconv.ParseFloat(row.PricePerSeat, 64)

	var pricePerMonth *float64
	if row.PricePerMonth.Valid {
		v, _ := strconv.ParseFloat(row.PricePerMonth.String, 64)
		pricePerMonth = &v
	}

	var seatMinimum *int
	if row.SeatMinimum.Valid {
		v := int(row.SeatMinimum.Int32)
		seatMinimum = &v
	}

	var displayFeatures []string
	_ = json.Unmarshal(row.DisplayFeatures, &displayFeatures)

	var includesPreviousPlan *string
	if row.IncludesPreviousPlan.Valid {
		includesPreviousPlan = &row.IncludesPreviousPlan.String
	}

	var stripePricingPlanID *string
	if row.StripePricingPlanID.Valid {
		stripePricingPlanID = &row.StripePricingPlanID.String
	}

	return domain.PricingPlan{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		TypeID:               row.TypeID,
		Name:                 row.Name,
		PlanTypeCode:         row.PlanTypeCode,
		PricePerSeat:         pricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		DisplayFeatures:      displayFeatures,
		DisplayOrder:         int(row.DisplayOrder),
		IsHighlighted:        row.IsHighlighted,
		ButtonText:           row.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
		StripePricingPlanID:  stripePricingPlanID,
	}
}

func mapPricingPlanBackwardRow(row sqlc.ListPricingPlansBackwardRow) domain.PricingPlan {
	pricePerSeat, _ := strconv.ParseFloat(row.PricePerSeat, 64)

	var pricePerMonth *float64
	if row.PricePerMonth.Valid {
		v, _ := strconv.ParseFloat(row.PricePerMonth.String, 64)
		pricePerMonth = &v
	}

	var seatMinimum *int
	if row.SeatMinimum.Valid {
		v := int(row.SeatMinimum.Int32)
		seatMinimum = &v
	}

	var displayFeatures []string
	_ = json.Unmarshal(row.DisplayFeatures, &displayFeatures)

	var includesPreviousPlan *string
	if row.IncludesPreviousPlan.Valid {
		includesPreviousPlan = &row.IncludesPreviousPlan.String
	}

	var stripePricingPlanID *string
	if row.StripePricingPlanID.Valid {
		stripePricingPlanID = &row.StripePricingPlanID.String
	}

	return domain.PricingPlan{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		TypeID:               row.TypeID,
		Name:                 row.Name,
		PlanTypeCode:         row.PlanTypeCode,
		PricePerSeat:         pricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		DisplayFeatures:      displayFeatures,
		DisplayOrder:         int(row.DisplayOrder),
		IsHighlighted:        row.IsHighlighted,
		ButtonText:           row.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
		StripePricingPlanID:  stripePricingPlanID,
	}
}

func mapGetPlanByCodeRow(row sqlc.GetPlanByCodeRow) domain.PricingPlan {
	pricePerSeat, _ := strconv.ParseFloat(row.PricePerSeat, 64)

	var pricePerMonth *float64
	if row.PricePerMonth.Valid {
		v, _ := strconv.ParseFloat(row.PricePerMonth.String, 64)
		pricePerMonth = &v
	}

	var seatMinimum *int
	if row.SeatMinimum.Valid {
		v := int(row.SeatMinimum.Int32)
		seatMinimum = &v
	}

	var displayFeatures []string
	_ = json.Unmarshal(row.DisplayFeatures, &displayFeatures)

	var includesPreviousPlan *string
	if row.IncludesPreviousPlan.Valid {
		includesPreviousPlan = &row.IncludesPreviousPlan.String
	}

	var stripePricingPlanID *string
	if row.StripePricingPlanID.Valid {
		stripePricingPlanID = &row.StripePricingPlanID.String
	}

	return domain.PricingPlan{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		TypeID:               row.TypeID,
		Name:                 row.Name,
		PlanTypeCode:         row.PlanTypeCode,
		PricePerSeat:         pricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		DisplayFeatures:      displayFeatures,
		DisplayOrder:         int(row.DisplayOrder),
		IsHighlighted:        row.IsHighlighted,
		ButtonText:           row.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
		StripePricingPlanID:  stripePricingPlanID,
	}
}

func mapGetPlanByTypeIDRow(row sqlc.GetPlanByTypeIDRow) domain.PricingPlan {
	pricePerSeat, _ := strconv.ParseFloat(row.PricePerSeat, 64)

	var pricePerMonth *float64
	if row.PricePerMonth.Valid {
		v, _ := strconv.ParseFloat(row.PricePerMonth.String, 64)
		pricePerMonth = &v
	}

	var seatMinimum *int
	if row.SeatMinimum.Valid {
		v := int(row.SeatMinimum.Int32)
		seatMinimum = &v
	}

	var displayFeatures []string
	_ = json.Unmarshal(row.DisplayFeatures, &displayFeatures)

	var includesPreviousPlan *string
	if row.IncludesPreviousPlan.Valid {
		includesPreviousPlan = &row.IncludesPreviousPlan.String
	}

	var stripePricingPlanID *string
	if row.StripePricingPlanID.Valid {
		stripePricingPlanID = &row.StripePricingPlanID.String
	}

	return domain.PricingPlan{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		TypeID:               row.TypeID,
		Name:                 row.Name,
		PlanTypeCode:         row.PlanTypeCode,
		PricePerSeat:         pricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		DisplayFeatures:      displayFeatures,
		DisplayOrder:         int(row.DisplayOrder),
		IsHighlighted:        row.IsHighlighted,
		ButtonText:           row.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
		StripePricingPlanID:  stripePricingPlanID,
	}
}

func (r *pricingPlanRepoImpl) GetPlanByTypeID(ctx context.Context, typeID string) (*domain.PricingPlan, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, pricingPlanRepoTracer, "repository.pricing_plan.get_plan_by_type_id")
	defer span.End()

	row, err := r.queries.GetPlanByTypeID(ctx, typeID)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	plan := mapGetPlanByTypeIDRow(row)
	return &plan, nil
}

func (r *pricingPlanRepoImpl) GetPlanByCode(ctx context.Context, planCode string) (*domain.PricingPlan, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, pricingPlanRepoTracer, "repository.pricing_plan.get_plan_by_code")
	defer span.End()

	row, err := r.queries.GetPlanByCode(ctx, planCode)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	plan := mapGetPlanByCodeRow(row)
	return &plan, nil
}

func (r *pricingPlanRepoImpl) ListPricingPlans(ctx context.Context, cursor *string, limit int32, query *string) ([]domain.PricingPlan, pagination.PageInfo, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, pricingPlanRepoTracer, "repository.pricing_plan.list_pricing_plans")
	defer span.End()

	searchQuery := gosql.NullString{}
	if query != nil && *query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
	}

	var cursorDir *pagination.Direction

	if cursor != nil {
		cur, err := pagination.DecodeCursor(*cursor)
		if err != nil {
			return nil, pagination.PageInfo{}, apierror.NewValidationError("invalid pagination cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPricingPlansBackward(ctx, sqlc.ListPricingPlansBackwardParams{
				CursorCreatedAt: cur.CreatedAt,
				CursorID:        cur.ID,
				SearchQuery:     searchQuery,
				Limit:           limit + 1,
			})
			if err != nil {
				return nil, pagination.PageInfo{}, tracing.Trace(span, db.MapSQLError(err))
			}
			plans := make([]domain.PricingPlan, len(rows))
			for i, row := range rows {
				plans[i] = mapPricingPlanBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPage(plans, limit, cursorDir, planCreatedAt, planID)
			return result, pageInfo, nil
		}

		// Forward
		rows, err := r.queries.ListPricingPlansForward(ctx, sqlc.ListPricingPlansForwardParams{
			CursorCreatedAt: gosql.NullTime{Time: cur.CreatedAt, Valid: true},
			CursorID:        gosql.NullInt64{Int64: cur.ID, Valid: true},
			SearchQuery:     searchQuery,
			Limit:           limit + 1,
		})
		if err != nil {
			return nil, pagination.PageInfo{}, tracing.Trace(span, db.MapSQLError(err))
		}
		plans := make([]domain.PricingPlan, len(rows))
		for i, row := range rows {
			plans[i] = mapPricingPlanForwardRow(row)
		}
		result, pageInfo := pagination.BuildPage(plans, limit, cursorDir, planCreatedAt, planID)
		return result, pageInfo, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPricingPlansForward(ctx, sqlc.ListPricingPlansForwardParams{
		SearchQuery: searchQuery,
		Limit:       limit + 1,
	})
	if err != nil {
		return nil, pagination.PageInfo{}, tracing.Trace(span, db.MapSQLError(err))
	}

	plans := make([]domain.PricingPlan, len(rows))
	for i, row := range rows {
		plans[i] = mapPricingPlanForwardRow(row)
	}
	result, pageInfo := pagination.BuildPage(plans, limit, cursorDir, planCreatedAt, planID)
	return result, pageInfo, nil
}

func (r *pricingPlanRepoImpl) GetPlanLimitsByTypeID(ctx context.Context, typeID string) ([]domain.PlanLimit, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, pricingPlanRepoTracer, "repository.pricing_plan.get_plan_limits_by_type_id")
	defer span.End()

	rows, err := r.queries.GetPlanLimitsByTypeID(ctx, typeID)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	limits := make([]domain.PlanLimit, len(rows))
	for i, row := range rows {
		var value *int
		if row.Value.Valid {
			v := int(row.Value.Int32)
			value = &v
		}

		limits[i] = domain.PlanLimit{
			Key:   row.Key,
			Value: value,
		}
	}

	return limits, nil
}
