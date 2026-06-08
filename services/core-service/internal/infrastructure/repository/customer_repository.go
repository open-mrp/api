package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

var customerRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.customer")

type customerRepoImpl struct {
	queries *sqlc.Queries
}

func NewCustomerRepo(queries *sqlc.Queries) domain.CustomerRepo {
	return &customerRepoImpl{queries: queries}
}

func customerCreatedAt(c *domain.Customer) time.Time { return c.CreatedAt }
func customerID(c *domain.Customer) string           { return c.ID }

func nullStringPtr(ns gosql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullCarrierBillingType(ns gosql.NullString) *constants.CarrierBillingType {
	if ns.Valid {
		cbt := constants.CarrierBillingType(ns.String)
		return &cbt
	}
	return nil
}

func nullPriorityCode(ns gosql.NullString) *constants.PriorityCode {
	if ns.Valid {
		pc := constants.PriorityCode(ns.String)
		return &pc
	}
	return nil
}

func nullBoolPtr(nb gosql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

func nullShippingTermType(isFreightExempt, isCarrierRate gosql.NullBool) *constants.ShippingTermType {
	if !isFreightExempt.Valid && !isCarrierRate.Valid {
		return nil
	}
	t := shippingTermTypeFromBooleans(isFreightExempt.Bool, isCarrierRate.Bool)
	return &t
}

func nullCommissionPolicy(ns gosql.NullString) *constants.CommissionPolicy {
	if ns.Valid {
		cp := constants.CommissionPolicy(ns.String)
		return &cp
	}
	return nil
}

func nullFreightPolicy(ns gosql.NullString) *constants.FreightPolicy {
	if ns.Valid {
		fp := constants.FreightPolicy(ns.String)
		return &fp
	}
	return nil
}

func nullAccountGroupType(ns gosql.NullString) *constants.AccountGroupType {
	if ns.Valid {
		agt := constants.AccountGroupType(ns.String)
		return &agt
	}
	return nil
}

func nullAccountUserStatus(ns gosql.NullString) *constants.AccountUserStatus {
	if ns.Valid {
		aus := constants.AccountUserStatus(ns.String)
		return &aus
	}
	return nil
}

func mapListCustomerForwardRow(row sqlc.ListCustomersForwardRow) *domain.Customer {
	var billToAddress *domain.CustomerAddress
	if row.DefaultBillingAddressID.Valid {
		billToAddress = buildCustomerAddress(
			row.DefaultBillingAddressID.String,
			row.DefaultBillingAddressName.String,
			row.DefaultBillingAddressPhone,
			row.DefaultBillingAddressEmail,
			row.DefaultBillingIsDropShip.Bool,
			row.DefaultBillingGeolocationID,
			row.DefaultBillingStreetLine1,
			row.DefaultBillingStreetLine2,
			row.DefaultBillingLocality,
			row.DefaultBillingState,
			row.DefaultBillingPostalCode,
			row.DefaultBillingCountry,
			row.DefaultBillingAddressCreatedAt.Time,
			row.DefaultBillingAddressUpdatedAt.Time,
		)
	}
	var shipToAddress *domain.CustomerAddress
	if row.DefaultShippingAddressID.Valid {
		shipToAddress = buildCustomerAddress(
			row.DefaultShippingAddressID.String,
			row.DefaultShippingAddressName.String,
			row.DefaultShippingAddressPhone,
			row.DefaultShippingAddressEmail,
			row.DefaultShippingIsDropShip.Bool,
			row.DefaultShippingGeolocationID,
			row.DefaultShippingStreetLine1,
			row.DefaultShippingStreetLine2,
			row.DefaultShippingLocality,
			row.DefaultShippingState,
			row.DefaultShippingPostalCode,
			row.DefaultShippingCountry,
			row.DefaultShippingAddressCreatedAt.Time,
			row.DefaultShippingAddressUpdatedAt.Time,
		)
	}
	return &domain.Customer{
		ID:                                 row.AccountID,
		Name:                               row.AccountName,
		Number:                             row.ExternalNumber,
		Status:                             constants.AccountStatusCode(row.Status.String),
		IsEdiEnabled:                       row.IsEdiEnabled,
		IsParentAccount:                    row.IsParentAccount,
		CommissionPolicy:                   constants.CommissionPolicy(row.CommissionStatusCode.String),
		FreightPolicy:                      constants.FreightPolicy(row.FreightStatusCode.String),
		Note:                               nullStringPtr(row.Notes),
		Email:                              nullStringPtr(row.Email),
		Phone:                              nullStringPtr(row.PhoneNumber),
		URL:                                nullStringPtr(row.WebsiteUrl),
		CarrierBillingType:                 nullCarrierBillingType(row.CarrierBillingType),
		CarrierBillingAccount:              nullStringPtr(row.CarrierBillingAccount),
		CreditLimitID:                      nullStringPtr(row.CreditLimitID),
		CreditLimitValue:                   nullStringPtr(row.CreditLimitValue),
		CreditLimitUnitID:                  nullStringPtr(row.CreditLimitUnitID),
		CreditLimitUnitAbbreviation:        nullStringPtr(row.CreditLimitUnitAbbreviation),
		CreditLimitUnitName:                nullStringPtr(row.CreditLimitUnitName),
		CreditLimitUnitType:                nullStringPtr(row.CreditLimitUnitType),
		DefaultCarrierID:                   nullStringPtr(row.DefaultCarrierID),
		DefaultCarrierName:                 nullStringPtr(row.DefaultCarrierName),
		DefaultCarrierIsPortalEnabled:      nullBoolPtr(row.DefaultCarrierIsPortalEnabled),
		DefaultCarrierCreatedAt:            nullTimePtr(row.DefaultCarrierCreatedAt),
		DefaultCarrierUpdatedAt:            nullTimePtr(row.DefaultCarrierUpdatedAt),
		DefaultServiceLevelID:              nullStringPtr(row.DefaultCarrierOptionID),
		DefaultServiceLevelName:            nullStringPtr(row.DefaultCarrierOptionName),
		DefaultServiceLevelToken:           nullStringPtr(row.DefaultCarrierOptionServiceLevelToken),
		DefaultServiceLevelIsPortalEnabled: nullBoolPtr(row.DefaultCarrierOptionIsPortalEnabled),
		DefaultServiceLevelCreatedAt:       nullTimePtr(row.DefaultCarrierOptionCreatedAt),
		DefaultServiceLevelUpdatedAt:       nullTimePtr(row.DefaultCarrierOptionUpdatedAt),
		DefaultPaymentTermID:               nullStringPtr(row.PaymentTermID),
		DefaultPaymentTermName:             nullStringPtr(row.PaymentTermName),
		DefaultPaymentTermIsActive:         nullBoolPtr(row.PaymentTermIsActive),
		DefaultPaymentTermCreatedAt:        nullTimePtr(row.PaymentTermCreatedAt),
		DefaultPaymentTermUpdatedAt:        nullTimePtr(row.PaymentTermUpdatedAt),
		DefaultShippingTermID:              nullStringPtr(row.ShippingTermID),
		DefaultShippingTermName:            nullStringPtr(row.ShippingTermName),
		DefaultShippingTermType:            nullShippingTermType(row.ShippingTermIsFreightExempt, row.ShippingTermIsCarrierRate),
		DefaultShippingTermCreatedAt:       nullTimePtr(row.ShippingTermCreatedAt),
		DefaultShippingTermUpdatedAt:       nullTimePtr(row.ShippingTermUpdatedAt),
		DefaultPriorityID:                  nullStringPtr(row.PriorityID),
		DefaultPriorityCode:                nullPriorityCode(row.PriorityCode),
		DefaultPriorityName:                nullStringPtr(row.PriorityName),
		DefaultSalesRepID:                  nullStringPtr(row.DefaultSalesRepID),
		DefaultSalesRepName:                nullStringPtr(row.DefaultSalesRepName),
		DefaultSalesRepStatus:              nullAccountUserStatus(row.DefaultSalesRepStatusCode),
		DefaultSalesRepCreatedAt:           nullTimePtr(row.DefaultSalesRepCreatedAt),
		DefaultSalesRepUpdatedAt:           nullTimePtr(row.DefaultSalesRepUpdatedAt),
		BillToAddressID:                    nullStringPtr(row.DefaultBillingAddressID),
		ShipToAddressID:                    nullStringPtr(row.DefaultShippingAddressID),
		BillToAddress:                      billToAddress,
		ShipToAddress:                      shipToAddress,
		TypeGroupID:                        nullStringPtr(row.TypeGroupID),
		TypeGroupName:                      nullStringPtr(row.TypeGroupName),
		TypeGroupCommissionPolicy:          nullCommissionPolicy(row.TypeGroupCommissionStatusCode),
		TypeGroupFreightPolicy:             nullFreightPolicy(row.TypeGroupFreightStatusCode),
		TypeGroupType:                      nullAccountGroupType(row.TypeGroupTypeCode),
		TypeGroupCreatedAt:                 nullTimePtr(row.TypeGroupCreatedAt),
		TypeGroupUpdatedAt:                 nullTimePtr(row.TypeGroupUpdatedAt),
		ParentAccountID:                    nullStringPtr(row.ParentAccountID),
		ParentAccountName:                  nullStringPtr(row.ParentAccountName),
		ParentAccountNumber:                nullStringPtr(row.ParentAccountNumber),
		ParentAccountCreatedAt:             nullTimePtr(row.ParentAccountCreatedAt),
		ParentAccountUpdatedAt:             nullTimePtr(row.ParentAccountUpdatedAt),
		CreatedAt:                          row.CreatedAt,
		UpdatedAt:                          row.UpdatedAt,
	}
}

func mapListCustomerBackwardRow(row sqlc.ListCustomersBackwardRow) *domain.Customer {
	var billToAddress *domain.CustomerAddress
	if row.DefaultBillingAddressID.Valid {
		billToAddress = buildCustomerAddress(
			row.DefaultBillingAddressID.String,
			row.DefaultBillingAddressName.String,
			row.DefaultBillingAddressPhone,
			row.DefaultBillingAddressEmail,
			row.DefaultBillingIsDropShip.Bool,
			row.DefaultBillingGeolocationID,
			row.DefaultBillingStreetLine1,
			row.DefaultBillingStreetLine2,
			row.DefaultBillingLocality,
			row.DefaultBillingState,
			row.DefaultBillingPostalCode,
			row.DefaultBillingCountry,
			row.DefaultBillingAddressCreatedAt.Time,
			row.DefaultBillingAddressUpdatedAt.Time,
		)
	}
	var shipToAddress *domain.CustomerAddress
	if row.DefaultShippingAddressID.Valid {
		shipToAddress = buildCustomerAddress(
			row.DefaultShippingAddressID.String,
			row.DefaultShippingAddressName.String,
			row.DefaultShippingAddressPhone,
			row.DefaultShippingAddressEmail,
			row.DefaultShippingIsDropShip.Bool,
			row.DefaultShippingGeolocationID,
			row.DefaultShippingStreetLine1,
			row.DefaultShippingStreetLine2,
			row.DefaultShippingLocality,
			row.DefaultShippingState,
			row.DefaultShippingPostalCode,
			row.DefaultShippingCountry,
			row.DefaultShippingAddressCreatedAt.Time,
			row.DefaultShippingAddressUpdatedAt.Time,
		)
	}
	return &domain.Customer{
		ID:                                 row.AccountID,
		Name:                               row.AccountName,
		Number:                             row.ExternalNumber,
		Status:                             constants.AccountStatusCode(row.Status.String),
		IsEdiEnabled:                       row.IsEdiEnabled,
		IsParentAccount:                    row.IsParentAccount,
		CommissionPolicy:                   constants.CommissionPolicy(row.CommissionStatusCode.String),
		FreightPolicy:                      constants.FreightPolicy(row.FreightStatusCode.String),
		Note:                               nullStringPtr(row.Notes),
		Email:                              nullStringPtr(row.Email),
		Phone:                              nullStringPtr(row.PhoneNumber),
		URL:                                nullStringPtr(row.WebsiteUrl),
		CarrierBillingType:                 nullCarrierBillingType(row.CarrierBillingType),
		CarrierBillingAccount:              nullStringPtr(row.CarrierBillingAccount),
		CreditLimitID:                      nullStringPtr(row.CreditLimitID),
		CreditLimitValue:                   nullStringPtr(row.CreditLimitValue),
		CreditLimitUnitID:                  nullStringPtr(row.CreditLimitUnitID),
		CreditLimitUnitAbbreviation:        nullStringPtr(row.CreditLimitUnitAbbreviation),
		CreditLimitUnitName:                nullStringPtr(row.CreditLimitUnitName),
		CreditLimitUnitType:                nullStringPtr(row.CreditLimitUnitType),
		DefaultCarrierID:                   nullStringPtr(row.DefaultCarrierID),
		DefaultCarrierName:                 nullStringPtr(row.DefaultCarrierName),
		DefaultCarrierIsPortalEnabled:      nullBoolPtr(row.DefaultCarrierIsPortalEnabled),
		DefaultCarrierCreatedAt:            nullTimePtr(row.DefaultCarrierCreatedAt),
		DefaultCarrierUpdatedAt:            nullTimePtr(row.DefaultCarrierUpdatedAt),
		DefaultServiceLevelID:              nullStringPtr(row.DefaultCarrierOptionID),
		DefaultServiceLevelName:            nullStringPtr(row.DefaultCarrierOptionName),
		DefaultServiceLevelToken:           nullStringPtr(row.DefaultCarrierOptionServiceLevelToken),
		DefaultServiceLevelIsPortalEnabled: nullBoolPtr(row.DefaultCarrierOptionIsPortalEnabled),
		DefaultServiceLevelCreatedAt:       nullTimePtr(row.DefaultCarrierOptionCreatedAt),
		DefaultServiceLevelUpdatedAt:       nullTimePtr(row.DefaultCarrierOptionUpdatedAt),
		DefaultPaymentTermID:               nullStringPtr(row.PaymentTermID),
		DefaultPaymentTermName:             nullStringPtr(row.PaymentTermName),
		DefaultPaymentTermIsActive:         nullBoolPtr(row.PaymentTermIsActive),
		DefaultPaymentTermCreatedAt:        nullTimePtr(row.PaymentTermCreatedAt),
		DefaultPaymentTermUpdatedAt:        nullTimePtr(row.PaymentTermUpdatedAt),
		DefaultShippingTermID:              nullStringPtr(row.ShippingTermID),
		DefaultShippingTermName:            nullStringPtr(row.ShippingTermName),
		DefaultShippingTermType:            nullShippingTermType(row.ShippingTermIsFreightExempt, row.ShippingTermIsCarrierRate),
		DefaultShippingTermCreatedAt:       nullTimePtr(row.ShippingTermCreatedAt),
		DefaultShippingTermUpdatedAt:       nullTimePtr(row.ShippingTermUpdatedAt),
		DefaultPriorityID:                  nullStringPtr(row.PriorityID),
		DefaultPriorityCode:                nullPriorityCode(row.PriorityCode),
		DefaultPriorityName:                nullStringPtr(row.PriorityName),
		DefaultSalesRepID:                  nullStringPtr(row.DefaultSalesRepID),
		DefaultSalesRepName:                nullStringPtr(row.DefaultSalesRepName),
		DefaultSalesRepStatus:              nullAccountUserStatus(row.DefaultSalesRepStatusCode),
		DefaultSalesRepCreatedAt:           nullTimePtr(row.DefaultSalesRepCreatedAt),
		DefaultSalesRepUpdatedAt:           nullTimePtr(row.DefaultSalesRepUpdatedAt),
		BillToAddressID:                    nullStringPtr(row.DefaultBillingAddressID),
		ShipToAddressID:                    nullStringPtr(row.DefaultShippingAddressID),
		BillToAddress:                      billToAddress,
		ShipToAddress:                      shipToAddress,
		TypeGroupID:                        nullStringPtr(row.TypeGroupID),
		TypeGroupName:                      nullStringPtr(row.TypeGroupName),
		TypeGroupCommissionPolicy:          nullCommissionPolicy(row.TypeGroupCommissionStatusCode),
		TypeGroupFreightPolicy:             nullFreightPolicy(row.TypeGroupFreightStatusCode),
		TypeGroupType:                      nullAccountGroupType(row.TypeGroupTypeCode),
		TypeGroupCreatedAt:                 nullTimePtr(row.TypeGroupCreatedAt),
		TypeGroupUpdatedAt:                 nullTimePtr(row.TypeGroupUpdatedAt),
		ParentAccountID:                    nullStringPtr(row.ParentAccountID),
		ParentAccountName:                  nullStringPtr(row.ParentAccountName),
		ParentAccountNumber:                nullStringPtr(row.ParentAccountNumber),
		ParentAccountCreatedAt:             nullTimePtr(row.ParentAccountCreatedAt),
		ParentAccountUpdatedAt:             nullTimePtr(row.ParentAccountUpdatedAt),
		CreatedAt:                          row.CreatedAt,
		UpdatedAt:                          row.UpdatedAt,
	}
}

func timeToNullTime(t *time.Time) gosql.NullTime {
	if t == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *t, Valid: true}
}

func stringToNullString(s *string) gosql.NullString {
	if s == nil || *s == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func ensureNullStringSlice(ids []string) []gosql.NullString {
	if len(ids) == 0 {
		return []gosql.NullString{{}}
	}
	result := make([]gosql.NullString, len(ids))
	for i, id := range ids {
		result[i] = gosql.NullString{String: id, Valid: true}
	}
	return result
}

func ensureStringSlice(ids []string) []string {
	if len(ids) == 0 {
		return []string{""}
	}
	return ids
}

func (r *customerRepoImpl) List(ctx context.Context, params domain.ListCustomersParams) (*domain.ListCustomersResult, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.list")
	defer span.End()

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + *params.Query + "%", Valid: true}
	}

	// Build filter flags and slices.
	includeCustomerGroupFilter := len(params.CustomerGroupIDs) > 0
	includePricingGroupFilter := len(params.PricingGroupIDs) > 0
	includeSalesRepFilter := len(params.SalesRepIDs) > 0
	includeStatusFilter := len(params.StatusCodes) > 0
	includeShippingTermFilter := len(params.ShippingTermIDs) > 0
	includePaymentTermFilter := len(params.PaymentTermIDs) > 0
	includeCommissionPolicyFilter := len(params.CommissionPolicyCodes) > 0
	includeFreightPolicyFilter := len(params.FreightPolicyCodes) > 0
	includeCarrierFilter := len(params.CarrierIDs) > 0
	includeServiceLevelFilter := len(params.ServiceLevelIDs) > 0
	includeParentAccountFilter := params.IsParentAccount != nil
	parentAccountFilterValue := false
	if params.IsParentAccount != nil {
		parentAccountFilterValue = *params.IsParentAccount
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListCustomersBackward(ctx, sqlc.ListCustomersBackwardParams{
				OwnerAccountID:                params.AccountID,
				SearchQuery:                   searchQuery,
				IncludeCustomerGroupFilter:    includeCustomerGroupFilter,
				CustomerGroupIds:              ensureNullStringSlice(params.CustomerGroupIDs),
				IncludePricingGroupFilter:     includePricingGroupFilter,
				PricingGroupIds:               ensureStringSlice(params.PricingGroupIDs),
				IncludeSalesRepFilter:         includeSalesRepFilter,
				SalesRepIds:                   ensureNullStringSlice(params.SalesRepIDs),
				IncludeStatusFilter:           includeStatusFilter,
				StatusCodes:                   ensureNullStringSlice(params.StatusCodes),
				IncludeShippingTermFilter:     includeShippingTermFilter,
				ShippingTermIds:               ensureNullStringSlice(params.ShippingTermIDs),
				IncludePaymentTermFilter:      includePaymentTermFilter,
				PaymentTermIds:                ensureNullStringSlice(params.PaymentTermIDs),
				IncludeCommissionStatusFilter: includeCommissionPolicyFilter,
				CommissionStatusCodes:         ensureNullStringSlice(params.CommissionPolicyCodes),
				IncludeFreightStatusFilter:    includeFreightPolicyFilter,
				FreightStatusCodes:            ensureNullStringSlice(params.FreightPolicyCodes),
				IncludeCarrierFilter:          includeCarrierFilter,
				CarrierIds:                    ensureNullStringSlice(params.CarrierIDs),
				IncludeCarrierOptionFilter:    includeServiceLevelFilter,
				CarrierOptionIds:              ensureNullStringSlice(params.ServiceLevelIDs),
				IncludeParentAccountFilter:    includeParentAccountFilter,
				ParentAccountFilterValue:      parentAccountFilterValue,
				City:                          stringToNullString(params.City),
				State:                         stringToNullString(params.State),
				PostalCode:                    stringToNullString(params.PostalCode),
				StartDate:                     timeToNullTime(params.StartDate),
				EndDate:                       timeToNullTime(params.EndDate),
				CursorCreatedAt:               cur.OccurredAt,
				CursorID:                      cur.ID,
				Limit:                         params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			relationIDs := make([]string, len(rows))
			items := make([]*domain.Customer, len(rows))
			for i, row := range rows {
				items[i] = mapListCustomerBackwardRow(row)
				relationIDs[i] = row.RelationID
			}
			if apiErr := r.stitchListCustomerIncludes(ctx, params.AccountID, items, relationIDs, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, customerCreatedAt, customerID)
			return &domain.ListCustomersResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListCustomersForward(ctx, sqlc.ListCustomersForwardParams{
			OwnerAccountID:                params.AccountID,
			SearchQuery:                   searchQuery,
			IncludeCustomerGroupFilter:    includeCustomerGroupFilter,
			CustomerGroupIds:              ensureNullStringSlice(params.CustomerGroupIDs),
			IncludePricingGroupFilter:     includePricingGroupFilter,
			PricingGroupIds:               ensureStringSlice(params.PricingGroupIDs),
			IncludeSalesRepFilter:         includeSalesRepFilter,
			SalesRepIds:                   ensureNullStringSlice(params.SalesRepIDs),
			IncludeStatusFilter:           includeStatusFilter,
			StatusCodes:                   ensureNullStringSlice(params.StatusCodes),
			IncludeShippingTermFilter:     includeShippingTermFilter,
			ShippingTermIds:               ensureNullStringSlice(params.ShippingTermIDs),
			IncludePaymentTermFilter:      includePaymentTermFilter,
			PaymentTermIds:                ensureNullStringSlice(params.PaymentTermIDs),
			IncludeCommissionStatusFilter: includeCommissionPolicyFilter,
			CommissionStatusCodes:         ensureNullStringSlice(params.CommissionPolicyCodes),
			IncludeFreightStatusFilter:    includeFreightPolicyFilter,
			FreightStatusCodes:            ensureNullStringSlice(params.FreightPolicyCodes),
			IncludeCarrierFilter:          includeCarrierFilter,
			CarrierIds:                    ensureNullStringSlice(params.CarrierIDs),
			IncludeCarrierOptionFilter:    includeServiceLevelFilter,
			CarrierOptionIds:              ensureNullStringSlice(params.ServiceLevelIDs),
			IncludeParentAccountFilter:    includeParentAccountFilter,
			ParentAccountFilterValue:      parentAccountFilterValue,
			City:                          stringToNullString(params.City),
			State:                         stringToNullString(params.State),
			PostalCode:                    stringToNullString(params.PostalCode),
			StartDate:                     timeToNullTime(params.StartDate),
			EndDate:                       timeToNullTime(params.EndDate),
			CursorCreatedAt:               gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                      gosql.NullString{String: cur.ID, Valid: true},
			Limit:                         params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		relationIDs := make([]string, len(rows))
		items := make([]*domain.Customer, len(rows))
		for i, row := range rows {
			items[i] = mapListCustomerForwardRow(row)
			relationIDs[i] = row.RelationID
		}
		if apiErr := r.stitchListCustomerIncludes(ctx, params.AccountID, items, relationIDs, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, customerCreatedAt, customerID)
		return &domain.ListCustomersResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListCustomersForward(ctx, sqlc.ListCustomersForwardParams{
		OwnerAccountID:                params.AccountID,
		SearchQuery:                   searchQuery,
		IncludeCustomerGroupFilter:    includeCustomerGroupFilter,
		CustomerGroupIds:              ensureNullStringSlice(params.CustomerGroupIDs),
		IncludePricingGroupFilter:     includePricingGroupFilter,
		PricingGroupIds:               ensureStringSlice(params.PricingGroupIDs),
		IncludeSalesRepFilter:         includeSalesRepFilter,
		SalesRepIds:                   ensureNullStringSlice(params.SalesRepIDs),
		IncludeStatusFilter:           includeStatusFilter,
		StatusCodes:                   ensureNullStringSlice(params.StatusCodes),
		IncludeShippingTermFilter:     includeShippingTermFilter,
		ShippingTermIds:               ensureNullStringSlice(params.ShippingTermIDs),
		IncludePaymentTermFilter:      includePaymentTermFilter,
		PaymentTermIds:                ensureNullStringSlice(params.PaymentTermIDs),
		IncludeCommissionStatusFilter: includeCommissionPolicyFilter,
		CommissionStatusCodes:         ensureNullStringSlice(params.CommissionPolicyCodes),
		IncludeFreightStatusFilter:    includeFreightPolicyFilter,
		FreightStatusCodes:            ensureNullStringSlice(params.FreightPolicyCodes),
		IncludeCarrierFilter:          includeCarrierFilter,
		CarrierIds:                    ensureNullStringSlice(params.CarrierIDs),
		IncludeCarrierOptionFilter:    includeServiceLevelFilter,
		CarrierOptionIds:              ensureNullStringSlice(params.ServiceLevelIDs),
		IncludeParentAccountFilter:    includeParentAccountFilter,
		ParentAccountFilterValue:      parentAccountFilterValue,
		City:                          stringToNullString(params.City),
		State:                         stringToNullString(params.State),
		PostalCode:                    stringToNullString(params.PostalCode),
		StartDate:                     timeToNullTime(params.StartDate),
		EndDate:                       timeToNullTime(params.EndDate),
		Limit:                         params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	relationIDs := make([]string, len(rows))
	items := make([]*domain.Customer, len(rows))
	for i, row := range rows {
		items[i] = mapListCustomerForwardRow(row)
		relationIDs[i] = row.RelationID
	}
	if apiErr := r.stitchListCustomerIncludes(ctx, params.AccountID, items, relationIDs, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, customerCreatedAt, customerID)
	return &domain.ListCustomersResult{Items: result, PageInfo: pageInfo}, nil
}

// stitchListCustomerIncludes fans out per-relation data (price groups, notification preferences)
// into the list items when the corresponding include keys were requested. Skipping them keeps the
// list path from paying for unused round-trips — the apiresource layer will collapse those fields
// to null for clients that didn't ask for them.
func (r *customerRepoImpl) stitchListCustomerIncludes(ctx context.Context, ownerAccountID string, items []*domain.Customer, relationIDs []string, includes []string) *apierror.APIError {
	if len(items) == 0 {
		return nil
	}
	wantPriceGroups := false
	wantNotifPrefs := false
	wantChildAccounts := false
	for _, inc := range includes {
		switch inc {
		case "price_groups":
			wantPriceGroups = true
		case "notification_preferences":
			wantNotifPrefs = true
		case "child_accounts":
			wantChildAccounts = true
		}
	}
	if !wantPriceGroups && !wantNotifPrefs && !wantChildAccounts {
		return nil
	}
	byRelationID := make(map[string]*domain.Customer, len(items))
	for i, rid := range relationIDs {
		byRelationID[rid] = items[i]
	}
	if wantPriceGroups {
		rows, err := r.queries.ListCustomersPriceGroups(ctx, relationIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, row := range rows {
			c, ok := byRelationID[row.AccountRelationID]
			if !ok {
				continue
			}
			c.PriceGroups = append(c.PriceGroups, domain.CustomerAccountGroup{
				ID:               row.ID,
				Name:             row.Name,
				CommissionPolicy: constants.CommissionPolicy(row.CommissionStatusCode),
				FreightPolicy:    constants.FreightPolicy(row.FreightStatusCode),
				Type:             constants.AccountGroupType(row.AccountGroupTypeCode),
				CreatedAt:        row.CreatedAt,
				UpdatedAt:        row.UpdatedAt,
			})
		}
	}
	if wantNotifPrefs {
		rows, err := r.queries.ListCustomersNotificationPreferences(ctx, relationIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		for _, row := range rows {
			if row.NotificationTypeCode != "invoice" {
				continue
			}
			if c, ok := byRelationID[row.AccountRelationID]; ok {
				c.AcceptsInvoiceEmails = true
			}
		}
	}
	if wantChildAccounts {
		byParent, apiErr := r.fetchChildAccountsByRelationIDs(ctx, ownerAccountID, relationIDs)
		if apiErr != nil {
			return apiErr
		}
		for rid, children := range byParent {
			if c, ok := byRelationID[rid]; ok {
				c.ChildAccounts = children
			}
		}
	}
	return nil
}

func (r *customerRepoImpl) Get(ctx context.Context, ownerAccountID, customerAccountID string, incs []string) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get")
	defer span.End()

	row, err := r.queries.GetCustomer(ctx, sqlc.GetCustomerParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var priceGroups []domain.CustomerAccountGroup
	if slices.Contains(incs, "price_groups") {
		priceGroupRows, err := r.queries.GetCustomerPriceGroups(ctx, row.RelationID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		priceGroups = make([]domain.CustomerAccountGroup, len(priceGroupRows))
		for i, pg := range priceGroupRows {
			priceGroups[i] = domain.CustomerAccountGroup{
				ID:               pg.ID,
				Name:             pg.Name,
				CommissionPolicy: constants.CommissionPolicy(pg.CommissionStatusCode),
				FreightPolicy:    constants.FreightPolicy(pg.FreightStatusCode),
				Type:             constants.AccountGroupType(pg.AccountGroupTypeCode),
				CreatedAt:        pg.CreatedAt,
				UpdatedAt:        pg.UpdatedAt,
			}
		}
	}

	acceptsInvoiceEmails := false
	if slices.Contains(incs, "notification_preferences") {
		notifRows, err := r.queries.GetCustomerNotificationPreferences(ctx, row.RelationID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, n := range notifRows {
			if n.NotificationTypeCode == "invoice" {
				acceptsInvoiceEmails = true
				break
			}
		}
	}

	var billToAddress *domain.CustomerAddress
	if slices.Contains(incs, "bill_to_address") && row.DefaultBillingAddressID.Valid {
		billToAddress = buildCustomerAddress(
			row.DefaultBillingAddressID.String,
			row.DefaultBillingAddressName.String,
			row.DefaultBillingAddressPhone,
			row.DefaultBillingAddressEmail,
			row.DefaultBillingIsDropShip.Bool,
			row.DefaultBillingGeolocationID,
			row.DefaultBillingStreetLine1,
			row.DefaultBillingStreetLine2,
			row.DefaultBillingLocality,
			row.DefaultBillingState,
			row.DefaultBillingPostalCode,
			row.DefaultBillingCountry,
			row.DefaultBillingAddressCreatedAt.Time,
			row.DefaultBillingAddressUpdatedAt.Time,
		)
	}

	var shipToAddress *domain.CustomerAddress
	if slices.Contains(incs, "ship_to_address") && row.DefaultShippingAddressID.Valid {
		shipToAddress = buildCustomerAddress(
			row.DefaultShippingAddressID.String,
			row.DefaultShippingAddressName.String,
			row.DefaultShippingAddressPhone,
			row.DefaultShippingAddressEmail,
			row.DefaultShippingIsDropShip.Bool,
			row.DefaultShippingGeolocationID,
			row.DefaultShippingStreetLine1,
			row.DefaultShippingStreetLine2,
			row.DefaultShippingLocality,
			row.DefaultShippingState,
			row.DefaultShippingPostalCode,
			row.DefaultShippingCountry,
			row.DefaultShippingAddressCreatedAt.Time,
			row.DefaultShippingAddressUpdatedAt.Time,
		)
	}

	customer := &domain.Customer{
		ID:                                 row.AccountID,
		Name:                               row.AccountName,
		Number:                             row.ExternalNumber,
		Status:                             constants.AccountStatusCode(row.Status.String),
		IsEdiEnabled:                       row.IsEdiEnabled,
		IsParentAccount:                    row.IsParentAccount,
		CommissionPolicy:                   constants.CommissionPolicy(row.CommissionStatusCode.String),
		FreightPolicy:                      constants.FreightPolicy(row.FreightStatusCode.String),
		Note:                               nullStringPtr(row.Notes),
		Email:                              nullStringPtr(row.Email),
		Phone:                              nullStringPtr(row.PhoneNumber),
		URL:                                nullStringPtr(row.WebsiteUrl),
		CarrierBillingType:                 nullCarrierBillingType(row.CarrierBillingType),
		CarrierBillingAccount:              nullStringPtr(row.CarrierBillingAccount),
		CreditLimitID:                      nullStringPtr(row.CreditLimitID),
		CreditLimitValue:                   nullStringPtr(row.CreditLimitValue),
		CreditLimitUnitID:                  nullStringPtr(row.CreditLimitUnitID),
		CreditLimitUnitAbbreviation:        nullStringPtr(row.CreditLimitUnitAbbreviation),
		CreditLimitUnitName:                nullStringPtr(row.CreditLimitUnitName),
		CreditLimitUnitType:                nullStringPtr(row.CreditLimitUnitType),
		AcceptsInvoiceEmails:               acceptsInvoiceEmails,
		DefaultCarrierID:                   nullStringPtr(row.DefaultCarrierID),
		DefaultCarrierName:                 nullStringPtr(row.DefaultCarrierName),
		DefaultCarrierIsPortalEnabled:      nullBoolPtr(row.DefaultCarrierIsPortalEnabled),
		DefaultCarrierCreatedAt:            nullTimePtr(row.DefaultCarrierCreatedAt),
		DefaultCarrierUpdatedAt:            nullTimePtr(row.DefaultCarrierUpdatedAt),
		DefaultServiceLevelID:              nullStringPtr(row.DefaultCarrierOptionID),
		DefaultServiceLevelName:            nullStringPtr(row.DefaultCarrierOptionName),
		DefaultServiceLevelToken:           nullStringPtr(row.DefaultCarrierOptionServiceLevelToken),
		DefaultServiceLevelIsPortalEnabled: nullBoolPtr(row.DefaultCarrierOptionIsPortalEnabled),
		DefaultServiceLevelCreatedAt:       nullTimePtr(row.DefaultCarrierOptionCreatedAt),
		DefaultServiceLevelUpdatedAt:       nullTimePtr(row.DefaultCarrierOptionUpdatedAt),
		DefaultPaymentTermID:               nullStringPtr(row.PaymentTermID),
		DefaultPaymentTermName:             nullStringPtr(row.PaymentTermName),
		DefaultPaymentTermIsActive:         nullBoolPtr(row.PaymentTermIsActive),
		DefaultPaymentTermCreatedAt:        nullTimePtr(row.PaymentTermCreatedAt),
		DefaultPaymentTermUpdatedAt:        nullTimePtr(row.PaymentTermUpdatedAt),
		DefaultShippingTermID:              nullStringPtr(row.ShippingTermID),
		DefaultShippingTermName:            nullStringPtr(row.ShippingTermName),
		DefaultShippingTermType:            nullShippingTermType(row.ShippingTermIsFreightExempt, row.ShippingTermIsCarrierRate),
		DefaultShippingTermCreatedAt:       nullTimePtr(row.ShippingTermCreatedAt),
		DefaultShippingTermUpdatedAt:       nullTimePtr(row.ShippingTermUpdatedAt),
		DefaultPriorityID:                  nullStringPtr(row.PriorityID),
		DefaultPriorityCode:                nullPriorityCode(row.PriorityCode),
		DefaultPriorityName:                nullStringPtr(row.PriorityName),
		DefaultSalesRepID:                  nullStringPtr(row.DefaultSalesRepID),
		DefaultSalesRepName:                nullStringPtr(row.DefaultSalesRepName),
		DefaultSalesRepStatus:              nullAccountUserStatus(row.DefaultSalesRepStatusCode),
		DefaultSalesRepCreatedAt:           nullTimePtr(row.DefaultSalesRepCreatedAt),
		DefaultSalesRepUpdatedAt:           nullTimePtr(row.DefaultSalesRepUpdatedAt),
		BillToAddressID:                    nullStringPtr(row.DefaultBillingAddressID),
		ShipToAddressID:                    nullStringPtr(row.DefaultShippingAddressID),
		BillToAddress:                      billToAddress,
		ShipToAddress:                      shipToAddress,
		TypeGroupID:                        nullStringPtr(row.TypeGroupID),
		TypeGroupName:                      nullStringPtr(row.TypeGroupName),
		TypeGroupCommissionPolicy:          nullCommissionPolicy(row.TypeGroupCommissionStatusCode),
		TypeGroupFreightPolicy:             nullFreightPolicy(row.TypeGroupFreightStatusCode),
		TypeGroupType:                      nullAccountGroupType(row.TypeGroupTypeCode),
		TypeGroupCreatedAt:                 nullTimePtr(row.TypeGroupCreatedAt),
		TypeGroupUpdatedAt:                 nullTimePtr(row.TypeGroupUpdatedAt),
		PriceGroups:                        priceGroups,
		ParentAccountID:                    nullStringPtr(row.ParentAccountID),
		ParentAccountName:                  nullStringPtr(row.ParentAccountName),
		ParentAccountNumber:                nullStringPtr(row.ParentAccountNumber),
		ParentAccountCreatedAt:             nullTimePtr(row.ParentAccountCreatedAt),
		ParentAccountUpdatedAt:             nullTimePtr(row.ParentAccountUpdatedAt),
		CreatedAt:                          row.CreatedAt,
		UpdatedAt:                          row.UpdatedAt,
	}

	if wantsInclude(incs, "child_accounts") {
		childrenByRelation, apiErr := r.fetchChildAccountsByRelationIDs(ctx, ownerAccountID, []string{row.RelationID})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		customer.ChildAccounts = childrenByRelation[row.RelationID]
	}

	return customer, nil
}

// wantsInclude returns true if the include key is present in the include list.
func wantsInclude(incs []string, key string) bool {
	return slices.Contains(incs, key)
}

// fetchChildAccountsByRelationIDs batches a single SQL query to fetch direct
// children for every parent relation ID in parentRelationIDs, grouped by
// parent relation ID.
func (r *customerRepoImpl) fetchChildAccountsByRelationIDs(ctx context.Context, ownerAccountID string, parentRelationIDs []string) (map[string][]domain.CustomerChildAccount, *apierror.APIError) {
	if len(parentRelationIDs) == 0 {
		return nil, nil
	}
	nullIDs := make([]gosql.NullString, len(parentRelationIDs))
	for i, id := range parentRelationIDs {
		nullIDs[i] = gosql.NullString{String: id, Valid: true}
	}
	rows, err := r.queries.ListChildAccountsByParentRelationIDs(ctx, sqlc.ListChildAccountsByParentRelationIDsParams{
		OwnerAccountID:    ownerAccountID,
		ParentRelationIds: nullIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	byParent := make(map[string][]domain.CustomerChildAccount, len(parentRelationIDs))
	for _, row := range rows {
		if !row.ParentRelationID.Valid {
			continue
		}
		byParent[row.ParentRelationID.String] = append(byParent[row.ParentRelationID.String], domain.CustomerChildAccount{
			ID:        row.AccountID,
			Name:      row.AccountName,
			Number:    row.ExternalNumber,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return byParent, nil
}

func buildCustomerAddress(
	id string,
	name string,
	phone gosql.NullString,
	email gosql.NullString,
	isDropShip bool,
	geolocationID gosql.NullString,
	streetLine1 gosql.NullString,
	streetLine2 gosql.NullString,
	locality gosql.NullString,
	state gosql.NullString,
	postalCode gosql.NullString,
	country gosql.NullString,
	createdAt time.Time,
	updatedAt time.Time,
) *domain.CustomerAddress {
	addr := &domain.CustomerAddress{
		ID:         id,
		Name:       name,
		Phone:      nullStringPtr(phone),
		Email:      nullStringPtr(email),
		IsDropShip: isDropShip,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	if geolocationID.Valid {
		addr.Geolocation = &domain.CustomerGeolocation{
			ID:          geolocationID.String,
			StreetLine1: nullStringPtr(streetLine1),
			StreetLine2: nullStringPtr(streetLine2),
			Locality:    nullStringPtr(locality),
			State:       nullStringPtr(state),
			PostalCode:  nullStringPtr(postalCode),
			Country:     country.String,
		}
	}

	return addr
}

func (r *customerRepoImpl) Create(ctx context.Context, accountID, relationID, brandingID string, params domain.CreateCustomerParams, customerNumber string) (*domain.Customer, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.create")
	defer span.End()

	// Create the account record.
	err := r.queries.InsertCustomerAccount(ctx, sqlc.InsertCustomerAccountParams{
		ID:                       accountID,
		Name:                     params.Name,
		DefaultBillingAddressID:  toNullString(params.BillToAddressID),
		DefaultShippingAddressID: toNullString(params.ShipToAddressID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Create account branding.
	err = r.queries.InsertCustomerAccountBranding(ctx, sqlc.InsertCustomerAccountBrandingParams{
		ID:             brandingID,
		OwnerAccountID: accountID,
		SupportEmail:   toNullString(params.Email),
		PhoneNumber:    toNullString(params.Phone),
		WebsiteUrl:     toNullString(params.URL),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Determine commission and freight status codes.
	commissionStatus := string(constants.CommissionPolicyApplied)
	if params.CommissionPolicy != nil {
		commissionStatus = string(*params.CommissionPolicy)
	}
	freightStatus := string(constants.FreightPolicyBilled)
	if params.FreightPolicy != nil {
		freightStatus = string(*params.FreightPolicy)
	}

	// Determine status code.
	statusCode := "normal"
	if params.StatusCode != nil {
		statusCode = *params.StatusCode
	}

	isEdi := false
	if params.IsEdiEnabled != nil {
		isEdi = *params.IsEdiEnabled
	}

	// Default priority code (column is NOT NULL).
	priorityCode := string(constants.PriorityCodeNormal)
	if params.DefaultPriorityCode != nil && *params.DefaultPriorityCode != "" {
		priorityCode = *params.DefaultPriorityCode
	}

	// Create the account relation.
	err = r.queries.InsertCustomerRelation(ctx, sqlc.InsertCustomerRelationParams{
		ID:                       relationID,
		OwnerAccountID:           params.OwnerAccountID,
		CounterpartyAccountID:    accountID,
		Alias:                    gosql.NullString{String: params.Name, Valid: true},
		ExternalNumber:           customerNumber,
		Notes:                    toNullString(params.Note),
		IsEdiEnabled:             isEdi,
		CommissionStatusCode:     gosql.NullString{String: commissionStatus, Valid: true},
		FreightStatusCode:        gosql.NullString{String: freightStatus, Valid: true},
		DefaultCarrierID:         toNullString(params.DefaultCarrierID),
		DefaultCarrierOptionID:   toNullString(params.DefaultServiceLevelID),
		DefaultSalesRepID:        toNullString(params.DefaultSalesRepID),
		AccountStatusCode:        gosql.NullString{String: statusCode, Valid: true},
		PaymentTermID:            toNullString(params.DefaultPaymentTermID),
		AccountGroupID:           toNullString(params.CustomerTypeGroupID),
		PriorityCode:             gosql.NullString{String: priorityCode, Valid: true},
		ShippingTermID:           toNullString(params.DefaultShippingTermID),
		CarrierBillingType:       toNullString(params.CarrierBillingType),
		CarrierBillingAccount:    toNullString(params.CarrierBillingAccount),
		DefaultBillingAddressID:  toNullString(params.BillToAddressID),
		DefaultShippingAddressID: toNullString(params.ShipToAddressID),
		CreditLimitID:            toNullString(params.CreditLimitID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.OwnerAccountID, accountID, nil)
}

func (r *customerRepoImpl) Update(ctx context.Context, relationID string, params domain.UpdateCustomerParams) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.update")
	defer span.End()

	// Map commission/freight policy to status codes.
	var commissionStatus gosql.NullString
	if params.CommissionPolicy != nil {
		commissionStatus = gosql.NullString{String: string(*params.CommissionPolicy), Valid: true}
	}

	var freightStatus gosql.NullString
	if params.FreightPolicy != nil {
		freightStatus = gosql.NullString{String: string(*params.FreightPolicy), Valid: true}
	}

	var isEdiEnabled gosql.NullBool
	if params.IsEdiEnabled != nil {
		isEdiEnabled = gosql.NullBool{Bool: *params.IsEdiEnabled, Valid: true}
	}

	err := r.queries.UpdateCustomer(ctx, sqlc.UpdateCustomerParams{
		ID:                       relationID,
		OwnerAccountID:           params.OwnerAccountID,
		Alias:                    stringToNullString(params.Name),
		ExternalNumber:           stringToNullString(params.Number),
		IsEdiEnabled:             isEdiEnabled,
		Notes:                    field.StringToNullString(params.Note),
		CommissionStatusCode:     commissionStatus,
		FreightStatusCode:        freightStatus,
		DefaultCarrierID:         stringToNullString(params.DefaultCarrierID),
		DefaultCarrierOptionID:   field.StringToNullString(params.DefaultServiceLevelID),
		DefaultSalesRepID:        field.StringToNullString(params.DefaultSalesRepID),
		AccountStatusCode:        stringToNullString(params.StatusCode),
		PaymentTermID:            stringToNullString(params.DefaultPaymentTermID),
		AccountGroupID:           stringToNullString(params.CustomerTypeGroupID),
		PriorityCode:             stringToNullString(params.DefaultPriorityCode),
		ShippingTermID:           stringToNullString(params.DefaultShippingTermID),
		CarrierBillingType:       stringToNullString(params.CarrierBillingType),
		CarrierBillingAccount:    field.StringToNullString(params.CarrierBillingAccount),
		DefaultBillingAddressID:  field.StringToNullString(params.BillToAddressID),
		DefaultShippingAddressID: field.StringToNullString(params.ShipToAddressID),
		CreditLimitID:            stringToNullString(params.CreditLimitID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) UpdateName(ctx context.Context, customerAccountID, name string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.update_name")
	defer span.End()

	_, err := r.queries.UpdateAccountName(ctx, sqlc.UpdateAccountNameParams{
		AccountID: customerAccountID,
		Name:      name,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) UpdateBranding(ctx context.Context, customerAccountID string, email, phone, url *string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.update_branding")
	defer span.End()

	brandingID, apiErr := id.GenID(id.AccountBrandingIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err := r.queries.UpsertCustomerBranding(ctx, sqlc.UpsertCustomerBrandingParams{
		ID:           brandingID,
		AccountID:    customerAccountID,
		SupportEmail: stringToNullString(email),
		PhoneNumber:  stringToNullString(phone),
		WebsiteUrl:   stringToNullString(url),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) Delete(ctx context.Context, ownerAccountID, customerAccountID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete")
	defer span.End()

	// Delete account users.
	err := r.queries.DeleteCustomerAccountUsers(ctx, customerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete account addresses.
	err = r.queries.DeleteCustomerAccountAddresses(ctx, customerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete the account relation.
	err = r.queries.DeleteCustomerByAccountID(ctx, sqlc.DeleteCustomerByAccountIDParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) BulkDelete(ctx context.Context, ownerAccountID string, customerIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.bulk_delete")
	defer span.End()

	err := r.queries.BulkDeleteCustomerRelations(ctx, sqlc.BulkDeleteCustomerRelationsParams{
		OwnerAccountID:         ownerAccountID,
		CounterpartyAccountIds: customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) ExistsByNumber(ctx context.Context, ownerAccountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.exists_by_number")
	defer span.End()

	exists, err := r.queries.CustomerExistsByExternalNumber(ctx, sqlc.CustomerExistsByExternalNumberParams{
		OwnerAccountID:        ownerAccountID,
		ExternalNumber:        number,
		ExcludeCounterpartyID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *customerRepoImpl) GetNextCustomerNumber(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_next_number")
	defer span.End()

	raw, err := r.queries.GetNextCustomerNumber(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	switch v := raw.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case []uint8:
		var n int64
		for _, b := range v {
			n = n*10 + int64(b-'0')
		}
		return n, nil
	default:
		return 0, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("unexpected type %T for customer number", raw), "Failed to parse customer number."))
	}
}

func (r *customerRepoImpl) UpdateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string, value string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.update_next_number")
	defer span.End()

	// Parse string to int32 for the sqlc query.
	var intVal int32
	for _, c := range value {
		intVal = intVal*10 + int32(c-'0')
	}

	err := r.queries.UpdateNextCustomerNumber(ctx, sqlc.UpdateNextCustomerNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
		Value:     intVal,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) InsertPriceGroup(ctx context.Context, id, relationID, groupID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.insert_price_group")
	defer span.End()

	err := r.queries.InsertAccountRelationPriceGroup(ctx, sqlc.InsertAccountRelationPriceGroupParams{
		ID:                id,
		AccountRelationID: relationID,
		AccountGroupID:    groupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeletePriceGroups(ctx context.Context, relationID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_price_groups")
	defer span.End()

	err := r.queries.DeleteAccountRelationPriceGroupsByRelationID(ctx, relationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetFrequentlyOrderedProducts(ctx context.Context, ownerAccountID, customerAccountID string) ([]*domain.FrequentlyOrderedProduct, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.frequently_ordered_products")
	defer span.End()

	rows, err := r.queries.GetFrequentlyOrderedProducts(ctx, sqlc.GetFrequentlyOrderedProductsParams{
		OwnerAccountID: ownerAccountID,
		BuyerAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	products := make([]*domain.FrequentlyOrderedProduct, len(rows))
	for i, row := range rows {
		productName := ""
		if row.ProductName.Valid {
			productName = row.ProductName.String
		}
		unitID := row.UnitID
		unitAbbr := row.UnitAbbreviation
		products[i] = &domain.FrequentlyOrderedProduct{
			ItemID:           row.ItemID,
			ProductName:      productName,
			UnitID:           &unitID,
			UnitAbbreviation: &unitAbbr,
			OrderCount:       safeconv.Int64ToInt32(row.OrderCount),
		}
	}

	return products, nil
}

func (r *customerRepoImpl) GetRelationID(ctx context.Context, ownerAccountID, customerAccountID string) (string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_relation_id")
	defer span.End()

	relID, err := r.queries.GetCustomerRelationID(ctx, sqlc.GetCustomerRelationIDParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return relID, nil
}

func (r *customerRepoImpl) MergeOrders(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_orders")
	defer span.End()

	err := r.queries.MergeCustomerOrders(ctx, sqlc.MergeCustomerOrdersParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeInvoices(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_invoices")
	defer span.End()

	err := r.queries.MergeCustomerInvoices(ctx, sqlc.MergeCustomerInvoicesParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeShipments(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_shipments")
	defer span.End()

	err := r.queries.MergeCustomerShipments(ctx, sqlc.MergeCustomerShipmentsParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeDeliveries(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_deliveries")
	defer span.End()

	err := r.queries.MergeCustomerDeliveries(ctx, sqlc.MergeCustomerDeliveriesParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeTransactions(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_transactions")
	defer span.End()

	err := r.queries.MergeCustomerTransactions(ctx, sqlc.MergeCustomerTransactionsParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeAccountPrices(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_account_prices")
	defer span.End()

	err := r.queries.MergeCustomerAccountPrices(ctx, sqlc.MergeCustomerAccountPricesParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeInventoryReceipts(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_inventory_receipts")
	defer span.End()

	err := r.queries.MergeCustomerInventoryReceipts(ctx, sqlc.MergeCustomerInventoryReceiptsParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeReceivingOrders(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_receiving_orders")
	defer span.End()

	err := r.queries.MergeCustomerReceivingOrders(ctx, sqlc.MergeCustomerReceivingOrdersParams{
		TargetAccountID:  targetAccountID,
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) MergeInventoryIssues(ctx context.Context, targetAccountID string, sourceAccountIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.merge_inventory_issues")
	defer span.End()

	err := r.queries.MergeCustomerInventoryIssues(ctx, sqlc.MergeCustomerInventoryIssuesParams{
		TargetAccountID:  targetAccountID,
		SourceAccountIds: sourceAccountIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeleteNotificationPreferences(ctx context.Context, relationIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_notification_preferences")
	defer span.End()

	err := r.queries.DeleteCustomerNotificationPreferences(ctx, relationIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeleteProductLineAccess(ctx context.Context, relationIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_product_line_access")
	defer span.End()

	err := r.queries.DeleteCustomerProductLineAccess(ctx, relationIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetRelationPriceGroupIDs(ctx context.Context, relationID string) ([]string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_relation_price_group_ids")
	defer span.End()

	ids, err := r.queries.GetRelationPriceGroupIDs(ctx, relationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return ids, nil
}

func (r *customerRepoImpl) GetRelationsPriceGroups(ctx context.Context, relationIDs []string) ([]domain.RelationPriceGroup, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_relations_price_groups")
	defer span.End()

	rows, err := r.queries.GetRelationsPriceGroups(ctx, relationIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.RelationPriceGroup, len(rows))
	for i, row := range rows {
		result[i] = domain.RelationPriceGroup{
			ID:             row.ID,
			AccountGroupID: row.AccountGroupID,
		}
	}
	return result, nil
}

func (r *customerRepoImpl) MoveRelationPriceGroups(ctx context.Context, targetRelationID string, ids []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.move_relation_price_groups")
	defer span.End()

	err := r.queries.MoveAccountRelationPriceGroups(ctx, sqlc.MoveAccountRelationPriceGroupsParams{
		TargetRelationID: targetRelationID,
		Ids:              ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeletePriceGroupsByIDs(ctx context.Context, ids []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_price_groups_by_ids")
	defer span.End()

	err := r.queries.DeleteAccountRelationPriceGroupsByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetRelationProductLineIDs(ctx context.Context, relationID string) ([]string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_relation_product_line_ids")
	defer span.End()

	ids, err := r.queries.GetRelationProductLineIDs(ctx, relationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return ids, nil
}

func (r *customerRepoImpl) GetRelationsProductLines(ctx context.Context, relationIDs []string) ([]domain.RelationProductLine, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_relations_product_lines")
	defer span.End()

	rows, err := r.queries.GetRelationsProductLines(ctx, relationIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.RelationProductLine, len(rows))
	for i, row := range rows {
		result[i] = domain.RelationProductLine{
			ID:            row.ID,
			ProductLineID: row.ProductLineID,
		}
	}
	return result, nil
}

func (r *customerRepoImpl) MoveRelationProductLines(ctx context.Context, targetRelationID string, ids []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.move_relation_product_lines")
	defer span.End()

	err := r.queries.MoveAccountRelationProductLines(ctx, sqlc.MoveAccountRelationProductLinesParams{
		TargetRelationID: targetRelationID,
		Ids:              ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeleteProductLinesByIDs(ctx context.Context, ids []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_product_lines_by_ids")
	defer span.End()

	err := r.queries.DeleteAccountRelationProductLinesByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) ReparentChildRelations(ctx context.Context, targetRelationID string, sourceRelationIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.reparent_child_relations")
	defer span.End()

	nullSourceIDs := make([]gosql.NullString, len(sourceRelationIDs))
	for i, id := range sourceRelationIDs {
		nullSourceIDs[i] = gosql.NullString{String: id, Valid: true}
	}

	err := r.queries.ReparentChildRelations(ctx, sqlc.ReparentChildRelationsParams{
		TargetRelationID:  gosql.NullString{String: targetRelationID, Valid: true},
		SourceRelationIds: nullSourceIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetAccountAddressIDs(ctx context.Context, accountID string) ([]string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_account_address_ids")
	defer span.End()

	ids, err := r.queries.GetAccountAddressIDs(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return ids, nil
}

func (r *customerRepoImpl) InsertAccountAddress(ctx context.Context, id, accountID, addressID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.insert_account_address")
	defer span.End()

	err := r.queries.InsertMergeAccountAddress(ctx, sqlc.InsertMergeAccountAddressParams{
		ID:        id,
		AccountID: accountID,
		AddressID: addressID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeleteAccountAddresses(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_account_addresses")
	defer span.End()

	err := r.queries.DeleteAccountAddressesByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetAccountUsers(ctx context.Context, accountID string) ([]domain.AccountUserRef, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_account_users")
	defer span.End()

	rows, err := r.queries.GetAccountUsersByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.AccountUserRef, len(rows))
	for i, row := range rows {
		result[i] = domain.AccountUserRef{
			ID:     row.ID,
			UserID: row.UserID,
		}
	}
	return result, nil
}

func (r *customerRepoImpl) MoveAccountUsers(ctx context.Context, targetAccountID string, ids []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.move_account_users")
	defer span.End()

	err := r.queries.MoveAccountUsers(ctx, sqlc.MoveAccountUsersParams{
		TargetAccountID: targetAccountID,
		Ids:             ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeleteAccountUsers(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_account_users")
	defer span.End()

	err := r.queries.DeleteAccountUsersByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetStripeCustomerID(ctx context.Context, ownerAccountID, customerAccountID string) (*string, *string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_stripe_customer_id")
	defer span.End()

	row, err := r.queries.GetCustomerStripeCustomerID(ctx, sqlc.GetCustomerStripeCustomerIDParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	return nullStringPtr(row.StripeCustomerID), nullStringPtr(row.StripeEmail), nil
}

func (r *customerRepoImpl) SetStripeCustomerID(ctx context.Context, ownerAccountID, customerAccountID, stripeCustomerID, stripeEmail string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.set_stripe_customer_id")
	defer span.End()

	err := r.queries.SetCustomerStripeCustomerID(ctx, sqlc.SetCustomerStripeCustomerIDParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: customerAccountID,
		StripeCustomerID:      gosql.NullString{String: stripeCustomerID, Valid: true},
		StripeEmail:           gosql.NullString{String: stripeEmail, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) GetCustomerEmail(ctx context.Context, customerAccountID string) (*string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_customer_email")
	defer span.End()

	email, err := r.queries.GetCustomerEmail(ctx, customerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return nullStringPtr(email), nil
}

func (r *customerRepoImpl) InsertCreditLimitQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.insert_credit_limit_quantity")
	defer span.End()

	err := r.queries.InsertCustomerCreditLimitQuantity(ctx, sqlc.InsertCustomerCreditLimitQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) UpdateCreditLimitQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.update_credit_limit_quantity")
	defer span.End()

	err := r.queries.UpdateCustomerCreditLimitQuantity(ctx, sqlc.UpdateCustomerCreditLimitQuantityParams{
		Value:  value,
		UnitID: unitID,
		ID:     id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerRepoImpl) DeleteCreditLimitQuantity(ctx context.Context, id string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.delete_credit_limit_quantity")
	defer span.End()

	err := r.queries.DeleteCustomerCreditLimitQuantity(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// toNullString is defined in unit_repository.go
