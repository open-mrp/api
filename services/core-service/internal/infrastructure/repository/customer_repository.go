package repository

import (
	"cmp"
	"context"
	gosql "database/sql"
	"errors"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/safeconv"
	"github.com/open-mrp/apikit/tracing"
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

func nullBoolPtr(nb gosql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

func boolToNullBool(v *bool) gosql.NullBool {
	if v == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: *v, Valid: true}
}

func nullInt32Ptr(ni gosql.NullInt32) *int32 {
	if ni.Valid {
		return &ni.Int32
	}
	return nil
}

func nullSupply(ns gosql.NullString) *constants.Supply {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	v := constants.Supply(ns.String)
	return &v
}

func supplyToNullString(v *constants.Supply) gosql.NullString {
	if v == nil || *v == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: string(*v), Valid: true}
}

// supplyPolicyToNullString is the planning policy a customer's supply implies, which the production schedule still
// reads: stock is built to stock, make to order, and buy (never produced) has none.
func supplyPolicyToNullString(v *constants.Supply) gosql.NullString {
	if v == nil {
		return gosql.NullString{}
	}
	policy, ok := v.FulfillmentPolicy()
	if !ok {
		return gosql.NullString{}
	}
	return gosql.NullString{String: string(policy), Valid: true}
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
		cp := constants.CommissionPolicyFromStatusCode(ns.String)
		return &cp
	}
	return nil
}

func nullFreightPolicy(ns gosql.NullString) *constants.FreightPolicy {
	if ns.Valid {
		fp := constants.FreightPolicyFromStatusCode(ns.String)
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

func stringToNullString(s *string) gosql.NullString {
	if s == nil || *s == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func (r *customerRepoImpl) List(ctx context.Context, params domain.ListCustomersParams) (*domain.ListCustomersResult, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.list")
	defer span.End()

	empty := &domain.ListCustomersResult{Items: []*domain.Customer{}, PageInfo: pagination.PageInfo{}}

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	search := customerListSearch{}
	if params.Query != nil && *params.Query != "" {
		search.like = "%" + db.EscapeLike(*params.Query) + "%"
		matches, ok, err := customerSearchMatches(ctx, r.queries.DB(), params.AccountID, search.like)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if ok && len(matches) == 0 {
			return empty, nil
		}
		if ok {
			search.matches = matches
		}
	}

	// Resolved up front: as a subquery the planner walks the owner's customers to find the group's few.
	var pricingRelationIDs []string
	if len(params.PricingGroupIDs) > 0 {
		ids, err := r.queries.ListPriceGroupRelationIDs(ctx, params.PricingGroupIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(ids) == 0 {
			return empty, nil
		}
		pricingRelationIDs = ids
	}

	query, args := customerListPageQuery(params, search, pricingRelationIDs, cursor, params.Limit+1)
	pageIDs, err := selectStrings(ctx, r.queries.DB(), query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rows, apiErr := r.getRowsByIDs(ctx, params.AccountID, pageIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	byID := make(map[string]sqlc.GetCustomerRow, len(rows))
	for _, row := range rows {
		byID[row.AccountID] = row
	}
	ordered := inPageOrder(pageIDs, byID)
	items := make([]*domain.Customer, len(ordered))
	relationIDs := make([]string, len(ordered))
	for i, row := range ordered {
		items[i] = customerFromRow(row, params.Includes)
		relationIDs[i] = row.RelationID
	}
	if apiErr := r.stitchListCustomerIncludes(ctx, params.AccountID, items, relationIDs, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var dir *pagination.Direction
	if cursor != nil {
		dir = &cursor.Direction
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, dir, customerCreatedAt, customerID)
	return &domain.ListCustomersResult{Items: result, PageInfo: pageInfo}, nil
}

// getRowsByIDs reads the owner's customers among ids, in no particular order.
func (r *customerRepoImpl) getRowsByIDs(ctx context.Context, ownerAccountID string, ids []string) ([]sqlc.GetCustomerRow, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetCustomersByIDs(ctx, sqlc.GetCustomersByIDsParams{
		OwnerAccountID:         ownerAccountID,
		CounterpartyAccountIds: ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	out := make([]sqlc.GetCustomerRow, len(rows))
	for i, row := range rows {
		out[i] = sqlc.GetCustomerRow(row)
	}
	return out, nil
}

// stitchListCustomerIncludes fans out per-relation data (price groups, notification preferences) into the list items when the corresponding include keys were requested. Skipping them keeps the list path from paying for unused round-trips — the apiresource layer will collapse those fields to null for clients that didn't ask for them.
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
				CommissionPolicy: constants.CommissionPolicyFromStatusCode(row.CommissionStatusCode),
				FreightPolicy:    constants.FreightPolicyFromStatusCode(row.FreightStatusCode),
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
				CommissionPolicy: constants.CommissionPolicyFromStatusCode(pg.CommissionStatusCode),
				FreightPolicy:    constants.FreightPolicyFromStatusCode(pg.FreightStatusCode),
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

	customer := customerFromRow(row, incs)
	customer.PriceGroups = priceGroups
	customer.AcceptsInvoiceEmails = acceptsInvoiceEmails

	if wantsInclude(incs, "child_accounts") {
		childrenByRelation, apiErr := r.fetchChildAccountsByRelationIDs(ctx, ownerAccountID, []string{row.RelationID})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		customer.ChildAccounts = childrenByRelation[row.RelationID]
	}

	return customer, nil
}

// GetByIDs is the batched form of Get with no includes. Ids that are not customers of the owner are
// absent from the result.
func (r *customerRepoImpl) GetByIDs(ctx context.Context, ownerAccountID string, customerAccountIDs []string) ([]*domain.Customer, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.get_by_ids")
	defer span.End()

	if len(customerAccountIDs) == 0 {
		return []*domain.Customer{}, nil
	}

	rows, apiErr := r.getRowsByIDs(ctx, ownerAccountID, customerAccountIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The addresses are in the row already; carrying them lets a customer reached through an
	// include (a transaction's `customer.bill_to_address`) show them.
	customers := make([]*domain.Customer, len(rows))
	for i, row := range rows {
		customers[i] = customerFromRow(row, []string{"bill_to_address", "ship_to_address"})
	}
	return customers, nil
}

// customerFromRow maps the columns every customer read carries. The bill-to and ship-to addresses
// are populated only when requested in incs.
func customerFromRow(row sqlc.GetCustomerRow, incs []string) *domain.Customer {
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

	return &domain.Customer{
		ID:                                 row.AccountID,
		Name:                               row.AccountName,
		Number:                             row.ExternalNumber,
		Status:                             constants.AccountStatusCode(row.Status.String),
		ArchiveStatus:                      constants.ArchiveStatusFromIsActive(row.IsActive),
		IsParentAccount:                    row.IsParentAccount,
		CommissionPolicy:                   constants.CommissionPolicyFromStatusCode(row.CommissionStatusCode.String),
		FreightPolicy:                      constants.FreightPolicyFromStatusCode(row.FreightStatusCode.String),
		DefaultLeadTimeDays:                nullInt32Ptr(row.DefaultLeadTimeDays),
		ReceiveCalendarID:                  nullStringToPtr(row.ReceiveCalendarID),
		Supply:                             nullSupply(row.SupplyCode),
		Instructions:                       nullStringPtr(row.Instructions),
		Email:                              nullStringPtr(row.Email),
		Phone:                              nullStringPtr(row.PhoneNumber),
		URL:                                nullStringPtr(row.WebsiteUrl),
		CarrierBillingType:                 nullCarrierBillingType(row.CarrierBillingType),
		CarrierBillingAccount:              nullStringPtr(row.CarrierBillingAccount),
		CreditLimitID:                      nullStringPtr(row.CreditLimitID),
		CreditLimitValue:                   nullStringPtr(row.CreditLimitValue),
		CreditLimit:                        storedMoneyPtr(row.CreditLimitValue, row.CreditLimitCurrency),
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
		ParentAccountRelationID:            nullStringPtr(row.ParentAccountRelationID),
		ParentAccountName:                  nullStringPtr(row.ParentAccountName),
		ParentAccountNumber:                nullStringPtr(row.ParentAccountNumber),
		ParentAccountCreatedAt:             nullTimePtr(row.ParentAccountCreatedAt),
		ParentAccountUpdatedAt:             nullTimePtr(row.ParentAccountUpdatedAt),
		CreatedAt:                          row.CreatedAt,
		UpdatedAt:                          row.UpdatedAt,
	}
}

// wantsInclude returns true if the include key is present in the include list.
func wantsInclude(incs []string, key string) bool {
	return slices.Contains(incs, key)
}

// fetchChildAccountsByRelationIDs batches a single SQL query to fetch direct children for every parent relation ID in parentRelationIDs, grouped by parent relation ID.
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
	commissionStatus := constants.CommissionPolicyApplied.StatusCode()
	if params.CommissionPolicy != nil {
		commissionStatus = params.CommissionPolicy.StatusCode()
	}
	freightStatus := constants.FreightPolicyBilled.StatusCode()
	if params.FreightPolicy != nil {
		freightStatus = params.FreightPolicy.StatusCode()
	}

	// Determine status code.
	statusCode := "normal"
	if params.StatusCode != nil {
		statusCode = *params.StatusCode
	}

	// Customers carry no priority any more, but the column is NOT NULL until it is dropped.
	priorityCode := string(constants.PriorityCodeNormal)

	// Create the account relation.
	err = r.queries.InsertCustomerRelation(ctx, sqlc.InsertCustomerRelationParams{
		ID:                       relationID,
		OwnerAccountID:           params.OwnerAccountID,
		CounterpartyAccountID:    accountID,
		Alias:                    gosql.NullString{String: params.Name, Valid: true},
		ExternalNumber:           customerNumber,
		Instructions:             toNullString(params.Instructions),
		ParentAccountRelationID:  toNullString(params.ParentRelationID),
		IsActive:                 params.ArchiveStatus.IsActive(),
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
		DefaultLeadTimeDays:      toNullInt32(params.DefaultLeadTimeDays),
		ReceiveCalendarID:        toNullString(params.ReceiveCalendarID),
		FulfillmentPolicyCode:    supplyPolicyToNullString(params.Supply),
		SupplyCode:               supplyToNullString(params.Supply),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, customerNumberDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.CreditLimitID != nil {
		if apiErr := db.MapSQLError(syncCreditLimit(ctx, r.queries.DB(), relationID)); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, params.OwnerAccountID, accountID, nil)
}

func (r *customerRepoImpl) Update(ctx context.Context, relationID string, params domain.UpdateCustomerParams) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.update")
	defer span.End()

	// Map commission/freight policy to status codes.
	var commissionStatus gosql.NullString
	if params.CommissionPolicy != nil {
		commissionStatus = gosql.NullString{String: params.CommissionPolicy.StatusCode(), Valid: true}
	}

	var freightStatus gosql.NullString
	if params.FreightPolicy != nil {
		freightStatus = gosql.NullString{String: params.FreightPolicy.StatusCode(), Valid: true}
	}

	err := r.queries.UpdateCustomer(ctx, sqlc.UpdateCustomerParams{
		ID:                       relationID,
		OwnerAccountID:           params.OwnerAccountID,
		Alias:                    stringToNullString(params.Name),
		ExternalNumber:           stringToNullString(params.Number),
		Instructions:             field.StringToNullString(params.Instructions),
		ParentAccountRelationID:  stringToNullString(params.ParentRelationID),
		IsActive:                 params.ArchiveStatus == nil || params.ArchiveStatus.IsActive(),
		CommissionStatusCode:     commissionStatus,
		FreightStatusCode:        freightStatus,
		DefaultCarrierID:         stringToNullString(params.DefaultCarrierID),
		DefaultCarrierOptionID:   field.StringToNullString(params.DefaultServiceLevelID),
		DefaultSalesRepID:        field.StringToNullString(params.DefaultSalesRepID),
		AccountStatusCode:        stringToNullString(params.StatusCode),
		PaymentTermID:            stringToNullString(params.DefaultPaymentTermID),
		AccountGroupID:           stringToNullString(params.CustomerTypeGroupID),
		ShippingTermID:           stringToNullString(params.DefaultShippingTermID),
		CarrierBillingType:       stringToNullString(params.CarrierBillingType),
		CarrierBillingAccount:    field.StringToNullString(params.CarrierBillingAccount),
		DefaultLeadTimeDays:      field.Int32ToNullInt32(params.DefaultLeadTimeDays),
		ReceiveCalendarID:        field.StringToNullString(params.ReceiveCalendarID),
		FulfillmentPolicyCode:    supplyPolicyToNullString(params.Supply.ValuePtr()),
		SupplyCode:               supplyToNullString(params.Supply.ValuePtr()),
		DefaultBillingAddressID:  field.StringToNullString(params.BillToAddressID),
		DefaultShippingAddressID: field.StringToNullString(params.ShipToAddressID),
		CreditLimitID:            stringToNullString(params.CreditLimitID),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, customerNumberDuplicateKeyMapping); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := db.MapSQLError(syncCreditLimit(ctx, r.queries.DB(), relationID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// customerInUseQuery probes, each from its own key, for the owner's sales orders and shipments to the customer and
// for customers under it.
const customerInUseQuery = `SELECT 'sales orders' FROM DUAL WHERE EXISTS (
    SELECT 1 FROM sales_order FORCE INDEX (sales_order_owner_buyer_created_idx) WHERE owner_account_id = ? AND buyer_account_id = ?)
UNION ALL
SELECT 'shipments' FROM DUAL WHERE EXISTS (
    SELECT 1 FROM shipment FORCE INDEX (shipment_account_buyer_created_idx) WHERE account_id = ? AND buyer_account_id = ?)
UNION ALL
SELECT 'child customers' FROM DUAL WHERE EXISTS (
    SELECT 1 FROM account_relation FORCE INDEX (account_relation_parent_account_relation_id_idx) WHERE parent_account_relation_id = ?)`

func (r *customerRepoImpl) InUseBy(ctx context.Context, ownerAccountID, customerAccountID, relationID string) ([]string, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.in_use_by")
	defer span.End()

	labels, err := selectStrings(ctx, r.queries.DB(), customerInUseQuery,
		ownerAccountID, customerAccountID, ownerAccountID, customerAccountID, relationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return labels, nil
}

func (r *customerRepoImpl) IsArchived(ctx context.Context, ownerAccountID, customerAccountID string) (bool, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.is_archived")
	defer span.End()

	var isActive bool
	err := r.queries.DB().QueryRowContext(ctx, `SELECT is_active FROM account_relation
WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		ownerAccountID, customerAccountID).Scan(&isActive)
	if errors.Is(err, gosql.ErrNoRows) {
		return false, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return !isActive, nil
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

	if apiErr := r.deleteRelationChildren(ctx, ownerAccountID, []string{customerAccountID}); apiErr != nil {
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

// deleteRelationChildren deletes the product line access, price groups, and notification preferences
// of the customers' relations, which nothing else removes once the relation row is gone.
func (r *customerRepoImpl) deleteRelationChildren(ctx context.Context, ownerAccountID string, customerIDs []string) *apierror.APIError {
	err := r.queries.DeleteCustomerRelationProductLines(ctx, sqlc.DeleteCustomerRelationProductLinesParams{
		OwnerAccountID:         ownerAccountID,
		CounterpartyAccountIds: customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	err = r.queries.DeleteCustomerRelationPriceGroups(ctx, sqlc.DeleteCustomerRelationPriceGroupsParams{
		OwnerAccountID:         ownerAccountID,
		CounterpartyAccountIds: customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	err = r.queries.DeleteCustomerRelationNotificationPreferences(ctx, sqlc.DeleteCustomerRelationNotificationPreferencesParams{
		OwnerAccountID:         ownerAccountID,
		CounterpartyAccountIds: customerIDs,
	})
	return db.MapSQLError(err)
}

func (r *customerRepoImpl) BulkDelete(ctx context.Context, ownerAccountID string, customerIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.bulk_delete")
	defer span.End()

	if apiErr := r.deleteRelationChildren(ctx, ownerAccountID, customerIDs); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

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

func (r *customerRepoImpl) AllocateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string) (int64, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.allocate_next_number")
	defer span.End()

	number, apiErr := customerNumbers(r.queries, accountID).next(ctx, sysPropertyID)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return number, nil
}

func (r *customerRepoImpl) LockNumbers(ctx context.Context, ownerAccountID string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.lock_numbers")
	defer span.End()

	_, err := r.queries.LockCustomerNumbers(ctx, ownerAccountID)
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

// frequentlyOrderedProductLimit caps how many products the reorder shortcut offers. Applied in Go rather than as a SQL LIMIT because the ranking it trims is decided in Go.
const frequentlyOrderedProductLimit = 12

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

	products := make([]*domain.FrequentlyOrderedProduct, 0, len(rows))
	for _, row := range rows {
		products = append(products, &domain.FrequentlyOrderedProduct{
			ItemID:     row.ItemID,
			OrderCount: safeconv.Int64ToInt32(row.OrderCount),
		})
	}

	// Most-ordered first, item id breaking ties so equally-ordered products do not shuffle between calls.
	slices.SortFunc(products, func(a, b *domain.FrequentlyOrderedProduct) int {
		if c := cmp.Compare(b.OrderCount, a.OrderCount); c != 0 {
			return c
		}
		return cmp.Compare(a.ItemID, b.ItemID)
	})

	if len(products) > frequentlyOrderedProductLimit {
		products = products[:frequentlyOrderedProductLimit]
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

func (r *customerRepoImpl) ListMergeRetaggableOrders(ctx context.Context, ownerAccountID string, sourceAccountIDs []string, salesRepID string) ([]domain.OrderSalesRep, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.list_merge_retaggable_orders")
	defer span.End()

	rows, err := r.queries.ListMergeRetaggableOrders(ctx, sqlc.ListMergeRetaggableOrdersParams{
		OwnerAccountID:   ownerAccountID,
		SourceAccountIds: sourceAccountIDs,
		SalesRepID:       gosql.NullString{String: salesRepID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	orders := make([]domain.OrderSalesRep, len(rows))
	for i, row := range rows {
		orders[i] = domain.OrderSalesRep{OrderID: row.ID, SalesRepID: nullStringPtr(row.SalesRepID)}
	}
	return orders, nil
}

func (r *customerRepoImpl) SetOrdersSalesRep(ctx context.Context, ownerAccountID, salesRepID string, orderIDs []string) *apierror.APIError {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.set_orders_sales_rep")
	defer span.End()

	if len(orderIDs) == 0 {
		return nil
	}
	err := r.queries.SetSalesOrdersSalesRep(ctx, sqlc.SetSalesOrdersSalesRepParams{
		SalesRepID:     gosql.NullString{String: salesRepID, Valid: true},
		OwnerAccountID: ownerAccountID,
		Ids:            orderIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
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

	err = r.queries.MergeCustomerPicks(ctx, sqlc.MergeCustomerPicksParams{
		OwnerAccountID:  ownerAccountID,
		TargetAccountID: targetAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.MergeCustomerShipmentBuyers(ctx, sqlc.MergeCustomerShipmentBuyersParams{
		OwnerAccountID:  ownerAccountID,
		TargetAccountID: targetAccountID,
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

func (r *customerRepoImpl) ListContacts(ctx context.Context, customerAccountIDs []string) ([]domain.CustomerContact, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.list_contacts")
	defer span.End()

	if len(customerAccountIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.ListCustomerContacts(ctx, customerAccountIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	contacts := make([]domain.CustomerContact, len(rows))
	for i, row := range rows {
		contacts[i] = domain.CustomerContact{
			CustomerAccountID: row.AccountID,
			Name:              nullStringPtr(row.Name),
			Email:             nullStringPtr(row.Email),
		}
	}
	return contacts, nil
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
		if apiErr.Code == apierror.CodeResourceNotFound {
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

// IsCommissionExempt reports whether the customer relationship, its type group, or any of its price groups is commission-exempt. Mirrors Dashboard's isCustomerOrGroupCommissionExempt, used in sales-rep auto-resolution.
func (r *customerRepoImpl) IsCommissionExempt(ctx context.Context, ownerAccountID, customerAccountID string) (bool, *apierror.APIError) {
	ctx, span := customerRepoTracer.Start(ctx, "repository.customer.is_commission_exempt")
	defer span.End()

	exempt, err := r.queries.IsCustomerCommissionExempt(ctx, sqlc.IsCustomerCommissionExemptParams{
		OwnerAccountID:    ownerAccountID,
		CustomerAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exempt, nil
}

// toNullString is defined in unit_repository.go

// commissionStatusCodes turns commission policy filter values into the codes stored on the row.
func commissionStatusCodes(policies []string) []string {
	out := make([]string, len(policies))
	for i, p := range policies {
		out[i] = constants.CommissionPolicyFromStatusCode(p).StatusCode()
	}
	return out
}

// freightStatusCodes turns freight policy filter values into the codes stored on the row.
func freightStatusCodes(policies []string) []string {
	out := make([]string, len(policies))
	for i, p := range policies {
		out[i] = constants.FreightPolicyFromStatusCode(p).StatusCode()
	}
	return out
}

// commissionPolicyPtr reads a nullable stored commission status code as a policy.
func commissionPolicyPtr(ns gosql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := string(constants.CommissionPolicyFromStatusCode(ns.String))
	return &s
}
