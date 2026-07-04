package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type AccountRepo interface {
	Create(ctx context.Context, id, name string, accountTypeCode AccountType, planCode constants.PlanCode) *apierror.APIError
	GetPlanCode(ctx context.Context, id string) (constants.PlanCode, *apierror.APIError)
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)
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
	GetBrandingLogoURL(ctx context.Context, accountID string) (*string, *apierror.APIError)
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
	Create(ctx context.Context, id, accountID, userID string, roleID, departmentID *string) *apierror.APIError
	Update(ctx context.Context, accountUserID string, roleID, departmentID *string) *apierror.APIError
	// ReactivateRemovedAccountUser reactivates a previously soft-removed link for (accountID, userID), setting its role/department, and returns the reactivated account_user id. Returns resource_not_found when no removed link exists.
	ReactivateRemovedAccountUser(ctx context.Context, accountID, userID string, roleID, departmentID *string) (string, *apierror.APIError)
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
	DeleteNotificationPreference(ctx context.Context, accountRelationID, recipientAccountUserID, notificationTypeCode string) *apierror.APIError
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
	ProductID      string
	ProductSKU     string
	QuantityUnitID string
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
	InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	InsertItem(ctx context.Context, itemID string, params CreateProductParams) *apierror.APIError
}

type ItemRepo interface {
	List(ctx context.Context, params ListItemsParams) (*ListItemsResult, *apierror.APIError)
	Get(ctx context.Context, params GetItemParams) (*Item, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Item, *apierror.APIError)
	GetInventory(ctx context.Context, accountID, itemID string) (*ItemInventory, *apierror.APIError)
	GetCostFlowConsumptions(ctx context.Context, stepID string) ([]CostFlowConsumption, *apierror.APIError)
	UpdateUnitCost(ctx context.Context, accountID, itemID string, cost decimal.Decimal, denominatorUnitID string) *apierror.APIError
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
	GetCategoryBaseUnitID(ctx context.Context, categoryID string) (string, *apierror.APIError)
	ListConsumptionChangeLogsForBurnRate(ctx context.Context, accountID, itemID string) ([]BurnRateConsumptionLog, *apierror.APIError)
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
	Get(ctx context.Context, params GetUnitParams) (*Unit, *apierror.APIError)
	// GetCurrencyBaseUnitID returns the global currency base unit ID used as the numerator unit when building monetary price rates.
	GetCurrencyBaseUnitID(ctx context.Context) (string, *apierror.APIError)
	// GetDimensionCodes returns a unit-id → unit_dimension_code map for the given IDs. Used to enforce unit-type constraints (e.g., currency-only numerator on cost rates) before persisting rate rows.
	GetDimensionCodes(ctx context.Context, ids []string) (map[string]string, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateUnitParams) (*Unit, *apierror.APIError)
	Update(ctx context.Context, params UpdateUnitParams) (*Unit, *apierror.APIError)
	Delete(ctx context.Context, params DeleteUnitParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	ExistsByAbbreviation(ctx context.Context, accountID, abbreviation string, excludeID *string) (bool, *apierror.APIError)
	FindByAbbreviations(ctx context.Context, accountID string, abbreviations []string) ([]*Unit, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Unit, *apierror.APIError)
}

type UnitGroupRepo interface {
	List(ctx context.Context, params ListUnitGroupsParams) (*ListUnitGroupsResult, *apierror.APIError)
	Get(ctx context.Context, params GetUnitGroupParams) (*UnitGroupFull, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateUnitGroupParams) (*UnitGroupFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateUnitGroupParams) (*UnitGroupFull, *apierror.APIError)
	Delete(ctx context.Context, params DeleteUnitGroupParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
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

// HubspotSyncRepo persists the HubSpot backfill state: jobs (state machine), the generic Augno->HubSpot id mapping, and the company-match review queue.
type HubspotSyncRepo interface {
	CreateJob(ctx context.Context, params CreateHubspotSyncJobParams) (*HubspotSyncJob, *apierror.APIError)
	GetJob(ctx context.Context, accountID, id string) (*HubspotSyncJob, *apierror.APIError)
	GetLatestJobForAccount(ctx context.Context, accountID string) (*HubspotSyncJob, *apierror.APIError)
	UpdateJob(ctx context.Context, params UpdateHubspotSyncJobParams) *apierror.APIError

	UpsertRecord(ctx context.Context, params UpsertHubspotSyncRecordParams) *apierror.APIError
	GetRecord(ctx context.Context, accountID, augnoType, augnoID string) (*HubspotSyncRecord, *apierror.APIError)

	CreateReview(ctx context.Context, params CreateHubspotCompanyReviewParams) (*HubspotCompanyReview, *apierror.APIError)
	GetReview(ctx context.Context, accountID, id string) (*HubspotCompanyReview, *apierror.APIError)
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
	IsInAccount(ctx context.Context, accountID, propertyID string) (bool, *apierror.APIError)
	DeleteAttributesByPropertyID(ctx context.Context, propertyID, accountID string) *apierror.APIError
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
}

// ProductionStepQueryRepo provides read-only methods the batch service needs from production steps.
type ProductionStepQueryRepo interface {
	Find(ctx context.Context, accountID, id string) (*ProductionStepDetail, *apierror.APIError)
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
	Create(ctx context.Context, id, responsibleUserID, number, accountID string) *apierror.APIError
	GetNextNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
}

// ProductionRunRepo provides full CRUD access for production run management.
type ProductionRunRepo interface {
	List(ctx context.Context, params ListProductionRunsParams) (*ListProductionRunsResult, *apierror.APIError)
	Get(ctx context.Context, params GetProductionRunParams) (*ProductionRun, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateProductionRunParams, number string) (*ProductionRun, *apierror.APIError)
	Update(ctx context.Context, params UpdateProductionRunParams) (*ProductionRun, *apierror.APIError)
	Delete(ctx context.Context, params DeleteProductionRunParams) *apierror.APIError
	ExistsByNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError)
	GetNextNumber(ctx context.Context, accountID string) (string, *apierror.APIError)
	IsCompleted(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	DeleteBatchesByRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError
	FindOrderIDsByRun(ctx context.Context, accountID, productionRunID string) ([]string, *apierror.APIError)
	UnlinkOrdersFromRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError
	DeleteReservedInventoryIssuesByOrder(ctx context.Context, accountID, orderID string) *apierror.APIError
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
}

// InventoryQueryRepo provides read-only access to inventory data.
type InventoryQueryRepo interface {
	FetchCurrentInventory(ctx context.Context, itemID, ownerAccountID string) (*InventorySnapshot, *apierror.APIError)
	FetchOnHandInventoryBulk(ctx context.Context, itemIDs []string, ownerAccountID string) ([]*BulkOnHandInventory, *apierror.APIError)
	FetchPhysicalInventory(ctx context.Context, itemID, ownerAccountID string) (float64, *apierror.APIError)
}

type ProductLineRepo interface {
	List(ctx context.Context, params ListProductLinesParams) (*ListProductLinesResult, *apierror.APIError)
	Get(ctx context.Context, params GetProductLineParams) (*ProductLineFull, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateProductLineParams) (*ProductLineFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateProductLineParams) (*ProductLineFull, *apierror.APIError)
	Delete(ctx context.Context, params DeleteProductLineParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	GetUnitGroup(ctx context.Context, unitGroupID string, includes []string) (*ProductLineUnitGroup, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*ProductLineFull, *apierror.APIError)
}

type ItemCategoryRepo interface {
	List(ctx context.Context, params ListItemCategoriesParams) (*ListItemCategoriesResult, *apierror.APIError)
	Get(ctx context.Context, params GetItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)
	Update(ctx context.Context, params UpdateItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)
	Delete(ctx context.Context, params DeleteItemCategoryParams) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, itemCategoryID string) (bool, *apierror.APIError)
	AddProperty(ctx context.Context, params AddItemCategoryPropertyParams) *apierror.APIError
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
	CreateInventoryReceipt(ctx context.Context, params CreateInventoryReceiptParams) *apierror.APIError
	// CreateInventoryIssue creates an inventory issue for negative delta.
	CreateInventoryIssue(ctx context.Context, params CreateInventoryIssueParams) *apierror.APIError
	// CreateInventoryLog creates a point-in-time inventory snapshot log.
	CreateInventoryLog(ctx context.Context, params CreateInventoryLogParams) *apierror.APIError
	// CreateInventoryChangeLog creates an audit trail entry for an inventory change.
	CreateInventoryChangeLog(ctx context.Context, params CreateInventoryChangeLogParams) *apierror.APIError
	// CreateQuantityForInventory creates a quantity record for use in inventory operations.
	CreateQuantityForInventory(ctx context.Context, quantityID, value, unitID string) *apierror.APIError
	// CreateRateForInventory creates a rate record for use in inventory operations.
	CreateRateForInventory(ctx context.Context, rateID, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
}

// OrderQueryRepo provides read-only queries for orders needed by the batch/production system.
type OrderQueryRepo interface {
	// FindIDByProductionRun returns the order ID for the given production run, or nil if no order exists.
	FindIDByProductionRun(ctx context.Context, accountID, productionRunID string) (*string, *apierror.APIError)
}

// InventoryReservationRepo manages inventory reservations for orders during production step execution.
type InventoryReservationRepo interface {
	// CreateMaterialReservation creates a reserved inventory issue for a material demand linked to an order.
	CreateMaterialReservation(ctx context.Context, params CreateMaterialReservationParams) *apierror.APIError
	// ReduceReservedForOrderItem reduces the reserved quantity for an order item by the given shortfall amount.
	ReduceReservedForOrderItem(ctx context.Context, params OrderReservationReductionParams) *apierror.APIError
	// ReduceReservedForOrderMaterials reduces reserved quantities for upstream materials of an order.
	ReduceReservedForOrderMaterials(ctx context.Context, orderID, accountID string, demands []MaterialDemandItem) *apierror.APIError
	// AllocateReservationsForConsumption allocates existing reservations for consumed materials. Returns the remaining quantity that could not be allocated from reservations.
	AllocateReservationsForConsumption(ctx context.Context, params ConsumptionAllocationParams) (*ConsumptionAllocationResult, *apierror.APIError)
	// AllocateOpenIssuesForItem performs FIFO allocation of all open inventory issues for the given item against available receipts. Used after receiving inventory.
	AllocateOpenIssuesForItem(ctx context.Context, accountID, itemID string) *apierror.APIError
}

// MaterialDemandRepo calculates material demand from a bill of materials.
type MaterialDemandRepo interface {
	// GetMaterialDemand calculates the material demand for producing the given items.
	GetMaterialDemand(ctx context.Context, accountID string, productItemID string, measure decimal.Decimal, unitID string) ([]MaterialDemandItem, *apierror.APIError)
}

// UnitConversionRepo provides unit conversion capabilities.
type UnitConversionRepo interface {
	// ConvertValue converts a measure from one unit to another within the same unit group. Returns the converted measure.
	ConvertValue(ctx context.Context, measure decimal.Decimal, fromUnitID, toUnitID string) (decimal.Decimal, *apierror.APIError)
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
	GetNextCustomerNumber(ctx context.Context, accountID string) (int64, *apierror.APIError)
	UpdateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string, value string) *apierror.APIError
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
	GetDemandForecast(ctx context.Context, params GetDemandForecastParams) (*DemandForecastResult, *apierror.APIError)
	GetOeeByDepartment(ctx context.Context, params AnalyzeOeeParams) ([]OeeDepartment, *apierror.APIError)
	GetWeeksOfSales(ctx context.Context, params AnalyzeWeeksOfSalesParams) (*WeeksOfSalesResult, *apierror.APIError)
}

type MachineRepo interface {
	List(ctx context.Context, params ListMachinesParams) (*ListMachinesResult, *apierror.APIError)
	Get(ctx context.Context, params GetMachineParams) (*Machine, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Machine, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateMachineParams) (*Machine, *apierror.APIError)
	Update(ctx context.Context, params UpdateMachineParams) (*Machine, *apierror.APIError)
	Delete(ctx context.Context, params DeleteMachineParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
}

type DepartmentRepo interface {
	List(ctx context.Context, params ListDepartmentsParams) (*ListDepartmentsResult, *apierror.APIError)
	Get(ctx context.Context, params GetDepartmentParams) (*Department, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Department, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateDepartmentParams) (*Department, *apierror.APIError)
	Update(ctx context.Context, params UpdateDepartmentParams) (*Department, *apierror.APIError)
	Delete(ctx context.Context, params DeleteDepartmentParams) *apierror.APIError
	ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError)
	SetMachinesDepartmentID(ctx context.Context, departmentID string, machineIDs []string) *apierror.APIError
	SetScanningStationsDepartmentID(ctx context.Context, departmentID, accountID string, scanningStationIDs []string) *apierror.APIError
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
	Create(ctx context.Context, id string, params CreateMaterialParams) *apierror.APIError
	Update(ctx context.Context, params UpdateMaterialParams) *apierror.APIError
	DeleteByID(ctx context.Context, accountID, materialID string) *apierror.APIError
	DeleteByItemID(ctx context.Context, accountID, itemID string) *apierror.APIError
	InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError
	InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError
	InsertItem(ctx context.Context, id string, params CreateMaterialParams) *apierror.APIError
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
	GetContactsByOrders(ctx context.Context, salesOrderIDs []string) (map[string]*SalesOrderContacts, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateSalesOrderParams) (*SalesOrder, *apierror.APIError)
	Update(ctx context.Context, params UpdateSalesOrderParams) (*SalesOrder, *apierror.APIError)
	Delete(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	UpdateStatus(ctx context.Context, accountID, salesOrderID, statusCode string, issuedAt, completedAt *time.Time) *apierror.APIError
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
	GetLinesForBOM(ctx context.Context, salesOrderID string) ([]SalesOrderLineForBOM, *apierror.APIError)
	SetProductionRunID(ctx context.Context, accountID, salesOrderID, productionRunID string) *apierror.APIError
	GetSaleLinesForIssue(ctx context.Context, salesOrderID string) ([]SalesOrderSaleLineForIssue, *apierror.APIError)
	CreateReservedInventoryIssue(ctx context.Context, id, accountID, itemID, quantityID, orderID string) *apierror.APIError
	DeleteInventoryAllocationsByReservedIssues(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	DeleteReservedInventoryIssues(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	GetAcknowledgementRecipients(ctx context.Context, salesOrderID string) ([]string, *apierror.APIError)
	MarkAcknowledgementSent(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
	CreateEmailContact(ctx context.Context, id, salesOrderID, accountUserID, notificationTypeCode string) *apierror.APIError
	DeleteEmailContactsByOrderAndType(ctx context.Context, salesOrderID, notificationTypeCode string) *apierror.APIError
	NoteFirstShipAt(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
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
	HasShippedAgainstOrderLine(ctx context.Context, salesOrderLineID string) (bool, *apierror.APIError)
	DeleteCascade(ctx context.Context, salesOrderLineID string) *apierror.APIError
	CreateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError
}

type PurchaseOrderRepo interface {
	List(ctx context.Context, params ListPurchaseOrdersParams) (*ListPurchaseOrdersResult, *apierror.APIError)
	Get(ctx context.Context, accountID, purchaseOrderID string) (*PurchaseOrder, *apierror.APIError)
	GetLines(ctx context.Context, salesOrderID string) ([]*PurchaseOrderLine, *apierror.APIError)
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
	InsertInventoryReceiptForDelivery(ctx context.Context, receiptID, accountID, itemID, quantityID, unitCostID string, storageLocationID, lotID, orderID *string) *apierror.APIError
	MarkPurchaseOrderFulfilled(ctx context.Context, purchaseOrderID, accountID string) *apierror.APIError
	FindOpenIssuesForItem(ctx context.Context, accountID, itemID string) ([]OpenInventoryIssue, *apierror.APIError)
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
	Update(ctx context.Context, params UpdateInvoiceParams) (*InvoiceSummary, *apierror.APIError)
	ListByCustomer(ctx context.Context, params ListCustomerInvoicesParams) (*ListCustomerInvoicesResult, *apierror.APIError)
	IsDuplicateNumber(ctx context.Context, accountID, number string) (bool, *apierror.APIError)
	GetEmailRecipients(ctx context.Context, invoiceID string) ([]string, *apierror.APIError)
	MarkEmailSent(ctx context.Context, accountID, invoiceID string) *apierror.APIError
	DeleteLinesByInvoice(ctx context.Context, invoiceID string) *apierror.APIError
	Delete(ctx context.Context, accountID, invoiceID string) *apierror.APIError
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
	GetDepartments(ctx context.Context, pickID string) ([]*PickDepartment, *apierror.APIError)
	UpdateNumber(ctx context.Context, accountID, pickID, number string) *apierror.APIError
	UpdateFinishedAt(ctx context.Context, accountID, pickID string, finishedAt time.Time) *apierror.APIError
	HasShippedItems(ctx context.Context, accountID, pickID string) (bool, *apierror.APIError)
	VoidAllLines(ctx context.Context, pickID string) *apierror.APIError
	DeleteDuplicatePickLines(ctx context.Context, accountID, pickID string) *apierror.APIError
	ClearFinishedAt(ctx context.Context, accountID, pickID string) *apierror.APIError
	PickAllLines(ctx context.Context, pickID string) *apierror.APIError
	GetShipmentNumbers(ctx context.Context, params GetPickShipmentsParams) (*PickShipmentsResult, *apierror.APIError)
	IsInAccount(ctx context.Context, accountID, pickID string) (bool, *apierror.APIError)
	FindLinesToPack(ctx context.Context, pickID string) ([]*PickLine, *apierror.APIError)
	PackLines(ctx context.Context, pickID string) *apierror.APIError
	MarkFinishedIfAllPacked(ctx context.Context, pickID string) *apierror.APIError
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
	UpdateQuantity(ctx context.Context, pickLineID, quantityValue string) *apierror.APIError
	PickRemainingQuantity(ctx context.Context, pickLineID string) *apierror.APIError
	VoidLine(ctx context.Context, pickLineID string) *apierror.APIError
	IsInPick(ctx context.Context, pickLineID, pickID string) (bool, *apierror.APIError)
	CreateForRemaining(ctx context.Context, id, quantityID, pickID, orderLineID string) *apierror.APIError
	CalculateRemainingForOrderLine(ctx context.Context, orderLineID string) (remainingValue string, unitID string, apiErr *apierror.APIError)
	HasUnpackedPickLineForOrderLine(ctx context.Context, orderLineID string) (bool, *apierror.APIError)
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
	GetNextSettlementNumber(ctx context.Context, accountID string) (int64, *apierror.APIError)
	UpdateNextSettlementNumber(ctx context.Context, sysPropertyID, accountID string, value int64) *apierror.APIError
	GetDollarUnitID(ctx context.Context) (string, *apierror.APIError)
	DeleteOrphanedAdjustmentTransactions(ctx context.Context, settlementID string) *apierror.APIError
	UpdateTransactionsFullyAllocated(ctx context.Context, transactionIDs []string, isFullyAllocated bool) *apierror.APIError
	UpdateInvoicePaymentStatus(ctx context.Context, invoiceID string, isPaidInFull, isOverPaid bool) *apierror.APIError
	GetInvoicePaymentFlags(ctx context.Context, invoiceIDs []string) ([]InvoicePaymentFlags, *apierror.APIError)
}

type TransactionRepo interface {
	Create(ctx context.Context, txID, number, typeCode, accountID, customerAccountID string, stripePaymentID *string, methodCode *string, adjustmentTypeCode *string, responsibleUserID *string, note *string, amountValue string, amountUnitID string) *apierror.APIError
	FindByStripePaymentID(ctx context.Context, stripePaymentID string) (*TransactionRecord, *apierror.APIError)
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
	GetNextCustomerNumber(ctx context.Context, accountID string) (int64, *apierror.APIError)
	UpdateNextCustomerNumber(ctx context.Context, sysPropertyID, accountID string, value int64) *apierror.APIError
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
	Delete(ctx context.Context, accountID, shipmentID string) *apierror.APIError
	MarkShipped(ctx context.Context, accountID, shipmentID, shippedByID string) *apierror.APIError
	MarkVoided(ctx context.Context, accountID, shipmentID string) *apierror.APIError
	FindInvoiceIDByShipment(ctx context.Context, accountID, shipmentID string) (*string, *apierror.APIError)
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
	ConnectProductionStepsByName(ctx context.Context, accountID, scanningStationID, name string) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError)
	FindType(ctx context.Context, accountID, id string) (string, *apierror.APIError)
}

type ShippingCaseRepo interface {
	Get(ctx context.Context, accountID, shippingCaseID string) (*ShippingCase, *apierror.APIError)
	Update(ctx context.Context, params UpdateShippingCaseParams) *apierror.APIError
	Delete(ctx context.Context, accountID, shippingCaseID string) *apierror.APIError
	IsInAccount(ctx context.Context, accountID, shippingCaseID string) (bool, *apierror.APIError)
	GetNumber(ctx context.Context, accountID, shippingCaseID string) (string, *apierror.APIError)
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
	Get(ctx context.Context, params GetLocationParams) (*Location, *apierror.APIError)
	GetByIDs(ctx context.Context, accountID string, ids []string) ([]*Location, *apierror.APIError)
	Create(ctx context.Context, id string, params CreateLocationParams) (*Location, *apierror.APIError)
	Update(ctx context.Context, params UpdateLocationParams) (*Location, *apierror.APIError)
	Delete(ctx context.Context, params DeleteLocationParams) *apierror.APIError
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
}
