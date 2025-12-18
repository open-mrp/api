package id

type IDPrefix string

// IDPrefixs contains all the type ID prefix values used in the system.
// This serves as a centralized reference for all type ID prefix values.
// These constants can be used when creating new type ID structs or for validation.
const (
	// Account-related prefix values
	AccountIDPrefix                               IDPrefix = "ac"
	AccountAddressIDPrefix                        IDPrefix = "acad"
	AccountBillingIDPrefix                        IDPrefix = "acbl"
	AccountBrandingIDPrefix                       IDPrefix = "acbr"
	AccountPortalIDPrefix                         IDPrefix = "acpl"
	AccountIntegrationIDPrefix                    IDPrefix = "acig"
	AccountGroupIDPrefix                          IDPrefix = "acgp"
	AccountGroupTypeIDPrefix                      IDPrefix = "acgpty"
	AccountTypeIDPrefix                           IDPrefix = "acty"
	AccountRelationIDPrefix                       IDPrefix = "acre"
	AccountStatusIDPrefix                         IDPrefix = "acst"
	AccountPriceIDPrefix                          IDPrefix = "acpr"
	AccountPriceItemCategoryIDPrefix              IDPrefix = "acpritcy"
	AccountGroupQuantityDiscountIDPrefix          IDPrefix = "acgpqtds"
	AccountPriceAttributeIDPrefix                 IDPrefix = "acprat"
	AccountRelationProductLineIDPrefix            IDPrefix = "acrepdln"
	AccountRelationPriceGroupIDPrefix             IDPrefix = "acreprgp"
	AccountRelationNotificationPreferenceIDPrefix IDPrefix = "acrentpf"
	AccountRelationNotificationTypeIDPrefix       IDPrefix = "acrentty"
	AccountGroupProductLineIDPrefix               IDPrefix = "acgppdln"
	AccountPreferenceIDPrefix                     IDPrefix = "acpf"
	AccountUserIDPrefix                           IDPrefix = "acus"

	// Action-related prefix values
	ActionTypeIDPrefix IDPrefix = "anty"

	// Address-related prefix values
	AddressIDPrefix IDPrefix = "ad"

	// API-related prefix values
	APILocationIDPrefix IDPrefix = "aplc"
	APIKeyIDPrefix      IDPrefix = "apky"

	// Attribute-related prefix values
	AttributeIDPrefix IDPrefix = "at"

	// Batch-related prefix values
	BatchIDPrefix IDPrefix = "bt"

	// Carrier-related prefix values
	CarrierIDPrefix       IDPrefix = "cr"
	CarrierOptionIDPrefix IDPrefix = "crop"

	// Color-related prefix values
	ColorIDPrefix IDPrefix = "cl"

	// Commission-related prefix values
	CommissionStatusIDPrefix IDPrefix = "cmst"

	// Consumption-related prefix values
	ConsumptionIDPrefix IDPrefix = "cn"

	// Contact-related prefix values
	ContactIDPrefix     IDPrefix = "ct"
	ContactTypeIDPrefix IDPrefix = "ctty"

	// Customer-related prefix values
	CustomerIDPrefix             IDPrefix = "cu"
	CustomerGroupIDPrefix        IDPrefix = "cugp"
	CustomerGroupTypeIDPrefix    IDPrefix = "cugpty"
	CustomerGroupProductIDPrefix IDPrefix = "cugppd"
	CustomerProductIDPrefix      IDPrefix = "cupd"
	CustomerStatusIDPrefix       IDPrefix = "cust"
	CustomerAddressesIDPrefix    IDPrefix = "cuad"
	CustomerEmailIDPrefix        IDPrefix = "cuem"
	CustomerPriceIDPrefix        IDPrefix = "cupr"
	CustomerPriceGroupsIDPrefix  IDPrefix = "cuprgp"

	// Department-related prefix values
	DepartmentIDPrefix IDPrefix = "dt"

	// Discount-related prefix values
	DiscountTypeIDPrefix         IDPrefix = "dsty"
	OrderDiscountIDPrefix        IDPrefix = "ords"
	QuantityDiscountIDPrefix     IDPrefix = "qtds"
	QuantityDiscountTierIDPrefix IDPrefix = "qtdstr"

	// Error-related prefix values
	ErrorSeverityIDPrefix IDPrefix = "ersv"
	ErrorLogIDPrefix      IDPrefix = "erlg"

	// Freight-related prefix values
	FreightStatusIDPrefix IDPrefix = "frst"

	// Geolocation-related prefix values
	GeolocationIDPrefix IDPrefix = "gl"

	// Integration-related prefix values
	IntegrationIDPrefix IDPrefix = "ig"

	// Invoice-related prefix values
	InvoiceIDPrefix          IDPrefix = "iv"
	InvoiceLineIDPrefix      IDPrefix = "ivln"
	InvoiceLineCacheIDPrefix IDPrefix = "ivlnca"

	// Item-related prefix values
	ItemIDPrefix             IDPrefix = "it"
	ItemCategoryIDPrefix     IDPrefix = "itcy"
	ItemCategoryTypeIDPrefix IDPrefix = "itcyty"
	ItemTypeIDPrefix         IDPrefix = "itty"

	// Label-related prefix values
	LabelSizeIDPrefix IDPrefix = "llsz"
	LabelTypeIDPrefix IDPrefix = "lbty"

	// Machine-related prefix values
	MachineIDPrefix IDPrefix = "mc"

	// Material-related prefix values
	MaterialIDPrefix         IDPrefix = "mt"
	SupplierMaterialIDPrefix IDPrefix = "sumt"

	// Organization-related prefix values
	OrganizationIDPrefix IDPrefix = "og"

	// Part-related prefix values
	PartIDPrefix IDPrefix = "pt"

	// Payment-related prefix values
	PaymentTermIDPrefix        IDPrefix = "pytm"
	OrderPaymentIntentIDPrefix IDPrefix = "sopi"

	// Permission-related prefix values
	PermissionIDPrefix      IDPrefix = "pm"
	PermissionGroupIDPrefix IDPrefix = "pmgp"

	// Pick-related prefix values
	PickIDPrefix           IDPrefix = "pk"
	PickLineIDPrefix       IDPrefix = "pkln"
	PickAssignmentIDPrefix IDPrefix = "pkas"

	// Preference-related prefix values
	PreferenceDefinitionIDPrefix IDPrefix = "pf"

	// Priority-related prefix values
	PriorityIDPrefix IDPrefix = "pi"

	// Product-related prefix values
	ProductIDPrefix           IDPrefix = "pd"
	ProductLineIDPrefix       IDPrefix = "pdln"
	ProductLineTargetIDPrefix IDPrefix = "pdlntg"
	ProductTypeIDPrefix       IDPrefix = "pdty"
	ProductImageIDPrefix      IDPrefix = "pdim"

	// Production-related prefix values
	ProductionIDPrefix                    IDPrefix = "pn"
	ProductionFormulaIDPrefix             IDPrefix = "pnfm"
	ProductionFormulaItemIDPrefix         IDPrefix = "pnfmit"
	ProductionProcessIDPrefix             IDPrefix = "pnps"
	ProductionProcessItemCategoryIDPrefix IDPrefix = "pnpsitcy"
	ProductionRunIDPrefix                 IDPrefix = "pnrn"
	ProductionStepIDPrefix                IDPrefix = "pnsp"
	ProductionStepTransformIDPrefix       IDPrefix = "pnsptn"

	// Property-related prefix values
	PropertyIDPrefix        IDPrefix = "pp"
	SysPropertyIDPrefix     IDPrefix = "sypp"
	SysPropertyTypeIDPrefix IDPrefix = "syppty"

	// Quantity-related prefix values
	QuantityIDPrefix IDPrefix = "qt"

	// Rate-related prefix values
	RateIDPrefix IDPrefix = "rt"

	// Role-related prefix values
	RoleIDPrefix           IDPrefix = "rl"
	RolePermissionIDPrefix IDPrefix = "rlpm"
	RoleTypeIDPrefix       IDPrefix = "rlty"

	// Sales Order-related prefix values
	SalesOrderIDPrefix          IDPrefix = "so"
	SalesOrderLineIDPrefix      IDPrefix = "soln"
	SalesOrderLineCacheIDPrefix IDPrefix = "solnca"
	SalesOrderStatusIDPrefix    IDPrefix = "sost"
	SalesOrderTypeIDPrefix      IDPrefix = "soty"

	// Territory-related prefix values
	TerritoryIDPrefix IDPrefix = "te"

	// Target-related prefix values
	TargetIDPrefix IDPrefix = "tg"

	// Scanning Station-related prefix values
	ScanningStationIDPrefix     IDPrefix = "sn"
	ScanningStationTypeIDPrefix IDPrefix = "snty"

	// Session-related prefix values
	SessionIDPrefix IDPrefix = "sess"

	// Shipment-related prefix values
	ShipmentIDPrefix       IDPrefix = "sh"
	ShipmentLineIDPrefix   IDPrefix = "shln"
	ShipmentStatusIDPrefix IDPrefix = "shst"
	ShippingCaseIDPrefix   IDPrefix = "shcs"
	ShippingTermIDPrefix   IDPrefix = "shtm"

	// Unit-related prefix values
	UnitIDPrefix            IDPrefix = "un"
	UnitDimensionIDPrefix   IDPrefix = "undm"
	UnitGroupIDPrefix       IDPrefix = "ungp"
	UnitGroupsUnitsIDPrefix IDPrefix = "ungpun"
	UnitConversionIDPrefix  IDPrefix = "uncv"

	// User-related prefix values
	UserIDPrefix        IDPrefix = "us"
	UserAccountIDPrefix IDPrefix = "usac"
	UserStatusIDPrefix  IDPrefix = "usst"
	UserTypeIDPrefix    IDPrefix = "usty"

	// Verification-related prefix values
	VerificationTokenIDPrefix IDPrefix = "vntk"

	// Visibility-related prefix values
	VisibilityTypeIDPrefix IDPrefix = "vsty"

	// Supplier-related prefix values
	SupplierIDPrefix IDPrefix = "su"

	// Adjustment-related prefix values
	AdjustmentTypeIDPrefix IDPrefix = "ajty"

	// Change Log-related prefix values
	ChangeLogIDPrefix IDPrefix = "chlg"

	// DC Location-related prefix values
	DCLocationIDPrefix IDPrefix = "dclc"

	// Email-related prefix values
	EmailLogIDPrefix       IDPrefix = "emlg"
	EmailRecipientIDPrefix IDPrefix = "emrc"
	OrderEmailIDPrefix     IDPrefix = "orem"

	// Inventory-related prefix values
	InventoryChangeLogIDPrefix IDPrefix = "inchlg"
	InventoryLogIDPrefix       IDPrefix = "inlg"

	// Registration-related prefix values
	RegistrationFlowIDPrefix IDPrefix = "rgfw"

	// Onboarding-related prefix values
	OnboardingStatusIDPrefix IDPrefix = "onst"

	// Settlement-related prefix values
	SettlementIDPrefix IDPrefix = "sl"

	// Stripe-related prefix values
	StripeEventLogIDPrefix IDPrefix = "spevlg"

	// Transaction-related prefix values
	TransactionIDPrefix           IDPrefix = "tx"
	TransactionAllocationIDPrefix IDPrefix = "txal"
	TransactionMethodIDPrefix     IDPrefix = "txmd"
	TransactionTypeIDPrefix       IDPrefix = "txty"

	// EDI-related prefix values
	EDIRunIDPrefix IDPrefix = "edirn"

	// Request-related prefix values
	RequestIDPrefix IDPrefix = "rq"
)
