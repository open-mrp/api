package id

import "strings"

// IDPrefix is a 2-letter prefix for a type-specific ID. This will be used to
// compose the prefix for the ID. Since each word has a unique 2 letter vocabulary,
// we can ensure the type ID provides a machine-readable and human-readable identifier.
type IDPrefix string

// Vocabulary contains all 2-letter word codes used to compose ID prefixes.
// Each code represents a semantic word/concept.
const (
	// Base entities
	VocAccount      = "ac"
	VocAddress      = "ad"
	VocAction       = "ax"
	VocAPI          = "ap"
	VocAttribute    = "at"
	VocAdjustment   = "aj"
	VocAllocation   = "al"
	VocAssignment   = "as"
	VocBatch        = "bt"
	VocBilling      = "bl"
	VocBranding     = "br"
	VocCache        = "ca"
	VocChange       = "ch"
	VocColor        = "cl"
	VocCommission   = "cm"
	VocConsumption  = "cp"
	VocCarrier      = "cr"
	VocCase         = "cs"
	VocContact      = "ct"
	VocCustomer     = "cu"
	VocConversion   = "ce"
	VocCategory     = "cg"
	VocDC           = "dc"
	VocDimension    = "dm"
	VocDiscount     = "ds"
	VocDefinition   = "df"
	VocDepartment   = "dp"
	VocEDI          = "ed"
	VocEmail        = "em"
	VocEnterprise   = "en"
	VocError        = "er"
	VocEvent        = "ev"
	VocFormula      = "fm"
	VocFreight      = "fr"
	VocFlow         = "fw"
	VocGeolocation  = "gl"
	VocGroup        = "gp"
	VocIdempotency  = "ip"
	VocIntegration  = "ig"
	VocImage        = "im"
	VocInventory    = "in"
	VocInquiry      = "ir"
	VocIntent       = "ie"
	VocItem         = "it"
	VocInvoice      = "iv"
	VocInbox        = "ix"
	VocKey          = "ke"
	VocLabel        = "lb"
	VocLocation     = "lc"
	VocLog          = "lg"
	VocLine         = "ln"
	VocMachine      = "mc"
	VocMethod       = "md"
	VocMessage      = "mg"
	VocMaterial     = "ml"
	VocNotification = "nf"
	VocOrganization = "og"
	VocOnboarding   = "ob"
	VocOption       = "op"
	VocOrder        = "or"
	VocOutbox       = "ox"
	VocProduct      = "pd"
	VocPreference   = "pf"
	VocPriority     = "pi"
	VocPick         = "pk"
	VocPortal       = "po"
	VocPlan         = "pl"
	VocPermission   = "pm"
	VocProduction   = "pn"
	VocProperty     = "pp"
	VocPrice        = "pr"
	VocProcess      = "ps"
	VocPart         = "pt"
	VocPayment      = "py"
	VocQuantity     = "qu"
	VocRecipient    = "rp"
	VocRelation     = "re"
	VocRegistration = "rg"
	VocRole         = "rl"
	VocRun          = "rn"
	VocRequest      = "rq"
	VocRate         = "rt"
	VocSandbox      = "sb"
	VocSession      = "se"
	VocShipment     = "sh"
	VocSettlement   = "sl"
	VocScanning     = "sg"
	VocService      = "si"
	VocStripe       = "sr"
	VocStep         = "st"
	VocStation      = "sc"
	VocStatus       = "ss"
	VocSupplier     = "su"
	VocSeverity     = "sv"
	VocSystem       = "sy"
	VocSize         = "sz"
	VocTerritory    = "te"
	VocTarget       = "ta"
	VocToken        = "tk"
	VocTerm         = "tm"
	VocTransform    = "tf"
	VocTier         = "tr"
	VocTransaction  = "tx"
	VocType         = "tp"
	VocUnit         = "un"
	VocUser         = "us"
	VocVisibility   = "vi"
	VocVerification = "ve"
)

// composePrefix concatenates vocabulary words to form a prefix.
// This ensures consistency and makes prefix construction explicit.
func composePrefix(words ...string) IDPrefix {
	return IDPrefix(strings.Join(words, ""))
}

// IDPrefixes contains all the type ID prefix values used in the system.
// This serves as a centralized reference for all type ID prefix values.
// These constants can be used when creating new type ID structs or for validation.
var (
	// Account-related prefix values
	AccountIDPrefix                               = composePrefix(VocAccount)
	AccountAddressIDPrefix                        = composePrefix(VocAccount, VocAddress)
	AccountBillingIDPrefix                        = composePrefix(VocAccount, VocBilling)
	AccountBrandingIDPrefix                       = composePrefix(VocAccount, VocBranding)
	AccountPortalIDPrefix                         = composePrefix(VocAccount, VocPortal)
	AccountIntegrationIDPrefix                    = composePrefix(VocAccount, VocIntegration)
	AccountGroupIDPrefix                          = composePrefix(VocAccount, VocGroup)
	AccountGroupTypeIDPrefix                      = composePrefix(VocAccount, VocGroup, VocType)
	AccountTypeIDPrefix                           = composePrefix(VocAccount, VocType)
	AccountRelationIDPrefix                       = composePrefix(VocAccount, VocRelation)
	AccountStatusIDPrefix                         = composePrefix(VocAccount, VocStatus)
	AccountPriceIDPrefix                          = composePrefix(VocAccount, VocPrice)
	AccountPriceItemCategoryIDPrefix              = composePrefix(VocAccount, VocPrice, VocItem, VocCategory)
	AccountGroupQuantityDiscountIDPrefix          = composePrefix(VocAccount, VocGroup, VocQuantity, VocDiscount)
	AccountPriceAttributeIDPrefix                 = composePrefix(VocAccount, VocPrice, VocAttribute)
	AccountRelationProductLineIDPrefix            = composePrefix(VocAccount, VocRelation, VocProduct, VocLine)
	AccountRelationPriceGroupIDPrefix             = composePrefix(VocAccount, VocRelation, VocPrice, VocGroup)
	AccountRelationNotificationPreferenceIDPrefix = composePrefix(VocAccount, VocRelation, VocNotification, VocPreference)
	AccountRelationNotificationTypeIDPrefix       = composePrefix(VocAccount, VocRelation, VocNotification, VocType)
	AccountGroupProductLineIDPrefix               = composePrefix(VocAccount, VocGroup, VocProduct, VocLine)
	AccountPreferenceIDPrefix                     = composePrefix(VocAccount, VocPreference)
	AccountUserIDPrefix                           = composePrefix(VocAccount, VocUser)

	// Action-related prefix values
	ActionTypeIDPrefix = composePrefix(VocAction, VocType)

	// Address-related prefix values
	AddressIDPrefix = composePrefix(VocAddress)

	// API-related prefix values
	APILocationIDPrefix = composePrefix(VocAPI, VocLocation)
	APIKeyIDPrefix      = composePrefix(VocAPI, VocKey)

	// Attribute-related prefix values
	AttributeIDPrefix = composePrefix(VocAttribute)

	// Batch-related prefix values
	BatchIDPrefix = composePrefix(VocBatch)

	// Carrier-related prefix values
	CarrierIDPrefix       = composePrefix(VocCarrier)
	CarrierOptionIDPrefix = composePrefix(VocCarrier, VocOption)

	// Color-related prefix values
	ColorIDPrefix = composePrefix(VocColor)

	// Commission-related prefix values
	CommissionStatusIDPrefix = composePrefix(VocCommission, VocStatus)

	// Consumption-related prefix values
	ConsumptionIDPrefix = composePrefix(VocConsumption)

	// Contact-related prefix values
	ContactIDPrefix     = composePrefix(VocContact)
	ContactTypeIDPrefix = composePrefix(VocContact, VocType)

	// Customer-related prefix values
	// !Depreciated
	// CustomerIDPrefix             = composePrefix(VocCustomer)
	// CustomerGroupIDPrefix        = composePrefix(VocCustomer, VocGroup)
	// CustomerGroupTypeIDPrefix    = composePrefix(VocCustomer, VocGroup, VocType)
	// CustomerGroupProductIDPrefix = composePrefix(VocCustomer, VocGroup, VocProduct)
	// CustomerProductIDPrefix      = composePrefix(VocCustomer, VocProduct)
	// CustomerStatusIDPrefix       = composePrefix(VocCustomer, VocStatus)
	// CustomerAddressesIDPrefix    = composePrefix(VocCustomer, VocAddress)
	// CustomerEmailIDPrefix        = composePrefix(VocCustomer, VocEmail)
	// CustomerPriceIDPrefix        = composePrefix(VocCustomer, VocPrice)
	// CustomerPriceGroupsIDPrefix  = composePrefix(VocCustomer, VocPrice, VocGroup)

	// Department-related prefix values
	DepartmentIDPrefix = composePrefix(VocDepartment)

	// Discount-related prefix values
	DiscountTypeIDPrefix         = composePrefix(VocDiscount, VocStatus, VocType)
	OrderDiscountIDPrefix        = composePrefix(VocOrder, VocDiscount)
	QuantityDiscountIDPrefix     = composePrefix(VocQuantity, VocDiscount)
	QuantityDiscountTierIDPrefix = composePrefix(VocQuantity, VocDiscount, VocTier)

	// Error-related prefix values
	ErrorSeverityIDPrefix = composePrefix(VocError, VocSeverity)
	ErrorLogIDPrefix      = composePrefix(VocError, VocLog)

	// Freight-related prefix values
	FreightStatusIDPrefix = composePrefix(VocFreight, VocStatus)

	// Geolocation-related prefix values
	GeolocationIDPrefix = composePrefix(VocGeolocation)

	// Integration-related prefix values
	IntegrationIDPrefix = composePrefix(VocIntegration)

	// Invoice-related prefix values
	InvoiceIDPrefix     = composePrefix(VocInvoice)
	InvoiceLineIDPrefix = composePrefix(VocInvoice, VocLine)

	// Item-related prefix values
	ItemIDPrefix             = composePrefix(VocItem)
	ItemCategoryIDPrefix     = composePrefix(VocItem, VocCategory)
	ItemCategoryTypeIDPrefix = composePrefix(VocItem, VocCategory, VocType)
	ItemTypeIDPrefix         = composePrefix(VocItem, VocType)

	// Label-related prefix values
	LabelSizeIDPrefix = composePrefix(VocLabel, VocSize)
	LabelTypeIDPrefix = composePrefix(VocLabel, VocType)

	// Machine-related prefix values
	MachineIDPrefix = composePrefix(VocMachine)

	// Material-related prefix values
	MaterialIDPrefix         = composePrefix(VocMaterial)
	SupplierMaterialIDPrefix = composePrefix(VocSupplier, VocMaterial)

	// Organization-related prefix values
	OrganizationIDPrefix = composePrefix(VocOrganization)

	// Part-related prefix values
	PartIDPrefix = composePrefix(VocPart)

	// Payment-related prefix values
	PaymentTermIDPrefix        = composePrefix(VocPayment, VocTerm)
	OrderPaymentIntentIDPrefix = composePrefix(VocOrder, VocPayment, VocIntent)

	// Permission-related prefix values
	PermissionIDPrefix      = composePrefix(VocPermission)
	PermissionGroupIDPrefix = composePrefix(VocPermission, VocGroup)

	// Pick-related prefix values
	PickIDPrefix           = composePrefix(VocPick)
	PickLineIDPrefix       = composePrefix(VocPick, VocLine)
	PickAssignmentIDPrefix = composePrefix(VocPick, VocAssignment)

	// Preference-related prefix values
	PreferenceDefinitionIDPrefix = composePrefix(VocPreference, VocDefinition)

	// Priority-related prefix values
	PriorityIDPrefix = composePrefix(VocPriority)

	// Product-related prefix values
	ProductIDPrefix           = composePrefix(VocProduct)
	ProductLineIDPrefix       = composePrefix(VocProduct, VocLine)
	ProductLineTargetIDPrefix = composePrefix(VocProduct, VocLine, VocTarget)
	ProductTypeIDPrefix       = composePrefix(VocProduct, VocType)
	ProductImageIDPrefix      = composePrefix(VocProduct, VocImage)

	// Production-related prefix values
	ProductionIDPrefix                    = composePrefix(VocProduction)
	ProductionFormulaIDPrefix             = composePrefix(VocProduction, VocFormula)
	ProductionFormulaItemIDPrefix         = composePrefix(VocProduction, VocFormula, VocItem)
	ProductionProcessIDPrefix             = composePrefix(VocProduction, VocProcess)
	ProductionProcessItemCategoryIDPrefix = composePrefix(VocProduction, VocProcess, VocItem, VocCategory)
	ProductionRunIDPrefix                 = composePrefix(VocProduction, VocRun)
	ProductionStepIDPrefix                = composePrefix(VocProduction, VocStep)
	ProductionStepTransformIDPrefix       = composePrefix(VocProduction, VocStep, VocTransform)

	// Property-related prefix values
	PropertyIDPrefix        = composePrefix(VocProperty)
	SysPropertyIDPrefix     = composePrefix(VocSystem, VocProperty)
	SysPropertyTypeIDPrefix = composePrefix(VocSystem, VocProperty, VocType)

	// Quantity-related prefix values
	QuantityIDPrefix = composePrefix(VocQuantity)

	// Rate-related prefix values
	RateIDPrefix = composePrefix(VocRate)

	// Role-related prefix values
	RoleIDPrefix           = composePrefix(VocRole)
	RolePermissionIDPrefix = composePrefix(VocRole, VocPermission)
	RoleTypeIDPrefix       = composePrefix(VocRole, VocType)

	// Order-related prefix values
	OrderIDPrefix       = composePrefix(VocOrder)
	OrderLineIDPrefix   = composePrefix(VocOrder, VocLine)
	OrderStatusIDPrefix = composePrefix(VocOrder, VocStatus)
	OrderTypeIDPrefix   = composePrefix(VocOrder, VocType)

	// Territory-related prefix values
	TerritoryIDPrefix = composePrefix(VocTerritory)

	// Target-related prefix values
	TargetIDPrefix = composePrefix(VocTarget)

	// Sandbox-related prefix values
	SandboxAccountIDPrefix = composePrefix(VocSandbox, VocAccount)

	// Scanning Station-related prefix values
	ScanningStationIDPrefix     = composePrefix(VocScanning, VocStation)
	ScanningStationTypeIDPrefix = composePrefix(VocScanning, VocStation, VocType)

	// Session-related prefix values
	SessionIDPrefix = IDPrefix(VocSession)

	// Shipment-related prefix values
	ShipmentIDPrefix       = composePrefix(VocShipment)
	ShipmentLineIDPrefix   = composePrefix(VocShipment, VocLine)
	ShipmentStatusIDPrefix = composePrefix(VocShipment, VocStatus)
	ShippingCaseIDPrefix   = composePrefix(VocShipment, VocCase)
	ShippingTermIDPrefix   = composePrefix(VocShipment, VocTerm)

	// Unit-related prefix values
	UnitIDPrefix            = composePrefix(VocUnit)
	UnitDimensionIDPrefix   = composePrefix(VocUnit, VocDimension)
	UnitGroupIDPrefix       = composePrefix(VocUnit, VocGroup)
	UnitGroupsUnitsIDPrefix = composePrefix(VocUnit, VocGroup, VocUnit)
	UnitConversionIDPrefix  = composePrefix(VocUnit, VocConversion)

	// User-related prefix values
	UserIDPrefix        = composePrefix(VocUser)
	UserAccountIDPrefix = composePrefix(VocUser, VocAccount)
	UserStatusIDPrefix  = composePrefix(VocUser, VocStatus)
	UserTypeIDPrefix    = composePrefix(VocUser, VocType)

	// Verification-related prefix values
	VerificationTokenIDPrefix = composePrefix(VocVerification, VocToken)

	// Visibility-related prefix values
	VisibilityTypeIDPrefix = composePrefix(VocVisibility, VocType)

	// Adjustment-related prefix values
	AdjustmentTypeIDPrefix = composePrefix(VocAdjustment, VocType)

	// Change Log-related prefix values
	ChangeLogIDPrefix = composePrefix(VocChange, VocLog)

	// DC Location-related prefix values
	DCLocationIDPrefix = composePrefix(VocDC, VocLocation)

	// Email-related prefix values
	EmailLogIDPrefix       = composePrefix(VocEmail, VocLog)
	EmailRecipientIDPrefix = composePrefix(VocEmail, VocRecipient)
	OrderEmailIDPrefix     = composePrefix(VocOrder, VocEmail)

	// Inventory-related prefix values
	InventoryChangeLogIDPrefix = composePrefix(VocInventory, VocChange, VocLog)
	InventoryLogIDPrefix       = composePrefix(VocInventory, VocLog)

	// Registration-related prefix values
	RegistrationFlowIDPrefix = composePrefix(VocRegistration, VocFlow)

	// Onboarding-related prefix values
	OnboardingStatusIDPrefix = composePrefix(VocOnboarding, VocStatus)

	// Settlement-related prefix values
	SettlementIDPrefix = composePrefix(VocSettlement)

	// Stripe-related prefix values
	StripeEventLogIDPrefix = composePrefix(VocStripe, VocEvent, VocLog)

	// Transaction-related prefix values
	TransactionIDPrefix           = composePrefix(VocTransaction)
	TransactionAllocationIDPrefix = composePrefix(VocTransaction, VocAllocation)
	TransactionMethodIDPrefix     = composePrefix(VocTransaction, VocMethod)
	TransactionTypeIDPrefix       = composePrefix(VocTransaction, VocType)

	// EDI-related prefix values
	EDIRunIDPrefix = composePrefix(VocEDI, VocRun)

	// Request-related prefix values
	RequestIDPrefix = composePrefix(VocRequest)

	// Plan Change-related prefix values
	PlanChangeIDPrefix = composePrefix(VocPlan, VocChange)

	// Enterprise Inquiry-related prefix values
	EnterpriseInquiryIDPrefix = composePrefix(VocEnterprise, VocInquiry)

	// Idempotency-related prefix values
	IdempotencyKeyIDPrefix        = composePrefix(VocIdempotency, VocKey)
	ServiceIdempotencyKeyIDPrefix = composePrefix(VocService, VocIdempotency, VocKey)

	// Messaging-related prefix values
	MessageIDPrefix = composePrefix(VocMessage)
)
