package domain

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type UserSvc interface {
	GetUser(ctx context.Context, userID string) (*UserRecord, *apierror.APIError)
	// BatchGetUsersByIDs returns users matching the given IDs that are affiliated with the target account.
	BatchGetUsersByIDs(ctx context.Context, ids []string) ([]*UserRecord, *apierror.APIError)
	UpdateUser(ctx context.Context, userID string, params UpdateUserParams) (*UserRecord, *apierror.APIError)
	UploadUserPhoto(ctx context.Context, userID string, file []byte, contentType string) *apierror.APIError
	GetUserPhotoURL(ctx context.Context, userID string) (*string, *apierror.APIError)
}

type SandboxSvc interface {
	// CreateSandbox creates a sandbox account with the given name.
	//
	// Preconditions:
	//   - The caller must be authorized to create sandbox accounts.
	//
	// Side effects:
	//   - Persists a new owner account and its associated sandbox account.
	CreateSandbox(ctx context.Context, name string, mode constants.SandboxMode) (*SandboxAccount, *apierror.APIError)

	// GetSandboxAccountByOwner returns the sandbox account ID associated with the given owner account.
	GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError)

	// ListSandboxAccounts returns a paginated list of sandbox accounts visible to the caller.
	//
	// Pagination:
	//   - If cursor is non-nil, results begin after the provided cursor.
	//   - limit controls the maximum number of results returned.
	ListSandboxAccounts(ctx context.Context, cursor *string, limit int32, query *string, includes []string) (*ListSandboxAccountsResult, *apierror.APIError)

	// GetSandbox returns a single sandbox account by its type ID. The caller must have read permission on the sandbox domain and the sandbox must belong to the caller's target account.
	GetSandbox(ctx context.Context, sandboxTypeID string, includes []string) (*SandboxAccount, *apierror.APIError)

	// DeleteSandbox deletes a sandbox account and its underlying account record. Account-scoped data is purged asynchronously via an outbox message.
	DeleteSandbox(ctx context.Context, sandboxTypeID string) *apierror.APIError

	// BatchGetSandboxesByIDs returns sandbox accounts matching the input type IDs that the caller's account is authorized to read. Used by the api-gateway resourcekit include resolver.
	BatchGetSandboxesByIDs(ctx context.Context, typeIDs []string) ([]*SandboxAccount, *apierror.APIError)
}

type UnitSvc interface {
	// ListUnits returns a paginated list of units visible to the caller's account. Includes both account-specific and global (system) units.
	ListUnits(ctx context.Context, params ListUnitsParams) (*ListUnitsResult, *apierror.APIError)

	// GetUnit returns a single unit by ID. The unit must belong to the caller's account or be a system (global) unit.
	GetUnit(ctx context.Context, unitID string) (*Unit, *apierror.APIError)

	// CreateUnit creates a new account-owned unit.
	CreateUnit(ctx context.Context, params CreateUnitParams) (*Unit, *apierror.APIError)

	// UpdateUnit partially updates an account-owned unit. System units cannot be updated.
	UpdateUnit(ctx context.Context, params UpdateUnitParams) (*Unit, *apierror.APIError)

	// DeleteUnit deletes an account-owned unit and cascades to unit_group_unit associations. System units cannot be deleted.
	DeleteUnit(ctx context.Context, unitID string) *apierror.APIError

	// ValidateUnits validates unit abbreviations and returns matching units. Performs case-insensitive matching against both account and system units.
	ValidateUnits(ctx context.Context, params ValidateUnitsParams) (*ValidateUnitsResult, *apierror.APIError)

	// BatchGetUnitsByIDs returns units by ID for the api-gateway include resolver.
	BatchGetUnitsByIDs(ctx context.Context, ids []string) ([]*Unit, *apierror.APIError)
}

type UnitGroupSvc interface {
	// ListUnitGroups returns a paginated list of unit groups visible to the caller's account. Includes both account-specific and system unit groups.
	ListUnitGroups(ctx context.Context, params ListUnitGroupsParams) (*ListUnitGroupsResult, *apierror.APIError)

	// GetUnitGroup returns a single unit group by ID with its unit conversions. Supports both internal and customer (cross-account) access.
	GetUnitGroup(ctx context.Context, params GetUnitGroupParams) (*UnitGroupFull, *apierror.APIError)

	// CreateUnitGroup creates a new account-owned unit group with optional unit conversions. Idempotent via idempotency keys.
	CreateUnitGroup(ctx context.Context, params CreateUnitGroupParams) (*UnitGroupFull, *apierror.APIError)

	// UpdateUnitGroup partially updates an account-owned unit group. System unit groups cannot be modified. Idempotent via idempotency keys.
	UpdateUnitGroup(ctx context.Context, params UpdateUnitGroupParams) (*UnitGroupFull, *apierror.APIError)

	// DeleteUnitGroup deletes an account-owned unit group and cascades to all unit_group_unit records. System unit groups cannot be deleted.
	DeleteUnitGroup(ctx context.Context, unitGroupID string) *apierror.APIError

	// UpsertUnitGroupUnit creates or updates a unit conversion within a unit group. The parent unit group must be account-owned.
	UpsertUnitGroupUnit(ctx context.Context, params UpsertUnitGroupUnitParams) (*UnitGroupUnit, *apierror.APIError)

	// DeleteUnitGroupUnit removes a unit conversion from a unit group. The parent unit group must be account-owned.
	DeleteUnitGroupUnit(ctx context.Context, params DeleteUnitGroupUnitParams) *apierror.APIError

	// ListUnitGroupUnits returns all unit conversions for a unit group.
	ListUnitGroupUnits(ctx context.Context, unitGroupID string, includes []string) ([]*UnitGroupUnit, *apierror.APIError)

	// GetUnitGroupUnit returns a single unit conversion by ID.
	GetUnitGroupUnit(ctx context.Context, params GetUnitGroupUnitParams) (*UnitGroupUnit, *apierror.APIError)

	// BatchGetUnitGroupsByIDs returns unit groups by ID for the api-gateway include resolver.
	BatchGetUnitGroupsByIDs(ctx context.Context, ids []string) ([]*UnitGroupFull, *apierror.APIError)

	// BatchGetUnitGroupUnitsByIDs returns unit group units by ID for the api-gateway include resolver.
	BatchGetUnitGroupUnitsByIDs(ctx context.Context, ids []string) ([]*UnitGroupUnit, *apierror.APIError)
}

type PaymentTermSvc interface {
	// ListPaymentTerms returns a paginated list of payment terms visible to the caller's account. Includes both account-specific and default (system) payment terms.
	ListPaymentTerms(ctx context.Context, params ListPaymentTermsParams) (*ListPaymentTermsResult, *apierror.APIError)

	// GetPaymentTerm returns a single payment term by ID. The payment term must belong to the caller's account or be a default (global) payment term.
	GetPaymentTerm(ctx context.Context, paymentTermID string) (*PaymentTerm, *apierror.APIError)

	// BatchGetPaymentTermsByIDs returns payment terms by ID for the api-gateway include resolver. Authorization matches GetPaymentTerm (caller's account + system terms).
	BatchGetPaymentTermsByIDs(ctx context.Context, ids []string) ([]*PaymentTerm, *apierror.APIError)

	// CreatePaymentTerm creates a new account-owned payment term.
	CreatePaymentTerm(ctx context.Context, params CreatePaymentTermParams) (*PaymentTerm, *apierror.APIError)

	// UpdatePaymentTerm partially updates an account-owned payment term. Default payment terms cannot be updated.
	UpdatePaymentTerm(ctx context.Context, params UpdatePaymentTermParams) (*PaymentTerm, *apierror.APIError)

	// DeletePaymentTerm deletes an account-owned payment term. Default payment terms cannot be deleted.
	DeletePaymentTerm(ctx context.Context, paymentTermID string) *apierror.APIError
}

type ShippingTermSvc interface {
	// ListShippingTerms returns a paginated list of shipping terms visible to the caller's account. Includes both account-specific and default (system) shipping terms.
	ListShippingTerms(ctx context.Context, params ListShippingTermsParams) (*ListShippingTermsResult, *apierror.APIError)

	// GetShippingTerm returns a single shipping term by ID. The shipping term must belong to the caller's account or be a default (global) shipping term.
	GetShippingTerm(ctx context.Context, params GetShippingTermParams) (*ShippingTerm, *apierror.APIError)

	// CreateShippingTerm creates a new account-owned shipping term.
	CreateShippingTerm(ctx context.Context, params CreateShippingTermParams) (*ShippingTerm, *apierror.APIError)

	// UpdateShippingTerm partially updates an account-owned shipping term. Default shipping terms cannot be updated.
	UpdateShippingTerm(ctx context.Context, params UpdateShippingTermParams) (*ShippingTerm, *apierror.APIError)

	// DeleteShippingTerm deletes an account-owned shipping term. Default shipping terms cannot be deleted.
	DeleteShippingTerm(ctx context.Context, shippingTermID string) *apierror.APIError

	// BatchGetShippingTermsByIDs returns shipping terms matching the given IDs that the caller's account is authorized to read.
	BatchGetShippingTermsByIDs(ctx context.Context, ids []string) ([]*ShippingTerm, *apierror.APIError)
}

type AccountGroupSvc interface {
	// ListAccountGroups returns a paginated list of account groups for the caller's account.
	ListAccountGroups(ctx context.Context, params ListAccountGroupsParams) (*ListAccountGroupsResult, *apierror.APIError)

	// GetAccountGroup returns a single account group by ID.
	GetAccountGroup(ctx context.Context, accountGroupID string) (*AccountGroup, *apierror.APIError)

	// CreateAccountGroup creates a new account group.
	CreateAccountGroup(ctx context.Context, params CreateAccountGroupParams) (*AccountGroup, *apierror.APIError)

	// UpdateAccountGroup partially updates an account group.
	UpdateAccountGroup(ctx context.Context, params UpdateAccountGroupParams) (*AccountGroup, *apierror.APIError)

	// DeleteAccountGroup deletes an account group.
	DeleteAccountGroup(ctx context.Context, accountGroupID string) *apierror.APIError

	// BatchGetAccountGroupsByIDs returns account groups matching the input IDs that the caller's account is authorized to read. Used by the api-gateway resourcekit include resolver.
	BatchGetAccountGroupsByIDs(ctx context.Context, ids []string) ([]*AccountGroup, *apierror.APIError)
}

type AccountGroupProductLineAccessSvc interface {
	// ListAccountGroupProductLineAccess returns a paginated list of product line access records grouped by account group.
	ListAccountGroupProductLineAccess(ctx context.Context, params ListAccountGroupProductLineAccessParams) (*ListAccountGroupProductLineAccessResult, *apierror.APIError)

	// GetAccountGroupProductLineAccess returns the product line access for a single account group.
	GetAccountGroupProductLineAccess(ctx context.Context, accountGroupID string) (*AccountGroupProductLineAccess, *apierror.APIError)

	// CreateAccountGroupProductLineAccess creates a new product line access record for an account group.
	CreateAccountGroupProductLineAccess(ctx context.Context, params CreateAccountGroupProductLineAccessParams) (*AccountGroupProductLineAccess, *apierror.APIError)

	// UpdateAccountGroupProductLineAccess replaces all product lines for an account group.
	UpdateAccountGroupProductLineAccess(ctx context.Context, params UpdateAccountGroupProductLineAccessParams) (*AccountGroupProductLineAccess, *apierror.APIError)

	// DeleteAccountGroupProductLineAccess removes all product line access for an account group.
	DeleteAccountGroupProductLineAccess(ctx context.Context, accountGroupID string) *apierror.APIError

	// BatchGetAccountGroupProductLineAccessByIDs returns access records for the given account_group_ids. Used by the api-gateway resourcekit resolver.
	BatchGetAccountGroupProductLineAccessByIDs(ctx context.Context, accountGroupIDs []string) ([]*AccountGroupProductLineAccess, *apierror.APIError)
}

type CustomerProductLineAccessSvc interface {
	// ListCustomerProductLineAccess returns a paginated list of product line access records grouped by customer.
	ListCustomerProductLineAccess(ctx context.Context, params ListCustomerProductLineAccessParams) (*ListCustomerProductLineAccessResult, *apierror.APIError)

	// GetCustomerProductLineAccess returns the product line access for a single customer.
	GetCustomerProductLineAccess(ctx context.Context, customerID string) (*CustomerProductLineAccess, *apierror.APIError)

	// CreateCustomerProductLineAccess creates a new product line access record for a customer.
	CreateCustomerProductLineAccess(ctx context.Context, params CreateCustomerProductLineAccessParams) (*CustomerProductLineAccess, *apierror.APIError)

	// UpdateCustomerProductLineAccess replaces all product lines for a customer.
	UpdateCustomerProductLineAccess(ctx context.Context, params UpdateCustomerProductLineAccessParams) (*CustomerProductLineAccess, *apierror.APIError)

	// DeleteCustomerProductLineAccess removes all product line access for a customer.
	DeleteCustomerProductLineAccess(ctx context.Context, customerID string) *apierror.APIError

	// BatchGetCustomerProductLineAccessByIDs returns access records for the given customer_ids. Used by the api-gateway resourcekit resolver.
	BatchGetCustomerProductLineAccessByIDs(ctx context.Context, customerIDs []string) ([]*CustomerProductLineAccess, *apierror.APIError)
}

type AddressSvc interface {
	// ListAddresses returns a paginated list of addresses for an account.
	ListAddresses(ctx context.Context, params ListAddressesParams) (*ListAddressesResult, *apierror.APIError)

	// GetAddress returns a single address by ID within an account.
	GetAddress(ctx context.Context, params GetAddressParams) (*Address, *apierror.APIError)

	// CreateAddress creates a new address linked to an account.
	CreateAddress(ctx context.Context, params CreateAddressParams) (*Address, *apierror.APIError)

	// UpdateAddress partially updates an address within an account.
	UpdateAddress(ctx context.Context, params UpdateAddressParams) (*Address, *apierror.APIError)

	// DeleteAddress deletes an address from an account.
	DeleteAddress(ctx context.Context, params DeleteAddressParams) *apierror.APIError

	// BatchGetAddressesByIDs returns addresses matching the input IDs that the caller's account is authorized to read. Used by the api-gateway resourcekit include resolver.
	BatchGetAddressesByIDs(ctx context.Context, ids []string) ([]*Address, *apierror.APIError)
}

type AddressValidationSvc interface {
	// Autocomplete returns address autocomplete suggestions.
	Autocomplete(ctx context.Context, input string, sessionToken *string) ([]AddressSuggestion, *apierror.APIError)

	// GetPlaceDetails returns parsed address components from a Google Places ID.
	GetPlaceDetails(ctx context.Context, placeID string, sessionToken *string) (*AddressDetailsResult, *apierror.APIError)

	// ValidateAddress validates an address using Google Address Validation API.
	ValidateAddress(ctx context.Context, addressLine1 string, addressLine2 *string, city, state, postalCode, country string) (*ValidatedAddress, *apierror.APIError)
}

type AccountStatusSvc interface {
	// ListAccountStatuses returns a paginated list of account statuses.
	ListAccountStatuses(ctx context.Context, params ListAccountStatusesParams) (*ListAccountStatusesResult, *apierror.APIError)

	// GetAccountStatus returns a single account status by ID or code.
	GetAccountStatus(ctx context.Context, identifier string) (*AccountStatus, *apierror.APIError)

	// BatchGetAccountStatusesByIDs returns account statuses by ID for the api-gateway include resolver.
	BatchGetAccountStatusesByIDs(ctx context.Context, ids []string) ([]*AccountStatus, *apierror.APIError)
}

type PrioritySvc interface {
	// ListPriorities returns a paginated list of priorities.
	ListPriorities(ctx context.Context, params ListPrioritiesParams) (*ListPrioritiesResult, *apierror.APIError)

	// GetPriority returns a single priority by ID or code.
	GetPriority(ctx context.Context, identifier string) (*Priority, *apierror.APIError)

	// BatchGetPrioritiesByIDs returns priorities by ID for the api-gateway include resolver. Priorities are system-wide, no per-caller scoping.
	BatchGetPrioritiesByIDs(ctx context.Context, ids []string) ([]*Priority, *apierror.APIError)
}

type AccountUserSvc interface {
	// ListAccountUsers returns a paginated list of account users.
	ListAccountUsers(ctx context.Context, params ListAccountUsersParams) (*ListAccountUsersResult, *apierror.APIError)

	// GetAccountUser returns a single account user by account_user ID.
	GetAccountUser(ctx context.Context, accountUserID string, includes []string) (*AccountUserDetail, *apierror.APIError)

	// CreateAccountUser creates a new account user.
	CreateAccountUser(ctx context.Context, params CreateAccountUserParams) (*AccountUserDetail, *apierror.APIError)

	// UpdateAccountUser partially updates an account user, optionally including notification preferences.
	UpdateAccountUser(ctx context.Context, params UpdateAccountUserParams, includes []string) (*AccountUserDetail, *apierror.APIError)

	// UpdateAccountUserStatus transitions an account user to the given target status.
	UpdateAccountUserStatus(ctx context.Context, accountUserID string, targetStatus constants.AccountUserStatus) *apierror.APIError

	// UpdateAccountUserPassword updates the password for a scanner-role account user.
	UpdateAccountUserPassword(ctx context.Context, accountUserID, requesterPassword, newPassword string) *apierror.APIError

	// BatchGetAccountUsersByIDs returns account users matching the given IDs.
	BatchGetAccountUsersByIDs(ctx context.Context, ids []string) ([]*AccountUserDetail, *apierror.APIError)
}

type SalesTargetSvc interface {
	// ListSalesTargets returns a paginated list of sales targets for an account user.
	ListSalesTargets(ctx context.Context, params ListSalesTargetsParams) (*ListSalesTargetsResult, *apierror.APIError)

	// CreateSalesTarget creates a new sales target.
	CreateSalesTarget(ctx context.Context, params CreateSalesTargetParams) (*SalesTarget, *apierror.APIError)

	// UpsertSalesTarget creates or updates a sales target by ID.
	UpsertSalesTarget(ctx context.Context, params UpsertSalesTargetParams) (*SalesTarget, *apierror.APIError)
}

type AdjustmentTypeSvc interface {
	// ListAdjustmentTypes returns a paginated list of adjustment types.
	ListAdjustmentTypes(ctx context.Context, params ListAdjustmentTypesParams) (*ListAdjustmentTypesResult, *apierror.APIError)
	// BatchGetAdjustmentTypesByIDs returns adjustment types by ID for the api-gateway include resolver.
	BatchGetAdjustmentTypesByIDs(ctx context.Context, ids []string) ([]*AdjustmentType, *apierror.APIError)
}

type AccountPriceSvc interface {
	// ListAccountPrices returns a paginated list of account prices for the caller's account. Customer actors can only see prices where they are the recipient.
	ListAccountPrices(ctx context.Context, params ListAccountPricesParams) (*ListAccountPricesResult, *apierror.APIError)

	// GetAccountPrice returns a single account price by ID.
	GetAccountPrice(ctx context.Context, accountPriceID string) (*AccountPrice, *apierror.APIError)

	// CreateAccountPrice creates a new account price with rate, category, and attribute associations.
	CreateAccountPrice(ctx context.Context, params CreateAccountPriceParams) (*AccountPrice, *apierror.APIError)

	// UpdateAccountPrice partially updates an account price. If categories or attributes are provided, they are replaced entirely (delete-all-then-recreate).
	UpdateAccountPrice(ctx context.Context, params UpdateAccountPriceParams) (*AccountPrice, *apierror.APIError)

	// DeleteAccountPrice deletes an account price and cascades to associations and rate.
	DeleteAccountPrice(ctx context.Context, accountPriceID string) *apierror.APIError
}

type AccountIntegrationSvc interface {
	// ListAccountIntegrations returns a paginated list of integrations for the caller's account.
	ListAccountIntegrations(ctx context.Context, params ListAccountIntegrationsParams) (*ListAccountIntegrationsResult, *apierror.APIError)

	// CreateAccountIntegration creates or upserts an integration. If one with the same code exists, it updates name and credentials instead of inserting.
	CreateAccountIntegration(ctx context.Context, params CreateAccountIntegrationParams) (*AccountIntegration, *apierror.APIError)

	// UpdateAccountIntegration updates name and/or is_active on an integration.
	UpdateAccountIntegration(ctx context.Context, params UpdateAccountIntegrationParams) (*AccountIntegration, *apierror.APIError)

	// DeleteAccountIntegration deletes an integration and returns the deleted resource.
	DeleteAccountIntegration(ctx context.Context, params DeleteAccountIntegrationParams) (*AccountIntegration, *apierror.APIError)

	// GetStripePublishableKey returns the Stripe publishable key for the account.
	GetStripePublishableKey(ctx context.Context) (string, *apierror.APIError)

	// HasStripeIntegration returns whether the account has a Stripe integration.
	HasStripeIntegration(ctx context.Context) (bool, *apierror.APIError)

	// BatchGetAccountIntegrationsByIDs returns account integrations matching the input IDs that the caller's account is authorized to read. Used by the api-gateway resourcekit include resolver.
	BatchGetAccountIntegrationsByIDs(ctx context.Context, ids []string) ([]*AccountIntegration, *apierror.APIError)
}

type PropertySvc interface {
	// ListProperties returns a paginated list of properties for the caller's account.
	ListProperties(ctx context.Context, params ListPropertiesParams, includes []string) (*ListPropertiesResult, *apierror.APIError)

	// GetProperty returns a single property by ID.
	GetProperty(ctx context.Context, propertyID string, includes []string) (*Property, *apierror.APIError)

	// CreateProperty creates a new property.
	CreateProperty(ctx context.Context, params CreatePropertyParams, includes []string) (*Property, *apierror.APIError)

	// UpdateProperty partially updates a property.
	UpdateProperty(ctx context.Context, params UpdatePropertyParams, includes []string) (*Property, *apierror.APIError)

	// DeleteProperty deletes a property and cascades to its attributes.
	DeleteProperty(ctx context.Context, propertyID string) *apierror.APIError

	// BatchGetPropertiesByIDs returns properties matching the input IDs that belong to the caller's account. Always populates attributes.
	BatchGetPropertiesByIDs(ctx context.Context, ids []string) ([]*Property, *apierror.APIError)
}

type AttributeSvc interface {
	// ListAttributes returns a paginated list of attributes for a property.
	ListAttributes(ctx context.Context, params ListAttributesParams) (*ListAttributesResult, *apierror.APIError)

	// GetAttribute returns a single attribute by ID within a property.
	GetAttribute(ctx context.Context, propertyID, attributeID string) (*Attribute, *apierror.APIError)

	// BatchGetAttributesByIDs returns attributes matching the input IDs that belong to the caller's account.
	BatchGetAttributesByIDs(ctx context.Context, ids []string) ([]*Attribute, *apierror.APIError)

	// CreateAttribute creates a new attribute under a property.
	CreateAttribute(ctx context.Context, params CreateAttributeParams) (*Attribute, *apierror.APIError)

	// UpdateAttribute partially updates an attribute.
	UpdateAttribute(ctx context.Context, params UpdateAttributeParams) (*Attribute, *apierror.APIError)

	// DeleteAttribute deletes an attribute.
	DeleteAttribute(ctx context.Context, params DeleteAttributeParams) *apierror.APIError
}

type CarrierSvc interface {
	// ListCarriers returns a paginated list of carriers visible to the caller's account.
	ListCarriers(ctx context.Context, params ListCarriersParams) (*ListCarriersResult, *apierror.APIError)

	// GetCarrier returns a single carrier by ID.
	GetCarrier(ctx context.Context, params GetCarrierParams) (*Carrier, *apierror.APIError)

	// BatchGetCarriersByIDs returns carriers by ID for the api-gateway include resolver. Authorization matches GetCarrier (caller's own account + system carriers). When serviceLevelsLimit > 0, each returned carrier carries a preview of up to N service_level IDs plus a has_more flag.
	BatchGetCarriersByIDs(ctx context.Context, ids []string, serviceLevelsLimit int32) ([]*Carrier, *apierror.APIError)

	// BatchGetServiceLevelsByIDs returns service levels by ID for the api-gateway include resolver. Authorization follows the parent carrier's account scope.
	BatchGetServiceLevelsByIDs(ctx context.Context, ids []string) ([]*ServiceLevel, *apierror.APIError)

	// CreateCarrier creates a new carrier, optionally registering with Shippo.
	CreateCarrier(ctx context.Context, params CreateCarrierParams) (*Carrier, *apierror.APIError)

	// UpdateCarrier partially updates a carrier.
	UpdateCarrier(ctx context.Context, params UpdateCarrierParams) (*Carrier, *apierror.APIError)

	// DeleteCarrier soft-deletes a carrier and cascades to options.
	DeleteCarrier(ctx context.Context, carrierID string) *apierror.APIError

	// InitiateOAuth starts the OAuth flow for a Shippo-managed carrier.
	InitiateOAuth(ctx context.Context, carrierID, redirectURI string, state *string) (string, *apierror.APIError)

	// GetOAuthStatus returns the OAuth connection status for a carrier.
	GetOAuthStatus(ctx context.Context, carrierID string) (string, *apierror.APIError)

	// SyncOptions syncs service levels from Shippo service levels.
	SyncOptions(ctx context.Context, carrierID string) (*Carrier, *apierror.APIError)
}

type ServiceLevelSvc interface {
	// ListServiceLevels returns a paginated list of service levels for a carrier.
	ListServiceLevels(ctx context.Context, params ListServiceLevelsParams) (*ListServiceLevelsResult, *apierror.APIError)

	// GetServiceLevel returns a single service level by ID.
	GetServiceLevel(ctx context.Context, carrierID, serviceLevelID string) (*ServiceLevel, *apierror.APIError)

	// CreateServiceLevel creates a new service level.
	CreateServiceLevel(ctx context.Context, params CreateServiceLevelParams) (*ServiceLevel, *apierror.APIError)

	// UpdateServiceLevel partially updates a service level.
	UpdateServiceLevel(ctx context.Context, params UpdateServiceLevelParams) (*ServiceLevel, *apierror.APIError)

	// DeleteServiceLevel deletes a service level.
	DeleteServiceLevel(ctx context.Context, carrierID, serviceLevelID string) *apierror.APIError
}

type ProductSvc interface {
	SearchProducts(ctx context.Context, accountID, query string) ([]ProductInfo, *apierror.APIError)
	ListProducts(ctx context.Context, accountID string) ([]ProductInfo, *apierror.APIError)
	GetCustomerByEmail(ctx context.Context, ownerAccountID, email string) (*CustomerByEmail, *apierror.APIError)
	FindContactsByEmail(ctx context.Context, email string) ([]ContactMatch, *apierror.APIError)

	// ListProductsFull returns a paginated list of products for the caller's account.
	ListProductsFull(ctx context.Context, params ListProductsFullParams) (*ListProductsFullResult, *apierror.APIError)

	// ExportProducts returns all matching products (no pagination) for export.
	ExportProducts(ctx context.Context, params ExportProductsParams) ([]*ProductFull, *apierror.APIError)

	// GetProduct returns a single product by item ID.
	GetProduct(ctx context.Context, params GetProductFullParams) (*ProductFull, *apierror.APIError)

	// CreateProduct creates a new product with its associated item and rates.
	CreateProduct(ctx context.Context, params CreateProductParams) (*ProductFull, *apierror.APIError)

	// UpdateProduct partially updates an existing product.
	UpdateProduct(ctx context.Context, params UpdateProductParams) (*ProductFull, *apierror.APIError)

	// DeleteProduct soft-deletes a product by its item ID.
	DeleteProduct(ctx context.Context, params DeleteProductParams) (*ProductFull, *apierror.APIError)

	// ChangeProductProductLine changes the product line assigned to a product.
	ChangeProductProductLine(ctx context.Context, params ChangeProductProductLineParams) (*ProductFull, *apierror.APIError)

	// ValidateProducts validates a map of SKUs and returns matching products.
	ValidateProducts(ctx context.Context, params ValidateProductsParams) (*ValidateProductsResult, *apierror.APIError)

	// BatchGetProductsByIDs returns products by their IDs for include resolution.
	BatchGetProductsByIDs(ctx context.Context, ids []string) ([]*ProductFull, *apierror.APIError)
}

type ItemSvc interface {
	// GetItemLotDefault resolves the lot an item is made in — how many, counted in what.
	GetItemLotDefault(ctx context.Context, itemID string) (*ItemLotDefault, *apierror.APIError)

	// ListItems returns a paginated list of items for the caller's account. Supports filtering by type, category, attribute, supplier, date range, and full-text search.
	ListItems(ctx context.Context, params ListItemsParams) (*ListItemsResult, *apierror.APIError)

	// GetItem returns a single item by ID within the caller's account.
	GetItem(ctx context.Context, itemID string, includes []string) (*Item, *apierror.APIError)

	// GetItemInventory returns inventory quantities (on-hand, reserved, ATP, short) for an item.
	GetItemInventory(ctx context.Context, itemID string) (*ItemInventory, *apierror.APIError)

	// GetItemCosts returns production cost breakdown for an item.
	GetItemCosts(ctx context.Context, itemID string) (*ItemCosts, *apierror.APIError)

	// GetItemTrends returns historical trend data for an item.
	GetItemTrends(ctx context.Context, itemID string, trendType string) (*ItemTrends, *apierror.APIError)

	// ExportItems returns all items with on-hand inventory for the caller's account.
	ExportItems(ctx context.Context) (*ExportItemsResult, *apierror.APIError)

	// UpdateItem partially updates an item (sku, description, notes).
	UpdateItem(ctx context.Context, params UpdateItemParams) (*Item, *apierror.APIError)

	// AddItemAttribute adds an attribute to an item.
	AddItemAttribute(ctx context.Context, itemID, attributeID string, includes []string) (*Item, *apierror.APIError)

	// RemoveItemAttribute removes an attribute from an item.
	RemoveItemAttribute(ctx context.Context, itemID, attributeID string, includes []string) (*Item, *apierror.APIError)

	// ChangeItemCategory changes the category of an item and updates rate units.
	ChangeItemCategory(ctx context.Context, itemID, categoryID string, includes []string) (*Item, *apierror.APIError)

	// UpdateItemInventory adjusts or reconciles inventory for an item.
	UpdateItemInventory(ctx context.Context, params UpdateItemInventoryParams) *apierror.APIError

	// BulkCreateItems creates multiple items in a single operation.
	BulkCreateItems(ctx context.Context, params BulkCreateItemsParams) ([]BulkCreateItemResult, *apierror.APIError)

	// BulkReconcileItems reconciles inventory for multiple items by SKU.
	BulkReconcileItems(ctx context.Context, params BulkReconcileItemsParams) (*BulkReconcileItemsResult, *apierror.APIError)

	// BatchGetItemsByIDs returns items by ID for the api-gateway include resolver. Always populates rates and attributes.
	BatchGetItemsByIDs(ctx context.Context, ids []string) ([]*Item, *apierror.APIError)

	// ListInventories returns all items with their on-hand inventory quantities.
	ListInventories(ctx context.Context, params ListInventoriesParams) (*ListInventoriesResult, *apierror.APIError)
}

type AccountSvc interface {
	// GetAccountContext returns contextual information for an account (including whether it is a sandbox).
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)

	// GetUserAccountAccess returns the user's access to an account, including role and permissions.
	//
	// If the user has no relationship to the account, returns (nil, false, nil).
	GetUserAccountAccess(ctx context.Context, userID, accountID string) (*AccountUserAccess, bool, *apierror.APIError)

	// GetRolePermissions returns the permission map for the given role ID.
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)

	// GetRoleInfo returns a role's name and type code.
	GetRoleInfo(ctx context.Context, roleID string) (*RoleInfo, *apierror.APIError)

	// GetAccountRelationByUserID returns the relationship between the target account and the account implied by the user. actorAccountID is required for owner-side matches (the relation's owner_account_id must equal it); pass "" to skip the owner-side fallback entirely.
	GetAccountRelationByUserID(ctx context.Context, targetAccountID, actorAccountID, userID string) (*AccountRelation, *apierror.APIError)

	// GetAccountRelationByAPIKeyID returns the relationship between the owner account and the account implied by the API key.
	GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)

	// MarkAccountUserUsed records that the account user was recently used.
	MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError

	// ListUserAccountAffiliations returns the accounts the user is affiliated with.
	//
	// Also returns, if available, the user's last used account ID.
	ListUserAccountAffiliations(ctx context.Context, userID string) ([]AccountAffiliation, *string, *apierror.APIError)

	// GetAdminRole returns the role ID used for administrative access.
	GetAdminRole(ctx context.Context) (string, *apierror.APIError)

	// UpdateAccountSubscription updates subscription fields on an account, resolving the account_plan_id from the plan_code.
	UpdateAccountSubscription(ctx context.Context, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError

	// ClearAccountStripeCustomer removes all Stripe-related fields from an account.
	ClearAccountStripeCustomer(ctx context.Context, accountID string) *apierror.APIError

	// GetAccountByStripeCustomerID resolves an account from a Stripe customer ID.
	GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (accountID string, planCode string, err *apierror.APIError)

	// CompleteRegistration creates the production account, sandbox, owner roles, account-user records, business address, and portal for a newly registered user. Returns the new account ID and sandbox account ID.
	CompleteRegistration(ctx context.Context, input CompleteRegistrationInput) (*CompleteRegistrationOutput, *apierror.APIError)

	// UpdateAgentSpendingCap sets or removes the monthly agent LLM spending cap for the caller's account. Pass nil to remove the cap.
	UpdateAgentSpendingCap(ctx context.Context, capCents *int64) (*int64, *apierror.APIError)

	// GetAccount returns the full account with optional branding and portal sub-resources.
	GetAccount(ctx context.Context, accountID string) (*Account, *apierror.APIError)

	// BatchGetAccountsByIDs returns accounts matching the given IDs that the caller is authorized to read. Used by the api-gateway include resolver for owner.account expansion across many parent resources.
	BatchGetAccountsByIDs(ctx context.Context, ids []string) ([]*Account, *apierror.APIError)

	// GetAccountBySlug returns a minimal public account by portal slug (unauthenticated).
	GetAccountBySlug(ctx context.Context, slug string) (*PublicAccountBySlug, *apierror.APIError)
	GetPortalProfileBySlug(ctx context.Context, slug string) (*PortalProfile, *apierror.APIError)

	// UpdateAccount partially updates an account's name, branding, and/or portal slug.
	UpdateAccount(ctx context.Context, params UpdateAccountParams) (*Account, *apierror.APIError)

	// UploadAccountPhoto uploads an account logo to S3 and updates the branding record.
	UploadAccountPhoto(ctx context.Context, accountID string, file []byte, contentType string) *apierror.APIError

	// GetAccountLogoURL returns a presigned S3 URL for the account's logo, or nil if none.
	GetAccountLogoURL(ctx context.Context, accountID string) (*string, *apierror.APIError)

	// UploadAccountFavicon uploads a customer-portal favicon to S3 and updates the branding record.
	UploadAccountFavicon(ctx context.Context, accountID string, file []byte, contentType string) *apierror.APIError

	// GetAccountFaviconURL returns a presigned S3 URL for the account's customer-portal favicon, or nil if none.
	GetAccountFaviconURL(ctx context.Context, accountID string) (*string, *apierror.APIError)
}

// CompleteRegistrationInput carries the data needed to finalize a registration.
type CompleteRegistrationInput struct {
	UserID               string
	PlanCode             string
	StripeCustomerID     string
	StripeSubscriptionID string
	AccountData          RegistrationAccountData
	BusinessAddress      *RegistrationAddress
}

// RegistrationAccountData holds the business profile information collected during onboarding.
type RegistrationAccountData struct {
	AccountName string
}

// RegistrationAddress is a structured postal address collected during registration.
type RegistrationAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// CompleteRegistrationOutput holds the IDs of the newly created accounts.
type CompleteRegistrationOutput struct {
	AccountID string
	SandboxID string
}

// BatchSvc handles all batch-related business logic.
type BatchSvc interface {
	// GetBatchFlow returns the flow graph for a batch.
	GetBatchFlow(ctx context.Context, batchID string) ([]BatchFlowNode, *apierror.APIError)

	// ListBatchesByScanningStation returns a paginated list of batches for a scanning station.
	ListBatchesByScanningStation(ctx context.Context, params ListBatchesByScanningStationParams) (*ListBatchesByScanningStationResult, *apierror.APIError)

	// GetPossibleNextSteps returns the possible next production steps for a batch at a scanning station.
	GetPossibleNextSteps(ctx context.Context, scanningStationID, batchID string) ([]ScanningProductionStepInfo, *apierror.APIError)

	// AnalyzeOpenBatches returns aggregated open batch summaries for analytics.
	AnalyzeOpenBatches(ctx context.Context, itemIDs, productLineIDs []string) ([]OpenBatchSummary, *apierror.APIError)

	// InitializeBatch initializes a batch at a scanning station.
	InitializeBatch(ctx context.Context, batchID, scanningStationID string) (*BaseBatch, *apierror.APIError)

	// MoveBatches moves one or more batches to a new production step.
	MoveBatches(ctx context.Context, params MoveBatchesParams) (*BaseBatch, *apierror.APIError)

	// MergeBatches merges multiple batches into a single batch.
	MergeBatches(ctx context.Context, params MergeBatchesParams) (*BaseBatch, *apierror.APIError)

	// SplitBatch splits a batch into firsts, seconds, and waste.
	SplitBatch(ctx context.Context, params SplitBatchParams) (*BaseBatch, *apierror.APIError)

	// GetRemainingQuantityToSplit returns the remaining quantity available to split from a batch flow.
	GetRemainingQuantityToSplit(ctx context.Context, batchIDs []string, productionStepID string) (*BatchQuantity, *apierror.APIError)

	// GetScanningStationConsumption returns consumption demand for a scanning station.
	GetScanningStationConsumption(ctx context.Context, params GetConsumptionParams) ([]ScanningConsumption, *apierror.APIError)

	// CloseBatch closes a batch.
	CloseBatch(ctx context.Context, batchID string) (*BaseBatch, *apierror.APIError)

	// DeleteBatch deletes a single batch.
	DeleteBatch(ctx context.Context, batchID string) (*BaseBatch, *apierror.APIError)

	// DeleteManyBatches deletes multiple batches.
	DeleteManyBatches(ctx context.Context, batchIDs []string) *apierror.APIError
}

type ChildAccountSvc interface {
	// ListChildAccounts returns a paginated list of child accounts for the target account.
	ListChildAccounts(ctx context.Context, cursor *string, limit int32, query *string) (*ListChildAccountsResult, *apierror.APIError)

	// AddChildAccount adds a child account relationship to the target account.
	AddChildAccount(ctx context.Context, childAccountID string) (*ChildAccount, *apierror.APIError)

	// RemoveChildAccount removes a child account relationship from the target account.
	RemoveChildAccount(ctx context.Context, childAccountID string) *apierror.APIError

	// BatchGetChildAccountsByIDs returns child account relations matching the input relation IDs. Used by the api-gateway resourcekit include resolver.
	BatchGetChildAccountsByIDs(ctx context.Context, relationIDs []string) ([]*ChildAccount, *apierror.APIError)
}

type ItemCategorySvc interface {
	// ListItemCategories returns a paginated list of item categories visible to the caller's account.
	ListItemCategories(ctx context.Context, params ListItemCategoriesParams) (*ListItemCategoriesResult, *apierror.APIError)

	// GetItemCategory returns a single item category by ID.
	GetItemCategory(ctx context.Context, params GetItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)

	// CreateItemCategory creates a new item category.
	CreateItemCategory(ctx context.Context, params CreateItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)

	// UpdateItemCategory partially updates an item category. Default categories cannot be updated.
	UpdateItemCategory(ctx context.Context, params UpdateItemCategoryParams) (*ItemCategoryFull, *apierror.APIError)

	// DeleteItemCategory deletes an item category. Default categories cannot be deleted.
	DeleteItemCategory(ctx context.Context, itemCategoryID string) *apierror.APIError

	// AddItemCategoryProperty adds a property to an item category.
	AddItemCategoryProperty(ctx context.Context, params AddItemCategoryPropertyParams) *apierror.APIError

	// RemoveItemCategoryProperty removes a property from an item category.
	RemoveItemCategoryProperty(ctx context.Context, params RemoveItemCategoryPropertyParams) *apierror.APIError

	// ChangeItemCategoryUnitGroup changes the unit group of an item category.
	ChangeItemCategoryUnitGroup(ctx context.Context, params ChangeItemCategoryUnitGroupParams) *apierror.APIError

	// BatchGetItemCategoriesByIDs returns item categories by ID for the api-gateway include resolver. Always populates properties and unit group with full tree.
	BatchGetItemCategoriesByIDs(ctx context.Context, ids []string) ([]*ItemCategoryFull, *apierror.APIError)
}

type ProductLineSvc interface {
	// ListProductLines returns a paginated list of product lines visible to the caller's account.
	ListProductLines(ctx context.Context, params ListProductLinesParams) (*ListProductLinesResult, *apierror.APIError)

	// GetProductLine returns a single product line by ID.
	GetProductLine(ctx context.Context, params GetProductLineParams) (*ProductLineFull, *apierror.APIError)

	// CreateProductLine creates a new product line.
	CreateProductLine(ctx context.Context, params CreateProductLineParams) (*ProductLineFull, *apierror.APIError)

	// UpdateProductLine partially updates a product line. Default product lines cannot be updated.
	UpdateProductLine(ctx context.Context, params UpdateProductLineParams) (*ProductLineFull, *apierror.APIError)

	// DeleteProductLine deletes a product line. Default product lines cannot be deleted.
	DeleteProductLine(ctx context.Context, productLineID string) *apierror.APIError

	// BatchGetProductLinesByIDs returns product lines by ID for the api-gateway include resolver.
	BatchGetProductLinesByIDs(ctx context.Context, ids []string) ([]*ProductLineFull, *apierror.APIError)
}

type ConsumptionSvc interface {
	// GetConsumption returns a single consumption by ID within a production step.
	GetConsumption(ctx context.Context, productionStepID, consumptionID string) (*Consumption, *apierror.APIError)

	// CreateConsumption creates a new consumption within a production step.
	CreateConsumption(ctx context.Context, params CreateConsumptionParams) (*Consumption, *apierror.APIError)

	// UpdateConsumption partially updates a consumption.
	UpdateConsumption(ctx context.Context, params UpdateConsumptionParams) (*Consumption, *apierror.APIError)

	// DeleteConsumption deletes a consumption from a production step and returns it.
	DeleteConsumption(ctx context.Context, params DeleteConsumptionParams) (*Consumption, *apierror.APIError)
}

type ProductionFlowSvc interface {
	// GetProductionFlow returns the production flow graph for a given item.
	GetProductionFlow(ctx context.Context, itemID string) ([]*ProductionFlowStep, *apierror.APIError)

	// ConnectSteps links two production steps in the flow DAG.
	ConnectSteps(ctx context.Context, sourceStepID, targetStepID string) *apierror.APIError
}

type CustomerSvc interface {
	// ListCustomers returns a paginated list of customers for the caller's account.
	ListCustomers(ctx context.Context, params ListCustomersParams) (*ListCustomersResult, *apierror.APIError)

	// GetCustomer returns a single customer by account ID. Supports customer actor access.
	GetCustomer(ctx context.Context, customerAccountID string, includes []string) (*Customer, *apierror.APIError)

	// CreateCustomer creates a new customer account.
	CreateCustomer(ctx context.Context, params CreateCustomerParams) (*Customer, *apierror.APIError)

	// DeleteCustomer deletes a customer and its associated relations.
	DeleteCustomer(ctx context.Context, params DeleteCustomerParams) *apierror.APIError

	// BulkDeleteCustomers deletes multiple customers at once.
	BulkDeleteCustomers(ctx context.Context, params BulkDeleteCustomersParams) *apierror.APIError

	// GetFrequentlyOrderedProducts returns the most frequently ordered products for a customer.
	GetFrequentlyOrderedProducts(ctx context.Context, customerAccountID string) ([]*FrequentlyOrderedProduct, *apierror.APIError)

	// ListCustomerNotificationRecipients returns the default order-notification recipients configured for a customer relationship.
	ListCustomerNotificationRecipients(ctx context.Context, customerAccountID string) ([]NotificationRecipient, *apierror.APIError)

	// UpdateCustomerNotificationRecipients replaces the default order-notification recipients configured for a customer relationship.
	UpdateCustomerNotificationRecipients(ctx context.Context, params UpdateCustomerNotificationRecipientsParams) ([]NotificationRecipient, *apierror.APIError)

	// UpdateCustomer partially updates a customer.
	UpdateCustomer(ctx context.Context, params UpdateCustomerParams) (*Customer, *apierror.APIError)

	// MergeCustomers merges source customers into a target customer.
	MergeCustomers(ctx context.Context, params MergeCustomersParams) (*Customer, *apierror.APIError)
}

type AnalyticsSvc interface {
	AnalyzeSales(ctx context.Context, params AnalyzeSalesParams) ([]SalesEntry, *apierror.APIError)
	AnalyzeOpenBatches(ctx context.Context, params AnalyzeOpenBatchesParams) ([]OpenBatchEntry, *apierror.APIError)
	AnalyzeProductionCosts(ctx context.Context, params AnalyzeProductionCostsParams) ([]ProductionCostEntry, *apierror.APIError)
	AnalyzeDeliveries(ctx context.Context, params AnalyzeDeliveriesParams) (*DeliveryAnalyticsResult, *apierror.APIError)
	AnalyzeManufacturing(ctx context.Context, params AnalyzeManufacturingParams) (float64, *apierror.APIError)
	AnalyzeManufacturingBatch(ctx context.Context, params AnalyzeManufacturingBatchParams) (*ManufacturingBatchResult, *apierror.APIError)
	AnalyzeOrders(ctx context.Context, params AnalyzeOrdersParams) ([]OrderEntry, *apierror.APIError)
	AnalyzeQuarterlyOrders(ctx context.Context, params AnalyzeQuarterlyOrdersParams) ([]YearlyQuarterlyData, *apierror.APIError)
	AnalyzeMaterials(ctx context.Context, params AnalyzeMaterialsParams) ([]MaterialAnalyticsEntry, *apierror.APIError)
	AnalyzeInventoryReceipts(ctx context.Context, params AnalyzeInventoryReceiptsParams) ([]InventoryReceiptEntry, *apierror.APIError)
	GetNewCustomersAnalytics(ctx context.Context, params GetNewCustomersAnalyticsParams) ([]NewCustomerEntry, *apierror.APIError)
	// GetDemandForecast returns per-item demand, revenue and sales history with seasonal-EMA forecasts and confidence bands.
	GetDemandForecast(ctx context.Context, params GetDemandForecastParams) (*DemandForecastResult, *apierror.APIError)
	// AnalyzeOee computes Availability x Performance x Quality per department from planned time, logged downtime and batch-ticket scan intervals.
	AnalyzeOee(ctx context.Context, params AnalyzeOeeParams) ([]OeeDepartment, *apierror.APIError)
	AnalyzeOeeTrend(ctx context.Context, params AnalyzeOeeTrendParams) ([]OeeTrendPeriod, *apierror.APIError)

	// AnalyzeScheduleAttainment measures actual production against the plan that was live at the time.
	AnalyzeScheduleAttainment(ctx context.Context, params AnalyzeScheduleAttainmentParams) (*ScheduleAttainmentResult, *apierror.APIError)
	// AnalyzeWeeksOfSales returns on-hand inventory expressed as weeks of average sales per product line.
	AnalyzeWeeksOfSales(ctx context.Context, params AnalyzeWeeksOfSalesParams) (*WeeksOfSalesResult, *apierror.APIError)
}

// MachineStatusSvc answers "what is the floor doing right now".
type MachineStatusSvc interface {
	ListMachineStatus(ctx context.Context, params ListMachineStatusParams) (*MachineStatusResult, *apierror.APIError)
}

type MachineSvc interface {
	// ListMachines returns a paginated list of machines for the caller's account.
	ListMachines(ctx context.Context, params ListMachinesParams) (*ListMachinesResult, *apierror.APIError)

	// GetMachine returns a single machine by ID.
	GetMachine(ctx context.Context, machineID string) (*Machine, *apierror.APIError)

	// CreateMachine creates a new machine.
	CreateMachine(ctx context.Context, params CreateMachineParams) (*Machine, *apierror.APIError)

	// UpdateMachine partially updates a machine.
	UpdateMachine(ctx context.Context, params UpdateMachineParams) (*Machine, *apierror.APIError)

	// DeleteMachine deletes a machine.
	DeleteMachine(ctx context.Context, machineID string) *apierror.APIError

	// BatchGetMachinesByIDs returns machines by their IDs for include resolution.
	BatchGetMachinesByIDs(ctx context.Context, ids []string) ([]*Machine, *apierror.APIError)
}

type ProductionScheduleSvc interface {
	// PreviewRegenerateProductionSchedule says what a re-solve would change about a draft, without changing it.
	PreviewRegenerateProductionSchedule(ctx context.Context, params RegenerateProductionScheduleParams) (*ScheduleRegeneratePreview, *apierror.APIError)
	// RegenerateProductionSchedule re-solves a draft in place, keeping its version number.
	RegenerateProductionSchedule(ctx context.Context, params RegenerateProductionScheduleParams) (*ProductionSchedule, *apierror.APIError)
	// PreviewProductionSchedule runs the solver and returns the plan without persisting it.
	PreviewProductionSchedule(ctx context.Context, params PreviewProductionScheduleParams) (*scheduling.SolverOutput, *apierror.APIError)

	// GenerateProductionSchedule solves and persists a new draft version.
	GenerateProductionSchedule(ctx context.Context, params GenerateProductionScheduleParams) (*ProductionSchedule, *apierror.APIError)

	// GetProductionSchedule returns one version by ID.
	GetProductionSchedule(ctx context.Context, scheduleID string) (*ProductionSchedule, *apierror.APIError)

	// GetCurrentProductionSchedule returns the published version covering today, or nil when there is none.
	GetCurrentProductionSchedule(ctx context.Context) (*ProductionSchedule, *apierror.APIError)

	// ListProductionSchedules returns a paginated list of versions.
	ListProductionSchedules(ctx context.Context, params ListProductionSchedulesParams) (*ListProductionSchedulesResult, *apierror.APIError)

	// ListProductionScheduleLines returns the planned campaigns for a version.
	ListProductionScheduleLines(ctx context.Context, params ListProductionScheduleLinesParams) ([]*ProductionScheduleLine, *apierror.APIError)

	// ListProductionScheduleItemPolicies returns the per-item policy snapshot behind a version.
	ListProductionScheduleItemPolicies(ctx context.Context, scheduleID string) ([]*ProductionScheduleItemPolicy, *apierror.APIError)
	// ListProductionScheduleFinishedPolicies returns the per-finished-SKU decomposition of a version's pooled constraint buffers.
	ListProductionScheduleFinishedPolicies(ctx context.Context, scheduleID string) ([]*ProductionScheduleFinishedPolicy, *apierror.APIError)

	// GetProductionScheduleSettings returns the merchant's planning assumptions, falling back to code defaults when nothing has been saved.
	GetProductionScheduleSettings(ctx context.Context) (*ProductionScheduleSettings, *apierror.APIError)

	// UpdateProductionScheduleSettings replaces the merchant's planning assumptions.
	UpdateProductionScheduleSettings(ctx context.Context, params UpdateProductionScheduleSettingsParams) (*ProductionScheduleSettings, *apierror.APIError)

	// ListResourceSettings returns per-machine, per-department and per-step overrides.
	ListResourceSettings(ctx context.Context) ([]*ProductionScheduleResourceSetting, *apierror.APIError)

	// UpsertResourceSetting writes one per-resource override.
	UpsertResourceSetting(ctx context.Context, params UpsertResourceSettingParams) (*ProductionScheduleResourceSetting, *apierror.APIError)

	// DeleteResourceSetting removes one per-resource override.
	DeleteResourceSetting(ctx context.Context, settingID string) *apierror.APIError

	// EnqueueScheduledGeneration reserves a version and queues its solve, in one transaction. Used by the generation cadence.
	EnqueueScheduledGeneration(ctx context.Context, params EnqueueGenerationParams) *apierror.APIError

	// RunScheduledGeneration solves into a row already created in `generating`. Used by the generate-command consumer.
	RunScheduledGeneration(ctx context.Context, params RunScheduledGenerationParams) *apierror.APIError

	// ListProductionScheduleDerivedLines returns derived downstream department work.
	ListProductionScheduleDerivedLines(ctx context.Context, params ListDerivedLinesParams) ([]*ProductionScheduleDerivedLine, *apierror.APIError)

	// ListScheduleDeviationTypes returns the global taxonomy of what a hand change can be.
	ListScheduleDeviationTypes(ctx context.Context) ([]*ScheduleDeviationType, *apierror.APIError)

	// ListProductionScheduleDeviations returns the append-only log of hand changes.
	ListProductionScheduleDeviations(ctx context.Context, params ListProductionScheduleDeviationsParams) (*ListProductionScheduleDeviationsResult, *apierror.APIError)

	// CreateProductionScheduleLine adds a campaign by hand and logs a deviation.
	CreateProductionScheduleLine(ctx context.Context, params CreateProductionScheduleLineParams) (*ProductionScheduleLine, *apierror.APIError)

	// UpdateProductionScheduleLine edits a campaign and logs a deviation.
	UpdateProductionScheduleLine(ctx context.Context, params UpdateProductionScheduleLineParams) (*ProductionScheduleLine, *apierror.APIError)

	// DeleteProductionScheduleLine removes a campaign and logs a deviation.
	DeleteProductionScheduleLine(ctx context.Context, params DeleteProductionScheduleLineParams) *apierror.APIError

	// PublishProductionSchedule freezes the first weeks, snapshots the frozen counts, and supersedes whatever it replaces.
	PublishProductionSchedule(ctx context.Context, scheduleID string) (*ProductionSchedule, *apierror.APIError)

	// ReleaseProductionScheduleWeek turns one planned week into a production run, with one batch per planned lot.
	ReleaseProductionScheduleWeek(ctx context.Context, params ReleaseScheduleWeekParams) (*ReleaseScheduleWeekResult, *apierror.APIError)

	// PreviewReleaseProductionScheduleWeek says what releasing a week would create, without creating it.
	PreviewReleaseProductionScheduleWeek(ctx context.Context, scheduleID string, weekIndex int32) (*ReleaseScheduleWeekPreview, *apierror.APIError)

	// ArchiveProductionSchedule retires a version without deleting its history.
	ArchiveProductionSchedule(ctx context.Context, scheduleID string) (*ProductionSchedule, *apierror.APIError)

	// DeleteProductionSchedule removes a draft. Published versions must be archived.
	DeleteProductionSchedule(ctx context.Context, scheduleID string) *apierror.APIError
}

type MachineDowntimeSvc interface {
	// ListDowntimeReasons returns the global downtime reason taxonomy, ordered for display.
	ListDowntimeReasons(ctx context.Context) ([]*MachineDowntimeReason, *apierror.APIError)

	// ListDowntimeEvents returns a paginated list of downtime events for the caller's account.
	ListDowntimeEvents(ctx context.Context, params ListMachineDowntimeEventsParams) (*ListMachineDowntimeEventsResult, *apierror.APIError)

	// GetDowntimeEvent returns a single downtime event by ID.
	GetDowntimeEvent(ctx context.Context, eventID string) (*MachineDowntimeEvent, *apierror.APIError)

	// CreateDowntimeEvent logs a stoppage. Department and production step are resolved from the machine; an open event (no end) records that the machine is still down.
	CreateDowntimeEvent(ctx context.Context, params CreateMachineDowntimeEventParams) (*MachineDowntimeEvent, *apierror.APIError)

	// UpdateDowntimeEvent closes or reclassifies an event.
	UpdateDowntimeEvent(ctx context.Context, params UpdateMachineDowntimeEventParams) (*MachineDowntimeEvent, *apierror.APIError)

	// DeleteDowntimeEvent removes a mis-logged event.
	DeleteDowntimeEvent(ctx context.Context, eventID string) *apierror.APIError

	// BatchGetDowntimeEventsByIDs returns downtime events by their IDs for include resolution.
	BatchGetDowntimeEventsByIDs(ctx context.Context, ids []string) ([]*MachineDowntimeEvent, *apierror.APIError)
}

type DemandOverrideSvc interface {
	// ListDemandOverrideTypes returns the global override type taxonomy.
	ListDemandOverrideTypes(ctx context.Context) ([]*DemandOverrideType, *apierror.APIError)

	// ListDemandOverrides returns a paginated list of demand overrides for the caller's account.
	ListDemandOverrides(ctx context.Context, params ListDemandOverridesParams) (*ListDemandOverridesResult, *apierror.APIError)

	// GetDemandOverride returns a single demand override by ID.
	GetDemandOverride(ctx context.Context, overrideID string) (*DemandOverride, *apierror.APIError)

	// CreateDemandOverride records demand the forecast cannot see. The scope reference is validated so an override can never silently match nothing.
	CreateDemandOverride(ctx context.Context, params CreateDemandOverrideParams) (*DemandOverride, *apierror.APIError)

	// UpdateDemandOverride partially updates an override. Type and value are validated as a pair against the resulting row.
	UpdateDemandOverride(ctx context.Context, params UpdateDemandOverrideParams) (*DemandOverride, *apierror.APIError)

	// DeleteDemandOverride removes an override.
	DeleteDemandOverride(ctx context.Context, overrideID string) *apierror.APIError

	// BatchGetDemandOverridesByIDs returns demand overrides by their IDs for include resolution.
	BatchGetDemandOverridesByIDs(ctx context.Context, ids []string) ([]*DemandOverride, *apierror.APIError)
}

type DepartmentSvc interface {
	// ListDepartments returns a paginated list of departments for the caller's account.
	ListDepartments(ctx context.Context, params ListDepartmentsParams) (*ListDepartmentsResult, *apierror.APIError)

	// GetDepartment returns a single department by ID.
	GetDepartment(ctx context.Context, departmentID string) (*Department, *apierror.APIError)

	// CreateDepartment creates a new department.
	CreateDepartment(ctx context.Context, params CreateDepartmentParams) (*Department, *apierror.APIError)

	// UpdateDepartment partially updates a department.
	UpdateDepartment(ctx context.Context, params UpdateDepartmentParams) (*Department, *apierror.APIError)

	// DeleteDepartment deletes a department.
	DeleteDepartment(ctx context.Context, departmentID string) *apierror.APIError

	// BatchGetDepartmentsByIDs returns departments matching the given IDs.
	BatchGetDepartmentsByIDs(ctx context.Context, ids []string) ([]*Department, *apierror.APIError)
}

type DeliverySvc interface {
	// ListDeliveries returns a paginated list of deliveries for the caller's account.
	ListDeliveries(ctx context.Context, params ListDeliveriesParams) (*ListDeliveriesResult, *apierror.APIError)

	// GetDelivery returns a single delivery by ID within the caller's account.
	GetDelivery(ctx context.Context, params GetDeliveryParams) (*Delivery, *apierror.APIError)
}

type EmailLogSvc interface {
	// ListEmailLogs returns a paginated list of email logs for the caller's account.
	ListEmailLogs(ctx context.Context, params ListEmailLogsParams) (*ListEmailLogsResult, *apierror.APIError)

	// GetEmailLog returns a single email log by ID within the caller's account.
	GetEmailLog(ctx context.Context, params GetEmailLogParams) (*EmailLog, *apierror.APIError)
}

type InventoryChangeLogSvc interface {
	// ListInventoryChangeLogs returns a paginated list of inventory change logs for the caller's account.
	ListInventoryChangeLogs(ctx context.Context, params ListInventoryChangeLogsParams) (*ListInventoryChangeLogsResult, *apierror.APIError)

	// GetInventoryChangeLog returns a single inventory change log by ID.
	GetInventoryChangeLog(ctx context.Context, id string) (*InventoryChangeLog, *apierror.APIError)

	// ExportInventoryChangeLogs returns all inventory change logs matching the provided filters for the caller's account.
	ExportInventoryChangeLogs(ctx context.Context, params ExportInventoryChangeLogsParams) ([]*InventoryChangeLog, *apierror.APIError)
}

type InvoiceSvc interface {
	// ListInvoices returns a paginated list of invoices for the caller's account.
	ListInvoices(ctx context.Context, params ListInvoicesParams) (*ListInvoicesResult, *apierror.APIError)

	// GetInvoice returns a single invoice by ID within the caller's account. Lines and allocations are fetched conditionally based on the includes parameter.
	GetInvoice(ctx context.Context, params GetInvoiceParams) (*Invoice, *apierror.APIError)

	// UpdateInvoice partially updates an invoice with idempotency support.
	UpdateInvoice(ctx context.Context, params UpdateInvoiceParams) (*InvoiceSummary, *apierror.APIError)

	// ListCustomerInvoices returns a paginated list of invoices for a customer account.
	ListCustomerInvoices(ctx context.Context, params ListCustomerInvoicesParams) (*ListCustomerInvoicesResult, *apierror.APIError)
}

type SettlementSvc interface {
	// ListSettlements returns a paginated list of settlements for the caller's account.
	ListSettlements(ctx context.Context, params ListSettlementsParams) (*ListSettlementsResult, *apierror.APIError)

	// GetSettlement returns a single settlement by ID within the caller's account.
	GetSettlement(ctx context.Context, params GetSettlementParams) (*Settlement, *apierror.APIError)

	// CreateSettlement creates a new settlement with transaction allocations and idempotency support.
	CreateSettlement(ctx context.Context, params CreateSettlementParams) (*Settlement, *apierror.APIError)

	// UpdateSettlement partially updates a settlement with idempotency support.
	UpdateSettlement(ctx context.Context, params UpdateSettlementParams) (*Settlement, *apierror.APIError)

	// DeleteSettlement deletes a settlement and cascades cleanup to allocations, orphaned transactions, and payment statuses.
	DeleteSettlement(ctx context.Context, params DeleteSettlementParams) (*Settlement, *apierror.APIError)
}

type TransactionSvc interface {
	ListTransactions(ctx context.Context, params ListTransactionsParams) (*ListTransactionsResult, *apierror.APIError)
	GetTransaction(ctx context.Context, params GetTransactionParams) (*Transaction, *apierror.APIError)
	CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, *apierror.APIError)
	UpdateTransaction(ctx context.Context, params UpdateTransactionParams) (*Transaction, *apierror.APIError)
	DeleteTransaction(ctx context.Context, params DeleteTransactionParams) (*Transaction, *apierror.APIError)
	ListAccountTransactions(ctx context.Context, params ListAccountTransactionsParams) (*ListAccountTransactionsResult, *apierror.APIError)
}

type TransactionAllocationSvc interface {
	// ListAllocationEntries returns a paginated list of allocation entries for the caller's account.
	ListAllocationEntries(ctx context.Context, params ListAllocationEntriesParams) (*ListAllocationEntriesResult, *apierror.APIError)

	// UpdateTransactionAllocation updates a transaction allocation's amount and/or created_at with idempotency support.
	UpdateTransactionAllocation(ctx context.Context, params UpdateTransactionAllocationParams) (*TransactionAllocation, *apierror.APIError)

	// DeleteTransactionAllocation deletes a transaction allocation.
	DeleteTransactionAllocation(ctx context.Context, params DeleteTransactionAllocationParams) *apierror.APIError

	// ListOpenCredits returns a list of open (not fully allocated) credit transactions.
	ListOpenCredits(ctx context.Context, params ListOpenCreditsParams) (*ListOpenCreditsResult, *apierror.APIError)
}

type ReceivableSvc interface {
	// ListReceivables returns a paginated list of receivable entries for the caller's account.
	ListReceivables(ctx context.Context, params ListReceivablesParams) (*ListReceivablesResult, *apierror.APIError)

	// ListReceivablesByCustomer returns a paginated list of receivable entries for a specific customer.
	ListReceivablesByCustomer(ctx context.Context, params ListReceivablesByCustomerParams) (*ListReceivablesByCustomerResult, *apierror.APIError)

	// ExportReceivablesByCustomer returns all receivable entries for a specific customer (no pagination).
	ExportReceivablesByCustomer(ctx context.Context, params ListReceivablesByCustomerParams) ([]ReceivableEntry, *apierror.APIError)

	// EmailReceivablesForCustomer sends a receivables statement to the specified email addresses.
	EmailReceivablesForCustomer(ctx context.Context, params EmailReceivablesParams) *apierror.APIError
}

type SalesOrderStatusSvc interface {
	// ListSalesOrderStatuses returns a paginated list of sales order statuses. These are global lookup values.
	ListSalesOrderStatuses(ctx context.Context, params ListSalesOrderStatusesParams) (*ListSalesOrderStatusesResult, *apierror.APIError)

	// BatchGetSalesOrderStatusesByIDs returns statuses by ID for the api-gateway include resolver.
	BatchGetSalesOrderStatusesByIDs(ctx context.Context, ids []string) ([]*SalesOrderStatus, *apierror.APIError)
}

type OrderDiscountSvc interface {
	// ListOrderDiscounts returns a paginated list of order discounts for the caller's account.
	ListOrderDiscounts(ctx context.Context, params ListOrderDiscountsParams) (*ListOrderDiscountsResult, *apierror.APIError)

	// GetOrderDiscount returns a single order discount by ID.
	GetOrderDiscount(ctx context.Context, orderDiscountID string) (*OrderDiscount, *apierror.APIError)

	// CreateOrderDiscount creates a new order discount.
	CreateOrderDiscount(ctx context.Context, params CreateOrderDiscountParams) (*OrderDiscount, *apierror.APIError)

	// UpdateOrderDiscount partially updates an order discount.
	UpdateOrderDiscount(ctx context.Context, params UpdateOrderDiscountParams) (*OrderDiscount, *apierror.APIError)

	// DeleteOrderDiscount deletes an order discount and returns the deleted resource.
	DeleteOrderDiscount(ctx context.Context, orderDiscountID string) (*OrderDiscount, *apierror.APIError)

	// FindOrderDiscountByCode finds an order discount by its code. Supports both internal and customer actors.
	FindOrderDiscountByCode(ctx context.Context, params FindOrderDiscountByCodeParams) (*OrderDiscount, *apierror.APIError)

	// BatchGetOrderDiscountsByIDs returns order discounts matching the input IDs that the caller's account is authorized to read. Used by the api-gateway resourcekit include resolver.
	BatchGetOrderDiscountsByIDs(ctx context.Context, ids []string) ([]*OrderDiscount, *apierror.APIError)
}

type VolumeDiscountSvc interface {
	// ListVolumeDiscounts returns a paginated list of volume discounts. Supports both internal and customer actors.
	ListVolumeDiscounts(ctx context.Context, params ListVolumeDiscountsParams) (*ListVolumeDiscountsResult, *apierror.APIError)

	// GetVolumeDiscount returns a single volume discount by ID. Supports both internal and customer actors.
	GetVolumeDiscount(ctx context.Context, params GetVolumeDiscountParams) (*VolumeDiscount, *apierror.APIError)

	// CreateVolumeDiscount creates a new volume discount with tiers and relations.
	CreateVolumeDiscount(ctx context.Context, params CreateVolumeDiscountParams) (*VolumeDiscount, *apierror.APIError)

	// UpdateVolumeDiscount partially updates a volume discount.
	UpdateVolumeDiscount(ctx context.Context, params UpdateVolumeDiscountParams) (*VolumeDiscount, *apierror.APIError)

	// DeleteVolumeDiscount deletes a volume discount and its tiers and relations.
	DeleteVolumeDiscount(ctx context.Context, volumeDiscountID string) *apierror.APIError
}

type SalesOrderSvc interface {
	// ListSalesOrders returns a paginated list of sales orders for the caller's account. Supports customer actor access via BuyerAccountID filter.
	ListSalesOrders(ctx context.Context, params ListSalesOrdersParams) (*ListSalesOrdersResult, *apierror.APIError)

	// GetSalesOrder returns a single sales order by ID. Lines are fetched conditionally based on the includes parameter.
	GetSalesOrder(ctx context.Context, params GetSalesOrderParams) (*SalesOrder, *apierror.APIError)

	// CreateSalesOrder creates a new sales order with lines, addresses, and optional discount.
	CreateSalesOrder(ctx context.Context, params CreateSalesOrderParams) (*SalesOrder, *apierror.APIError)

	// UpdateSalesOrder partially updates a sales order.
	UpdateSalesOrder(ctx context.Context, params UpdateSalesOrderParams) (*SalesOrder, *apierror.APIError)

	// DeleteSalesOrder deletes a sales order and cascades to related records.
	DeleteSalesOrder(ctx context.Context, params DeleteSalesOrderParams) *apierror.APIError

	// BulkDeleteSalesOrders deletes multiple sales orders.
	BulkDeleteSalesOrders(ctx context.Context, params BulkDeleteSalesOrdersParams) *apierror.APIError

	// ChangeSalesOrderStatus changes the status of a sales order.
	ChangeSalesOrderStatus(ctx context.Context, params ChangeSalesOrderStatusParams) (*SalesOrder, *apierror.APIError)

	// CheckoutSalesOrder initiates a Stripe checkout for a sales order.
	CheckoutSalesOrder(ctx context.Context, params CheckoutSalesOrderParams) (*CheckoutSalesOrderResult, *apierror.APIError)
	QuoteSalesOrderLinePrices(ctx context.Context, params QuoteSalesOrderLinePricesParams) ([]SalesOrderLineQuote, *apierror.APIError)

	// QuoteSalesOrderFreight re-estimates an existing order's freight charge from its current ship-to, carrier, service level, and lines, without mutating the order.
	QuoteSalesOrderFreight(ctx context.Context, params QuoteSalesOrderFreightParams) (*SalesOrderFreightQuote, *apierror.APIError)

	// CreateSalesOrderProductionRun creates a production run from a sales order.
	CreateSalesOrderProductionRun(ctx context.Context, params CreateSalesOrderProductionRunParams) (*CreateSalesOrderProductionRunResult, *apierror.APIError)

	// CreateCustomerCheckoutSession creates an embedded Stripe checkout session for a customer actor.
	CreateCustomerCheckoutSession(ctx context.Context, params CreateCustomerCheckoutSessionParams) (*CreateCustomerCheckoutSessionResult, *apierror.APIError)

	// RecordOrderPayment links a succeeded Stripe payment intent to a sales order (called from ProcessAccountStripeWebhook and from the billing-service's platform webhook consumer). Idempotent: a payment intent already linked is a no-op.
	RecordOrderPayment(ctx context.Context, salesOrderID, paymentIntentID string) *apierror.APIError

	// ProcessAccountStripeWebhook verifies a webhook event from an account's connected Stripe account against the account's stored webhook secret, then links succeeded payment intents to their sales orders via RecordOrderPayment. Events that are not order payments are acknowledged and ignored so Stripe does not retry them.
	ProcessAccountStripeWebhook(ctx context.Context, accountID string, rawPayload []byte, signature string) *apierror.APIError
}

type SalesOrderLineSvc interface {
	// CreateSalesOrderLine creates a new line on a sales order.
	CreateSalesOrderLine(ctx context.Context, params CreateSalesOrderLineParams) (*SalesOrderLine, *apierror.APIError)

	// UpdateSalesOrderLine partially updates a sales order line.
	UpdateSalesOrderLine(ctx context.Context, params UpdateSalesOrderLineParams) (*SalesOrderLine, *apierror.APIError)

	// DeleteSalesOrderLine deletes a sales order line and cascades to pick/shipment/invoice lines.
	DeleteSalesOrderLine(ctx context.Context, params DeleteSalesOrderLineParams) *apierror.APIError

	// ReorderSalesOrderLines re-sequences the order's product lines to match the given order, keeping credit/freight lines at the bottom. Returns the order's lines in their new order.
	ReorderSalesOrderLines(ctx context.Context, params ReorderSalesOrderLinesParams) ([]*SalesOrderLine, *apierror.APIError)
}

type PurchaseOrderSvc interface {
	ListPurchaseOrders(ctx context.Context, params ListPurchaseOrdersParams) (*ListPurchaseOrdersResult, *apierror.APIError)
	GetPurchaseOrder(ctx context.Context, params GetPurchaseOrderParams) (*PurchaseOrder, *apierror.APIError)
	CreatePurchaseOrder(ctx context.Context, params CreatePurchaseOrderParams) (*PurchaseOrder, *apierror.APIError)
	UpdatePurchaseOrder(ctx context.Context, params UpdatePurchaseOrderParams) (*PurchaseOrder, *apierror.APIError)
	DeletePurchaseOrder(ctx context.Context, params DeletePurchaseOrderParams) *apierror.APIError
	BulkDeletePurchaseOrders(ctx context.Context, params BulkDeletePurchaseOrdersParams) *apierror.APIError
	ChangePurchaseOrderStatus(ctx context.Context, params ChangePurchaseOrderStatusParams) (*PurchaseOrder, *apierror.APIError)
}

type PurchaseOrderLineSvc interface {
	CreatePurchaseOrderLine(ctx context.Context, params CreatePurchaseOrderLineParams) (*PurchaseOrderLine, *apierror.APIError)
	UpdatePurchaseOrderLine(ctx context.Context, params UpdatePurchaseOrderLineParams) (*PurchaseOrderLine, *apierror.APIError)
	DeletePurchaseOrderLine(ctx context.Context, params DeletePurchaseOrderLineParams) *apierror.APIError
}

type PartSvc interface {
	// CreatePart creates a new part with its associated item and rates.
	CreatePart(ctx context.Context, params CreatePartParams) (*Part, *apierror.APIError)

	// GetPart returns a single part by ID.
	GetPart(ctx context.Context, params GetPartParams) (*Part, *apierror.APIError)

	// ListParts returns a paginated list of parts for the caller's account.
	ListParts(ctx context.Context, params ListPartsParams) (*ListPartsResult, *apierror.APIError)

	// ExportParts returns all matching parts (no pagination) for export.
	ExportParts(ctx context.Context, params ExportPartsParams) ([]*Part, *apierror.APIError)

	// UpdatePart partially updates a part's item fields.
	UpdatePart(ctx context.Context, params UpdatePartParams) (*Part, *apierror.APIError)

	// DeletePart soft-deletes a part by its item ID.
	DeletePart(ctx context.Context, itemID string) (*Part, *apierror.APIError)

	// BatchGetPartsByIDs returns parts by their IDs for include resolution.
	BatchGetPartsByIDs(ctx context.Context, ids []string) ([]*Part, *apierror.APIError)
}

type MaterialSvc interface {
	// ListMaterials returns a paginated list of materials for the caller's account.
	ListMaterials(ctx context.Context, params ListMaterialsParams) (*ListMaterialsResult, *apierror.APIError)

	// ExportMaterials returns all matching materials (no pagination) for export.
	ExportMaterials(ctx context.Context, params ExportMaterialsParams) ([]*Material, *apierror.APIError)

	// GetMaterial returns a single material by ID.
	GetMaterial(ctx context.Context, params GetMaterialParams) (*Material, *apierror.APIError)

	// CreateMaterial creates a new material.
	CreateMaterial(ctx context.Context, params CreateMaterialParams) (*Material, *apierror.APIError)

	// UpdateMaterial partially updates a material.
	UpdateMaterial(ctx context.Context, params UpdateMaterialParams) (*Material, *apierror.APIError)

	// DeleteMaterial soft-deletes a material by its item ID.
	DeleteMaterial(ctx context.Context, itemID string) (*Material, *apierror.APIError)

	// BatchGetMaterialsByIDs returns materials by their IDs for include resolution.
	BatchGetMaterialsByIDs(ctx context.Context, ids []string) ([]*Material, *apierror.APIError)
}

type SupplierMaterialSvc interface {
	// ListSupplierMaterials returns a paginated list of supplier materials.
	ListSupplierMaterials(ctx context.Context, params ListSupplierMaterialsParams) (*ListSupplierMaterialsResult, *apierror.APIError)

	// GetSupplierMaterial returns a single supplier material by supplier and material ID.
	GetSupplierMaterial(ctx context.Context, supplierAccountID, materialID string) (*SupplierMaterial, *apierror.APIError)

	// CreateSupplierMaterial creates a new supplier material association.
	CreateSupplierMaterial(ctx context.Context, params CreateSupplierMaterialParams) (*SupplierMaterial, *apierror.APIError)

	// UpdateSupplierMaterial partially updates a supplier material.
	UpdateSupplierMaterial(ctx context.Context, params UpdateSupplierMaterialParams) (*SupplierMaterial, *apierror.APIError)

	// DeleteSupplierMaterial deletes a supplier material association.
	DeleteSupplierMaterial(ctx context.Context, params DeleteSupplierMaterialParams) (*SupplierMaterial, *apierror.APIError)
}

type PermissionGroupSvc interface {
	// ListPermissionGroups returns a paginated list of permission groups with their nested permissions. Permission groups are global (not account-scoped).
	ListPermissionGroups(ctx context.Context, params ListPermissionGroupsParams) (*ListPermissionGroupsResult, *apierror.APIError)

	// BatchGetPermissionGroupsByIDs returns permission groups by their IDs for include resolution.
	BatchGetPermissionGroupsByIDs(ctx context.Context, ids []string) ([]*PermissionGroup, *apierror.APIError)
}

type PickSvc interface {
	// ListPicks returns a paginated list of picks for the caller's account.
	ListPicks(ctx context.Context, params ListPicksParams) (*ListPicksResult, *apierror.APIError)

	// GetPick returns a single pick by ID, optionally including lines and departments.
	GetPick(ctx context.Context, pickID string, includes []string) (*Pick, *apierror.APIError)

	// UpdatePick partially updates a pick's metadata (number).
	UpdatePick(ctx context.Context, params UpdatePickParams) (*Pick, *apierror.APIError)

	// PickAllLines picks all unpacked lines to their remaining quantities.
	PickAllLines(ctx context.Context, pickID string) (*Pick, *apierror.APIError)

	// VoidPick voids all lines in a pick, setting quantities to zero.
	VoidPick(ctx context.Context, pickID string) (*Pick, *apierror.APIError)

	// PackPick packs eligible lines and creates a shipment with shipping cases.
	PackPick(ctx context.Context, pickID string, shipmentCaseCount int32) (*PackPickResult, *apierror.APIError)

	// GetPickShipments returns shipment numbers associated with a pick's order.
	GetPickShipments(ctx context.Context, params GetPickShipmentsParams) (*PickShipmentsResult, *apierror.APIError)
}

type PickLineSvc interface {
	// UpdatePickLine updates a pick line's quantity value.
	UpdatePickLine(ctx context.Context, params UpdatePickLineParams) (*PickLine, *apierror.APIError)

	// PickPickLine picks a single line to its remaining quantity.
	PickPickLine(ctx context.Context, pickID, pickLineID string) (*PickLine, *apierror.APIError)

	// VoidPickLine voids a single pick line by setting its quantity to zero.
	VoidPickLine(ctx context.Context, pickID, pickLineID string) (*PickLine, *apierror.APIError)
}

type ReceivingOrderSvc interface {
	// ListReceivingOrders returns a paginated list of receiving orders.
	ListReceivingOrders(ctx context.Context, params ListReceivingOrdersParams) (*ListReceivingOrdersResult, *apierror.APIError)

	// GetReceivingOrder returns a single receiving order by ID with lines.
	GetReceivingOrder(ctx context.Context, params GetReceivingOrderParams) (*ReceivingOrder, *apierror.APIError)

	// StockReceivingOrder stocks a receiving order, creating deliveries and inventory records.
	StockReceivingOrder(ctx context.Context, params StockReceivingOrderParams) (*ReceivingOrder, *apierror.APIError)

	// ReceiveReceivingOrder receives all unstocked lines, setting their quantities to remaining.
	ReceiveReceivingOrder(ctx context.Context, receivingOrderID string) (*ReceivingOrder, *apierror.APIError)

	// VoidReceivingOrder voids all lines in a receiving order.
	VoidReceivingOrder(ctx context.Context, receivingOrderID string) (*ReceivingOrder, *apierror.APIError)
}

type ReceivingOrderLineSvc interface {
	// UpdateReceivingOrderLine updates a receiving order line's quantity.
	UpdateReceivingOrderLine(ctx context.Context, params UpdateReceivingOrderLineParams) (*ReceivingOrderLine, *apierror.APIError)

	// VoidReceivingOrderLine voids a single receiving order line.
	VoidReceivingOrderLine(ctx context.Context, receivingOrderID, lineID string) (*ReceivingOrderLine, *apierror.APIError)

	// ReceiveReceivingOrderLine receives a single line, setting its quantity to remaining.
	ReceiveReceivingOrderLine(ctx context.Context, receivingOrderID, lineID string) (*ReceivingOrderLine, *apierror.APIError)
}

type ProductTypeSvc interface {
	// ListProductTypes returns a paginated list of product types. Product types are global (not account-scoped).
	ListProductTypes(ctx context.Context, params ListProductTypesParams) (*ListProductTypesResult, *apierror.APIError)

	// GetProductType returns a single product type by ID or code.
	GetProductType(ctx context.Context, identifier string) (*ProductType, *apierror.APIError)

	// CreateProductType creates a new product type.
	CreateProductType(ctx context.Context, params CreateProductTypeParams) (*ProductType, *apierror.APIError)

	// UpdateProductType partially updates a product type.
	UpdateProductType(ctx context.Context, params UpdateProductTypeParams) (*ProductType, *apierror.APIError)

	// DeleteProductType deletes a product type by ID.
	DeleteProductType(ctx context.Context, id string) *apierror.APIError

	// BatchGetProductTypesByIDs returns product types matching the input IDs. Used by the api-gateway resourcekit include resolver.
	BatchGetProductTypesByIDs(ctx context.Context, ids []string) ([]*ProductType, *apierror.APIError)
}

type ProductionRunSvc interface {
	ListProductionRuns(ctx context.Context, params ListProductionRunsParams) (*ListProductionRunsResult, *apierror.APIError)
	GetProductionRun(ctx context.Context, params GetProductionRunParams) (*ProductionRun, *apierror.APIError)
	CreateProductionRun(ctx context.Context, params CreateProductionRunParams) (*ProductionRun, *apierror.APIError)
	UpdateProductionRun(ctx context.Context, params UpdateProductionRunParams) (*ProductionRun, *apierror.APIError)
	DeleteProductionRun(ctx context.Context, params DeleteProductionRunParams) *apierror.APIError
	AddBatchesToProductionRun(ctx context.Context, params AddBatchesToProductionRunParams) ([]*BaseBatch, *apierror.APIError)
	ListBatchesByProductionRun(ctx context.Context, params ListBatchesByProductionRunParams) (*ListBatchesByProductionRunResult, *apierror.APIError)
}

type ProductionStepSvc interface {
	// ListProductionSteps returns a paginated list of production steps.
	ListProductionSteps(ctx context.Context, params ListProductionStepsParams) (*ListProductionStepsResult, *apierror.APIError)

	// GetProductionStep returns a single production step by ID.
	GetProductionStep(ctx context.Context, id string) (*ProductionStep, *apierror.APIError)

	// CreateProductionStep creates a new production step with rates, production, and consumptions.
	CreateProductionStep(ctx context.Context, params CreateProductionStepParams) (*ProductionStep, *apierror.APIError)

	// UpdateProductionStep partially updates a production step.
	UpdateProductionStep(ctx context.Context, params UpdateProductionStepParams) (*ProductionStep, *apierror.APIError)

	// DeleteProductionStep deletes a production step and its associated data.
	DeleteProductionStep(ctx context.Context, id string) *apierror.APIError

	// BulkCreateProductionSteps creates multiple production steps in a single operation.
	BulkCreateProductionSteps(ctx context.Context, params BulkCreateProductionStepsParams) ([]BulkCreateProductionStepResult, *apierror.APIError)
}

type ProductionSvc interface {
	// GetProduction returns a single production output by ID within a production step.
	GetProduction(ctx context.Context, productionStepID, productionID string) (*Production, *apierror.APIError)

	// UpdateProduction partially updates a production output.
	UpdateProduction(ctx context.Context, params UpdateProductionParams) (*Production, *apierror.APIError)
}

type MeasureSvc interface {
	UpdateQuantity(ctx context.Context, params UpdateQuantityParams) (*Quantity, *apierror.APIError)
	UpdateRate(ctx context.Context, params UpdateRateParams) (*Rate, *apierror.APIError)
}

type UtilsSvc interface {
	CheckDuplicate(ctx context.Context, params CheckDuplicateParams) (*CheckDuplicateResult, *apierror.APIError)
	EmailRecord(ctx context.Context, params EmailRecordParams) *apierror.APIError
	RequestDemo(ctx context.Context, params RequestDemoParams) *apierror.APIError
	SubmitFeedback(ctx context.Context, params SubmitFeedbackParams) *apierror.APIError
}

type CatalogSvc interface {
	// ListCatalogProductLines returns a paginated list of product lines available in the catalog. Supports both internal and customer actors via CheckIsAssignedActor.
	ListCatalogProductLines(ctx context.Context, params ListCatalogProductLinesParams) (*ListCatalogProductLinesResult, *apierror.APIError)

	// ListCatalogProducts returns a paginated list of products in a specific product line, grouped by item category. Supports both internal and customer actors via CheckIsAssignedActor.
	ListCatalogProducts(ctx context.Context, params ListCatalogProductsParams) (*ListCatalogProductsResult, *apierror.APIError)
}

type EDISvc interface {
	// ListDCLocations returns a paginated list of DC locations.
	ListDCLocations(ctx context.Context, params ListDCLocationsParams) (*ListDCLocationsResult, *apierror.APIError)

	// GetDCLocation returns a single DC location by ID.
	GetDCLocation(ctx context.Context, dcLocationID string) (*DCLocation, *apierror.APIError)

	// CreateDCLocation creates a new DC location.
	CreateDCLocation(ctx context.Context, params CreateDCLocationParams) (*DCLocation, *apierror.APIError)

	// UpdateDCLocation partially updates a DC location.
	UpdateDCLocation(ctx context.Context, params UpdateDCLocationParams) (*DCLocation, *apierror.APIError)

	// DeleteDCLocation deletes a DC location.
	DeleteDCLocation(ctx context.Context, dcLocationID string) *apierror.APIError

	// ListEDIRuns returns a paginated list of EDI runs.
	ListEDIRuns(ctx context.Context, params ListEDIRunsParams) (*ListEDIRunsResult, *apierror.APIError)

	// GetEDIRun returns a single EDI run by ID.
	GetEDIRun(ctx context.Context, ediRunID string) (*EDIRun, *apierror.APIError)

	// PullOrders processes EDI operations (pull orders from FTP, process invoices).
	PullOrders(ctx context.Context) *apierror.APIError

	// ResubmitInvoice resubmits an invoice via EDI.
	ResubmitInvoice(ctx context.Context, invoiceID string) *apierror.APIError

	// BatchGetDCLocationsByIDs returns DC locations matching the input IDs. Used by the api-gateway resourcekit include resolver.
	BatchGetDCLocationsByIDs(ctx context.Context, ids []string) ([]*DCLocation, *apierror.APIError)

	// BatchGetEDIRunsByIDs returns EDI runs matching the input IDs. Used by the api-gateway resourcekit include resolver.
	BatchGetEDIRunsByIDs(ctx context.Context, ids []string) ([]*EDIRun, *apierror.APIError)
}

type RegistrationFlowSvc interface {
	ListRegistrationFlows(ctx context.Context, params ListRegistrationFlowsParams) (*ListRegistrationFlowsResult, *apierror.APIError)
	GetRegistrationFlow(ctx context.Context, flowID string) (*RegistrationFlow, *apierror.APIError)
	GetRegistrationFlowBySlug(ctx context.Context, slug string) (*RegistrationFlow, *apierror.APIError)
	CreateRegistrationFlow(ctx context.Context, params CreateRegistrationFlowParams) (*RegistrationFlow, *apierror.APIError)
	UpdateRegistrationFlow(ctx context.Context, params UpdateRegistrationFlowParams) (*RegistrationFlow, *apierror.APIError)
	DeleteRegistrationFlow(ctx context.Context, flowID string) *apierror.APIError
	RegisterCustomer(ctx context.Context, params RegisterCustomerParams) *apierror.APIError
}

type ScanningStationSvc interface {
	ListScanningStations(ctx context.Context, params ListScanningStationsParams) (*ListScanningStationsResult, *apierror.APIError)
	GetScanningStation(ctx context.Context, params GetScanningStationParams) (*ScanningStation, *apierror.APIError)
	BatchGetScanningStationsByIDs(ctx context.Context, ids []string) ([]*ScanningStation, *apierror.APIError)
	CreateScanningStation(ctx context.Context, params CreateScanningStationParams) (*ScanningStation, *apierror.APIError)
	UpdateScanningStation(ctx context.Context, params UpdateScanningStationParams) (*ScanningStation, *apierror.APIError)
	DeleteScanningStation(ctx context.Context, scanningStationID string) *apierror.APIError
	ConnectProductionStepsByName(ctx context.Context, params ConnectProductionStepsByNameParams) *apierror.APIError
}

type LocationSvc interface {
	ListLocations(ctx context.Context, params ListLocationsParams) (*ListLocationsResult, *apierror.APIError)
	GetLocation(ctx context.Context, params GetLocationParams) (*Location, *apierror.APIError)
	CreateLocation(ctx context.Context, params CreateLocationParams) (*Location, *apierror.APIError)
	UpdateLocation(ctx context.Context, params UpdateLocationParams) (*Location, *apierror.APIError)
	DeleteLocation(ctx context.Context, params DeleteLocationParams) *apierror.APIError
	ListLocationTypes(ctx context.Context, params ListLocationTypesParams) (*ListLocationTypesResult, *apierror.APIError)
	GetLocationType(ctx context.Context, params GetLocationTypeParams) (*LocationType, *apierror.APIError)
	BatchGetLocationsByIDs(ctx context.Context, ids []string) ([]*Location, *apierror.APIError)
}

type SupplierSvc interface {
	ListSuppliers(ctx context.Context, params ListSuppliersParams) (*ListSuppliersResult, *apierror.APIError)
	GetSupplier(ctx context.Context, params GetSupplierParams) (*Supplier, *apierror.APIError)
	CreateSupplier(ctx context.Context, params CreateSupplierParams) (*Supplier, *apierror.APIError)
	UpdateSupplier(ctx context.Context, params UpdateSupplierParams) (*Supplier, *apierror.APIError)
	DeleteSupplier(ctx context.Context, params DeleteSupplierParams) (*Supplier, *apierror.APIError)
	BulkDeleteSuppliers(ctx context.Context, params BulkDeleteSuppliersParams) *apierror.APIError
}

type SysPropertySvc interface {
	ListSysProperties(ctx context.Context, params ListSysPropertiesParams) (*ListSysPropertiesResult, *apierror.APIError)
	GetSysProperty(ctx context.Context, sysPropertyID string) (*SysProperty, *apierror.APIError)
	GetSysPropertyValue(ctx context.Context, code string) (*SysPropertyValue, *apierror.APIError)
	GetLatestSysPropertyValue(ctx context.Context, typeCode constants.SysPropertyTypeCode) (string, *apierror.APIError)
	UpdateSysProperty(ctx context.Context, params UpdateSysPropertyParams) (*SysProperty, *apierror.APIError)

	// BatchGetSysPropertiesByIDs returns sys properties matching the input IDs. Used by the api-gateway resourcekit include resolver.
	BatchGetSysPropertiesByIDs(ctx context.Context, ids []string) ([]*SysProperty, *apierror.APIError)
}

type TenancySvc interface {
	GetTenancy(ctx context.Context, userID string, targetAccountID *string) (*Tenancy, *apierror.APIError)
	SwitchAccount(ctx context.Context, userID, accountID string) (*Tenancy, *apierror.APIError)
	GetCurrentUser(ctx context.Context, userID string, targetAccountID *string) (*UserRecord, *apierror.APIError)
	ListCustomerAccountsForUser(ctx context.Context, userID, vendorAccountID string) ([]CustomerAccountSummary, *apierror.APIError)
}

type TerritorySvc interface {
	ListTerritories(ctx context.Context, params ListTerritoriesParams) (*ListTerritoriesResult, *apierror.APIError)
	GetTerritory(ctx context.Context, params GetTerritoryParams) (*Territory, *apierror.APIError)
	CreateTerritory(ctx context.Context, params CreateTerritoryParams) (*Territory, *apierror.APIError)
	UpdateTerritory(ctx context.Context, params UpdateTerritoryParams) (*Territory, *apierror.APIError)
	DeleteTerritory(ctx context.Context, params DeleteTerritoryParams) *apierror.APIError
	BatchGetTerritoriesByIDs(ctx context.Context, ids []string) ([]*Territory, *apierror.APIError)
}

type ShippingCaseSvc interface {
	GetShippingCase(ctx context.Context, accountID, shippingCaseID string) (*ShippingCase, *apierror.APIError)
	UpdateShippingCase(ctx context.Context, params UpdateShippingCaseParams) (*ShippingCase, *apierror.APIError)
	DeleteShippingCase(ctx context.Context, accountID, shippingCaseID string) *apierror.APIError
	GetShippingCaseLabel(ctx context.Context, accountID, shippingCaseID string) (*string, *apierror.APIError)
}

type ShipmentSvc interface {
	ListShipments(ctx context.Context, params ListShipmentsParams) (*ListShipmentsResult, *apierror.APIError)
	GetShipment(ctx context.Context, params GetShipmentParams) (*Shipment, *apierror.APIError)
	UpdateShipment(ctx context.Context, params UpdateShipmentParams) (*Shipment, *apierror.APIError)
	DeleteShipment(ctx context.Context, params DeleteShipmentParams) *apierror.APIError
	ShipShipment(ctx context.Context, params ShipShipmentParams) (*Shipment, *apierror.APIError)
	VoidShipment(ctx context.Context, params VoidShipmentParams) (*Shipment, *apierror.APIError)
	EstimateRate(ctx context.Context, params EstimateRateParams) (float64, *apierror.APIError)
	RateShop(ctx context.Context, params RateShopParams) (*RateShopResult, *apierror.APIError)
}

type ShipmentLineSvc interface {
	ListShipmentLines(ctx context.Context, params ListShipmentLinesParams) (*ListShipmentLinesResult, *apierror.APIError)
	GetShipmentLine(ctx context.Context, accountID, shipmentID, shipmentLineID string) (*ShipmentLine, *apierror.APIError)
	CreateShipmentLine(ctx context.Context, params CreateShipmentLineEndpointParams) (*ShipmentLine, *apierror.APIError)
	UpdateShipmentLine(ctx context.Context, params UpdateShipmentLineEndpointParams) (*ShipmentLine, *apierror.APIError)
	DeleteShipmentLine(ctx context.Context, params DeleteShipmentLineEndpointParams) *apierror.APIError
}

type RoleSvc interface {
	ListRoles(ctx context.Context, params ListRolesParams) (*ListRolesResult, *apierror.APIError)
	GetRole(ctx context.Context, roleID string, incs []string) (*RoleWithPermissions, *apierror.APIError)
	CreateRole(ctx context.Context, params CreateRoleParams) (*RoleWithPermissions, *apierror.APIError)
	UpdateRole(ctx context.Context, params UpdateRoleParams) (*RoleWithPermissions, *apierror.APIError)
	DeleteRole(ctx context.Context, roleID string) *apierror.APIError
	BatchGetRolesByIDs(ctx context.Context, ids []string) ([]*RoleWithPermissions, *apierror.APIError)
}

type StripeWebhookSvc interface {
	HandleAccountStripeWebhook(ctx context.Context, params HandleStripeWebhookParams) *apierror.APIError
}

type PortalDomainSvc interface {
	CreatePortalDomain(ctx context.Context, domainName string) (*PortalDomain, *apierror.APIError)
	GetPortalDomain(ctx context.Context, portalDomainID string) (*PortalDomain, *apierror.APIError)
	ListPortalDomains(ctx context.Context) ([]*PortalDomain, *apierror.APIError)
	VerifyPortalDomain(ctx context.Context, portalDomainID string) (*PortalDomain, *apierror.APIError)
	DeletePortalDomain(ctx context.Context, portalDomainID string) *apierror.APIError
	ResolvePortalHost(ctx context.Context, domainName string) (*PublicAccountBySlug, *apierror.APIError)
	BatchGetPortalDomainsByIDs(ctx context.Context, ids []string) ([]*PortalDomain, *apierror.APIError)
}

// CustomerRegistrar registers the authenticated buyer as a customer of a seller account. Implemented by the registration-flow service; injected into the portal registration-session service so completion reuses the existing one-shot registration logic.
type CustomerRegistrar interface {
	RegisterCustomer(ctx context.Context, params RegisterCustomerParams) *apierror.APIError
}

// PortalRegistrationSessionSvc drives a buyer's session-based registration into a seller's customer portal.
type PortalRegistrationSessionSvc interface {
	// CreateOrResumeSession returns the buyer's active session for the seller (resuming a non-expired incomplete one), or starts a new one.
	CreateOrResumeSession(ctx context.Context, sellerSlug string) (*PortalRegistrationSession, *apierror.APIError)
	GetSession(ctx context.Context, typeID string) (*PortalRegistrationSession, *apierror.APIError)
	UpdateSession(ctx context.Context, params UpdatePortalRegistrationSessionParams) (*PortalRegistrationSession, *apierror.APIError)
	CompleteSession(ctx context.Context, typeID string) (*PortalRegistrationSession, *apierror.APIError)
	AbandonSession(ctx context.Context, typeID string) (*PortalRegistrationSession, *apierror.APIError)
	// ListSessions returns the seller account's registration sessions for the customer-service follow-up view (seller-facing; scoped to the caller's account).
	ListSessions(ctx context.Context, params ListPortalRegistrationSessionsParams) (*ListPortalRegistrationSessionsResult, *apierror.APIError)
}

// CreateAccountParams holds the parameters for creating a production account during registration.
type CreateAccountParams struct {
	ID                   string
	Name                 string
	PlanCode             string
	StripeCustomerID     string
	StripeSubscriptionID string
}
