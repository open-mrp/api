package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type AccountRepo interface {
	Create(ctx context.Context, id, name string, accountTypeCode AccountType, planCode constants.PlanCode) *apierror.APIError
	GetPlanCode(ctx context.Context, id string) (constants.PlanCode, *apierror.APIError)
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)
	// GetAccountNames resolves account IDs to their display names. IDs with no matching account are omitted from the result.
	GetAccountNames(ctx context.Context, ids []string) (map[string]string, *apierror.APIError)
	// GetPlanIDAndPeriodEnd returns the account's active plan id (for limit lookups) and current subscription period end (for deriving billing-period start).
	GetPlanIDAndPeriodEnd(ctx context.Context, accountID string) (planID *string, periodEnd *time.Time, apiErr *apierror.APIError)
	Delete(ctx context.Context, id string) *apierror.APIError
	GetPlanTypeIDByCode(ctx context.Context, planCode string) (string, *apierror.APIError)
	UpdateSubscription(ctx context.Context, accountID string, status *string, planCode string, accountPlanID *string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError
	ClearStripeCustomer(ctx context.Context, accountID string) *apierror.APIError
	ClearPricingPlanSubscription(ctx context.Context, accountID string) *apierror.APIError
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (accountID string, planCode string, err *apierror.APIError)
	GetSandboxLimit(ctx context.Context, accountID string) (*int32, *apierror.APIError)
	GetSeatLimitByPlanCode(ctx context.Context, planCode string) (*int32, *apierror.APIError)
	CountNonSandboxByPlanCode(ctx context.Context, planCode string) (int64, *apierror.APIError)
	UpdateAgentSpendingCap(ctx context.Context, accountID string, capCents *int64) *apierror.APIError
	GetAgentSpendingCap(ctx context.Context, accountID string) (*int64, *apierror.APIError)
	HasActiveBillingPlan(ctx context.Context, accountID string) (bool, *apierror.APIError)
	GetName(ctx context.Context, accountID string) (string, *apierror.APIError)
	GetPortalSlug(ctx context.Context, accountID string) (*string, *apierror.APIError)
	GetByID(ctx context.Context, accountID string) (*Account, *apierror.APIError)
	// GetByIDs returns accounts matching the given IDs. Caller authorization is enforced at the service layer.
	GetByIDs(ctx context.Context, ids []string) ([]*Account, *apierror.APIError)
	GetBySlug(ctx context.Context, slug string) (*PublicAccountBySlug, *apierror.APIError)
	UpdateName(ctx context.Context, accountID, name string) *apierror.APIError
	UpdateBranding(ctx context.Context, accountID string, params UpdateAccountParams) *apierror.APIError
	UpdatePortalSlug(ctx context.Context, accountID, slug string) *apierror.APIError
	ExistsPortalSlug(ctx context.Context, slug, excludeAccountID string) (bool, *apierror.APIError)
	UpdateBrandingLogoURL(ctx context.Context, accountID, logoURL string) *apierror.APIError
	GetBrandingLogoKey(ctx context.Context, accountID string) (*string, *apierror.APIError)
	UpdateBrandingFaviconURL(ctx context.Context, accountID, faviconURL string) *apierror.APIError
	GetBrandingFaviconKey(ctx context.Context, accountID string) (*string, *apierror.APIError)
	ListPlanLimits(ctx context.Context, accountPlanID string) (map[string]*int32, *apierror.APIError)
	ListPlanFeatures(ctx context.Context, accountPlanID string) (map[string]bool, *apierror.APIError)
}

type AccountUserRepo interface {
	FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*AccountUser, *apierror.APIError)
	// ResolveAccountUserID resolves either an account_user id or a user id to the account_user id within the given account.
	ResolveAccountUserID(ctx context.Context, accountID, userOrAccountUserID string) (string, *apierror.APIError)
	FindAffiliationsByUserID(ctx context.Context, userID string) ([]AccountAffiliation, *apierror.APIError)
	FindLastUsedAccountID(ctx context.Context, userID string) (string, *apierror.APIError)
	UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *apierror.APIError
	GetAdminRoleID(ctx context.Context) (string, *apierror.APIError)
	DeactivateExcept(ctx context.Context, accountID, keepUserID string, limit int32) (int64, *apierror.APIError)
	EnsureActive(ctx context.Context, accountID, userID string) *apierror.APIError
	CountActive(ctx context.Context, accountID string) (int64, *apierror.APIError)
	ReactivateUsers(ctx context.Context, accountID string, limit int32) (int64, *apierror.APIError)
	List(ctx context.Context, params ListAccountUsersParams) (*ListAccountUsersResult, *apierror.APIError)
	GetDetail(ctx context.Context, accountID, userID string, includes []string) (*AccountUserDetail, *apierror.APIError)
	GetDetailByAccountAndID(ctx context.Context, accountID, accountUserID string, includes []string) (*AccountUserDetail, *apierror.APIError)
	Create(ctx context.Context, id, accountID, userID string, roleID, departmentID *string, isCommissionEligible bool) *apierror.APIError
	Update(ctx context.Context, accountUserID string, roleID, departmentID *string, isCommissionEligible bool) *apierror.APIError
	// ReactivateRemovedAccountUser reactivates a previously soft-removed link for (accountID, userID), setting its role/department, and returns the reactivated account_user id. Returns resource_not_found when no removed link exists.
	ReactivateRemovedAccountUser(ctx context.Context, accountID, userID string, roleID, departmentID *string, isCommissionEligible bool) (string, *apierror.APIError)
	SoftDelete(ctx context.Context, accountUserID string) *apierror.APIError
	UpdateStatus(ctx context.Context, accountUserID string, status constants.AccountUserStatus) *apierror.APIError
	CountByRoleID(ctx context.Context, accountID, roleID string) (int64, *apierror.APIError)
	RevokeRefreshTokensByUserID(ctx context.Context, userID string) *apierror.APIError
	FindFirstAccountIDByUserID(ctx context.Context, userID string) (string, *apierror.APIError)
	FindTenancyAccountsByUserID(ctx context.Context, userID string) ([]TenancyAccount, *apierror.APIError)
	MarkUsedByAccountAndUser(ctx context.Context, accountID, userID string) *apierror.APIError
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*AccountUserDetail, *apierror.APIError)
}

type UserRepo interface {
	FindByID(ctx context.Context, userID string) (*UserRecord, *apierror.APIError)
	// GetByIDs returns the users matching the given IDs that are affiliated with the given account.
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*UserRecord, *apierror.APIError)
	FindByEmail(ctx context.Context, email string) (*UserRecord, *apierror.APIError)
	FindByUsername(ctx context.Context, username string) (*UserRecord, *apierror.APIError)
	CreateUser(ctx context.Context, id string, params CreateUserRecordParams) *apierror.APIError
	UpdateProfile(ctx context.Context, userID string, name, email, username, imageURL *string, emailVerified *time.Time) *apierror.APIError
	UpdatePassword(ctx context.Context, userID, hashedPassword string) *apierror.APIError
	GetHashedPassword(ctx context.Context, userID string) (string, *apierror.APIError)
	UpdateImageURL(ctx context.Context, userID string, imageURL *string) *apierror.APIError
}

type SandboxAccountRepo interface {
	FindFirstByOwnerAccountID(ctx context.Context, ownerAccountID string) (string, *apierror.APIError)
	FindByTypeID(ctx context.Context, typeID string, includes []string) (*SandboxAccount, *apierror.APIError)
	GetByTypeIDs(ctx context.Context, ownerAccountID string, typeIDs []string) ([]*SandboxAccount, *apierror.APIError)
	List(ctx context.Context, ownerAccountID string, cursor *string, limit int32, query *string, includes []string) (*ListSandboxAccountsResult, *apierror.APIError)
	Create(ctx context.Context, typeID, ownerAccountID, accountID string) *apierror.APIError
	CountByOwnerAccountID(ctx context.Context, ownerAccountID string) (int64, *apierror.APIError)
	DeleteByID(ctx context.Context, id int64) *apierror.APIError
}

type AccountRelationRepo interface {
	FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)
	FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)
	FindByCounterpartyAccountAndUserID(ctx context.Context, counterpartyAccountID, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)
	FindByCounterpartyAccountAndAPIKeyID(ctx context.Context, counterpartyAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)
	FindCustomerByEmail(ctx context.Context, ownerAccountID, email string) (*CustomerByEmail, *apierror.APIError)
	FindContactsByEmail(ctx context.Context, ownerAccountID, email string) ([]ContactMatch, *apierror.APIError)
	HasRelation(ctx context.Context, ownerAccountID, counterpartyAccountID string) (bool, *apierror.APIError)
	CountOtherOwnerRelations(ctx context.Context, counterpartyAccountID, excludeOwnerAccountID string) (int64, *apierror.APIError)
	FindRelationByOwnerAndCounterparty(ctx context.Context, ownerAccountID, counterpartyAccountID string) (string, *apierror.APIError)
	CreateNotificationPreference(ctx context.Context, id, accountRelationID, recipientAccountUserID string, notificationTypeCode string) *apierror.APIError
	ListNotificationPreferences(ctx context.Context, accountRelationID, recipientAccountUserID string) ([]NotificationPreference, *apierror.APIError)
	ListNotificationRecipients(ctx context.Context, accountRelationID string) ([]NotificationRecipientRef, *apierror.APIError)
	DeleteNotificationPreference(ctx context.Context, accountRelationID, recipientAccountUserID, notificationTypeCode string) *apierror.APIError
	DeleteNotificationPreferencesByTypes(ctx context.Context, accountRelationID string, notificationTypeCodes []string) *apierror.APIError
	ListChildAccounts(ctx context.Context, params ListChildAccountsParams) (*ListChildAccountsResult, *apierror.APIError)
	GetChildAccountDetail(ctx context.Context, ownerAccountID, counterpartyAccountID string) (*ChildAccount, *apierror.APIError)
	GetChildAccountsByRelationIDs(ctx context.Context, ownerAccountID string, relationIDs []string) ([]*ChildAccount, *apierror.APIError)
	SetParentRelation(ctx context.Context, ownerAccountID, childRelationID, parentRelationID string) *apierror.APIError
	ClearParentRelation(ctx context.Context, ownerAccountID, childRelationID, parentRelationID string) *apierror.APIError
	GetParentRelationID(ctx context.Context, relationID string) (*string, *apierror.APIError)
	FindCustomerAccountsByVendorAndUser(ctx context.Context, vendorAccountID, userID string) ([]CustomerAccountSummary, *apierror.APIError)
}

// SystemProductInfo holds the minimal info needed to synthesize an order line using one of the account's built-in system products (credit, shipping).
type SystemProductInfo struct {
	ProductID          string
	ProductSKU         string
	ProductDescription *string
	QuantityUnitID     string
}

type ProductRepo interface {
	SearchBySKU(ctx context.Context, accountID, query string) ([]ProductInfo, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string) ([]ProductInfo, *apierror.APIError)
	// GetSystemProduct fetches the account's built-in product matching the given product_type_code (e.g. "credit", "shipping") along with the base unit of its item category. Returns nil if no such product exists.
	GetSystemProduct(ctx context.Context, accountID, productTypeCode string) (*SystemProductInfo, *apierror.APIError)

	List(ctx context.Context, params ListProductsFullParams) (*ListProductsFullResult, *apierror.APIError)
	Export(ctx context.Context, params ExportProductsParams) ([]*ProductFull, *apierror.APIError)
	Get(ctx context.Context, params GetProductFullParams) (*ProductFull, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*ProductFull, *apierror.APIError)
	Create(ctx context.Context, productID, itemID string, params CreateProductParams) (*ProductFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateProductParams) (*ProductFull, *apierror.APIError)
	SoftDelete(ctx context.Context, params DeleteProductParams) *apierror.APIError
	ChangeProductLine(ctx context.Context, params ChangeProductProductLineParams) (*ProductFull, *apierror.APIError)
	ValidateProducts(ctx context.Context, params ValidateProductsParams) (*ValidateProductsResult, *apierror.APIError)
	ExistsBySKU(ctx context.Context, accountID, sku string, excludeItemID *string) (bool, *apierror.APIError)
	// FindBySKUs batch-resolves existing products by SKU within the account, returning the
	// product/item IDs and unit_value/unit_cost rate IDs needed to update them.
	FindBySKUs(ctx context.Context, accountID string, skus []string) ([]*ProductSKUMatch, *apierror.APIError)
	InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	InsertItem(ctx context.Context, params InsertProductItemParams) *apierror.APIError
}

type ItemRepo interface {
	List(ctx context.Context, params ListItemsParams) (*ListItemsResult, *apierror.APIError)
	Get(ctx context.Context, params GetItemParams) (*Item, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Item, *apierror.APIError)
	GetInventory(ctx context.Context, accountID, itemID string) (*ItemInventory, *apierror.APIError)
	GetCostFlowConsumptions(ctx context.Context, stepID string) ([]CostFlowConsumption, *apierror.APIError)
	// FindItemsProducedFromConsumed returns the items produced by every step that consumes any of the given ones — one generation outwards in the cost graph.
	FindItemsProducedFromConsumed(ctx context.Context, accountID string, itemIDs []string) ([]string, *apierror.APIError)
	UpdateUnitCost(ctx context.Context, accountID, itemID string, cost decimal.Decimal, denominatorUnitID string) *apierror.APIError
	// GetStockingUnit resolves the unit an item is counted in, via its category's unit group, and that group. It is the only denominator the item's unit cost may carry, whatever unit the production step producing it is written in.
	GetStockingUnit(ctx context.Context, accountID, itemID string) (*ItemStockingUnit, *apierror.APIError)
	GetTrends(ctx context.Context, accountID, itemID, trendType string) (*ItemTrends, *apierror.APIError)
	ExportWithInventory(ctx context.Context, accountID string) (*ExportItemsResult, *apierror.APIError)
	Update(ctx context.Context, params UpdateItemParams) *apierror.APIError
	CheckSKUExists(ctx context.Context, accountID, sku, excludeID string) (bool, *apierror.APIError)
	AddAttribute(ctx context.Context, params AddItemAttributeParams) *apierror.APIError
	RemoveAttribute(ctx context.Context, params RemoveItemAttributeParams) *apierror.APIError
	ChangeCategory(ctx context.Context, params ChangeItemCategoryParams) *apierror.APIError
	UpdateRateUnits(ctx context.Context, accountID, itemID, newUnitID string) *apierror.APIError
	UpdateMaterialOrderPointUnit(ctx context.Context, accountID, itemID, newUnitID string) *apierror.APIError
	UpdateConsumptionProductionQuantityUnits(ctx context.Context, accountID, itemID, newUnitID string) *apierror.APIError
	// GetCategoryBaseUnitID resolves a category's base unit id and its category type
	// code, so create paths can enforce item-type/category-type matching.
	GetCategoryBaseUnitID(ctx context.Context, categoryID string) (string, string, *apierror.APIError)
	// GetCategoryBaseUnitIDs batch-resolves category base unit ids and category type
	// codes. Existing categories map to their ref (base unit id empty when the unit
	// group has no base unit); missing categories are absent from the map. Used by
	// bulk upsert category validation.
	GetCategoryBaseUnitIDs(ctx context.Context, categoryIDs []string) (map[string]CategoryRef, *apierror.APIError)
	ListConsumptionChangeLogsForBurnRate(ctx context.Context, accountID, itemID string) ([]BurnRateConsumptionLog, *apierror.APIError)
	// ListStaleBurnRateItems returns up to limit items whose burn rate has not been recomputed
	// since staleBefore, stalest first, for the periodic sweeper to re-enqueue.
	ListStaleBurnRateItems(ctx context.Context, staleBefore time.Time, limit int32) ([]StaleBurnRateItem, *apierror.APIError)
	FetchItemsBySKU(ctx context.Context, accountID string, skus []string) ([]ItemSKUInfo, *apierror.APIError)
	// FindBySKU returns the existing item's ID and its unit_value rate ID for the given SKU within the account. Returns (nil, nil, nil) when no match exists. Used by bulk upsert flows.
	FindBySKU(ctx context.Context, accountID, sku string) (itemID *string, unitValueRateID *string, apiErr *apierror.APIError)
	// UpdateRateValue updates a rate's numeric value in place.
	UpdateRateValue(ctx context.Context, rateID, value string) *apierror.APIError
	// UpdateRate updates value and unit IDs on an existing rate row.
	UpdateRate(ctx context.Context, rateID string, params CreateRateParams) *apierror.APIError
	// ClearItemDirtyFlag sets is_dirty = 0 on an item (e.g. after manual unit cost edit).
	ClearItemDirtyFlag(ctx context.Context, accountID, itemID string) *apierror.APIError
	// LoadAttributes fetches the attributes for an item and populates item.Attributes.
	LoadAttributes(ctx context.Context, item *Item) *apierror.APIError
}

type RolePermissionRepo interface {
	FindByRoleID(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)
	ListByRoleID(ctx context.Context, roleID string) ([]*RolePermission, *apierror.APIError)
	ListByRoleIDs(ctx context.Context, roleIDs []string) (map[string][]*RolePermission, *apierror.APIError)
	Create(ctx context.Context, permID, roleID string, input CreateRolePermissionInput) *apierror.APIError
	DeleteByRoleID(ctx context.Context, roleID string) *apierror.APIError
}

type RoleRepo interface {
	GetByID(ctx context.Context, roleID string) (*RoleInfo, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Role, *apierror.APIError)
	FindByTypeCode(ctx context.Context, typeCode string, accountID string) (*RoleInfo, *apierror.APIError)
	List(ctx context.Context, params ListRolesParams) (*ListRolesPage, *apierror.APIError)
	Get(ctx context.Context, roleID, accountID string) (*Role, *apierror.APIError)
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	Create(ctx context.Context, roleID string, params CreateRoleParams) *apierror.APIError
	UpdateName(ctx context.Context, roleID, accountID, name string) *apierror.APIError
	Delete(ctx context.Context, roleID, accountID string) *apierror.APIError
}

// RegistrationRepo handles the multi-step account creation during registration: account record, owner role with all permissions, account-user join, business address, and account portal.
type RegistrationRepo interface {
	// CreateAccountForRegistration creates a production account with Stripe billing fields populated.
	CreateAccountForRegistration(ctx context.Context, params CreateAccountParams) *apierror.APIError

	// CreateAccountUser creates an account-user join record linking the user to the account with the given role.
	CreateAccountUser(ctx context.Context, accountID, userID, roleID string) *apierror.APIError

	// CreateBusinessAddress creates a geolocation, address, and account-address chain and sets the address as the account's default billing and shipping address.
	CreateBusinessAddress(ctx context.Context, accountID, accountName string, address RegistrationAddress) *apierror.APIError

	// CreateAccountPortal creates a portal record with a slug derived from the account ID.
	CreateAccountPortal(ctx context.Context, accountID string) *apierror.APIError

	// CreateSystemProducts creates the shipping and credit system products required by every account. This includes units, unit groups, item categories, product lines, rates, items, and product records.
	CreateSystemProducts(ctx context.Context, accountID string) *apierror.APIError

	// CreateAccountBranding creates a default (empty) branding record for the account so portal and notification templates can reference it.
	CreateAccountBranding(ctx context.Context, accountID string) *apierror.APIError
}

type UnitRepo interface {
	List(ctx context.Context, params ListUnitsParams) (*ListUnitsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportUnitsParams) ([]*Unit, *apierror.APIError)
	Get(ctx context.Context, params GetUnitParams) (*Unit, *apierror.APIError)
	// GetCurrencyBaseUnitID returns the global currency base unit ID used as the numerator unit when building monetary price rates.
	GetCurrencyBaseUnitID(ctx context.Context) (string, *apierror.APIError)
	// GetFreightWeightUnitID returns the global unit shipping-case freight weights are recorded in (pounds), which is what carriers are quoted and billed on.
	GetFreightWeightUnitID(ctx context.Context) (string, *apierror.APIError)
	// GetDimensionCodes returns a unit-id → unit_dimension_code map for the given IDs. Used to enforce unit-type constraints (e.g., currency-only numerator on cost rates) before persisting rate rows.
	GetDimensionCodes(ctx context.Context, ids []string) (map[string]string, *apierror.APIError)
	// IsUnitInGroup reports whether a unit is usable as a unit group's member, counting the group's base unit. A dimension code is too coarse to answer this: a carton and an each are both counts, and only the group says how many of one make the other.
	IsUnitInGroup(ctx context.Context, unitGroupID, unitID string) (bool, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateUnitParams) (*Unit, *apierror.APIError)
	Update(ctx context.Context, params UpdateUnitParams) (*Unit, *apierror.APIError)
	Delete(ctx context.Context, params DeleteUnitParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	ExistsByAbbreviation(ctx context.Context, accountID, abbreviation string, excludeID *string) (bool, *apierror.APIError)
	FindByAbbreviations(ctx context.Context, accountID string, abbreviations []string) ([]*Unit, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Unit, *apierror.APIError)
	FindByAbbreviationsOrNames(ctx context.Context, accountID string, abbreviations, names []string) ([]*Unit, *apierror.APIError)
}

type UnitGroupRepo interface {
	List(ctx context.Context, params ListUnitGroupsParams) (*ListUnitGroupsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportUnitGroupsParams) ([]*UnitGroupFull, *apierror.APIError)
	Get(ctx context.Context, params GetUnitGroupParams) (*UnitGroupFull, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateUnitGroupParams) (*UnitGroupFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateUnitGroupParams) (*UnitGroupFull, *apierror.APIError)
	Delete(ctx context.Context, params DeleteUnitGroupParams) *apierror.APIError
	Exists(ctx context.Context, params UnitGroupExistsParams) (bool, *apierror.APIError)
	GetTypesByIDs(ctx context.Context, accountID string, ids []string) (map[string]string, *apierror.APIError)
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	FindByNames(ctx context.Context, accountID string, names []string) ([]*UnitGroupFull, *apierror.APIError)
	FindUnitsByGroupIDs(ctx context.Context, unitGroupIDs []string) ([]*UnitGroupUnit, *apierror.APIError)
	UpsertUnitGroupUnit(ctx context.Context, id string, params UpsertUnitGroupUnitParams) (*UnitGroupUnit, *apierror.APIError)
	DeleteUnitGroupUnit(ctx context.Context, params DeleteUnitGroupUnitParams) *apierror.APIError
	DeleteAllUnitGroupUnits(ctx context.Context, accountID, unitGroupID string) *apierror.APIError
	ListUnits(ctx context.Context, unitGroupID string, includes []string) ([]*UnitGroupUnit, *apierror.APIError)
	GetUnit(ctx context.Context, params GetUnitGroupUnitParams) (*UnitGroupUnit, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*UnitGroupFull, *apierror.APIError)
	GetUnitGroupUnitsByIDs(ctx context.Context, accountID string, ids []string) ([]*UnitGroupUnit, *apierror.APIError)
}

type PaymentTermRepo interface {
	List(ctx context.Context, params ListPaymentTermsParams) (*ListPaymentTermsResult, *apierror.APIError)
	Get(ctx context.Context, params GetPaymentTermParams) (*PaymentTerm, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*PaymentTerm, *apierror.APIError)
	Create(ctx context.Context, id string, params CreatePaymentTermParams) (*PaymentTerm, *apierror.APIError)
	Update(ctx context.Context, params UpdatePaymentTermParams) (*PaymentTerm, *apierror.APIError)
	Delete(ctx context.Context, params DeletePaymentTermParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
}

type ShippingTermRepo interface {
	List(ctx context.Context, params ListShippingTermsParams) (*ListShippingTermsResult, *apierror.APIError)
	Get(ctx context.Context, params GetShippingTermParams) (*ShippingTerm, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*ShippingTerm, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateShippingTermParams) (*ShippingTerm, *apierror.APIError)
	Update(ctx context.Context, params UpdateShippingTermParams) (*ShippingTerm, *apierror.APIError)
	Delete(ctx context.Context, params DeleteShippingTermParams) *apierror.APIError
	InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	DeleteQuantity(ctx context.Context, id string) *apierror.APIError
	InsertFreeShippingRule(ctx context.Context, id, shippingTermID, serviceLevelID string) *apierror.APIError
	DeleteFreeShippingRulesByShippingTermID(ctx context.Context, shippingTermID string) *apierror.APIError
}

type AccountGroupRepo interface {
	List(ctx context.Context, params ListAccountGroupsParams) (*ListAccountGroupsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, id string) (*AccountGroup, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*AccountGroup, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateAccountGroupParams) (*AccountGroup, *apierror.APIError)
	Update(ctx context.Context, params UpdateAccountGroupParams) (*AccountGroup, *apierror.APIError)
	Delete(ctx context.Context, params DeleteAccountGroupParams) *apierror.APIError
	CheckAccountGroupNotInUse(ctx context.Context, accountGroup *AccountGroup) *apierror.APIError
	DeleteAccountRelationPriceGroupsByAccountGroupID(ctx context.Context, accountGroupID string) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
}

type DeletedRecordRepo interface {
	Create(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string, data any) *apierror.APIError
	Exists(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string) (bool, *apierror.APIError)
}

type AccountGroupProductLineAccessRepo interface {
	List(ctx context.Context, params ListAccountGroupProductLineAccessParams) (*ListAccountGroupProductLineAccessResult, *apierror.APIError)
	Get(ctx context.Context, accountID, accountGroupID string) (*AccountGroupProductLineAccess, *apierror.APIError)
	Create(ctx context.Context, params CreateAccountGroupProductLineAccessParams) (*AccountGroupProductLineAccess, *apierror.APIError)
	Update(ctx context.Context, params UpdateAccountGroupProductLineAccessParams) (*AccountGroupProductLineAccess, *apierror.APIError)
	Delete(ctx context.Context, accountID, accountGroupID string) *apierror.APIError
	ExistsByAccountGroupID(ctx context.Context, accountGroupID string) (bool, *apierror.APIError)
}

type CustomerProductLineAccessRepo interface {
	List(ctx context.Context, params ListCustomerProductLineAccessParams) (*ListCustomerProductLineAccessResult, *apierror.APIError)
	Get(ctx context.Context, accountID, customerID string) (*CustomerProductLineAccess, *apierror.APIError)
	Create(ctx context.Context, params CreateCustomerProductLineAccessParams) (*CustomerProductLineAccess, *apierror.APIError)
	Update(ctx context.Context, params UpdateCustomerProductLineAccessParams) (*CustomerProductLineAccess, *apierror.APIError)
	Delete(ctx context.Context, accountID, customerID string) *apierror.APIError
	ExistsByCustomerID(ctx context.Context, accountID, customerID string) (bool, *apierror.APIError)
}

type AddressRepo interface {
	List(ctx context.Context, params ListAddressesParams) (*ListAddressesResult, *apierror.APIError)
	Get(ctx context.Context, params GetAddressParams) (*Address, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Address, *apierror.APIError)
	Create(ctx context.Context, addressID, geolocationID, accountAddressID string, params CreateAddressParams) (*Address, *apierror.APIError)
	Update(ctx context.Context, params UpdateAddressParams) (*Address, *apierror.APIError)
	Delete(ctx context.Context, params DeleteAddressParams) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, addressID string) (bool, *apierror.APIError)
	GetGeolocationSharedCount(ctx context.Context, geolocationID string) (int64, *apierror.APIError)
	GetGeolocationIDByAddressID(ctx context.Context, addressID string) (string, *apierror.APIError)
	CreateGeolocation(ctx context.Context, id string, params CreateAddressParams) *apierror.APIError
	UpdateGeolocation(ctx context.Context, geolocationID string, params UpdateAddressParams) *apierror.APIError
	RelinkGeolocation(ctx context.Context, addressID, geolocationID string) *apierror.APIError
	CheckAddressNotInUse(ctx context.Context, addressID string) *apierror.APIError
	// SwitchAccountDefaultAddressToRelation realigns any account default billing/shipping pointer at
	// the given address to the account-relation default (owner→this account), falling back to NULL
	// when the relation has no usable default. It keeps a non-active account from being left with a
	// dangling default when the address is deleted (there are no FKs to cascade). Call inside the
	// delete transaction.
	SwitchAccountDefaultAddressToRelation(ctx context.Context, addressID string) *apierror.APIError
}

type AccountStatusRepo interface {
	List(ctx context.Context, params ListAccountStatusesParams) (*ListAccountStatusesResult, *apierror.APIError)
	Get(ctx context.Context, identifier string) (*AccountStatus, *apierror.APIError)
	GetByIDs(ctx context.Context, ids []string) ([]*AccountStatus, *apierror.APIError)
}

type PriorityRepo interface {
	List(ctx context.Context, params ListPrioritiesParams) (*ListPrioritiesResult, *apierror.APIError)
	Get(ctx context.Context, identifier string) (*Priority, *apierror.APIError)
	GetByIDs(ctx context.Context, ids []string) ([]*Priority, *apierror.APIError)
}

type AdjustmentTypeRepo interface {
	List(ctx context.Context, params ListAdjustmentTypesParams) (*ListAdjustmentTypesResult, *apierror.APIError)
	GetByIDs(ctx context.Context, ids []string) ([]*AdjustmentType, *apierror.APIError)
}

type AccountPriceRepo interface {
	List(ctx context.Context, params ListAccountPricesParams) (*ListAccountPricesResult, *apierror.APIError)
	// ResolveRecipientAccountIDs returns the customer plus its parent account, if it has one.
	ResolveRecipientAccountIDs(ctx context.Context, ownerAccountID, customerAccountID string) ([]string, *apierror.APIError)
	Get(ctx context.Context, accountID, accountPriceID string) (*AccountPrice, *apierror.APIError)
	Create(ctx context.Context, accountPriceID, rateID string, params CreateAccountPriceParams) (*AccountPrice, *apierror.APIError)
	Update(ctx context.Context, params UpdateAccountPriceParams) (*AccountPrice, *apierror.APIError)
	Delete(ctx context.Context, accountID, accountPriceID string) *apierror.APIError
}

type AccountIntegrationRepo interface {
	List(ctx context.Context, params ListAccountIntegrationsParams) (*ListAccountIntegrationsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, id string) (*AccountIntegration, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*AccountIntegration, *apierror.APIError)
	FindByCode(ctx context.Context, accountID string, code constants.IntegrationCode) (*AccountIntegration, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateAccountIntegrationParams, encryptedCredentials string) (*AccountIntegration, *apierror.APIError)
	UpdateCredentials(ctx context.Context, accountID, id, name, encryptedCredentials string) (*AccountIntegration, *apierror.APIError)
	Update(ctx context.Context, params UpdateAccountIntegrationParams) (*AccountIntegration, *apierror.APIError)
	Delete(ctx context.Context, params DeleteAccountIntegrationParams) (*AccountIntegration, *apierror.APIError)
	GetEncryptedCredentials(ctx context.Context, accountID string, code constants.IntegrationCode) (credentials string, isActive bool, err *apierror.APIError)
	HasIntegration(ctx context.Context, accountID string, code constants.IntegrationCode) (bool, *apierror.APIError)
}

// HubspotSyncRepo persists the HubSpot backfill state: jobs (state machine), the generic OpenMRP->HubSpot id mapping, and the company-match review queue.
type HubspotSyncRepo interface {
	CreateJob(ctx context.Context, params CreateHubspotSyncJobParams) (*HubspotSyncJob, *apierror.APIError)
	GetJob(ctx context.Context, accountID, id string) (*HubspotSyncJob, *apierror.APIError)
	GetLatestJobForAccount(ctx context.Context, accountID string) (*HubspotSyncJob, *apierror.APIError)
	UpdateJob(ctx context.Context, params UpdateHubspotSyncJobParams) *apierror.APIError
	// ClaimJobForExecute atomically moves a review_pending/failed job to executing, reporting whether this caller won the transition. Losing means another execute already claimed the job.
	ClaimJobForExecute(ctx context.Context, accountID, jobID string) (bool, *apierror.APIError)

	UpsertRecord(ctx context.Context, params UpsertHubspotSyncRecordParams) *apierror.APIError
	GetRecord(ctx context.Context, accountID, augnoType, augnoID string) (*HubspotSyncRecord, *apierror.APIError)
	// ListRecords pages the account's mappings for one OpenMRP type, resolving each entity's display name.
	ListRecords(ctx context.Context, params ListHubspotSyncRecordsParams) (*ListHubspotSyncRecordsResult, *apierror.APIError)

	CreateReview(ctx context.Context, params CreateHubspotCompanyReviewParams) (*HubspotCompanyReview, *apierror.APIError)
	GetReview(ctx context.Context, accountID, id string) (*HubspotCompanyReview, *apierror.APIError)
	// GetReviewsByIDs reads many reviews at once, so a bulk resolution can validate every id it was handed in a single round trip.
	GetReviewsByIDs(ctx context.Context, accountID string, ids []string) ([]*HubspotCompanyReview, *apierror.APIError)
	ListReviewsForJob(ctx context.Context, jobID string, status *string) ([]*HubspotCompanyReview, *apierror.APIError)
	CountPendingReviews(ctx context.Context, jobID string) (int64, *apierror.APIError)
	ResolveReview(ctx context.Context, params ResolveHubspotCompanyReviewParams) *apierror.APIError
}

type SalesTargetRepo interface {
	List(ctx context.Context, params ListSalesTargetsParams) (*ListSalesTargetsResult, *apierror.APIError)
	Get(ctx context.Context, targetID string) (*SalesTarget, *apierror.APIError)
	Exists(ctx context.Context, targetID string) (bool, *apierror.APIError)
	IsInAccount(ctx context.Context, targetID, accountID string) (bool, *apierror.APIError)
	SalesRepExistsInAccount(ctx context.Context, salesRepID, accountID string) (bool, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateSalesTargetParams, amountID string) *apierror.APIError
	Update(ctx context.Context, params UpsertSalesTargetParams) *apierror.APIError
	InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
}

type PropertyRepo interface {
	List(ctx context.Context, params ListPropertiesParams) (*ListPropertiesResult, *apierror.APIError)
	Get(ctx context.Context, params GetPropertyParams) (*Property, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Property, *apierror.APIError)
	Create(ctx context.Context, id string, params CreatePropertyParams) (*Property, *apierror.APIError)
	Update(ctx context.Context, params UpdatePropertyParams) (*Property, *apierror.APIError)
	Delete(ctx context.Context, params DeletePropertyParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	FindByNames(ctx context.Context, accountID string, names []string) ([]*Property, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, propertyID string) (bool, *apierror.APIError)
	DeleteAttributesByPropertyID(ctx context.Context, propertyID, accountID string) *apierror.APIError
	Export(ctx context.Context, params ExportPropertiesParams) ([]*Property, *apierror.APIError)
}

type AttributeRepo interface {
	List(ctx context.Context, params ListAttributesParams) (*ListAttributesResult, *apierror.APIError)
	ListByPropertyIDs(ctx context.Context, accountID string, propertyIDs []string) ([]*Attribute, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Attribute, *apierror.APIError)
	Get(ctx context.Context, params GetAttributeParams) (*Attribute, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateAttributeParams) (*Attribute, *apierror.APIError)
	Update(ctx context.Context, params UpdateAttributeParams) (*Attribute, *apierror.APIError)
	Delete(ctx context.Context, params DeleteAttributeParams) *apierror.APIError
	ExistsByValueInAccount(ctx context.Context, accountID, value string, excludeID *string) (bool, *apierror.APIError)
	FindByTextsInAccount(ctx context.Context, accountID string, texts []string) ([]*AttributeTextMatch, *apierror.APIError)
	CountByProperty(ctx context.Context, propertyID, accountID string) (int64, *apierror.APIError)
	ShiftOrdersUp(ctx context.Context, propertyID, accountID string, fromOrder int32) *apierror.APIError
	ShiftOrdersDown(ctx context.Context, propertyID, accountID string, afterOrder int32) *apierror.APIError
	ShiftOrdersUpBounded(ctx context.Context, propertyID, accountID string, fromOrder, toOrder int32) *apierror.APIError
	ShiftOrdersDownBounded(ctx context.Context, propertyID, accountID string, afterOrder, upToOrder int32) *apierror.APIError
}

type CarrierRepo interface {
	List(ctx context.Context, params ListCarriersParams) (*ListCarriersResult, *apierror.APIError)
	Get(ctx context.Context, params GetCarrierParams) (*Carrier, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Carrier, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateCarrierParams) (*Carrier, *apierror.APIError)
	Update(ctx context.Context, params UpdateCarrierParams) (*Carrier, *apierror.APIError)
	SoftDelete(ctx context.Context, accountID, carrierID string) *apierror.APIError
	DeleteOptionsByCarrierID(ctx context.Context, accountID, carrierID string) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	ListOptionsByCarrierID(ctx context.Context, accountID, carrierID string) ([]*ServiceLevel, *apierror.APIError)
	// ListOptionIDsForCarriers returns all carrier_option IDs grouped by carrier_id, ordered (carrier_id, created_at ASC, id ASC) — callers truncate to a per-carrier preview limit in Go.
	ListOptionIDsForCarriers(ctx context.Context, accountID string, carrierIDs []string) (map[string][]string, *apierror.APIError)
	// GetOptionsByIDs returns full ServiceLevel records by id with the same account-scoping rule as ListOptionsByCarrierID (the parent carrier must be the caller's own or a system carrier).
	GetOptionsByIDs(ctx context.Context, accountID string, ids []string) ([]*ServiceLevel, *apierror.APIError)
}

type CarrierTransitEstimateRepo interface {
	// Resolve returns both transit candidates for a lane in one round trip. A lane whose service level does not exist (or belongs to another account) resolves to nil rather than an error: the caller's fallback is to stamp no transit at all.
	Resolve(ctx context.Context, accountID string, lane TransitLane) (*CarrierTransitCandidates, *apierror.APIError)
	// Upsert writes a harvested estimate, leaving an operator-entered row for the same lane untouched.
	Upsert(ctx context.Context, params UpsertTransitEstimateParams) *apierror.APIError
}

// OperatingCalendarRepo reads and writes the day-sets a commitment is resolved against.
type OperatingCalendarRepo interface {
	Get(ctx context.Context, accountID, calendarID string) (*OperatingCalendar, *apierror.APIError)
	GetByCode(ctx context.Context, accountID, code string) (*OperatingCalendar, *apierror.APIError)
	List(ctx context.Context, params ListOperatingCalendarsParams) ([]OperatingCalendar, *apierror.APIError)
	// ResolveShip returns the calendar the account tenders freight on, preferring an explicitly configured one over the account default. Nil means none is configured, and the caller falls back to Monday-to-Friday.
	ResolveShip(ctx context.Context, accountID string) (*OperatingCalendar, *apierror.APIError)
	// ResolveReceive walks address -> customer -> customer's group -> account setting -> account default in one query. Nil means none is configured anywhere on the chain.
	ResolveReceive(ctx context.Context, query ReceiveCalendarQuery) (*OperatingCalendar, *apierror.APIError)
	// ListClosures returns closures for a set of calendars inside a bounded window, keyed by calendar ID.
	ListClosures(ctx context.Context, query ClosureWindowQuery) (map[string][]OperatingCalendarClosure, *apierror.APIError)
	Create(ctx context.Context, params CreateOperatingCalendarParams) *apierror.APIError
	Update(ctx context.Context, params UpdateOperatingCalendarParams) *apierror.APIError
	// ClearDefault demotes every other calendar of the same kind, so exactly one default survives per kind.
	ClearDefault(ctx context.Context, accountID, kindCode, keepID string) *apierror.APIError
	// Delete is a soft delete: an issued order's commitment was resolved against this calendar, and a hard delete would orphan the links that explain it.
	Delete(ctx context.Context, accountID, calendarID string) *apierror.APIError
	CountReferences(ctx context.Context, accountID, calendarID string) (*OperatingCalendarReferences, *apierror.APIError)
	GetClosure(ctx context.Context, accountID, closureID string) (*OperatingCalendarClosure, *apierror.APIError)
	// GetClosureByDate reads a closure back by the key it is unique on, so a re-closed date returns the row that was already there rather than a discarded ID.
	GetClosureByDate(ctx context.Context, accountID, calendarID string, closedOn time.Time) (*OperatingCalendarClosure, *apierror.APIError)
	// UpsertClosures is idempotent, so re-seeding a year neither duplicates a closure nor renames one an operator has relabelled.
	UpsertClosures(ctx context.Context, closures []UpsertClosureParams) *apierror.APIError
	DeleteClosure(ctx context.Context, accountID, closureID string) *apierror.APIError
}

type ServiceLevelRepo interface {
	List(ctx context.Context, params ListServiceLevelsParams) (*ListServiceLevelsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, serviceLevelID string) (*ServiceLevel, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateServiceLevelParams) (*ServiceLevel, *apierror.APIError)
	Update(ctx context.Context, params UpdateServiceLevelParams) (*ServiceLevel, *apierror.APIError)
	Delete(ctx context.Context, accountID, serviceLevelID string) *apierror.APIError
	IsInCarrier(ctx context.Context, serviceLevelID, carrierID string) (bool, *apierror.APIError)
	ExistsByCodeInCarrier(ctx context.Context, carrierID, code string, excludeID *string) (bool, *apierror.APIError)
	ClearDefaultsForCarrier(ctx context.Context, accountID, carrierID string) *apierror.APIError
}

// BatchRepo handles all batch data access.
type BatchRepo interface {
	Find(ctx context.Context, accountID, batchID string) (*Batch, *apierror.APIError)
	FindBatchFlow(ctx context.Context, accountID, batchID string) ([]BatchFlowNode, *apierror.APIError)
	FindByScanningStation(ctx context.Context, params ListBatchesByScanningStationParams) (*ListBatchesByScanningStationResult, *apierror.APIError)
	FindPossibleNextSteps(ctx context.Context, accountID, scanningStationID, batchID string) ([]ScanningProductionStepInfo, *apierror.APIError)
	FindPossibleInitSteps(ctx context.Context, accountID, scanningStationID, batchID string) ([]ScanningProductionStepInfo, *apierror.APIError)
	FindOpenBatches(ctx context.Context, accountID string, itemIDs, productLineIDs []string) ([]OpenBatchSummary, *apierror.APIError)
	FindFurthestRightBatchInFlow(ctx context.Context, accountID, batchID string) (*BaseBatch, *apierror.APIError)
	FindNextAvailableBatchInFlow(ctx context.Context, accountID, batchID, productionStepID string) (*BaseBatch, *apierror.APIError)
	FindAvailableBatchesInFlow(ctx context.Context, accountID string, batchIDs []string, productionStepID string) ([]BaseBatch, *apierror.APIError)
	FindOutputBatches(ctx context.Context, accountID, batchID string) ([]BaseBatch, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateBatchParams) (*BaseBatch, *apierror.APIError)
	MarkAsScanned(ctx context.Context, accountID, batchID string) *apierror.APIError
	ConnectProductionStep(ctx context.Context, accountID, batchID, productionStepID string) *apierror.APIError
	ConnectScanningStation(ctx context.Context, accountID, batchID, scanningStationID string) *apierror.APIError
	ConnectOneToOne(ctx context.Context, accountID, sourceBatchID, targetBatchID string, autoClose bool) *apierror.APIError
	ConnectManyToOne(ctx context.Context, accountID string, sourceBatchIDs []string, targetBatchID string, autoClose bool) *apierror.APIError
	Close(ctx context.Context, accountID, batchID string) (*BaseBatch, *apierror.APIError)
	CloseIfLastStep(ctx context.Context, accountID, batchID, productionStepID string) *apierror.APIError
	CloseIfFullyUsed(ctx context.Context, accountID string, batch BaseBatch, producedUnit LightUnit, productionStepID string) *apierror.APIError
	Delete(ctx context.Context, accountID, batchID string) (*BaseBatch, *apierror.APIError)
	DeleteMany(ctx context.Context, accountID string, batchIDs []string) *apierror.APIError
	// CountDownstreamBatches reports how many batches were fed by this one. A batch something downstream still feeds on cannot be undone.
	CountDownstreamBatches(ctx context.Context, batchID string) (int64, *apierror.APIError)
	// FindInputBatchIDs returns the batches that fed the given one.
	FindInputBatchIDs(ctx context.Context, batchID string) ([]string, *apierror.APIError)
	// FindLineageShortfall walks up a batch's lineage for the production run it belongs to and the seconds and waste accumulated along the way.
	FindLineageShortfall(ctx context.Context, batchID string) (*LineageShortfall, *apierror.APIError)
	// Unscan returns a batch to the state it was in before it was scanned, leaving the row in place so the production run that created it still holds that unit of work.
	Unscan(ctx context.Context, accountID, batchID string) (*BaseBatch, *apierror.APIError)
	// Reopen clears a batch's closed_at.
	Reopen(ctx context.Context, accountID, batchID string) *apierror.APIError
	// ReassignMachine points a batch at one machine and drops any other machine link it had, for a ticket moved to a campaign running somewhere else.
	ReassignMachine(ctx context.Context, accountID, batchID, machineID string) *apierror.APIError
	// ReopenIfNotFullyUsed reopens a batch that is no longer fully consumed — the mirror of CloseIfFullyUsed, run after a downstream batch is deleted and the quantity it was holding comes back.
	ReopenIfNotFullyUsed(ctx context.Context, accountID string, batch BaseBatch, producedUnit LightUnit, productionStepID string) *apierror.APIError
}

// ProductionStepQueryRepo provides read-only methods the batch service needs from production steps.
type ProductionStepQueryRepo interface {
	Find(ctx context.Context, accountID, id string) (*ProductionStepDetail, *apierror.APIError)
	Export(ctx context.Context, params ExportProductionStepsParams) ([]*ProductionStepExport, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	IsMultiPart(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	IsLastStep(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	IsInputOfStep(ctx context.Context, accountID, currentStepID, inputStepID string) (bool, *apierror.APIError)
	FindProducedItemID(ctx context.Context, accountID, id string) (string, *apierror.APIError)
	FindProducedUnit(ctx context.Context, accountID, id string) (*LightUnit, *apierror.APIError)
	FindIDByScanningStationAndProducedBlock(ctx context.Context, accountID, scanningStationID, itemID string) (string, *apierror.APIError)
	FindOneByScanningStationAndProducedBlock(ctx context.Context, accountID, scanningStationID, itemID string) (*ProductionStepDetail, *apierror.APIError)
	CalculateNextStepQuantities(ctx context.Context, accountID, itemID string, batchQuantity BatchQuantity, stepID string) (*NextStepQuantitiesResult, *apierror.APIError)
}

// ScanningStationQueryRepo provides read-only access to scanning station data.
type ScanningStationQueryRepo interface {
	IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	FindType(ctx context.Context, accountID, id string) (string, *apierror.APIError)
}

// ProductionRunQueryRepo provides limited write access for starting and closing production runs.
type ProductionRunQueryRepo interface {
	Start(ctx context.Context, accountID, id string) *apierror.APIError
	CloseIfAllBatchesScannedOrDeleted(ctx context.Context, accountID, id string) *apierror.APIError
	// Reopen undoes a completion after one of the run's batches goes back to unscanned, clearing started_at as well when nothing in the run is scanned any more.
	Reopen(ctx context.Context, accountID, id string) *apierror.APIError
	Create(ctx context.Context, id, responsibleUserID, number, accountID string) *apierror.APIError
	GetNextNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
}

// ProductionRunRepo provides full CRUD access for production run management.
type ProductionRunRepo interface {
	List(ctx context.Context, params ListProductionRunsParams) (*ListProductionRunsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportProductionRunsParams) ([]*ProductionRunExport, *apierror.APIError)
	Get(ctx context.Context, params GetProductionRunParams) (*ProductionRun, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateProductionRunParams, number string) (*ProductionRun, *apierror.APIError)
	Update(ctx context.Context, params UpdateProductionRunParams) (*ProductionRun, *apierror.APIError)
	Delete(ctx context.Context, params DeleteProductionRunParams) *apierror.APIError
	ExistsByNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError)
	GetNextNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
	// reserves count sequential numbers from a single locked read, for batch writes
	GetNextNumbers(ctx context.Context, accountID string, count int) ([]string, *apierror.APIError)
	IsCompleted(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	DeleteBatchesByRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError
	FindOrderIDsByRun(ctx context.Context, accountID, productionRunID string) ([]string, *apierror.APIError)
	UnlinkOrdersFromRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError
	ListBatchesByRun(ctx context.Context, params ListBatchesByProductionRunParams) (*ListBatchesByProductionRunResult, *apierror.APIError)
	SetBatchProductionRunID(ctx context.Context, accountID, batchID, productionRunID string) *apierror.APIError
}

// UnitGroupQueryRepo provides read-only access to unit group data.
type UnitGroupQueryRepo interface {
	FindByItem(ctx context.Context, accountID, itemID string) (*UnitGroup, *apierror.APIError)
}

// UnitQueryRepo provides read-only access to unit data.
type UnitQueryRepo interface {
	Find(ctx context.Context, accountID, id string) (*LightUnit, *apierror.APIError)
	GetDimensionCodes(ctx context.Context, ids []string) (map[string]string, *apierror.APIError)
}

// InventoryQueryRepo provides read-only access to inventory data.
type InventoryQueryRepo interface {
	FetchCurrentInventory(ctx context.Context, itemID, ownerAccountID string) (*InventorySnapshot, *apierror.APIError)
	FetchOnHandInventoryBulk(ctx context.Context, itemIDs []string, ownerAccountID string) ([]*BulkOnHandInventory, *apierror.APIError)
	FetchPhysicalInventory(ctx context.Context, itemID, ownerAccountID, unitID string) (decimal.Decimal, *apierror.APIError)
	// FetchPhysicalInventoryBaseForItems returns each item's physical inventory in base units, so the batch-scan audit trail can level many items with one query instead of one per item.
	FetchPhysicalInventoryBaseForItems(ctx context.Context, accountID string, itemIDs []string) (map[string]decimal.Decimal, *apierror.APIError)
}

type ProductLineRepo interface {
	List(ctx context.Context, params ListProductLinesParams) (*ListProductLinesResult, *apierror.APIError)
	Export(ctx context.Context, params ExportProductLinesParams) ([]*ProductLineFull, *apierror.APIError)
	Get(ctx context.Context, params GetProductLineParams) (*ProductLineFull, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateProductLineParams) (*ProductLineFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateProductLineParams) (*ProductLineFull, *apierror.APIError)
	Delete(ctx context.Context, params DeleteProductLineParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	FindByNames(ctx context.Context, accountID string, names []string) ([]*ProductLineFull, *apierror.APIError)
	GetUnitGroup(ctx context.Context, accountID, unitGroupID string, includes []string) (*ProductLineUnitGroup, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*ProductLineFull, *apierror.APIError)
	// IsUnitInGroup reports whether a unit can be used by a line on this unit group. A lot counted in a unit the line cannot express is not a lot anybody can act on.
	IsUnitInGroup(ctx context.Context, unitGroupID, unitID string) (bool, *apierror.APIError)

	// Lot conventions. These back both the solver's resolution and the one-item lookup a person gets when adding a batch by hand.
	GetItemLotOverride(ctx context.Context, accountID, itemID string) (float64, *apierror.APIError)
	// GetProductLineLotForItem returns the convention of the line an item sells under, or nil for an intermediate item that is not itself sold.
	GetProductLineLotForItem(ctx context.Context, accountID, itemID string) (*ProductLineLotDefault, *apierror.APIError)
	// GetDownstreamProductLineLot returns the convention an intermediate item inherits from what it becomes, highest-demand line first.
	GetDownstreamProductLineLot(ctx context.Context, accountID, itemID string) (*ProductLineLotDefault, *apierror.APIError)
	// GetFlowProductLineLot walks the production flow from an intermediate item to the first thing it becomes that has a lot convention. It answers for an item that has never been produced, which the demand-weighted lookup cannot.
	GetFlowProductLineLot(ctx context.Context, accountID, itemID string, maxDepth int) (*ProductLineLotDefault, *apierror.APIError)
}

type ItemCategoryRepo interface {
	List(ctx context.Context, params ListItemCategoriesParams) (*ListItemCategoriesResult, *apierror.APIError)
	Export(ctx context.Context, params ExportItemCategoriesParams) ([]*ItemCategoryFull, *apierror.APIError)
	Get(ctx context.Context, params GetItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)
	UpdateWithUnitGroup(ctx context.Context, params UpdateItemCategoryWithUnitGroupParams) (*ItemCategoryFull, *apierror.APIError)
	Delete(ctx context.Context, params DeleteItemCategoryParams) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, itemCategoryID string) (bool, *apierror.APIError)
	FindByNames(ctx context.Context, accountID string, names []string) ([]*ItemCategoryFull, *apierror.APIError)
	AddProperty(ctx context.Context, params AddItemCategoryPropertyParams) *apierror.APIError
	UpsertProperty(ctx context.Context, itemCategoryID, propertyID string) *apierror.APIError
	RemoveProperty(ctx context.Context, params RemoveItemCategoryPropertyParams) *apierror.APIError
	ChangeUnitGroup(ctx context.Context, params ChangeItemCategoryUnitGroupParams) *apierror.APIError
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*ItemCategoryFull, *apierror.APIError)
	GetProperties(ctx context.Context, itemCategoryID string) ([]*ItemCategoryProperty, *apierror.APIError)
	GetUnitGroup(ctx context.Context, unitGroupID string, includes []string) (*ItemCategoryUnitGroup, *apierror.APIError)
	IsPropertyInAccount(ctx context.Context, accountID, propertyID string) (bool, *apierror.APIError)
	PropertyExistsByNameInCategory(ctx context.Context, accountID, itemCategoryID, name string, excludePropertyID *string) (bool, *apierror.APIError)
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}

type ConsumptionRepo interface {
	Get(ctx context.Context, accountID, productionStepID, consumptionID string) (*Consumption, *apierror.APIError)
	Create(ctx context.Context, consumptionID, quantityID, wasteQuantityID string, params CreateConsumptionParams) (*Consumption, *apierror.APIError)
	UpdateItem(ctx context.Context, accountID, consumptionID, itemID string, instructions *string) *apierror.APIError
	UpdateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError
	Delete(ctx context.Context, accountID, consumptionID string) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, consumptionID string) (bool, *apierror.APIError)
	GetQuantityIDs(ctx context.Context, consumptionID string) (quantityID string, wasteQuantityID string, apiErr *apierror.APIError)
	InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	DeleteQuantity(ctx context.Context, id string) *apierror.APIError
	GetItemID(ctx context.Context, consumptionID string) (string, *apierror.APIError)
	GetInstructions(ctx context.Context, consumptionID string) (*string, *apierror.APIError)
}

// ProductionStepRepo provides CRUD access to production step data.
type ProductionStepRepo interface {
	List(ctx context.Context, params ListProductionStepsParams) (*ListProductionStepsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, id string) (*ProductionStep, *apierror.APIError)
	InsertStep(ctx context.Context, id, name string, notes *string, levelingFactor, allowances, laborRateID, laborTimeID, overheadRateID string, scanningStationID, departmentID *string, accountID string) *apierror.APIError
	Update(ctx context.Context, params UpdateProductionStepParams) *apierror.APIError
	Delete(ctx context.Context, accountID, id string) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	FindIDByName(ctx context.Context, accountID, name string) (*string, *apierror.APIError)
	DeleteParentChildLinks(ctx context.Context, id string) *apierror.APIError
	GetInputSteps(ctx context.Context, id string) ([]LightProductionStep, *apierror.APIError)
	GetOutputSteps(ctx context.Context, id string) ([]LightProductionStep, *apierror.APIError)
	GetMachines(ctx context.Context, id string) ([]LightMachine, *apierror.APIError)
	InsertRate(ctx context.Context, id string, params CreateRateParams) *apierror.APIError
	InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	InsertProduction(ctx context.Context, id, itemID, quantityID, productionStepID string) *apierror.APIError
	DeleteConsumptionsByStepID(ctx context.Context, stepID string) *apierror.APIError
	DeleteProductionsByStepID(ctx context.Context, stepID string) *apierror.APIError
	UpdateStepFull(ctx context.Context, id, accountID, levelingFactor, allowances string, scanningStationID *string) *apierror.APIError
	// FindByNames resolves existing production steps by name (case-insensitive) in
	// one query. Names must be pre-lowercased by the caller.
	FindByNames(ctx context.Context, accountID string, names []string) ([]*ProductionStepBulkRow, *apierror.APIError)
	// UpdateForBulkUpsert writes the full step row for a bulk upsert update; the rate
	// IDs point at freshly inserted rate rows.
	UpdateForBulkUpsert(ctx context.Context, params UpdateProductionStepForBulkUpsertParams) *apierror.APIError
}

// ProductionRepo provides CRUD access to production (output) data.
type ProductionRepo interface {
	Get(ctx context.Context, accountID, productionStepID, productionID string) (*Production, *apierror.APIError)
	UpdateItem(ctx context.Context, productionID, itemID string) *apierror.APIError
	UpdateQuantity(ctx context.Context, productionID, value, unitID string) *apierror.APIError
	GetQuantityID(ctx context.Context, productionID string) (string, *apierror.APIError)
}

// InventoryMutationRepo handles inventory receipt and issue creation for production step execution.
type InventoryMutationRepo interface {
	// UpdateInventory creates an inventory receipt (positive measure) or issue (negative measure) for the given item. This is the core inventory mutation used by the executeProductionStep consumer.
	UpdateInventory(ctx context.Context, params InventoryUpdateParams) *apierror.APIError
	// CreateInventoryReceipt creates an inventory receipt for positive delta.
	CreateInventoryReceipt(ctx context.Context, scope *ledgerlock.Scope, params CreateInventoryReceiptParams) *apierror.APIError
	// CreateInventoryIssue creates an inventory issue for negative delta.
	CreateInventoryIssue(ctx context.Context, scope *ledgerlock.Scope, params CreateInventoryIssueParams) *apierror.APIError
	// CreateInventoryLog creates a point-in-time inventory snapshot log.
	CreateInventoryLog(ctx context.Context, params CreateInventoryLogParams) *apierror.APIError
	// CreateInventoryChangeLog creates an audit trail entry for an inventory change.
	CreateInventoryChangeLog(ctx context.Context, params CreateInventoryChangeLogParams) *apierror.APIError
	// CreateQuantityForInventory creates a quantity record for use in inventory operations.
	CreateQuantityForInventory(ctx context.Context, quantityID, value, unitID string) *apierror.APIError
	// CreateRateForInventory creates a rate record for use in inventory operations.
	CreateRateForInventory(ctx context.Context, rateID, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	// LockItemForLedger takes the item's ordering root. Callers do not call it directly: ledgerlock.Acquire does, as the first statement of a ledger-writing transaction. See docs/patterns/architecture-patterns.md, "Inventory ledger lock order".
	LockItemForLedger(ctx context.Context, itemID string) *apierror.APIError
	// ListItemIDsForBatchReversal names every item a batch's reversal will write, so the caller can take their ordering roots before opening the transaction that writes them. Non-locking; it decides nothing.
	ListItemIDsForBatchReversal(ctx context.Context, accountID, batchID string) ([]string, *apierror.APIError)
	// ReverseInventoryForBatch undoes every inventory movement a scan recorded against a batch and returns the corrections it made, so the caller can write the audit trail and request allocation. Refuses when the batch's output has already been drawn on, since reversing it would drive inventory negative.
	ReverseInventoryForBatch(ctx context.Context, scope *ledgerlock.Scope, params ReverseInventoryForBatchParams) ([]InventoryReversalDelta, *apierror.APIError)
	// CountAllocatedReceiptsForBatch reports how many of a batch's produced receipts have already been drawn against. Used as a pre-flight guard before a batch is deleted.
	CountAllocatedReceiptsForBatch(ctx context.Context, accountID, batchID string) (int64, *apierror.APIError)
	// ReverseInventoryForOrderItem hands a consumed measure back to the order's reservation, walking the issues it opened newest first and splitting the last one when it overshoots. The caller requests allocation so the freed receipts can cover other open issues.
	ReverseInventoryForOrderItem(ctx context.Context, scope *ledgerlock.Scope, accountID, orderID, itemID string, measure decimal.Decimal) *apierror.APIError
}

// OrderQueryRepo provides read-only queries for orders needed by the batch/production system.
type OrderQueryRepo interface {
	// FindIDByProductionRun returns the order ID for the given production run, or nil if no order exists.
	FindIDByProductionRun(ctx context.Context, accountID, productionRunID string) (*string, *apierror.APIError)
}

// InventoryReservationRepo manages inventory reservations for orders during production step execution.
type InventoryReservationRepo interface {
	// CreateMaterialReservation creates a reserved inventory issue for a material demand linked to an order.
	CreateMaterialReservation(ctx context.Context, scope *ledgerlock.Scope, params CreateMaterialReservationParams) *apierror.APIError
	// ReduceReservedForOrderItem reduces the reserved quantity for an order item by the given shortfall amount.
	ReduceReservedForOrderItem(ctx context.Context, scope *ledgerlock.Scope, params OrderReservationReductionParams) *apierror.APIError
	// ReduceReservedForOrderMaterials reduces reserved quantities for upstream materials of an order.
	ReduceReservedForOrderMaterials(ctx context.Context, scope *ledgerlock.Scope, orderID, accountID string, demands []MaterialDemandItem) *apierror.APIError
	// AllocateReservationsForConsumption allocates existing reservations for consumed materials. Returns the remaining quantity that could not be allocated from reservations.
	AllocateReservationsForConsumption(ctx context.Context, scope *ledgerlock.Scope, params ConsumptionAllocationParams) (*ConsumptionAllocationResult, *apierror.APIError)
	// LockItemForLedger takes the item's ordering root. Callers do not call it directly: ledgerlock.Acquire does, as the first statement of a ledger-writing transaction. See docs/patterns/architecture-patterns.md, "Inventory ledger lock order".
	LockItemForLedger(ctx context.Context, itemID string) *apierror.APIError
	// ListOpenIssueIDsForItem names one page of the item's open demand (up to limit, oldest first, resuming after the (afterCreatedAt, afterID) cursor). It takes no locks and decides nothing: every id it returns is re-read under FOR UPDATE by AllocateOneOpenIssue.
	ListOpenIssueIDsForItem(ctx context.Context, accountID, itemID string, afterCreatedAt time.Time, afterID string, limit int32) ([]OpenIssueRef, *apierror.APIError)
	// CountAvailableReceiptsForItem reports how many receipts the item has to draw on, so an uncoverable backlog costs one read rather than a transaction per issue.
	CountAvailableReceiptsForItem(ctx context.Context, accountID, itemID string) (int64, *apierror.APIError)
	// AllocateOneOpenIssue covers one open issue against available receipts. Each call is meant to be its own transaction; the issue is re-read by primary key under FOR UPDATE and skipped if it is no longer open.
	AllocateOneOpenIssue(ctx context.Context, scope *ledgerlock.Scope, accountID, itemID, issueID string) *apierror.APIError
	// ListReservedItemIDsForOrders names the items the given orders hold reservations on, so a release can take their ordering root as its transaction's first statement. Read on the pool, before the transaction opens.
	ListReservedItemIDsForOrders(ctx context.Context, accountID string, orderIDs []string) ([]string, *apierror.APIError)
	// ReleaseReservedIssuesForOrder deletes an order's reservations along with the allocations covering them, returning the receipts those allocations were holding down to `available`. Returns the items it touched, whose open demand the caller must enqueue allocation for after committing.
	ReleaseReservedIssuesForOrder(ctx context.Context, scope *ledgerlock.Scope, accountID, orderID string) ([]string, *apierror.APIError)
}

// MaterialDemandRepo calculates material demand from a bill of materials.
type MaterialDemandRepo interface {
	// GetMaterialDemand calculates the material demand for producing the given items.
	GetMaterialDemand(ctx context.Context, accountID string, productItemID string, measure decimal.Decimal, unitID string) ([]MaterialDemandItem, *apierror.APIError)
	// GetMaterialDemandForOrder calculates the aggregated material demand across a set of order lines (one entry per material).
	GetMaterialDemandForOrder(ctx context.Context, accountID string, lines []MaterialDemandLineInput) ([]MaterialDemandItem, *apierror.APIError)
}

// UnitConversionRepo provides unit conversion capabilities.
type UnitConversionRepo interface {
	// ConvertValue converts a measure from one unit to another within the same unit group. Returns the converted measure.
	ConvertValue(ctx context.Context, measure decimal.Decimal, fromUnitID, toUnitID string) (decimal.Decimal, *apierror.APIError)
	// GetUnitFactors returns the base-conversion factors for each requested unit ID. Unknown IDs are omitted.
	GetUnitFactors(ctx context.Context, accountID string, unitIDs []string) (map[string]UnitFactors, *apierror.APIError)
}

type ProductionFlowRepo interface {
	LinkFlow(ctx context.Context, productionStepID, accountID string) *apierror.APIError
	DisconnectSteps(ctx context.Context, sourceID, targetID string) *apierror.APIError
	FindSourceStepsByConsumption(ctx context.Context, productionStepID, consumptionID, accountID string) ([]string, *apierror.APIError)
	FindDownstreamStepByItem(ctx context.Context, productionStepID, itemID, accountID string) (*string, *apierror.APIError)
	GetAllStepEdgesForAccount(ctx context.Context, accountID string) ([]StepEdge, *apierror.APIError)
	ConnectStepsIdempotent(ctx context.Context, sourceID, targetID string) *apierror.APIError
	GetFlowStep(ctx context.Context, accountID, stepID string) (*ProductionFlowStep, *apierror.APIError)
	FindStepsByProducedItem(ctx context.Context, accountID, itemID string) ([]string, *apierror.APIError)
}

type CustomerRepo interface {
	List(ctx context.Context, params ListCustomersParams) (*ListCustomersResult, *apierror.APIError)
	Get(ctx context.Context, ownerAccountID, customerAccountID string, includes []string) (*Customer, *apierror.APIError)
	Create(ctx context.Context, accountID, relationID, brandingID string, params CreateCustomerParams, customerNumber string) (*Customer, *apierror.APIError)
	Update(ctx context.Context, relationID string, params UpdateCustomerParams) *apierror.APIError
	UpdateName(ctx context.Context, customerAccountID, name string) *apierror.APIError
	UpdateBranding(ctx context.Context, customerAccountID string, email, phone, url *string) *apierror.APIError
	Delete(ctx context.Context, ownerAccountID, customerAccountID string) *apierror.APIError
	BulkDelete(ctx context.Context, ownerAccountID string, customerIDs []string) *apierror.APIError
	IsCommissionExempt(ctx context.Context, ownerAccountID, customerAccountID string) (bool, *apierror.APIError)
	ExistsByNumber(ctx context.Context, ownerAccountID, number string, excludeID *string) (bool, *apierror.APIError)
	// AllocateNextCustomerNumber reserves the account's next customer number in one locked statement, so two registrations landing together cannot be handed the same one.
	AllocateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string) (int64, *apierror.APIError)
	InsertPriceGroup(ctx context.Context, id, relationID, groupID string) *apierror.APIError
	DeletePriceGroups(ctx context.Context, relationID string) *apierror.APIError
	GetFrequentlyOrderedProducts(ctx context.Context, ownerAccountID, customerAccountID string) ([]*FrequentlyOrderedProduct, *apierror.APIError)
	GetRelationID(ctx context.Context, ownerAccountID, customerAccountID string) (string, *apierror.APIError)
	MergeOrders(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeInvoices(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeShipments(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeDeliveries(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeTransactions(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeAccountPrices(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeInventoryReceipts(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeReceivingOrders(ctx context.Context, ownerAccountID, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	MergeInventoryIssues(ctx context.Context, targetAccountID string, sourceAccountIDs []string) *apierror.APIError
	DeleteNotificationPreferences(ctx context.Context, relationIDs []string) *apierror.APIError
	DeleteProductLineAccess(ctx context.Context, relationIDs []string) *apierror.APIError
	GetRelationPriceGroupIDs(ctx context.Context, relationID string) ([]string, *apierror.APIError)
	GetRelationsPriceGroups(ctx context.Context, relationIDs []string) ([]RelationPriceGroup, *apierror.APIError)
	MoveRelationPriceGroups(ctx context.Context, targetRelationID string, ids []string) *apierror.APIError
	DeletePriceGroupsByIDs(ctx context.Context, ids []string) *apierror.APIError
	GetRelationProductLineIDs(ctx context.Context, relationID string) ([]string, *apierror.APIError)
	GetRelationsProductLines(ctx context.Context, relationIDs []string) ([]RelationProductLine, *apierror.APIError)
	MoveRelationProductLines(ctx context.Context, targetRelationID string, ids []string) *apierror.APIError
	DeleteProductLinesByIDs(ctx context.Context, ids []string) *apierror.APIError
	ReparentChildRelations(ctx context.Context, targetRelationID string, sourceRelationIDs []string) *apierror.APIError
	GetAccountAddressIDs(ctx context.Context, accountID string) ([]string, *apierror.APIError)
	InsertAccountAddress(ctx context.Context, id, accountID, addressID string) *apierror.APIError
	DeleteAccountAddresses(ctx context.Context, accountID string) *apierror.APIError
	GetAccountUsers(ctx context.Context, accountID string) ([]AccountUserRef, *apierror.APIError)
	MoveAccountUsers(ctx context.Context, targetAccountID string, ids []string) *apierror.APIError
	DeleteAccountUsers(ctx context.Context, accountID string) *apierror.APIError
	GetStripeCustomerID(ctx context.Context, ownerAccountID, customerAccountID string) (stripeCustomerID *string, stripeEmail *string, err *apierror.APIError)
	SetStripeCustomerID(ctx context.Context, ownerAccountID, customerAccountID, stripeCustomerID, stripeEmail string) *apierror.APIError
	GetCustomerEmail(ctx context.Context, customerAccountID string) (*string, *apierror.APIError)
	InsertCreditLimitQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	UpdateCreditLimitQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	DeleteCreditLimitQuantity(ctx context.Context, id string) *apierror.APIError
}

type AnalyticsRepo interface {
	GetSalesEntries(ctx context.Context, params AnalyzeSalesParams) ([]SalesEntry, *apierror.APIError)
	GetOpenBatchEntries(ctx context.Context, params AnalyzeOpenBatchesParams) ([]OpenBatchEntry, *apierror.APIError)
	GetProductionCostEntries(ctx context.Context, params AnalyzeProductionCostsParams) ([]ProductionCostEntry, *apierror.APIError)
	GetDeliveryAnalytics(ctx context.Context, params AnalyzeDeliveriesParams) (*DeliveryAnalyticsResult, *apierror.APIError)
	GetManufacturingMetric(ctx context.Context, params AnalyzeManufacturingParams) (float64, *apierror.APIError)
	GetManufacturingBatch(ctx context.Context, params AnalyzeManufacturingBatchParams) (*ManufacturingBatchResult, *apierror.APIError)
	GetOrderEntries(ctx context.Context, params AnalyzeOrdersParams) ([]OrderEntry, *apierror.APIError)
	GetQuarterlyOrders(ctx context.Context, params AnalyzeQuarterlyOrdersParams) ([]YearlyQuarterlyData, *apierror.APIError)
	GetMaterialAnalytics(ctx context.Context, params AnalyzeMaterialsParams) ([]MaterialAnalyticsEntry, *apierror.APIError)
	GetInventoryReceiptAnalytics(ctx context.Context, params AnalyzeInventoryReceiptsParams) ([]InventoryReceiptEntry, *apierror.APIError)
	GetNewCustomerEntries(ctx context.Context, params GetNewCustomersAnalyticsParams) ([]NewCustomerEntry, *apierror.APIError)
	GetDemandForecastMonthlyDemand(ctx context.Context, params GetDemandForecastWindowParams) ([]DemandForecastMonthlyDemandRow, *apierror.APIError)
	GetDemandForecastMonthlyRevenue(ctx context.Context, params GetDemandForecastWindowParams) ([]DemandForecastMonthlyRevenueRow, *apierror.APIError)
	GetOeeDepartmentData(ctx context.Context, params GetOeeWindowParams) ([]OeeDepartmentDataRow, *apierror.APIError)
	GetOeeEstimatedRuntime(ctx context.Context, params GetOeeWindowParams) ([]OeeEstimatedRuntimeRow, *apierror.APIError)
	GetOeeEstimatedRuntimeForMachines(ctx context.Context, params GetOeeWindowParams) ([]OeeEstimatedRuntimeRow, *apierror.APIError)
	GetOeeTrendEstimatedRuntimeForMachines(ctx context.Context, params GetOeeWindowParams) ([]OeeTrendEstimatedRuntimeRow, *apierror.APIError)
	GetOeeDowntimeByDepartment(ctx context.Context, params GetOeeWindowParams) ([]OeeDowntimeRow, *apierror.APIError)
	GetOeeTrendDepartmentDataByWeek(ctx context.Context, params GetOeeWindowParams) ([]OeeTrendDepartmentWeekRow, *apierror.APIError)
	GetOeeTrendDowntimeIntervals(ctx context.Context, params GetOeeWindowParams) ([]OeeDowntimeIntervalRow, *apierror.APIError)
	CountMachinesByDepartment(ctx context.Context, accountID string) ([]DepartmentMachineCountRow, *apierror.APIError)
	GetSaleProductItemIDs(ctx context.Context, accountID string) ([]SaleProductItemRow, *apierror.APIError)
	GetProductLineInfo(ctx context.Context, accountID string, productLineIDs []string) ([]ProductLineInfoRow, *apierror.APIError)
	GetOrderQuantityByProductLine(ctx context.Context, params GetOrderQuantityByProductLineParams) (*OrderQuantityByProductLineRow, *apierror.APIError)
}

// MachineStatusRepo reads the raw pieces the floor-status view is assembled from.
type MachineStatusRepo interface {
	// ListMachinesForStatus returns every machine that can carry work, whether or not the plan has given it any, ordered by name then id.
	ListMachinesForStatus(ctx context.Context, accountID string) ([]MachineForStatusRow, *apierror.APIError)
	// ListOpenDowntimeForStatus returns the machines that are down right now — one row per machine, as the open-event guard enforces on write.
	ListOpenDowntimeForStatus(ctx context.Context, accountID string) ([]OpenDowntimeForStatusRow, *apierror.APIError)
	// ListScheduleLinesForStatus returns a published schedule's lines from the given week forward, with per-campaign scan progress, ordered by machine, week, sequence, id.
	ListScheduleLinesForStatus(ctx context.Context, params ListScheduleLinesForStatusParams) ([]ScheduleLineForStatusRow, *apierror.APIError)
}

type MachineRepo interface {
	List(ctx context.Context, params ListMachinesParams) (*ListMachinesResult, *apierror.APIError)
	// Export returns every matching machine up to params.Limit, unpaginated.
	Export(ctx context.Context, params ExportMachinesParams) ([]*Machine, *apierror.APIError)
	Get(ctx context.Context, params GetMachineParams) (*Machine, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Machine, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateMachineParams) (*Machine, *apierror.APIError)
	Update(ctx context.Context, params UpdateMachineParams) (*Machine, *apierror.APIError)
	Delete(ctx context.Context, params DeleteMachineParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	// FindByNames resolves existing machines by name (case-insensitive) in one query.
	// Used by bulk upsert; names must be pre-lowercased by the caller.
	FindByNames(ctx context.Context, accountID string, names []string) ([]*Machine, *apierror.APIError)
	// ExistsBySerialNumber reports whether another machine in the account already
	// uses the serial number (case-insensitive), optionally excluding one machine.
	ExistsBySerialNumber(ctx context.Context, accountID, serialNumber string, excludeID *string) (bool, *apierror.APIError)
	// FindBySerialNumbers resolves existing machines by serial number
	// (case-insensitive) in one query.
	FindBySerialNumbers(ctx context.Context, accountID string, serialNumbers []string) ([]*Machine, *apierror.APIError)
}

type ProductionScheduleRepo interface {
	NextVersion(ctx context.Context, accountID string) (int32, *apierror.APIError)
	Create(ctx context.Context, schedule *ProductionSchedule) *apierror.APIError
	CreateLines(ctx context.Context, accountID, scheduleID string, lines []*ProductionScheduleLine) *apierror.APIError
	CreateItemPolicies(ctx context.Context, accountID, scheduleID string, policies []*ProductionScheduleItemPolicy) *apierror.APIError
	// ReplaceFinishedPolicies rewrites a version's finished-goods targets — the per-SKU decomposition of the pooled greige buffers.
	ReplaceFinishedPolicies(ctx context.Context, accountID, scheduleID string, policies []*ProductionScheduleFinishedPolicy) *apierror.APIError
	ListFinishedPolicies(ctx context.Context, accountID, scheduleID string) ([]*ProductionScheduleFinishedPolicy, *apierror.APIError)
	// ReplaceFinishingLines rewrites a version's stage-two plan. Wholesale rather than patched: the finished mix is a pure function of the knit plan, the order book and each SKU's position, so a partial update could leave a week holding lines for a campaign the re-solve no longer produces.
	ReplaceFinishingLines(ctx context.Context, accountID, scheduleID string, lines []*ProductionScheduleFinishingLine) *apierror.APIError
	ListFinishingLines(ctx context.Context, params ListProductionScheduleFinishingLinesParams) ([]*ProductionScheduleFinishingLine, *apierror.APIError)
	DeleteFinishingLines(ctx context.Context, accountID, scheduleID string) *apierror.APIError
	// DeleteItemPolicies clears a version's policy snapshot. A regenerate re-solves the same version, and the snapshot describes one solve rather than an accumulation.
	DeleteItemPolicies(ctx context.Context, accountID, scheduleID string) *apierror.APIError
	Get(ctx context.Context, params GetProductionScheduleParams) (*ProductionSchedule, *apierror.APIError)
	// GetCurrent returns the published version covering the date, or nil when none does — an account with no live plan is a normal state, not an error.
	GetCurrent(ctx context.Context, accountID string, asOf time.Time) (*ProductionSchedule, *apierror.APIError)
	List(ctx context.Context, params ListProductionSchedulesParams) (*ListProductionSchedulesResult, *apierror.APIError)
	ListLines(ctx context.Context, params ListProductionScheduleLinesParams) ([]*ProductionScheduleLine, *apierror.APIError)
	ListItemPolicies(ctx context.Context, accountID, scheduleID string) ([]*ProductionScheduleItemPolicy, *apierror.APIError)
	Delete(ctx context.Context, accountID, scheduleID string) *apierror.APIError

	// Lifecycle: hand edits, the deviation log, and publish.
	ListScheduleDeviationTypes(ctx context.Context) ([]*ScheduleDeviationType, *apierror.APIError)
	GetLine(ctx context.Context, accountID, lineID string) (*ProductionScheduleLine, *apierror.APIError)
	UpdateLine(ctx context.Context, params UpdateLineRepoParams) (*ProductionScheduleLine, *apierror.APIError)
	DeleteLine(ctx context.Context, accountID, lineID string) *apierror.APIError
	NextSequenceIndex(ctx context.Context, accountID, scheduleID string, weekIndex int32) (int32, *apierror.APIError)
	CreateDeviation(ctx context.Context, id string, deviation *ProductionScheduleDeviation) *apierror.APIError
	ListDeviations(ctx context.Context, params ListProductionScheduleDeviationsParams) (*ListProductionScheduleDeviationsResult, *apierror.APIError)
	// SumFrozenLines returns the counts captured onto the version at publish. They are snapshotted rather than recomputed, so adherence keeps its original denominator.
	SumFrozenLines(ctx context.Context, accountID, scheduleID string, frozenThrough time.Time) (*FrozenLineTotals, *apierror.APIError)
	FreezeLines(ctx context.Context, accountID, scheduleID string, frozenThrough time.Time) *apierror.APIError
	Publish(ctx context.Context, accountID, scheduleID string, frozenThrough time.Time, totals *FrozenLineTotals, publishedByID *string) *apierror.APIError
	ListPublishedOverlapping(ctx context.Context, accountID, excludeID string, start, end time.Time) ([]string, *apierror.APIError)
	Supersede(ctx context.Context, accountID, scheduleID, supersededByID string) *apierror.APIError
	SetStatus(ctx context.Context, accountID, scheduleID, statusCode string) *apierror.APIError

	// Releasing a week to the floor.
	CountReleasedLinesForWeek(ctx context.Context, accountID, scheduleID string, weekIndex int32) (*WeekReleaseState, *apierror.APIError)
	// UnreleaseLinesForRun returns a week to planned when the run holding its work is deleted, so it can be released again.
	UnreleaseLinesForRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError
	// MarkLineReleased links a campaign to the run now carrying it. It is a no-op on a line that is already released, so a racing double release cannot re-point work.
	MarkLineReleased(ctx context.Context, accountID, lineID, productionRunID string) *apierror.APIError
	// ListCarryForwardBatches returns an item's unworked tickets from weeks that have already begun, oldest first, so a release can move them rather than print their replacements.
	ListCarryForwardBatches(ctx context.Context, params ListCarryForwardBatchesParams) ([]*CarryForwardBatch, *apierror.APIError)

	// Generation cadence.
	ListGenerationCadences(ctx context.Context) ([]GenerationCadence, *apierror.APIError)
	StampGenerationRun(ctx context.Context, accountID string, at time.Time) *apierror.APIError
	ReapStalledGenerations(ctx context.Context, before time.Time) *apierror.APIError
	CreateGeneratingSchedule(ctx context.Context, params CreateGeneratingScheduleParams) *apierror.APIError
	FillGeneratedSchedule(ctx context.Context, schedule *ProductionSchedule) *apierror.APIError
	// RefreshRegenerated re-stamps a draft with the metadata of the solve that just replaced its lines.
	RefreshRegenerated(ctx context.Context, schedule *ProductionSchedule) *apierror.APIError
	FailGeneration(ctx context.Context, accountID, scheduleID, reason string) *apierror.APIError

	// Merchant-editable planning assumptions.
	GetSettings(ctx context.Context, accountID string) (*ProductionScheduleSettings, *apierror.APIError)

	// ReplaceLineOrders rewrites which campaigns are building which orders for one version.
	ReplaceLineOrders(ctx context.Context, accountID, scheduleID string, links []CreateLineOrderParams) *apierror.APIError
	// ListLineOrders returns which campaigns are building which orders for one version.
	ListLineOrders(ctx context.Context, accountID, scheduleID string) ([]*ProductionScheduleLineOrder, *apierror.APIError)

	// ListItemSettings returns every per-item planning override in the account.
	ListItemSettings(ctx context.Context, accountID string) ([]*ProductionScheduleItemPlanningSetting, *apierror.APIError)
	// GetItemSetting returns one item's override, or nil when it has none.
	GetItemSetting(ctx context.Context, accountID, itemID string) (*ProductionScheduleItemPlanningSetting, *apierror.APIError)
	// UpsertItemSetting writes one item's override.
	UpsertItemSetting(ctx context.Context, params UpsertItemSettingParams) *apierror.APIError
	// DeleteItemSetting removes one item's override; false when there was none.
	DeleteItemSetting(ctx context.Context, accountID, itemID string) (bool, *apierror.APIError)
	UpsertSettings(ctx context.Context, settings *ProductionScheduleSettings) *apierror.APIError
	ListResourceSettings(ctx context.Context, accountID string) ([]*ProductionScheduleResourceSetting, *apierror.APIError)
	UpsertResourceSetting(ctx context.Context, id string, params UpsertResourceSettingParams) *apierror.APIError
	DeleteResourceSetting(ctx context.Context, accountID, settingID string) *apierror.APIError

	// Derived department work.
	LoadStepGraph(ctx context.Context, accountID string) (*StepGraph, *apierror.APIError)
	ReplaceDerivedLines(ctx context.Context, accountID, scheduleID string, lines []*ProductionScheduleDerivedLine) *apierror.APIError
	ListDerivedLines(ctx context.Context, params ListDerivedLinesParams) ([]*ProductionScheduleDerivedLine, *apierror.APIError)
}

// ScheduleAttainmentRepo is the thin read surface behind schedule attainment. Each method is one query mapped to domain rows; choosing the baseline that was live for a week — and every ratio built on that choice — lives in the analytics service.
type ScheduleAttainmentRepo interface {
	// SelectAttainmentBaselines returns every published version whose horizon overlaps the window, newest publish first.
	SelectAttainmentBaselines(ctx context.Context, params SelectAttainmentBaselinesParams) ([]AttainmentBaselineRow, *apierror.APIError)

	// SumPlannedByWeek returns planned quantity and run hours per (week, machine, item) for one baseline version.
	SumPlannedByWeek(ctx context.Context, params SumPlannedByWeekParams) ([]AttainmentPlannedRow, *apierror.APIError)

	// SumScheduledHoursByDepartmentWeek returns scheduled machine time (run + changeover) per department per week for one baseline version — the denominator OEE availability is measured against.
	SumScheduledHoursByDepartmentWeek(ctx context.Context, params SumPlannedByWeekParams) ([]ScheduledHoursRow, *apierror.APIError)

	// SumActualsByWeek returns what was actually produced, bucketed to the Monday of the scan week so it lines up with a schedule line's week_start_date. An unscanned batch was never produced, so it is excluded.
	SumActualsByWeek(ctx context.Context, params SumActualsByWeekParams) ([]AttainmentActualRow, *apierror.APIError)

	// CountDeviationsForBaselines counts frozen-week changes per baseline version, which is the numerator of frozen adherence.
	CountDeviationsForBaselines(ctx context.Context, accountID string, scheduleIDs []string) ([]AttainmentDeviationRow, *apierror.APIError)

	// GetMachineLabels returns machine names for the given ids.
	GetMachineLabels(ctx context.Context, accountID string, ids []string) ([]AttainmentLabelRow, *apierror.APIError)

	// GetDepartmentLabels returns department names for the given ids.
	GetDepartmentLabels(ctx context.Context, accountID string, ids []string) ([]AttainmentLabelRow, *apierror.APIError)

	// GetItemLabels returns item SKUs for the given ids.
	GetItemLabels(ctx context.Context, accountID string, ids []string) ([]AttainmentLabelRow, *apierror.APIError)
}

// ProductionScheduleInputRepo is the thin read surface behind solver-input assembly. Each method is one query mapped to domain or scheduling types; the assembly itself — genealogy attribution, demand pooling, settings defaulting — lives in the production schedule service.
type ProductionScheduleInputRepo interface {
	// GetConstraintMachines returns every planned machine in the constraint department, in name order.
	GetConstraintMachines(ctx context.Context, accountID, departmentID string) ([]scheduling.Machine, *apierror.APIError)

	// CountConstraintMachinesWithoutStep reports how many constraint machines cannot carry a plan downstream because they have no production step.
	CountConstraintMachinesWithoutStep(ctx context.Context, accountID, departmentID string) (int, *apierror.APIError)

	// GetConstraintDepartmentLaborRate returns the hourly labor rate configured on the constraint department, or nil when it has none.
	GetConstraintDepartmentLaborRate(ctx context.Context, accountID, departmentID string) (*float64, *apierror.APIError)

	// GetConstraintBatchMeasurements returns one row per historical batch produced on the given machines inside the window.
	GetConstraintBatchMeasurements(ctx context.Context, params GetConstraintBatchMeasurementsParams) ([]ConstraintBatchRow, *apierror.APIError)
	// GetItemRunRateHistory returns the run-rate samples behind an item's most recent scans, newest first, ignoring any measurement window. Only scans whose step carries a labor time come back.
	GetItemRunRateHistory(ctx context.Context, accountID, itemID string, limit int32) ([]ItemRunRateSample, *apierror.APIError)
	// GetFinishingMachines returns every machine outside the constraint department — the second stage, selected as the complement of the constraint rather than as a list of its own.
	GetFinishingMachines(ctx context.Context, accountID, constraintDepartmentID string) ([]scheduling.Machine, *apierror.APIError)
	// GetFinishingBatchMeasurements returns the second stage's production history for the given finished goods, which is what its run rates are measured from.
	GetFinishingBatchMeasurements(ctx context.Context, params GetFinishingBatchMeasurementsParams) ([]FinishingBatchRow, *apierror.APIError)

	// GetStepConsumptionItems returns the input items each production step consumes.
	GetStepConsumptionItems(ctx context.Context, stepIDs []string) ([]StepConsumptionRow, *apierror.APIError)

	// GetSeedBatchesForItems returns every scanned batch for the given items inside the demand window, to start the genealogy walk from.
	GetSeedBatchesForItems(ctx context.Context, params GetSeedBatchesParams) ([]SeedBatchRow, *apierror.APIError)

	// GetBatchFlowChildren returns the immediate downstream batches of the given parent batches.
	GetBatchFlowChildren(ctx context.Context, accountID string, parentBatchIDs []string) ([]BatchFlowChildRow, *apierror.APIError)

	// GetEchelonOnHand returns available inventory per item, net of allocations, normalized through the unit ratio.
	GetEchelonOnHand(ctx context.Context, accountID string, itemIDs []string) (map[string]float64, *apierror.APIError)

	// GetProductsForItems returns the sellable products carried by the given items.
	GetProductsForItems(ctx context.Context, accountID string, itemIDs []string) ([]SellableProductRow, *apierror.APIError)

	// GetPooledOrderDemandByProduct returns monthly sold quantity per product inside the window.
	GetPooledOrderDemandByProduct(ctx context.Context, params GetPooledOrderDemandParams) ([]PooledMonthlyDemandRow, *apierror.APIError)

	// ListDeliveryOutcomes returns every order whose commitment came due inside the window, with what happened to it, narrowed by the given filters.
	ListDeliveryOutcomes(ctx context.Context, accountID string, start, end time.Time, filters DeliveryFilters) ([]scheduling.DeliveryOutcome, *apierror.APIError)

	// CountUncommittedOrders counts issued orders in the window carrying no ship-by date, under the same filters, so the excluded count describes the same slice the rates do.
	CountUncommittedOrders(ctx context.Context, accountID string, start, end time.Time, filters DeliveryFilters) (int, *apierror.APIError)

	// GetOpenOrderRequirements returns the outstanding quantity on every issued, unshipped line for the given products.
	GetOpenOrderRequirements(ctx context.Context, accountID string, productIDs []string) ([]OpenOrderRequirementRow, *apierror.APIError)

	// GetProductDemandByCustomer returns monthly sold quantity per product and buyer, for measuring customer concentration.
	GetProductDemandByCustomer(ctx context.Context, params GetPooledOrderDemandParams) ([]CustomerDemandRow, *apierror.APIError)

	// GetCustomerFulfillmentProfiles returns every customer's resolved lead time and stated policy.
	GetCustomerFulfillmentProfiles(ctx context.Context, accountID string, accountDefaultLeadTimeDays int) ([]CustomerFulfillmentProfile, *apierror.APIError)

	// GetActiveDemandOverrides returns the demand overrides in force at the planning date.
	GetActiveDemandOverrides(ctx context.Context, accountID string, asOf time.Time) ([]scheduling.DemandOverride, *apierror.APIError)

	// GetItemsForProductLines maps product lines to the items sold under them.
	GetItemsForProductLines(ctx context.Context, accountID string, productLineIDs []string) ([]ProductLineItemRow, *apierror.APIError)

	// ListProductLineLotDefaults returns every product line in the account that has a lot convention.
	ListProductLineLotDefaults(ctx context.Context, accountID string) ([]scheduling.ProductLineLot, *apierror.APIError)

	// ListProductLineFulfillmentPolicies returns every product line in the account that sets a fulfillment policy, keyed by line id.
	ListProductLineFulfillmentPolicies(ctx context.Context, accountID string) (map[string]string, *apierror.APIError)

	// GetAllSellableProducts returns every product in the account, which is the candidate set for a fulfillment recommendation.
	GetAllSellableProducts(ctx context.Context, accountID string) ([]SellableProductRow, *apierror.APIError)

	// GetItemUnitCosts returns each item's unit cost, keyed by item id.
	GetItemUnitCosts(ctx context.Context, accountID string, itemIDs []string) (map[string]float64, *apierror.APIError)

	// ListItemProductLines maps items to the product line they sell under.
	ListItemProductLines(ctx context.Context, accountID string, itemIDs []string) ([]ItemProductLineRow, *apierror.APIError)

	// GetAccountScheduleSettings returns the account's stored planning assumptions as one raw row, or nil when the account has never configured scheduling.
	GetAccountScheduleSettings(ctx context.Context, accountID string) (*ProductionScheduleSettingsRow, *apierror.APIError)

	// ListScheduleItemSettings returns the account's per-item planning overrides.
	ListScheduleItemSettings(ctx context.Context, accountID string) ([]ProductionScheduleItemSetting, *apierror.APIError)
}

type MachineDowntimeRepo interface {
	ListReasons(ctx context.Context) ([]*MachineDowntimeReason, *apierror.APIError)
	GetReason(ctx context.Context, code string) (*MachineDowntimeReason, *apierror.APIError)
	List(ctx context.Context, params ListMachineDowntimeEventsParams) (*ListMachineDowntimeEventsResult, *apierror.APIError)
	Get(ctx context.Context, params GetMachineDowntimeEventParams) (*MachineDowntimeEvent, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*MachineDowntimeEvent, *apierror.APIError)
	// GetOpenForMachine returns the machine's currently-open event, or nil when the machine is running. Used to reject a second concurrent open event.
	GetOpenForMachine(ctx context.Context, accountID, machineID string) (*MachineDowntimeEvent, *apierror.APIError)
	Create(ctx context.Context, id string, event *MachineDowntimeEvent) (*MachineDowntimeEvent, *apierror.APIError)
	Update(ctx context.Context, event *MachineDowntimeEvent) (*MachineDowntimeEvent, *apierror.APIError)
	Delete(ctx context.Context, params DeleteMachineDowntimeEventParams) *apierror.APIError
}

type DemandOverrideRepo interface {
	ListTypes(ctx context.Context) ([]*DemandOverrideType, *apierror.APIError)
	List(ctx context.Context, params ListDemandOverridesParams) (*ListDemandOverridesResult, *apierror.APIError)
	Get(ctx context.Context, params GetDemandOverrideParams) (*DemandOverride, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*DemandOverride, *apierror.APIError)
	Create(ctx context.Context, id string, override *DemandOverride) (*DemandOverride, *apierror.APIError)
	Update(ctx context.Context, params UpdateDemandOverrideParams) (*DemandOverride, *apierror.APIError)
	Delete(ctx context.Context, params DeleteDemandOverrideParams) *apierror.APIError
}

type DepartmentRepo interface {
	List(ctx context.Context, params ListDepartmentsParams) (*ListDepartmentsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportDepartmentsParams) ([]*Department, *apierror.APIError)
	Get(ctx context.Context, params GetDepartmentParams) (*Department, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Department, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateDepartmentParams) (*Department, *apierror.APIError)
	Update(ctx context.Context, params UpdateDepartmentParams) (*Department, *apierror.APIError)
	Delete(ctx context.Context, params DeleteDepartmentParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	FindByNames(ctx context.Context, accountID string, names []string) ([]*Department, *apierror.APIError)
	SetMachinesDepartmentID(ctx context.Context, departmentID, accountID string, machineIDs []string) *apierror.APIError
	SetScanningStationsDepartmentID(ctx context.Context, departmentID, accountID string, scanningStationIDs []string) *apierror.APIError
	InsertLaborRate(ctx context.Context, rateID string, params CreateRateParams) *apierror.APIError
	UpdateLaborRate(ctx context.Context, rateID string, params CreateRateParams) *apierror.APIError
}

type DeliveryRepo interface {
	List(ctx context.Context, params ListDeliveriesParams) (*ListDeliveriesResult, *apierror.APIError)
	Get(ctx context.Context, params GetDeliveryParams) (*Delivery, *apierror.APIError)
	CountByPurchaseOrder(ctx context.Context, purchaseOrderID string) (int64, *apierror.APIError)
	CreateDelivery(ctx context.Context, id, number, salesOrderID, accountID, statusCode string, acceptedAt, rejectedAt *time.Time) *apierror.APIError
	CreateDeliveryLine(ctx context.Context, id, deliveryID, receivingOrderLineID, quantityID, unitCostID string, storageLocationID, lotID *string, acceptedAt, rejectedAt *time.Time) *apierror.APIError
}

type EmailLogRepo interface {
	List(ctx context.Context, params ListEmailLogsParams) (*ListEmailLogsResult, *apierror.APIError)
	Get(ctx context.Context, params GetEmailLogParams) (*EmailLog, *apierror.APIError)
}

type InventoryChangeLogRepo interface {
	List(ctx context.Context, params ListInventoryChangeLogsParams) (*ListInventoryChangeLogsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, id string) (*InventoryChangeLog, *apierror.APIError)
	ListAll(ctx context.Context, params ExportInventoryChangeLogsParams) ([]*InventoryChangeLog, *apierror.APIError)
}

type SalesOrderStatusRepo interface {
	List(ctx context.Context, params ListSalesOrderStatusesParams) (*ListSalesOrderStatusesResult, *apierror.APIError)
	GetByIDs(ctx context.Context, ids []string) ([]*SalesOrderStatus, *apierror.APIError)
}

type OrderDiscountRepo interface {
	List(ctx context.Context, params ListOrderDiscountsParams) (*ListOrderDiscountsResult, *apierror.APIError)
	Get(ctx context.Context, params GetOrderDiscountParams) (*OrderDiscount, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*OrderDiscount, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateOrderDiscountParams) (*OrderDiscount, *apierror.APIError)
	Update(ctx context.Context, params UpdateOrderDiscountParams) (*OrderDiscount, *apierror.APIError)
	Delete(ctx context.Context, params DeleteOrderDiscountParams) (*OrderDiscount, *apierror.APIError)
	ExistsByCode(ctx context.Context, accountID, code string, excludeID *string) (bool, *apierror.APIError)
	FindByCode(ctx context.Context, accountID, code string) (*OrderDiscount, *apierror.APIError)
	CheckDuplicateUsage(ctx context.Context, accountID, buyerAccountID, orderDiscountID string, salesOrderID *string) (bool, *apierror.APIError)
}

type VolumeDiscountRepo interface {
	List(ctx context.Context, params ListVolumeDiscountsParams) (*ListVolumeDiscountsResult, *apierror.APIError)
	Get(ctx context.Context, params GetVolumeDiscountParams) (*VolumeDiscount, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateVolumeDiscountParams) (*VolumeDiscount, *apierror.APIError)
	Update(ctx context.Context, params UpdateVolumeDiscountParams) (*VolumeDiscount, *apierror.APIError)
	Delete(ctx context.Context, params DeleteVolumeDiscountParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
}

type MaterialRepo interface {
	List(ctx context.Context, params ListMaterialsParams) (*ListMaterialsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportMaterialsParams) ([]*Material, *apierror.APIError)
	GetByID(ctx context.Context, params GetMaterialParams) (*Material, *apierror.APIError)
	GetByItemID(ctx context.Context, accountID, itemID string) (*Material, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Material, *apierror.APIError)
	Create(ctx context.Context, materialID, itemID, orderPointID, leadTimeID string) *apierror.APIError
	Update(ctx context.Context, params UpdateMaterialParams) *apierror.APIError
	DeleteByID(ctx context.Context, accountID, materialID string) *apierror.APIError
	DeleteByItemID(ctx context.Context, accountID, itemID string) *apierror.APIError
	InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	InsertItem(ctx context.Context, params InsertMaterialItemParams) *apierror.APIError
	// FindBySKUs batch-resolves existing materials by SKU within the account, returning the
	// IDs needed to update them. Used by bulk upsert.
	FindBySKUs(ctx context.Context, accountID string, skus []string) ([]*MaterialSKUMatch, *apierror.APIError)
	UpdateItem(ctx context.Context, params UpdateMaterialParams) *apierror.APIError
}

type SupplierMaterialRepo interface {
	List(ctx context.Context, params ListSupplierMaterialsParams) (*ListSupplierMaterialsResult, *apierror.APIError)
	GetBySupplierAndMaterialID(ctx context.Context, ownerAccountID, supplierAccountID, materialID string) (*SupplierMaterial, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateSupplierMaterialParams) (*SupplierMaterial, *apierror.APIError)
	Update(ctx context.Context, params UpdateSupplierMaterialParams) (*SupplierMaterial, *apierror.APIError)
	Delete(ctx context.Context, params DeleteSupplierMaterialParams) (*SupplierMaterial, *apierror.APIError)
	ExistsByMaterialAndSupplier(ctx context.Context, ownerAccountID, materialID, supplierAccountID string) (bool, *apierror.APIError)
}

type SalesOrderRepo interface {
	List(ctx context.Context, params ListSalesOrdersParams) (*ListSalesOrdersResult, *apierror.APIError)
	Get(ctx context.Context, accountID, salesOrderID string) (*SalesOrder, *apierror.APIError)
	GetForCustomer(ctx context.Context, accountID, buyerAccountID, salesOrderID string) (*SalesOrder, *apierror.APIError)
	GetLines(ctx context.Context, salesOrderID string) ([]*SalesOrderLine, *apierror.APIError)
	GetShipmentIDs(ctx context.Context, salesOrderID string) ([]string, *apierror.APIError)
	GetInvoiceIDs(ctx context.Context, salesOrderID string) ([]string, *apierror.APIError)
	GetContactsByOrders(ctx context.Context, salesOrderIDs []string) (map[string]*SalesOrderContacts, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateSalesOrderParams) (*SalesOrder, *apierror.APIError)
	Update(ctx context.Context, params UpdateSalesOrderParams) (*SalesOrder, *apierror.APIError)
	Delete(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	UpdateStatus(ctx context.Context, accountID, salesOrderID, statusCode string, issuedAt, completedAt *time.Time) *apierror.APIError
	GetCustomerLeadTimeChain(ctx context.Context, accountID, buyerAccountID string) (*CustomerLeadTimeChain, *apierror.APIError)
	SetShipByCommitment(ctx context.Context, accountID, salesOrderID string, commitment *ShipByCommitment) *apierror.APIError
	IsOrderForCustomer(ctx context.Context, salesOrderID, buyerAccountID string) (bool, *apierror.APIError)
	AreAllLineProductLinesCommissionExempt(ctx context.Context, productIDs []string) (bool, *apierror.APIError)
	GetAccountOriginAddress(ctx context.Context, accountID string) (*ShippingAddress, *apierror.APIError)
	GetProductTypesAndLines(ctx context.Context, productIDs []string) ([]ProductTypeLine, *apierror.APIError)
	IsDuplicateOrderNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError)
	IsDuplicateCustomerPO(ctx context.Context, accountID, buyerAccountID, customerPO string, excludeID *string) (bool, *apierror.APIError)
	CountSalesOrdersForBuyerAccounts(ctx context.Context, ownerAccountID string, buyerAccountIDs []string) (int64, *apierror.APIError)
	GetNextOrderNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
	GetPickID(ctx context.Context, salesOrderID string) (*string, *apierror.APIError)
	DeleteCascade(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	CreatePick(ctx context.Context, pickID, number, salesOrderID, accountID string) *apierror.APIError
	CreatePickLine(ctx context.Context, pickLineID, pickID, quantityID, salesOrderLineID string) *apierror.APIError
	DeleteQuantitiesByPickLines(ctx context.Context, salesOrderID string) *apierror.APIError
	DeletePickLinesBySalesOrder(ctx context.Context, salesOrderID string) *apierror.APIError
	DeletePickBySalesOrder(ctx context.Context, salesOrderID string) *apierror.APIError
	CheckPaymentStatus(ctx context.Context, salesOrderID string) (bool, *apierror.APIError)
	GetPaymentStatuses(ctx context.Context, accountID string, salesOrderIDs []string) (map[string]constants.SalesOrderPaymentStatus, *apierror.APIError)
	// GetPaymentIntentIDs returns the Stripe payment intent IDs linked to each of the given orders (scoped to the owning account), keyed by sales order ID. Orders with no payments are absent from the map.
	GetPaymentIntentIDs(ctx context.Context, accountID string, salesOrderIDs []string) (map[string][]string, *apierror.APIError)
	// GetFulfillmentProgress returns each order's picked/packed/invoiced completion fractions (0..1), aggregated over its sale-type lines, keyed by sales order ID. Orders with no sale lines are absent from the map.
	GetFulfillmentProgress(ctx context.Context, salesOrderIDs []string) (map[string]SalesOrderFulfillmentProgress, *apierror.APIError)
	GetLinesForBOM(ctx context.Context, salesOrderID string) ([]SalesOrderLineForBOM, *apierror.APIError)
	SetProductionRunID(ctx context.Context, accountID, salesOrderID, productionRunID string) *apierror.APIError
	GetSaleLinesForIssue(ctx context.Context, salesOrderID string) ([]SalesOrderSaleLineForIssue, *apierror.APIError)
	CreateReservedInventoryIssue(ctx context.Context, id, accountID, itemID, quantityID, orderID string) *apierror.APIError
	GetAcknowledgementRecipients(ctx context.Context, salesOrderID string) ([]string, *apierror.APIError)
	MarkAcknowledgementSent(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	CreateEmailContact(ctx context.Context, id, salesOrderID, accountUserID, notificationTypeCode string) *apierror.APIError
	DeleteEmailContactsByOrderAndType(ctx context.Context, salesOrderID, notificationTypeCode string) *apierror.APIError
	NoteFirstShipAt(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	// MarkFulfilled sets the order status to fulfilled and stamps completed_at (idempotent).
	MarkFulfilled(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	// GetSalesRepEmail returns the order's sales rep email, or nil if it has no rep or no email.
	GetSalesRepEmail(ctx context.Context, accountID, salesOrderID string) (*string, *apierror.APIError)
	MarkUnfulfilled(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	HasShippedShipment(ctx context.Context, salesOrderID string) (bool, *apierror.APIError)
}

type SalesOrderLineRepo interface {
	List(ctx context.Context, salesOrderID string) ([]*SalesOrderLine, *apierror.APIError)
	Get(ctx context.Context, salesOrderLineID string) (*SalesOrderLine, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateSalesOrderLineParams) (*SalesOrderLine, *apierror.APIError)
	Update(ctx context.Context, params UpdateSalesOrderLineParams) (*SalesOrderLine, *apierror.APIError)
	Delete(ctx context.Context, salesOrderLineID string) *apierror.APIError
	IsInOrder(ctx context.Context, salesOrderLineID, salesOrderID, accountID string) (bool, *apierror.APIError)
	GetNextLineItemNumber(ctx context.Context, salesOrderID string) (int32, *apierror.APIError)
	// HasShipmentAgainstOrderLine reports whether the order line is part of any shipment (packed or shipped).
	HasShipmentAgainstOrderLine(ctx context.Context, salesOrderLineID string) (bool, *apierror.APIError)
	// SyncInvoiceLineQuantities keeps the invoice lines referencing the order line in sync
	// with a quantity edit: their unit always follows the order line's unit (a unit edit is
	// a correction, and rollups sum these values without conversion), while their value
	// follows only when it still mirrors the pre-update ordered quantity — partial-shipment
	// snapshots keep the amount that was actually billed.
	SyncInvoiceLineQuantities(ctx context.Context, salesOrderLineID, previousQuantityValue, quantityValue, quantityUnitID string) *apierror.APIError
	// SyncShipmentLineQuantities applies the same sync rule as SyncInvoiceLineQuantities to
	// the shipment lines referencing the order line.
	SyncShipmentLineQuantities(ctx context.Context, salesOrderLineID, previousQuantityValue, quantityValue, quantityUnitID string) *apierror.APIError
	// SyncPickLineQuantityUnits relabels the pick line quantities referencing the order line
	// to the given unit. Pick line values are picking progress and are reconciled separately,
	// so only the unit follows a unit edit on the order line.
	SyncPickLineQuantityUnits(ctx context.Context, salesOrderLineID, quantityUnitID string) *apierror.APIError
	DeleteCascade(ctx context.Context, salesOrderLineID string) *apierror.APIError
	CreateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError
	// GetLineOrder returns the order's lines in current display order, flagging credit/freight (system) lines.
	GetLineOrder(ctx context.Context, salesOrderID string) ([]*SalesOrderLinePosition, *apierror.APIError)
	// SetLineItemNumber sets a single line's line_item_number.
	SetLineItemNumber(ctx context.Context, salesOrderLineID string, lineItemNumber int32) *apierror.APIError
}

type PurchaseOrderRepo interface {
	List(ctx context.Context, params ListPurchaseOrdersParams) (*ListPurchaseOrdersResult, *apierror.APIError)
	Get(ctx context.Context, accountID, purchaseOrderID string) (*PurchaseOrder, *apierror.APIError)
	GetLines(ctx context.Context, salesOrderID string) ([]*PurchaseOrderLine, *apierror.APIError)
	GetLinesByIDs(ctx context.Context, accountID string, ids []string) ([]*PurchaseOrderLine, *apierror.APIError)
	Create(ctx context.Context, id string, params CreatePurchaseOrderParams) (*PurchaseOrder, *apierror.APIError)
	Update(ctx context.Context, params UpdatePurchaseOrderParams) (*PurchaseOrder, *apierror.APIError)
	Delete(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError
	UpdateStatus(ctx context.Context, accountID, purchaseOrderID, statusCode string, issuedAt, completedAt *time.Time) *apierror.APIError
	IsDuplicateOrderNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError)
	GetNextOrderNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
	DeleteCascade(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError
	GetSupplierID(ctx context.Context, accountID, purchaseOrderID string) (string, *apierror.APIError)
	UpdateAcknowledgmentSent(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError
	CreateEmailContact(ctx context.Context, id, salesOrderID, accountUserID, notificationTypeCode string) *apierror.APIError
	DeleteEmailContactsByOrder(ctx context.Context, salesOrderID string) *apierror.APIError
	GetEmailContacts(ctx context.Context, salesOrderID string) ([]*PurchaseOrderEmailContact, *apierror.APIError)
	GetSubmissionRecipients(ctx context.Context, purchaseOrderID string) ([]string, *apierror.APIError)
	MarkSubmissionSent(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError
}

type PurchaseOrderLineRepo interface {
	Get(ctx context.Context, salesOrderLineID, salesOrderID string) (*PurchaseOrderLine, *apierror.APIError)
	Create(ctx context.Context, id string, params CreatePurchaseOrderLineParams) (*PurchaseOrderLine, *apierror.APIError)
	Update(ctx context.Context, params UpdatePurchaseOrderLineParams) (*PurchaseOrderLine, *apierror.APIError)
	Delete(ctx context.Context, salesOrderLineID, salesOrderID string) *apierror.APIError
	IsInOrder(ctx context.Context, salesOrderLineID, salesOrderID string) (bool, *apierror.APIError)
	GetNextLineItemNumber(ctx context.Context, salesOrderID string) (int32, *apierror.APIError)
	DeleteCascade(ctx context.Context, salesOrderLineID string) *apierror.APIError
	CreateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError
	CreateRate(ctx context.Context, rateID, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	UpdateQuantityValue(ctx context.Context, quantityID, value string) *apierror.APIError
	UpdateRateValue(ctx context.Context, rateID, value string, numeratorUnitID, denominatorUnitID *string) *apierror.APIError
}

type ReceivingOrderRepo interface {
	// Existing methods
	Create(ctx context.Context, id, number, orderID, accountID string) *apierror.APIError
	CreateLine(ctx context.Context, id, receivingOrderID, quantityID, salesOrderLineID string) *apierror.APIError
	GetByOrderID(ctx context.Context, orderID string) (*string, *apierror.APIError)
	DeleteLinesByOrderID(ctx context.Context, orderID string) *apierror.APIError
	DeleteByOrderID(ctx context.Context, orderID string) *apierror.APIError
	MarkComplete(ctx context.Context, orderID string) *apierror.APIError
	MarkIncomplete(ctx context.Context, orderID string) *apierror.APIError
	DeleteLinesByOrderLineID(ctx context.Context, salesOrderLineID string) *apierror.APIError

	// New methods for the receiving order endpoints
	List(ctx context.Context, params ListReceivingOrdersParams) (*ListReceivingOrdersResult, *apierror.APIError)
	Get(ctx context.Context, accountID, receivingOrderID string) (*ReceivingOrder, *apierror.APIError)
	ListLines(ctx context.Context, receivingOrderID string) ([]*ReceivingOrderLine, *apierror.APIError)
	FindUnstockedLineIDs(ctx context.Context, receivingOrderID, accountID string, enforceNonZero bool) ([]UnstockedLine, *apierror.APIError)
	StockLines(ctx context.Context, lineIDs []string, accountID string) *apierror.APIError
	MarkCompleteIfAllStocked(ctx context.Context, id, accountID string) (bool, *apierror.APIError)
	MarkIncompleteByID(ctx context.Context, id, accountID string) *apierror.APIError
	BulkCreateForRemainingQuantities(ctx context.Context, receivingOrderID string, orderLineIDs []string, accountID string) *apierror.APIError
	BulkReceiveRemainingQuantities(ctx context.Context, receivingOrderID string, orderLineIDs []string, accountID string) *apierror.APIError
	VoidAllLines(ctx context.Context, receivingOrderID, accountID string) *apierror.APIError
	DeleteDuplicateLines(ctx context.Context, receivingOrderID, accountID string) *apierror.APIError
	UpdateLineQuantity(ctx context.Context, lineID string, quantityValue string) *apierror.APIError
	VoidLine(ctx context.Context, lineID, accountID string) *apierror.APIError
	GetLine(ctx context.Context, lineID string) (*ReceivingOrderLine, *apierror.APIError)
	IsLineInReceivingOrder(ctx context.Context, lineID, receivingOrderID string) (bool, *apierror.APIError)
	CalculateQuantityYetToBeReceived(ctx context.Context, lineID, accountID string) (string, string, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, receivingOrderID string) (bool, *apierror.APIError)
	GetLineUnitPrices(ctx context.Context, receivingOrderID string) ([]ReceivingOrderLineUnitPrice, *apierror.APIError)
	GetPurchaseOrderID(ctx context.Context, receivingOrderID, accountID string) (string, *apierror.APIError)
	UpsertLot(ctx context.Context, lotID, accountID, itemID, lotNumber string) (string, *apierror.APIError)
	// LockItemForLedger takes the item's ordering root; ledgerlock.Acquire calls it. See docs/patterns/architecture-patterns.md, "Inventory ledger lock order".
	LockItemForLedger(ctx context.Context, itemID string) *apierror.APIError
	InsertInventoryReceiptForDelivery(ctx context.Context, scope *ledgerlock.Scope, receiptID, accountID, itemID, quantityID, unitCostID string, storageLocationID, lotID, orderID *string) *apierror.APIError
	MarkPurchaseOrderFulfilled(ctx context.Context, purchaseOrderID, accountID string) *apierror.APIError
	GetAllocationSumForIssue(ctx context.Context, issueID string) (string, *apierror.APIError)
	HasUnstockedLineForOrderLine(ctx context.Context, salesOrderLineID string) (bool, *apierror.APIError)
	CreateLineForRemainingQuantity(ctx context.Context, receivingOrderID, salesOrderLineID, accountID string) *apierror.APIError
}

type PartRepo interface {
	Create(ctx context.Context, partID, itemID string, params CreatePartParams) (*Part, *apierror.APIError)
	Get(ctx context.Context, params GetPartParams) (*Part, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Part, *apierror.APIError)
	List(ctx context.Context, params ListPartsParams) (*ListPartsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportPartsParams) ([]*Part, *apierror.APIError)
	Delete(ctx context.Context, params DeletePartParams) *apierror.APIError
	ExistsBySKU(ctx context.Context, accountID, sku string, excludeItemID *string) (bool, *apierror.APIError)
	// FindBySKUs batch-resolves existing parts by SKU within the account, returning the
	// part/item IDs and unit_value/unit_cost rate IDs needed to update them.
	FindBySKUs(ctx context.Context, accountID string, skus []string) ([]*PartSKUMatch, *apierror.APIError)
	InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	InsertItem(ctx context.Context, itemID string, params CreatePartParams, unitValueID, burnRateID, unitCostID string) *apierror.APIError
	TouchUpdatedAt(ctx context.Context, partID string) *apierror.APIError
	UpdateItem(ctx context.Context, params PartUpdateItemParams) *apierror.APIError
}

type PermissionGroupRepo interface {
	List(ctx context.Context, params ListPermissionGroupsParams) (*ListPermissionGroupsResult, *apierror.APIError)
	GetByIDs(ctx context.Context, ids []string) ([]*PermissionGroup, *apierror.APIError)
}

type InvoiceRepo interface {
	List(ctx context.Context, params ListInvoicesParams) (*ListInvoicesResult, *apierror.APIError)
	Get(ctx context.Context, params GetInvoiceParams) (*Invoice, *apierror.APIError)
	CountSince(ctx context.Context, accountID string, since time.Time) (int64, *apierror.APIError)
	GetLines(ctx context.Context, invoiceID string) ([]*InvoiceLine, *apierror.APIError)
	GetAllocations(ctx context.Context, invoiceID string) ([]*InvoiceAllocation, *apierror.APIError)
	GetAllocationsForInvoices(ctx context.Context, invoiceIDs []string) (map[string][]*InvoiceAllocation, *apierror.APIError)
	Update(ctx context.Context, params UpdateInvoiceParams) (*Invoice, *apierror.APIError)
	ListByCustomer(ctx context.Context, params ListCustomerInvoicesParams) (*ListCustomerInvoicesResult, *apierror.APIError)
	IsDuplicateNumber(ctx context.Context, accountID, number string) (bool, *apierror.APIError)
	GetEmailRecipients(ctx context.Context, invoiceID string) ([]string, *apierror.APIError)
	MarkEmailSent(ctx context.Context, accountID, invoiceID string) *apierror.APIError
	DeleteLinesByInvoice(ctx context.Context, invoiceID string) *apierror.APIError
	Delete(ctx context.Context, accountID, invoiceID string) *apierror.APIError
	// CreateFromShipment writes the invoice a shipment bills for: one line per shipped line plus the
	// order's non-shippable lines (freight/tax/discount/service) at full ordered quantity. Returns
	// the new invoice id.
	CreateFromShipment(ctx context.Context, params CreateInvoiceFromShipmentParams) (string, *apierror.APIError)
}

type ReceivableRepo interface {
	List(ctx context.Context, params ListReceivablesParams) (*ListReceivablesResult, *apierror.APIError)
	ListByCustomer(ctx context.Context, params ListReceivablesByCustomerParams) (*ListReceivablesByCustomerResult, *apierror.APIError)
	ListAllByCustomer(ctx context.Context, accountID, customerAccountID string, cutoffDate *time.Time) ([]ReceivableEntry, *apierror.APIError)
	ListOpenCreditsByCustomer(ctx context.Context, accountID, customerAccountID string) ([]OpenCredit, *apierror.APIError)
}

type PickRepo interface {
	List(ctx context.Context, params ListPicksParams) (*ListPicksResult, *apierror.APIError)
	Get(ctx context.Context, accountID, pickID string) (*Pick, *apierror.APIError)
	GetLines(ctx context.Context, pickID string) ([]*PickLine, *apierror.APIError)
	// GetLinesForPicks returns the lines for a page of picks in one query, keyed by pick id, so the
	// list endpoint's lines expansion does not fan out into one query per pick.
	GetLinesForPicks(ctx context.Context, pickIDs []string) (map[string][]*PickLine, *apierror.APIError)
	UpdateFinishedAt(ctx context.Context, accountID, pickID string, finishedAt time.Time) *apierror.APIError
	HasShippedItems(ctx context.Context, accountID, pickID string) (bool, *apierror.APIError)
	VoidAllLines(ctx context.Context, pickID string) *apierror.APIError
	DeleteDuplicatePickLines(ctx context.Context, accountID, pickID string) *apierror.APIError
	ClearFinishedAt(ctx context.Context, accountID, pickID string) *apierror.APIError
	PickAllLines(ctx context.Context, pickID string) *apierror.APIError
	// GetShipmentIDs returns the ids of shipments raised against the pick's order, oldest first.
	GetShipmentIDs(ctx context.Context, accountID, pickID string) ([]string, *apierror.APIError)
	// GetShipmentIDsForPicks returns shipment ids for a page of picks in one query, keyed by pick
	// id (oldest first within each), so the list endpoint's shipments expansion does not fan out
	// into one query per pick.
	GetShipmentIDsForPicks(ctx context.Context, accountID string, pickIDs []string) (map[string][]string, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, pickID string) (bool, *apierror.APIError)
	FindLinesToPack(ctx context.Context, pickID string) ([]*PickLine, *apierror.APIError)
	PackLines(ctx context.Context, pickID string) *apierror.APIError
	MarkFinishedIfAllPacked(ctx context.Context, pickID string) *apierror.APIError
	// CloseOpenPickLines packs every still-open pick line (used when the order is closed).
	CloseOpenPickLines(ctx context.Context, pickID string) *apierror.APIError
	// ReopenIncompletePickLines reopens pick lines whose picked quantity is below the ordered quantity (used when a fulfilled order is reopened).
	ReopenIncompletePickLines(ctx context.Context, pickID string) *apierror.APIError
	CountLines(ctx context.Context, pickID string) (int64, *apierror.APIError)
	CountShipmentsByOrder(ctx context.Context, salesOrderID string) (int64, *apierror.APIError)
	GetSalesOrderForPick(ctx context.Context, accountID, pickID string) (*PickSalesOrder, *apierror.APIError)
	CreateShipment(ctx context.Context, params CreateShipmentFromPickParams) *apierror.APIError
	CreateShipmentLine(ctx context.Context, params CreateShipmentLineParams) *apierror.APIError
	CreateShippingCase(ctx context.Context, params CreateShippingCaseParams) *apierror.APIError
	CreateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	FindIDByShipmentOrder(ctx context.Context, accountID, shipmentID string) (string, *apierror.APIError)
}

type PickLineRepo interface {
	Get(ctx context.Context, pickLineID string) (*PickLine, *apierror.APIError)
	// UpdateQuantity writes the line's picked quantity; a nil value or unit leaves that half unchanged.
	UpdateQuantity(ctx context.Context, pickLineID string, quantityValue, quantityUnitID *string) *apierror.APIError
	PickRemainingQuantity(ctx context.Context, pickLineID string) *apierror.APIError
	VoidLine(ctx context.Context, pickLineID string) *apierror.APIError
	IsInPick(ctx context.Context, pickLineID, pickID string) (bool, *apierror.APIError)
	CreateForRemaining(ctx context.Context, id, quantityID, pickID, orderLineID string) *apierror.APIError
	CalculateRemainingForOrderLine(ctx context.Context, orderLineID string) (remainingValue string, unitID string, apiErr *apierror.APIError)
	HasUnpackedPickLineForOrderLine(ctx context.Context, orderLineID string) (bool, *apierror.APIError)
	// GetOrderLinePackProgress returns the order line's ordered quantity, the total already packed, and the quantity unit. outstanding = ordered - packed decides whether an open pick line is still needed.
	GetOrderLinePackProgress(ctx context.Context, orderLineID string) (orderedValue string, packedValue string, unitID string, apiErr *apierror.APIError)
	// DeleteUnpackedForOrderLine deletes every unpacked (open) pick line for the order line, along with their quantity rows. Packed lines are left untouched.
	DeleteUnpackedForOrderLine(ctx context.Context, orderLineID string) *apierror.APIError
	UnpackByShipment(ctx context.Context, shipmentID string) *apierror.APIError
}

type ProductTypeRepo interface {
	List(ctx context.Context, params ListProductTypesParams) (*ListProductTypesResult, *apierror.APIError)
	Get(ctx context.Context, identifier string) (*ProductType, *apierror.APIError)
	GetByIDs(ctx context.Context, ids []string) ([]*ProductType, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateProductTypeParams) (*ProductType, *apierror.APIError)
	Update(ctx context.Context, params UpdateProductTypeParams) (*ProductType, *apierror.APIError)
	Delete(ctx context.Context, id string) *apierror.APIError
	ExistsByName(ctx context.Context, name string, excludeID *string) (bool, *apierror.APIError)
	ExistsByCode(ctx context.Context, code string, excludeID *string) (bool, *apierror.APIError)
	ExistsByID(ctx context.Context, id string) (bool, *apierror.APIError)
}

type QuantityRepo interface {
	Get(ctx context.Context, id string) (*Quantity, *apierror.APIError)
	Update(ctx context.Context, params UpdateQuantityParams) (*Quantity, *apierror.APIError)
}

type RateRepo interface {
	Get(ctx context.Context, id string) (*Rate, *apierror.APIError)
	Update(ctx context.Context, params UpdateRateParams) (*Rate, *apierror.APIError)
}

type SettlementRepo interface {
	List(ctx context.Context, params ListSettlementsParams) (*ListSettlementsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, settlementID string) (*Settlement, *apierror.APIError)
	GetAllocations(ctx context.Context, settlementID string) ([]*TransactionAllocation, *apierror.APIError)
	InsertSettlement(ctx context.Context, id, number string, params CreateSettlementParams) *apierror.APIError
	Update(ctx context.Context, params UpdateSettlementParams) (*Settlement, *apierror.APIError)
	Delete(ctx context.Context, accountID, settlementID string) *apierror.APIError
	IsDuplicateNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError)
	CreateAllocation(ctx context.Context, allocationID, quantityID, settlementID, dollarUnitID string, params CreateSettlementAllocationParams) *apierror.APIError
	DeleteAllocations(ctx context.Context, settlementID string) ([]*TransactionAllocation, *apierror.APIError)
	GetAllocationTransactionIDs(ctx context.Context, settlementID string) ([]string, *apierror.APIError)
	GetAllocationInvoiceIDs(ctx context.Context, settlementID string) ([]string, *apierror.APIError)
	// AllocateNextSettlementNumber reserves the account's next settlement number in one locked statement.
	AllocateNextSettlementNumber(ctx context.Context, sysPropertyID, accountID string) (int64, *apierror.APIError)
	GetDollarUnitID(ctx context.Context) (string, *apierror.APIError)
	DeleteOrphanedAdjustmentTransactions(ctx context.Context, settlementID string) *apierror.APIError
	UpdateTransactionsFullyAllocated(ctx context.Context, transactionIDs []string, isFullyAllocated bool) *apierror.APIError
	UpdateInvoicePaymentStatus(ctx context.Context, invoiceID string, isPaidInFull, isOverPaid bool) *apierror.APIError
	GetInvoicePaymentFlags(ctx context.Context, invoiceIDs []string) ([]InvoicePaymentFlags, *apierror.APIError)
}

type TransactionRepo interface {
	Create(ctx context.Context, txID, number, typeCode, accountID, customerAccountID string, stripePaymentID *string, methodCode *string, adjustmentTypeCode *string, responsibleUserID *string, note *string, amountValue string, amountUnitID string) *apierror.APIError
	// FindByStripePaymentID returns the transaction linked to the given Stripe payment intent, or nil when none exists.
	FindByStripePaymentID(ctx context.Context, stripePaymentID string) (*TransactionRecord, *apierror.APIError)
	// UpdateFundsReceivedByStripePaymentIDs stamps funds_received_at on every transaction of the account whose stripe_payment_id is in the given set (called when a Stripe payout lands).
	UpdateFundsReceivedByStripePaymentIDs(ctx context.Context, accountID string, stripePaymentIDs []string, fundsReceivedAt time.Time) *apierror.APIError
	UpdateNote(ctx context.Context, txID, note string) *apierror.APIError
	Delete(ctx context.Context, txID string) *apierror.APIError
	DeleteAllocations(ctx context.Context, transactionID string) *apierror.APIError
	DeleteQuantity(ctx context.Context, quantityID string) *apierror.APIError
	FetchAndIncrementTransactionNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
	List(ctx context.Context, params ListTransactionsParams) (*ListTransactionsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, transactionID string) (*Transaction, *apierror.APIError)
	GetAllocations(ctx context.Context, transactionID string) ([]*TransactionAllocation, *apierror.APIError)
	Update(ctx context.Context, params UpdateTransactionParams) (*Transaction, *apierror.APIError)
	ExistsByNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError)
	ResolveResponsibleUserID(ctx context.Context, accountID, userOrAccountUserID string) (string, *apierror.APIError)
	ListByCustomer(ctx context.Context, params ListAccountTransactionsParams) (*ListAccountTransactionsResult, *apierror.APIError)
	GetDollarUnitID(ctx context.Context) (string, *apierror.APIError)
}

type TransactionAllocationRepo interface {
	ListEntries(ctx context.Context, params ListAllocationEntriesParams) (*ListAllocationEntriesResult, *apierror.APIError)
	GetByID(ctx context.Context, accountID, allocationID string) (*TransactionAllocation, *apierror.APIError)
	UpdateAmount(ctx context.Context, amountID, newValue string) *apierror.APIError
	Delete(ctx context.Context, accountID, allocationID string) *apierror.APIError
	ListOpenCredits(ctx context.Context, params ListOpenCreditsParams) (*ListOpenCreditsResult, *apierror.APIError)
	GetDollarUnitID(ctx context.Context) (string, *apierror.APIError)
}

type CatalogRepo interface {
	// ListProductLines returns the distinct product lines that have portal-ready products for a given account.
	ListProductLines(ctx context.Context, accountID string) ([]*CatalogProductLine, *apierror.APIError)

	// ListProductLinesForCustomer returns the distinct product lines that the given customer has access to.
	ListProductLinesForCustomer(ctx context.Context, accountID, customerAccountID string) ([]*CatalogProductLine, *apierror.APIError)

	// ListProducts returns products in a specific product line grouped by item category.
	ListProducts(ctx context.Context, accountID, productLineID string) ([]*CatalogCategory, *apierror.APIError)

	// ListProductsForCustomer returns products in a specific product line that the given customer has access to.
	ListProductsForCustomer(ctx context.Context, accountID, customerAccountID, productLineID string) ([]*CatalogCategory, *apierror.APIError)
}

type EDIRepo interface {
	ListDCLocations(ctx context.Context, params ListDCLocationsParams) (*ListDCLocationsResult, *apierror.APIError)
	GetDCLocation(ctx context.Context, params GetDCLocationParams) (*DCLocation, *apierror.APIError)
	GetDCLocationsByIDs(ctx context.Context, ownerAccountID string, ids []string) ([]*DCLocation, *apierror.APIError)
	CreateDCLocation(ctx context.Context, id string, params CreateDCLocationParams) (*DCLocation, *apierror.APIError)
	UpdateDCLocation(ctx context.Context, params UpdateDCLocationParams) (*DCLocation, *apierror.APIError)
	DeleteDCLocation(ctx context.Context, params DeleteDCLocationParams) *apierror.APIError
	ListEDIRuns(ctx context.Context, params ListEDIRunsParams) (*ListEDIRunsResult, *apierror.APIError)
	GetEDIRun(ctx context.Context, accountID, ediRunID string) (*EDIRun, *apierror.APIError)
	GetEDIRunsByIDs(ctx context.Context, accountID string, ids []string) ([]*EDIRun, *apierror.APIError)
}

type RegistrationFlowRepo interface {
	List(ctx context.Context, params ListRegistrationFlowsParams) (*ListRegistrationFlowsResult, *apierror.APIError)
	Get(ctx context.Context, accountID, id string) (*RegistrationFlow, *apierror.APIError)
	GetByAccountID(ctx context.Context, accountID string) ([]*RegistrationFlow, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateRegistrationFlowParams) (*RegistrationFlow, *apierror.APIError)
	Update(ctx context.Context, params UpdateRegistrationFlowParams) (*RegistrationFlow, *apierror.APIError)
	Delete(ctx context.Context, params DeleteRegistrationFlowParams) *apierror.APIError
}

type CustomerRegistrationRepo interface {
	FindCustomerAccountByExternalNumber(ctx context.Context, ownerAccountID, externalNumber string) (string, *apierror.APIError)
	CreateAccountUserLink(ctx context.Context, linkID, accountID, userID string) *apierror.APIError
	// AllocateNextCustomerNumber reserves the account's next customer number in one locked statement.
	AllocateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string) (int64, *apierror.APIError)
	CreateNewCustomerAccount(ctx context.Context, params CreateNewCustomerAccountParams) (string, *apierror.APIError)
	GetUserEmailByID(ctx context.Context, userID string) (string, *apierror.APIError)
}

type OrderPaymentIntentRepo interface {
	Create(ctx context.Context, id, paymentIntentID, salesOrderID string) *apierror.APIError
	FindByPaymentIntentID(ctx context.Context, paymentIntentID string) (*OrderPaymentIntent, *apierror.APIError)
	Delete(ctx context.Context, id string) *apierror.APIError
}

type ShipmentRepo interface {
	List(ctx context.Context, params ListShipmentsParams) (*ListShipmentsResult, *apierror.APIError)
	Get(ctx context.Context, params GetShipmentParams) (*Shipment, *apierror.APIError)
	Update(ctx context.Context, params UpdateShipmentParams) (*Shipment, *apierror.APIError)
	SyncShippingForOrder(ctx context.Context, params SyncShipmentShippingParams) *apierror.APIError
	// SyncShipToForOrder re-points every shipment on an order to the given ship-to address, independently of the carrier.
	SyncShipToForOrder(ctx context.Context, accountID, salesOrderID, shippingAddressID string) *apierror.APIError
	Delete(ctx context.Context, accountID, shipmentID string) *apierror.APIError
	MarkShipped(ctx context.Context, accountID, shipmentID, shippedByID string) *apierror.APIError
	MarkVoided(ctx context.Context, accountID, shipmentID string) *apierror.APIError
	FindInvoiceIDByShipment(ctx context.Context, accountID, shipmentID string) (*string, *apierror.APIError)
	// LinkInvoice points the shipment at the invoice created for it, so void can find it later.
	LinkInvoice(ctx context.Context, accountID, shipmentID, invoiceID string) *apierror.APIError
	// SetMasterTracking stamps the shipment's carrier master tracking number.
	SetMasterTracking(ctx context.Context, accountID, shipmentID, trackingNumber string) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, shipmentID string) (bool, *apierror.APIError)
}

type ShipmentLineRepo interface {
	List(ctx context.Context, params ListShipmentLinesParams) (*ListShipmentLinesResult, *apierror.APIError)
	Get(ctx context.Context, shipmentLineID string) (*ShipmentLine, *apierror.APIError)
	Create(ctx context.Context, id, quantityID string, params CreateShipmentLineEndpointParams) (*ShipmentLine, *apierror.APIError)
	Update(ctx context.Context, params UpdateShipmentLineEndpointParams) (*ShipmentLine, *apierror.APIError)
	Delete(ctx context.Context, shipmentLineID string) *apierror.APIError
	IsInShipment(ctx context.Context, shipmentLineID, shipmentID string) (bool, *apierror.APIError)
	ListByShipment(ctx context.Context, shipmentID string) ([]*ShipmentLine, *apierror.APIError)
	DeleteByShipment(ctx context.Context, shipmentID string) *apierror.APIError
	// GetSalesOrderLineCapacity reports which order a sales order line belongs to and how much of
	// it is still unshipped. excludeShipmentLineID omits the line an update is replacing.
	GetSalesOrderLineCapacity(ctx context.Context, salesOrderLineID string, excludeShipmentLineID *string) (*SalesOrderLineShipmentCapacity, *apierror.APIError)
}

type ScanningStationRepo interface {
	List(ctx context.Context, params ListScanningStationsParams) (*ListScanningStationsResult, *apierror.APIError)
	Get(ctx context.Context, params GetScanningStationParams) (*ScanningStation, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*ScanningStation, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateScanningStationParams) (*ScanningStation, *apierror.APIError)
	Update(ctx context.Context, params UpdateScanningStationParams) (*ScanningStation, *apierror.APIError)
	Delete(ctx context.Context, params DeleteScanningStationParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	FindIDByName(ctx context.Context, accountID, name string) (*string, *apierror.APIError)
	// FindByNames resolves existing scanning stations by name (case-insensitive) in
	// one query. Names must be pre-lowercased by the caller.
	FindByNames(ctx context.Context, accountID string, names []string) ([]*ScanningStation, *apierror.APIError)
	ConnectProductionStepsByName(ctx context.Context, accountID, scanningStationID, name string) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	FindType(ctx context.Context, accountID, id string) (string, *apierror.APIError)
	// Export returns every matching station up to params.Limit, unpaginated.
	Export(ctx context.Context, params ExportScanningStationsParams) ([]*ScanningStation, *apierror.APIError)
}

type ShippingCaseRepo interface {
	Get(ctx context.Context, accountID, shippingCaseID string) (*ShippingCase, *apierror.APIError)
	Update(ctx context.Context, params UpdateShippingCaseParams) *apierror.APIError
	// RepointToCarrier moves every case on a shipment onto the given carrier, so per-case tracking deep-links keep resolving against the carrier that actually carries them.
	RepointToCarrier(ctx context.Context, accountID, shipmentID, carrierID string) *apierror.APIError
	Delete(ctx context.Context, accountID, shippingCaseID string) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, shippingCaseID string) (bool, *apierror.APIError)
	GetNumber(ctx context.Context, accountID, shippingCaseID string) (string, *apierror.APIError)
	// GetSalesOrderID walks the case to its order, so audit events can stamp that order as their root.
	GetSalesOrderID(ctx context.Context, accountID, shippingCaseID string) (string, *apierror.APIError)
	ListByShipment(ctx context.Context, shipmentID string) ([]*ShippingCase, *apierror.APIError)
	MarkShippedByShipment(ctx context.Context, shipmentID string) *apierror.APIError
	VoidByShipment(ctx context.Context, shipmentID string) *apierror.APIError
	UpdateWithShipmentInfo(ctx context.Context, shippingCaseID, trackingNumber, shippoTransactionID, shippingLabelURL string) *apierror.APIError
	AddSscc(ctx context.Context, shippingCaseID, sscc string) *apierror.APIError
	FindAndIncrementSsccCounter(ctx context.Context, accountID string) (int64, *apierror.APIError)
	DeleteByShipment(ctx context.Context, shipmentID string) *apierror.APIError
}

type LocationRepo interface {
	List(ctx context.Context, params ListLocationsParams) (*ListLocationsResult, *apierror.APIError)
	Export(ctx context.Context, params ExportLocationsParams) ([]*Location, *apierror.APIError)
	Get(ctx context.Context, params GetLocationParams) (*Location, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Location, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateLocationParams) (*Location, *apierror.APIError)
	Update(ctx context.Context, params UpdateLocationParams) (*Location, *apierror.APIError)
	Delete(ctx context.Context, params DeleteLocationParams) *apierror.APIError
	FindByNames(ctx context.Context, accountID string, names []string) ([]*Location, *apierror.APIError)
	LinkParent(ctx context.Context, accountID, childID, parentID string) *apierror.APIError
	ListTypes(ctx context.Context, params ListLocationTypesParams) (*ListLocationTypesResult, *apierror.APIError)
	GetType(ctx context.Context, idOrCode string) (*LocationType, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	CountChildren(ctx context.Context, accountID, parentID string) (int64, *apierror.APIError)
}

type StripeEventLogRepo interface {
	Exists(ctx context.Context, eventID, objectID string) (bool, *apierror.APIError)
	Create(ctx context.Context, id, eventID, objectID, eventType string) *apierror.APIError
}

type SupplierRepo interface {
	List(ctx context.Context, params ListSuppliersParams) (*ListSuppliersResult, *apierror.APIError)
	Get(ctx context.Context, params GetSupplierParams) (*Supplier, *apierror.APIError)
	Create(ctx context.Context, accountID, relationID string, params CreateSupplierParams, billToAddressID, shipToAddressID *string) (*Supplier, *apierror.APIError)
	Update(ctx context.Context, params UpdateSupplierParams) (*Supplier, *apierror.APIError)
	Delete(ctx context.Context, ownerAccountID, supplierAccountID string) (*Supplier, *apierror.APIError)
	BulkDelete(ctx context.Context, ownerAccountID string, supplierAccountIDs []string) *apierror.APIError
	ExistsByNumber(ctx context.Context, ownerAccountID, number string, excludeID *string) (bool, *apierror.APIError)
	// FindByNames resolves supplier display names to supplier account IDs within the
	// owner account (case-insensitive). Used by bulk upsert to attach existing suppliers.
	FindByNames(ctx context.Context, ownerAccountID string, names []string) ([]*SupplierNameMatch, *apierror.APIError)
}

type SysPropertyRepo interface {
	List(ctx context.Context, params ListSysPropertiesParams) (*ListSysPropertiesResult, *apierror.APIError)
	Get(ctx context.Context, accountID, id string) (*SysProperty, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*SysProperty, *apierror.APIError)
	GetByTypeCode(ctx context.Context, accountID string, typeCode constants.SysPropertyTypeCode) (*SysProperty, *apierror.APIError)
	Create(ctx context.Context, id, accountID string, typeCode constants.SysPropertyTypeCode, value int32) (*SysProperty, *apierror.APIError)
	UpdateValue(ctx context.Context, accountID, id string, value int32) (*SysProperty, *apierror.APIError)
	IncrementValue(ctx context.Context, accountID, id string) (*SysProperty, *apierror.APIError)
	IsDuplicate(ctx context.Context, accountID string, typeCode constants.SysPropertyTypeCode, value string) (bool, *apierror.APIError)
}

type TerritoryRepo interface {
	List(ctx context.Context, params ListTerritoriesParams) (*ListTerritoriesResult, *apierror.APIError)
	Get(ctx context.Context, params GetTerritoryParams) (*Territory, *apierror.APIError)
	Create(ctx context.Context, territoryID string, params CreateTerritoryParams) (*Territory, *apierror.APIError)
	Update(ctx context.Context, params UpdateTerritoryParams) (*Territory, *apierror.APIError)
	Delete(ctx context.Context, params DeleteTerritoryParams) *apierror.APIError
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Territory, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, territoryID string) (bool, *apierror.APIError)
	// FindSalesRepByZipcode returns the sales_rep (account_user) ID for the territory whose zipcode range includes the given zipcode, if any.
	FindSalesRepByZipcode(ctx context.Context, accountID string, zipcode int32) (*string, *apierror.APIError)
	// FindSalesRepByState returns the sales_rep (account_user) ID for the state-only territory matching the given state, if any.
	FindSalesRepByState(ctx context.Context, accountID, state string) (*string, *apierror.APIError)
}

// PricingRepo loads the data the sales-order-line pricing engine needs.
type PricingRepo interface {
	// LoadPricingBundle fetches, in a small number of queries, all product list prices, unit conversion data, unit-group discounts, account-price overrides (for the buyer and its parent account), and applicable volume discounts.
	LoadPricingBundle(ctx context.Context, params LoadPricingBundleParams) (*PricingBundle, *apierror.APIError)
	// ProductQuantityUnits returns, per product, the set of unit IDs its unit group allows a quantity to be expressed in. Products the account does not own are absent, which the caller reports as an unknown product rather than as an unusable unit.
	ProductQuantityUnits(ctx context.Context, accountID string, productIDs []string) (map[string]map[string]struct{}, *apierror.APIError)
}

type JobRepo interface {
	Get(ctx context.Context, jobID, accountID string) (*Job, *apierror.APIError)
	Create(ctx context.Context, params CreateJobRepositoryParams) *apierror.APIError
	// returns the number of rows changed; the query guards on the terminal timestamps, so an
	// already-settled job matches zero rows, which serializes a cancel against a completion.
	Update(ctx context.Context, params UpdateJobRepositoryParams) (int64, *apierror.APIError)
}

// persists customer portal custom domains. The single-row getters return (nil, nil) when no matching row exists so callers can distinguish absence from failure.
// persists a buyer's customer-portal registration session.
type PortalRegistrationSessionRepo interface {
	Create(ctx context.Context, typeID string, params CreatePortalRegistrationSessionParams) (*PortalRegistrationSession, *apierror.APIError)
	GetByTypeID(ctx context.Context, typeID string) (*PortalRegistrationSession, *apierror.APIError)
	// GetIncomplete returns the newest non-completed, non-abandoned session for the (user, seller) pair, or nil.
	GetIncomplete(ctx context.Context, userID, sellerAccountID string) (*PortalRegistrationSession, *apierror.APIError)
	Update(ctx context.Context, params UpdatePortalRegistrationSessionParams) (*PortalRegistrationSession, *apierror.APIError)
	Complete(ctx context.Context, typeID, customerID string) (*PortalRegistrationSession, *apierror.APIError)
	Abandon(ctx context.Context, typeID string) *apierror.APIError
	// ListSessions returns a seller's registration sessions (keyset-paginated), for customer-service follow-up.
	ListSessions(ctx context.Context, params ListPortalRegistrationSessionsParams) (*ListPortalRegistrationSessionsResult, *apierror.APIError)
}

type PortalDomainRepo interface {
	Create(ctx context.Context, portalDomainID, accountID, domainName string) (*PortalDomain, *apierror.APIError)
	GetByID(ctx context.Context, accountID, portalDomainID string) (*PortalDomain, *apierror.APIError)
	GetByAccountID(ctx context.Context, accountID string) (*PortalDomain, *apierror.APIError)
	// GetByDomain looks a domain up without account scoping; used for global-uniqueness checks.
	GetByDomain(ctx context.Context, domainName string) (*PortalDomain, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string) ([]*PortalDomain, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*PortalDomain, *apierror.APIError)
	// UpdateProviderState persists the latest required DNS records and status reported by the serving provider.
	UpdateProviderState(ctx context.Context, portalDomainID string, status constants.PortalDomainStatus, dnsRecords []PortalDNSRecord) *apierror.APIError
	MarkVerified(ctx context.Context, portalDomainID string) *apierror.APIError
	Delete(ctx context.Context, accountID, portalDomainID string) (bool, *apierror.APIError)
	// ResolveVerifiedHost returns the public account whose verified portal domain matches the given host, or a not-found error.
	ResolveVerifiedHost(ctx context.Context, domainName string) (*PublicAccountBySlug, *apierror.APIError)
}
