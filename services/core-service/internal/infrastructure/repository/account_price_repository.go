package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var accountPriceRepoTracer = tracing.GetTracer("core-service.account_price_repository")

type accountPriceRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountPriceRepo(queries *sqlc.Queries) domain.AccountPriceRepo {
	return &accountPriceRepoImpl{queries: queries}
}

func accountPriceCreatedAt(ap *domain.AccountPrice) time.Time { return ap.CreatedAt }
func accountPriceID(ap *domain.AccountPrice) string           { return ap.ID }

func mapForwardAccountPriceRow(row sqlc.ListAccountPricesForwardRow) *domain.AccountPrice {
	ap := &domain.AccountPrice{
		ID:                               row.ID,
		OwnerAccountID:                   row.OwnerAccountID,
		RecipientAccountID:               row.RecipientAccountID,
		RecipientAccountName:             row.RecipientAccountName,
		RecipientAccountNumber:           row.RecipientAccountNumber,
		RecipientAccountIsEdiEnabled:     row.RecipientAccountIsEdiEnabled,
		RecipientAccountRelationshipType: row.RecipientAccountRelationshipType,
		RecipientAccountCreatedAt:        row.RecipientAccountCreatedAt,
		RecipientAccountUpdatedAt:        row.RecipientAccountUpdatedAt,
		ProductLineID:                    row.ProductLineID,
		ProductLineName:                  row.ProductLineName,
		ProductLineIsCommissionExempt:    row.ProductLineIsCommissionExempt,
		ProductLineIsFreightExempt:       row.ProductLineIsFreightExempt,
		ProductLineCreatedAt:             row.ProductLineCreatedAt,
		ProductLineUpdatedAt:             row.ProductLineUpdatedAt,
		RateID:                           row.RateID,
		RateValue:                        row.RateValue,
		RateCreatedAt:                    row.RateCreatedAt,
		RateUpdatedAt:                    row.RateUpdatedAt,
		NumeratorUnitID:                  row.NumeratorUnitID,
		NumeratorUnitName:                row.NumeratorUnitName,
		NumeratorUnitAbbr:                row.NumeratorUnitAbbreviation,
		NumeratorUnitType:                row.NumeratorUnitType,
		NumeratorUnitRatioNumerator:      row.NumeratorUnitRatioNumerator,
		NumeratorUnitRatioDenominator:    row.NumeratorUnitRatioDenominator,
		NumeratorUnitOffsetNumerator:     row.NumeratorUnitOffsetNumerator,
		NumeratorUnitOffsetDenominator:   row.NumeratorUnitOffsetDenominator,
		NumeratorUnitCreatedAt:           row.NumeratorUnitCreatedAt,
		NumeratorUnitUpdatedAt:           row.NumeratorUnitUpdatedAt,
		DenominatorUnitID:                row.DenominatorUnitID,
		DenominatorUnitName:              row.DenominatorUnitName,
		DenominatorUnitAbbr:              row.DenominatorUnitAbbreviation,
		DenominatorUnitType:              row.DenominatorUnitType,
		DenominatorUnitRatioNumerator:    row.DenominatorUnitRatioNumerator,
		DenominatorUnitRatioDenominator:  row.DenominatorUnitRatioDenominator,
		DenominatorUnitOffsetNumerator:   row.DenominatorUnitOffsetNumerator,
		DenominatorUnitOffsetDenominator: row.DenominatorUnitOffsetDenominator,
		DenominatorUnitCreatedAt:         row.DenominatorUnitCreatedAt,
		DenominatorUnitUpdatedAt:         row.DenominatorUnitUpdatedAt,
		CreatedAt:                        row.CreatedAt,
		UpdatedAt:                        row.UpdatedAt,
	}
	if row.RecipientAccountStatus.Valid {
		ap.RecipientAccountStatus = row.RecipientAccountStatus.String
	}
	if row.RecipientAccountCommissionPolicy.Valid {
		ap.RecipientAccountCommissionPolicy = row.RecipientAccountCommissionPolicy.String
	}
	return ap
}

func mapBackwardAccountPriceRow(row sqlc.ListAccountPricesBackwardRow) *domain.AccountPrice {
	ap := &domain.AccountPrice{
		ID:                               row.ID,
		OwnerAccountID:                   row.OwnerAccountID,
		RecipientAccountID:               row.RecipientAccountID,
		RecipientAccountName:             row.RecipientAccountName,
		RecipientAccountNumber:           row.RecipientAccountNumber,
		RecipientAccountIsEdiEnabled:     row.RecipientAccountIsEdiEnabled,
		RecipientAccountRelationshipType: row.RecipientAccountRelationshipType,
		RecipientAccountCreatedAt:        row.RecipientAccountCreatedAt,
		RecipientAccountUpdatedAt:        row.RecipientAccountUpdatedAt,
		ProductLineID:                    row.ProductLineID,
		ProductLineName:                  row.ProductLineName,
		ProductLineIsCommissionExempt:    row.ProductLineIsCommissionExempt,
		ProductLineIsFreightExempt:       row.ProductLineIsFreightExempt,
		ProductLineCreatedAt:             row.ProductLineCreatedAt,
		ProductLineUpdatedAt:             row.ProductLineUpdatedAt,
		RateID:                           row.RateID,
		RateValue:                        row.RateValue,
		RateCreatedAt:                    row.RateCreatedAt,
		RateUpdatedAt:                    row.RateUpdatedAt,
		NumeratorUnitID:                  row.NumeratorUnitID,
		NumeratorUnitName:                row.NumeratorUnitName,
		NumeratorUnitAbbr:                row.NumeratorUnitAbbreviation,
		NumeratorUnitType:                row.NumeratorUnitType,
		NumeratorUnitRatioNumerator:      row.NumeratorUnitRatioNumerator,
		NumeratorUnitRatioDenominator:    row.NumeratorUnitRatioDenominator,
		NumeratorUnitOffsetNumerator:     row.NumeratorUnitOffsetNumerator,
		NumeratorUnitOffsetDenominator:   row.NumeratorUnitOffsetDenominator,
		NumeratorUnitCreatedAt:           row.NumeratorUnitCreatedAt,
		NumeratorUnitUpdatedAt:           row.NumeratorUnitUpdatedAt,
		DenominatorUnitID:                row.DenominatorUnitID,
		DenominatorUnitName:              row.DenominatorUnitName,
		DenominatorUnitAbbr:              row.DenominatorUnitAbbreviation,
		DenominatorUnitType:              row.DenominatorUnitType,
		DenominatorUnitRatioNumerator:    row.DenominatorUnitRatioNumerator,
		DenominatorUnitRatioDenominator:  row.DenominatorUnitRatioDenominator,
		DenominatorUnitOffsetNumerator:   row.DenominatorUnitOffsetNumerator,
		DenominatorUnitOffsetDenominator: row.DenominatorUnitOffsetDenominator,
		DenominatorUnitCreatedAt:         row.DenominatorUnitCreatedAt,
		DenominatorUnitUpdatedAt:         row.DenominatorUnitUpdatedAt,
		CreatedAt:                        row.CreatedAt,
		UpdatedAt:                        row.UpdatedAt,
	}
	if row.RecipientAccountStatus.Valid {
		ap.RecipientAccountStatus = row.RecipientAccountStatus.String
	}
	if row.RecipientAccountCommissionPolicy.Valid {
		ap.RecipientAccountCommissionPolicy = row.RecipientAccountCommissionPolicy.String
	}
	return ap
}

func mapGetAccountPriceRow(row sqlc.GetAccountPriceRow) *domain.AccountPrice {
	ap := &domain.AccountPrice{
		ID:                               row.ID,
		OwnerAccountID:                   row.OwnerAccountID,
		RecipientAccountID:               row.RecipientAccountID,
		RecipientAccountName:             row.RecipientAccountName,
		RecipientAccountNumber:           row.RecipientAccountNumber,
		RecipientAccountIsEdiEnabled:     row.RecipientAccountIsEdiEnabled,
		RecipientAccountRelationshipType: row.RecipientAccountRelationshipType,
		RecipientAccountCreatedAt:        row.RecipientAccountCreatedAt,
		RecipientAccountUpdatedAt:        row.RecipientAccountUpdatedAt,
		ProductLineID:                    row.ProductLineID,
		ProductLineName:                  row.ProductLineName,
		ProductLineIsCommissionExempt:    row.ProductLineIsCommissionExempt,
		ProductLineIsFreightExempt:       row.ProductLineIsFreightExempt,
		ProductLineCreatedAt:             row.ProductLineCreatedAt,
		ProductLineUpdatedAt:             row.ProductLineUpdatedAt,
		RateID:                           row.RateID,
		RateValue:                        row.RateValue,
		RateCreatedAt:                    row.RateCreatedAt,
		RateUpdatedAt:                    row.RateUpdatedAt,
		NumeratorUnitID:                  row.NumeratorUnitID,
		NumeratorUnitName:                row.NumeratorUnitName,
		NumeratorUnitAbbr:                row.NumeratorUnitAbbreviation,
		NumeratorUnitType:                row.NumeratorUnitType,
		NumeratorUnitRatioNumerator:      row.NumeratorUnitRatioNumerator,
		NumeratorUnitRatioDenominator:    row.NumeratorUnitRatioDenominator,
		NumeratorUnitOffsetNumerator:     row.NumeratorUnitOffsetNumerator,
		NumeratorUnitOffsetDenominator:   row.NumeratorUnitOffsetDenominator,
		NumeratorUnitCreatedAt:           row.NumeratorUnitCreatedAt,
		NumeratorUnitUpdatedAt:           row.NumeratorUnitUpdatedAt,
		DenominatorUnitID:                row.DenominatorUnitID,
		DenominatorUnitName:              row.DenominatorUnitName,
		DenominatorUnitAbbr:              row.DenominatorUnitAbbreviation,
		DenominatorUnitType:              row.DenominatorUnitType,
		DenominatorUnitRatioNumerator:    row.DenominatorUnitRatioNumerator,
		DenominatorUnitRatioDenominator:  row.DenominatorUnitRatioDenominator,
		DenominatorUnitOffsetNumerator:   row.DenominatorUnitOffsetNumerator,
		DenominatorUnitOffsetDenominator: row.DenominatorUnitOffsetDenominator,
		DenominatorUnitCreatedAt:         row.DenominatorUnitCreatedAt,
		DenominatorUnitUpdatedAt:         row.DenominatorUnitUpdatedAt,
		CreatedAt:                        row.CreatedAt,
		UpdatedAt:                        row.UpdatedAt,
	}
	if row.RecipientAccountStatus.Valid {
		ap.RecipientAccountStatus = row.RecipientAccountStatus.String
	}
	if row.RecipientAccountCommissionPolicy.Valid {
		ap.RecipientAccountCommissionPolicy = row.RecipientAccountCommissionPolicy.String
	}
	return ap
}

func (r *accountPriceRepoImpl) fetchCategoriesAndAttributes(ctx context.Context, ap *domain.AccountPrice) *apierror.APIError {
	catRows, err := r.queries.GetAccountPriceCategories(ctx, ap.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	cats := make([]domain.AccountPriceCategory, len(catRows))
	for i, c := range catRows {
		cats[i] = domain.AccountPriceCategory{
			ID:        c.ID,
			Name:      c.Name,
			Type:      c.Type,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}
	ap.Categories = cats

	attrRows, err := r.queries.GetAccountPriceAttributes(ctx, ap.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	attrs := make([]domain.AccountPriceAttribute, len(attrRows))
	for i, a := range attrRows {
		attrs[i] = domain.AccountPriceAttribute{
			ID:        a.ID,
			Value:     a.Text,
			ColorCode: a.ColorCode,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		}
	}
	ap.Attributes = attrs

	return nil
}

// ResolveRecipientAccountIDs expands a customer into the set of accounts whose prices reach
// it: itself, plus its parent when it is a child account. A price recorded against a parent
// applies to orders its children place, so filtering on the customer alone would hide prices
// that in fact price their orders.
func (r *accountPriceRepoImpl) ResolveRecipientAccountIDs(ctx context.Context, ownerAccountID, customerAccountID string) ([]string, *apierror.APIError) {
	ctx, span := accountPriceRepoTracer.Start(ctx, "repository.account_price.resolve_recipient_account_ids")
	defer span.End()

	ids := []string{customerAccountID}

	parentID, err := r.queries.GetBuyerParentAccountID(ctx, sqlc.GetBuyerParentAccountIDParams{
		OwnerAccountID:    ownerAccountID,
		CustomerAccountID: customerAccountID,
	})
	switch {
	case err == nil:
		if parentID != "" && parentID != customerAccountID {
			ids = append(ids, parentID)
		}
	case errors.Is(err, gosql.ErrNoRows):
		// Standalone or parent account; only its own prices apply.
	default:
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return ids, nil
}

// sqlc renders an empty IN list as invalid SQL, so the flag carries "no filter" and the
// slice is only read when it is set.
func buildRecipientFilter(recipientAccountIDs []string) (bool, []string) {
	if len(recipientAccountIDs) == 0 {
		return false, []string{""}
	}
	return true, recipientAccountIDs
}

func buildAccountPriceSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *accountPriceRepoImpl) List(ctx context.Context, params domain.ListAccountPricesParams) (*domain.ListAccountPricesResult, *apierror.APIError) {
	ctx, span := accountPriceRepoTracer.Start(ctx, "repository.account_price.list")
	defer span.End()

	includeRecipientFilter, recipientAccountIDs := buildRecipientFilter(params.RecipientAccountIDs)
	searchQuery := buildAccountPriceSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAccountPricesBackward(ctx, sqlc.ListAccountPricesBackwardParams{
				OwnerAccountID:         params.AccountID,
				IncludeRecipientFilter: includeRecipientFilter,
				RecipientAccountIds:    recipientAccountIDs,
				SearchQuery:            searchQuery,
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			prices := make([]*domain.AccountPrice, len(rows))
			for i, row := range rows {
				prices[i] = mapBackwardAccountPriceRow(row)
				if apiErr := r.fetchCategoriesAndAttributes(ctx, prices[i]); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
			result, pageInfo := pagination.BuildPageString(prices, params.Limit, cursorDir, accountPriceCreatedAt, accountPriceID)
			return &domain.ListAccountPricesResult{AccountPrices: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListAccountPricesForward(ctx, sqlc.ListAccountPricesForwardParams{
			OwnerAccountID:         params.AccountID,
			IncludeRecipientFilter: includeRecipientFilter,
			RecipientAccountIds:    recipientAccountIDs,
			SearchQuery:            searchQuery,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		prices := make([]*domain.AccountPrice, len(rows))
		for i, row := range rows {
			prices[i] = mapForwardAccountPriceRow(row)
			if apiErr := r.fetchCategoriesAndAttributes(ctx, prices[i]); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		result, pageInfo := pagination.BuildPageString(prices, params.Limit, cursorDir, accountPriceCreatedAt, accountPriceID)
		return &domain.ListAccountPricesResult{AccountPrices: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListAccountPricesForward(ctx, sqlc.ListAccountPricesForwardParams{
		OwnerAccountID:         params.AccountID,
		IncludeRecipientFilter: includeRecipientFilter,
		RecipientAccountIds:    recipientAccountIDs,
		SearchQuery:            searchQuery,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	prices := make([]*domain.AccountPrice, len(rows))
	for i, row := range rows {
		prices[i] = mapForwardAccountPriceRow(row)
		if apiErr := r.fetchCategoriesAndAttributes(ctx, prices[i]); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	result, pageInfo := pagination.BuildPageString(prices, params.Limit, cursorDir, accountPriceCreatedAt, accountPriceID)
	return &domain.ListAccountPricesResult{AccountPrices: result, PageInfo: pageInfo}, nil
}

func (r *accountPriceRepoImpl) Get(ctx context.Context, accountID, accountPriceID string) (*domain.AccountPrice, *apierror.APIError) {
	ctx, span := accountPriceRepoTracer.Start(ctx, "repository.account_price.get")
	defer span.End()

	row, err := r.queries.GetAccountPrice(ctx, sqlc.GetAccountPriceParams{
		ID:             accountPriceID,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ap := mapGetAccountPriceRow(row)
	if apiErr := r.fetchCategoriesAndAttributes(ctx, ap); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return ap, nil
}

func (r *accountPriceRepoImpl) Create(ctx context.Context, accountPriceID, rateID string, params domain.CreateAccountPriceParams) (*domain.AccountPrice, *apierror.APIError) {
	ctx, span := accountPriceRepoTracer.Start(ctx, "repository.account_price.create")
	defer span.End()

	// Insert rate
	err := r.queries.InsertRate(ctx, sqlc.InsertRateParams{
		ID:                rateID,
		Value:             params.RateValue,
		NumeratorUnitID:   params.RateNumeratorUnitID,
		DenominatorUnitID: params.RateDenominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Insert account_price
	err = r.queries.InsertAccountPrice(ctx, sqlc.InsertAccountPriceParams{
		ID:                 accountPriceID,
		OwnerAccountID:     params.AccountID,
		RecipientAccountID: params.RecipientAccountID,
		ProductLineID:      params.ProductLineID,
		UnitValueID:        rateID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Insert category associations
	for _, categoryID := range params.CategoryIDs {
		joinID, apiErr := id.GenID(id.AccountPriceItemCategoryIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		err = r.queries.InsertAccountPriceCategory(ctx, sqlc.InsertAccountPriceCategoryParams{
			ID:             joinID,
			AccountPriceID: accountPriceID,
			ItemCategoryID: categoryID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Insert attribute associations
	for _, attributeID := range params.AttributeIDs {
		joinID, apiErr := id.GenID(id.AccountPriceAttributeIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		err = r.queries.InsertAccountPriceAttribute(ctx, sqlc.InsertAccountPriceAttributeParams{
			ID:             joinID,
			AccountPriceID: accountPriceID,
			AttributeID:    attributeID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, params.AccountID, accountPriceID)
}

func (r *accountPriceRepoImpl) Update(ctx context.Context, params domain.UpdateAccountPriceParams) (*domain.AccountPrice, *apierror.APIError) {
	ctx, span := accountPriceRepoTracer.Start(ctx, "repository.account_price.update")
	defer span.End()

	// Get the rate ID for this account price
	rateID, err := r.queries.GetRateIDByAccountPriceID(ctx, sqlc.GetRateIDByAccountPriceIDParams{
		ID:             params.AccountPriceID,
		OwnerAccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Update account_price
	err = r.queries.UpdateAccountPrice(ctx, sqlc.UpdateAccountPriceParams{
		ID:                 params.AccountPriceID,
		OwnerAccountID:     params.AccountID,
		RecipientAccountID: toNullString(params.RecipientAccountID),
		ProductLineID:      toNullString(params.ProductLineID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Update rate
	err = r.queries.UpdateRate(ctx, sqlc.UpdateRateParams{
		ID:                rateID,
		Value:             toNullString(params.RateValue),
		NumeratorUnitID:   toNullString(params.RateNumeratorUnitID),
		DenominatorUnitID: toNullString(params.RateDenominatorUnitID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// If categories provided, delete-all-then-recreate
	if params.CategoryIDs != nil {
		err = r.queries.DeleteAccountPriceCategoriesByPriceID(ctx, params.AccountPriceID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, categoryID := range *params.CategoryIDs {
			joinID, apiErr := id.GenID(id.AccountPriceItemCategoryIDPrefix, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			err = r.queries.InsertAccountPriceCategory(ctx, sqlc.InsertAccountPriceCategoryParams{
				ID:             joinID,
				AccountPriceID: params.AccountPriceID,
				ItemCategoryID: categoryID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// If attributes provided, delete-all-then-recreate
	if params.AttributeIDs != nil {
		err = r.queries.DeleteAccountPriceAttributesByPriceID(ctx, params.AccountPriceID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, attributeID := range *params.AttributeIDs {
			joinID, apiErr := id.GenID(id.AccountPriceAttributeIDPrefix, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			err = r.queries.InsertAccountPriceAttribute(ctx, sqlc.InsertAccountPriceAttributeParams{
				ID:             joinID,
				AccountPriceID: params.AccountPriceID,
				AttributeID:    attributeID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	return r.Get(ctx, params.AccountID, params.AccountPriceID)
}

func (r *accountPriceRepoImpl) Delete(ctx context.Context, accountID, accountPriceID string) *apierror.APIError {
	ctx, span := accountPriceRepoTracer.Start(ctx, "repository.account_price.delete")
	defer span.End()

	// Get the rate ID before deleting the account price
	rateID, err := r.queries.GetRateIDByAccountPriceID(ctx, sqlc.GetRateIDByAccountPriceIDParams{
		ID:             accountPriceID,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete category associations
	err = r.queries.DeleteAccountPriceCategoriesByPriceID(ctx, accountPriceID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete attribute associations
	err = r.queries.DeleteAccountPriceAttributesByPriceID(ctx, accountPriceID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete account_price
	err = r.queries.DeleteAccountPrice(ctx, sqlc.DeleteAccountPriceParams{
		ID:             accountPriceID,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete rate
	err = r.queries.DeleteRate(ctx, rateID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
