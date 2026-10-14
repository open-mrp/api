package id

import apikitid "github.com/open-mrp/apikit/id"

// Vocabulary contains all 2-letter word codes used to compose ID prefixes. Each code represents a semantic word/concept.
const (
	// Base entities
	VocAccount      = "ac"
	VocAddress      = "ad"
	VocAction       = "ax"
	VocAgent        = "ag"
	VocAPI          = "ap"
	VocArtifact     = "af"
	VocAttribute    = "at"
	VocAdjustment   = "aj"
	VocAudit        = "au"
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
	VocCalendar     = "cd"
	VocCase         = "cs"
	VocContact      = "ct"
	VocCustomer     = "cu"
	VocClosure      = "cn"
	VocConversion   = "ce"
	VocCategory     = "cg"
	VocDC           = "dc"
	VocDelivery     = "dv"
	VocDimension    = "dm"
	VocDiscount     = "ds"
	VocDefinition   = "df"
	VocDepartment   = "dp"
	VocDocument     = "do"
	VocDomain       = "dn"
	VocDraft        = "dr"
	VocEDI          = "ed"
	VocEmail        = "em"
	VocEnterprise   = "en"
	VocError        = "er"
	VocEstimate     = "es"
	VocEvent        = "ev"
	VocFormula      = "fm"
	VocFreight      = "fr"
	VocFlow         = "fw"
	VocFollowup     = "fu"
	VocGeolocation  = "gl"
	VocGroup        = "gp"
	VocIdempotency  = "ip"
	VocIntegration  = "ig"
	VocImage        = "im"
	VocInventory    = "in"
	VocInquiry      = "ir"
	VocIssue        = "rs" // "rs" derives from "receipt/shipment" — inventory issues are the counterpart to inventory receipts
	VocIntent       = "ie"
	VocItem         = "it"
	VocInvoice      = "iv"
	VocInbox        = "ix"
	VocJob          = "jb"
	VocKey          = "ke"
	VocLabel        = "lb"
	VocLevel        = "lv"
	VocLocation     = "lc"
	VocLog          = "lg"
	VocLine         = "ln"
	VocLink         = "lk"
	VocLot          = "lt"
	VocMachine      = "mc"
	VocMemory       = "mm"
	VocMethod       = "md"
	VocMessage      = "mg"
	VocMaterial     = "ml"
	VocNotification = "nf"
	VocOrganization = "og"
	VocOnboarding   = "ob"
	VocOperating    = "oc"
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
	VocRecord       = "rd"
	VocReview       = "rv"
	VocReply        = "ry"
	VocRoute        = "ru"
	VocSandbox      = "sb"
	VocSession      = "se"
	VocShipment     = "sh"
	VocSettlement   = "sl"
	VocScanning     = "sg"
	VocService      = "si"
	VocStripe       = "sr"
	VocStep         = "st"
	VocStation      = "sn"
	VocStatus       = "ss"
	VocSupplier     = "su"
	VocSupport      = "sp"
	VocSender       = "sx"
	VocSeverity     = "sv"
	VocSystem       = "sy"
	VocSize         = "sz"
	VocTerritory    = "te"
	VocTarget       = "ta"
	VocToken        = "tk"
	VocTool         = "tl"
	VocTerm         = "tm"
	VocTransform    = "tf"
	VocTransit      = "ts"
	VocTier         = "tr"
	VocTransaction  = "tx"
	VocType         = "tp"
	VocUnit         = "un"
	VocUser         = "us"
	VocVisibility   = "vi"
	VocVerification = "ve"
	VocReceiving    = "rc"
	VocConversation = "cv"
	VocSchedule     = "sc"
	VocAttachment   = "ah"
	VocAnnouncement = "an"
	VocBlock        = "bk"
	VocReport       = "ro"
	VocDemand       = "de"
	VocDeviation    = "dw"
	VocDerived      = "dl"
	VocDowntime     = "dt"
	VocOverride     = "ov"
	VocResource     = "rr"
	VocSetting      = "sd"
	VocShift        = "sf"
	VocPolicy       = "pc"
	VocFinished     = "fi"
	VocWork         = "wk"
	VocCenter       = "cc"
	VocOperation    = "oe"
	VocList         = "ls"
	VocRule         = "rw"
	VocCredit       = "cx"
	VocNote         = "nt"
	VocApplication  = "aa"
	VocMovement     = "mv"
	VocDevice       = "dx"
	VocPairing      = "pa"
	VocSubscription = "sk"
	VocVersion      = "vr"
	VocCall         = "ck"
	VocTemplate     = "tt"
)

// IDPrefixes contains all the type ID prefix values used in the system. This serves as a centralized reference for all type ID prefix values. These constants can be used when creating new type ID structs or for validation.
var (
	// Account-related prefix values
	AccountIDPrefix                               = apikitid.ComposePrefix(VocAccount)
	AccountAddressIDPrefix                        = apikitid.ComposePrefix(VocAccount, VocAddress)
	AccountBillingIDPrefix                        = apikitid.ComposePrefix(VocAccount, VocBilling)
	AccountBrandingIDPrefix                       = apikitid.ComposePrefix(VocAccount, VocBranding)
	AccountPortalIDPrefix                         = apikitid.ComposePrefix(VocAccount, VocPortal)
	AccountIntegrationIDPrefix                    = apikitid.ComposePrefix(VocAccount, VocIntegration)
	AccountGroupIDPrefix                          = apikitid.ComposePrefix(VocAccount, VocGroup)
	AccountGroupTypeIDPrefix                      = apikitid.ComposePrefix(VocAccount, VocGroup, VocType)
	AccountTypeIDPrefix                           = apikitid.ComposePrefix(VocAccount, VocType)
	AccountRelationIDPrefix                       = apikitid.ComposePrefix(VocAccount, VocRelation)
	AccountStatusIDPrefix                         = apikitid.ComposePrefix(VocAccount, VocStatus)
	AccountPriceIDPrefix                          = apikitid.ComposePrefix(VocAccount, VocPrice)
	AccountPriceItemCategoryIDPrefix              = apikitid.ComposePrefix(VocAccount, VocPrice, VocItem, VocCategory)
	AccountGroupQuantityDiscountIDPrefix          = apikitid.ComposePrefix(VocAccount, VocGroup, VocQuantity, VocDiscount)
	AccountPriceAttributeIDPrefix                 = apikitid.ComposePrefix(VocAccount, VocPrice, VocAttribute)
	AccountRelationProductLineIDPrefix            = apikitid.ComposePrefix(VocAccount, VocRelation, VocProduct, VocLine)
	AccountRelationPriceGroupIDPrefix             = apikitid.ComposePrefix(VocAccount, VocRelation, VocPrice, VocGroup)
	AccountRelationNotificationPreferenceIDPrefix = apikitid.ComposePrefix(VocAccount, VocRelation, VocNotification, VocPreference)
	AccountRelationNotificationTypeIDPrefix       = apikitid.ComposePrefix(VocAccount, VocRelation, VocNotification, VocType)
	AccountGroupProductLineIDPrefix               = apikitid.ComposePrefix(VocAccount, VocGroup, VocProduct, VocLine)
	AccountPreferenceIDPrefix                     = apikitid.ComposePrefix(VocAccount, VocPreference)
	AccountUserIDPrefix                           = apikitid.ComposePrefix(VocAccount, VocUser)

	// Action-related prefix values
	ActionTypeIDPrefix = apikitid.ComposePrefix(VocAction, VocType)

	// Agent-related prefix values
	AgentDefinitionIDPrefix     = apikitid.ComposePrefix(VocAgent, VocDefinition)
	AgentConfigIDPrefix         = apikitid.ComposePrefix(VocAgent, VocAccount)
	AgentRunIDPrefix            = apikitid.ComposePrefix(VocAgent, VocRun)
	AgentActionIDPrefix         = apikitid.ComposePrefix(VocAgent, VocAction)
	AgentArtifactIDPrefix       = apikitid.ComposePrefix(VocAgent, VocArtifact)
	AgentMemoryIDPrefix         = apikitid.ComposePrefix(VocAgent, VocMemory)
	AgentTokenUsageIDPrefix     = apikitid.ComposePrefix(VocAgent, VocToken)
	TokenPackPurchaseIDPrefix   = apikitid.ComposePrefix(VocToken, VocPayment)
	AgentDefinitionToolIDPrefix = apikitid.ComposePrefix(VocAgent, VocDefinition, VocTool)
	AgentAccountStatusIDPrefix  = apikitid.ComposePrefix(VocAgent, VocAccount, VocStatus)
	AgentRunEventIDPrefix       = apikitid.ComposePrefix(VocAgent, VocRun, VocEvent)
	ToolDefinitionIDPrefix      = apikitid.ComposePrefix(VocTool, VocDefinition)

	// Address-related prefix values
	AddressIDPrefix = apikitid.ComposePrefix(VocAddress)

	// API-related prefix values
	APILocationIDPrefix = apikitid.ComposePrefix(VocAPI, VocLocation)
	APIKeyIDPrefix      = apikitid.ComposePrefix(VocAPI, VocKey)
	DocAPIKeyIDPrefix   = apikitid.ComposePrefix(VocDocument, VocAPI, VocKey)

	// Document-related prefix values
	DocumentSettingIDPrefix = apikitid.ComposePrefix(VocDocument, VocSetting)

	// Attribute-related prefix values
	AttributeIDPrefix = apikitid.ComposePrefix(VocAttribute)

	// Batch-related prefix values
	BatchIDPrefix                = apikitid.ComposePrefix(VocBatch)
	BatchCountAdjustmentIDPrefix = apikitid.ComposePrefix(VocBatch, VocAdjustment)

	// Carrier-related prefix values
	CarrierIDPrefix                = apikitid.ComposePrefix(VocCarrier)
	CarrierTransitEstimateIDPrefix = apikitid.ComposePrefix(VocCarrier, VocTransit, VocEstimate)

	// Operating calendar-related prefix values
	OperatingCalendarIDPrefix        = apikitid.ComposePrefix(VocOperating, VocCalendar)
	OperatingCalendarClosureIDPrefix = apikitid.ComposePrefix(VocOperating, VocCalendar, VocClosure)

	// Service level-related prefix values
	ServiceLevelIDPrefix = apikitid.ComposePrefix(VocService, VocLevel)

	// Color-related prefix values
	ColorIDPrefix = apikitid.ComposePrefix(VocColor)

	// Commission-related prefix values
	CommissionPolicyIDPrefix = apikitid.ComposePrefix(VocCommission, VocStatus)

	// Consumption-related prefix values
	ConsumptionIDPrefix = apikitid.ComposePrefix(VocConsumption)

	// Contact-related prefix values
	ContactIDPrefix     = apikitid.ComposePrefix(VocContact)
	ContactTypeIDPrefix = apikitid.ComposePrefix(VocContact, VocType)

	// Customer-related prefix values
	// !Depreciated
	// CustomerIDPrefix             = apikitid.ComposePrefix(VocCustomer)
	// CustomerGroupIDPrefix        = apikitid.ComposePrefix(VocCustomer, VocGroup)
	// CustomerGroupTypeIDPrefix    = apikitid.ComposePrefix(VocCustomer, VocGroup, VocType)
	// CustomerGroupProductIDPrefix = apikitid.ComposePrefix(VocCustomer, VocGroup, VocProduct)
	// CustomerProductIDPrefix      = apikitid.ComposePrefix(VocCustomer, VocProduct)
	// CustomerStatusIDPrefix       = apikitid.ComposePrefix(VocCustomer, VocStatus)
	// CustomerAddressesIDPrefix    = apikitid.ComposePrefix(VocCustomer, VocAddress)
	// CustomerEmailIDPrefix        = apikitid.ComposePrefix(VocCustomer, VocEmail)
	// CustomerPriceIDPrefix        = apikitid.ComposePrefix(VocCustomer, VocPrice)
	// CustomerPriceGroupsIDPrefix  = apikitid.ComposePrefix(VocCustomer, VocPrice, VocGroup)

	// Delivery-related prefix values
	DeliveryIDPrefix     = apikitid.ComposePrefix(VocDelivery)
	DeliveryLineIDPrefix = apikitid.ComposePrefix(VocDelivery, VocLine)

	// Department-related prefix values
	DepartmentIDPrefix = apikitid.ComposePrefix(VocDepartment)

	// Discount-related prefix values
	DiscountTypeIDPrefix         = apikitid.ComposePrefix(VocDiscount, VocStatus, VocType)
	OrderDiscountIDPrefix        = apikitid.ComposePrefix(VocOrder, VocDiscount)
	QuantityDiscountIDPrefix     = apikitid.ComposePrefix(VocQuantity, VocDiscount)
	QuantityDiscountTierIDPrefix = apikitid.ComposePrefix(VocQuantity, VocDiscount, VocTier)

	// Error-related prefix values
	ErrorSeverityIDPrefix = apikitid.ComposePrefix(VocError, VocSeverity)
	ErrorLogIDPrefix      = apikitid.ComposePrefix(VocError, VocLog)

	// Freight-related prefix values
	FreightPolicyIDPrefix = apikitid.ComposePrefix(VocFreight, VocStatus)

	// Geolocation-related prefix values
	GeolocationIDPrefix = apikitid.ComposePrefix(VocGeolocation)

	// Integration-related prefix values
	IntegrationIDPrefix = apikitid.ComposePrefix(VocIntegration)

	// HubSpot sync (backfill/reconciliation) prefix values
	HubspotSyncJobIDPrefix       = apikitid.ComposePrefix(VocIntegration, VocJob)
	HubspotSyncRecordIDPrefix    = apikitid.ComposePrefix(VocIntegration, VocRecord)
	HubspotCompanyReviewIDPrefix = apikitid.ComposePrefix(VocIntegration, VocReview)

	// Invoice-related prefix values
	InvoiceIDPrefix     = apikitid.ComposePrefix(VocInvoice)
	InvoiceLineIDPrefix = apikitid.ComposePrefix(VocInvoice, VocLine)

	// Item-related prefix values
	ItemIDPrefix             = apikitid.ComposePrefix(VocItem)
	ItemCategoryIDPrefix     = apikitid.ComposePrefix(VocItem, VocCategory)
	ItemCategoryTypeIDPrefix = apikitid.ComposePrefix(VocItem, VocCategory, VocType)
	ItemTypeIDPrefix         = apikitid.ComposePrefix(VocItem, VocType)

	// Label-related prefix values
	LabelSizeIDPrefix = apikitid.ComposePrefix(VocLabel, VocSize)
	LabelTypeIDPrefix = apikitid.ComposePrefix(VocLabel, VocType)

	// Machine-related prefix values
	MachineIDPrefix               = apikitid.ComposePrefix(VocMachine)
	MachineDowntimeEventIDPrefix  = apikitid.ComposePrefix(VocMachine, VocDowntime)
	MachineDowntimeReasonIDPrefix = apikitid.ComposePrefix(VocMachine, VocDowntime, VocType)

	// Demand-override prefix values
	DemandOverrideIDPrefix     = apikitid.ComposePrefix(VocDemand, VocOverride)
	DemandOverrideTypeIDPrefix = apikitid.ComposePrefix(VocDemand, VocOverride, VocType)

	// Material-related prefix values
	MaterialIDPrefix         = apikitid.ComposePrefix(VocMaterial)
	SupplierMaterialIDPrefix = apikitid.ComposePrefix(VocSupplier, VocMaterial)

	// Organization-related prefix values
	OrganizationIDPrefix = apikitid.ComposePrefix(VocOrganization)

	// Part-related prefix values
	PartIDPrefix = apikitid.ComposePrefix(VocPart)

	// Payment-related prefix values
	PaymentTermIDPrefix        = apikitid.ComposePrefix(VocPayment, VocTerm)
	OrderPaymentIntentIDPrefix = apikitid.ComposePrefix(VocOrder, VocPayment, VocIntent)

	// Permission-related prefix values
	PermissionIDPrefix      = apikitid.ComposePrefix(VocPermission)
	PermissionGroupIDPrefix = apikitid.ComposePrefix(VocPermission, VocGroup)

	// Pick-related prefix values
	PickIDPrefix           = apikitid.ComposePrefix(VocPick)
	PickLineIDPrefix       = apikitid.ComposePrefix(VocPick, VocLine)
	PickAssignmentIDPrefix = apikitid.ComposePrefix(VocPick, VocAssignment)

	// Preference-related prefix values
	PreferenceDefinitionIDPrefix = apikitid.ComposePrefix(VocPreference, VocDefinition)

	// Priority-related prefix values
	PriorityIDPrefix = apikitid.ComposePrefix(VocPriority)

	// Product-related prefix values
	ProductIDPrefix           = apikitid.ComposePrefix(VocProduct)
	ProductLineIDPrefix       = apikitid.ComposePrefix(VocProduct, VocLine)
	ProductLineTargetIDPrefix = apikitid.ComposePrefix(VocProduct, VocLine, VocTarget)
	ProductTypeIDPrefix       = apikitid.ComposePrefix(VocProduct, VocType)
	ProductImageIDPrefix      = apikitid.ComposePrefix(VocProduct, VocImage)

	// Production-related prefix values
	ProductionIDPrefix                    = apikitid.ComposePrefix(VocProduction)
	ProductionFormulaIDPrefix             = apikitid.ComposePrefix(VocProduction, VocFormula)
	ProductionFormulaItemIDPrefix         = apikitid.ComposePrefix(VocProduction, VocFormula, VocItem)
	ProductionProcessIDPrefix             = apikitid.ComposePrefix(VocProduction, VocProcess)
	ProductionProcessItemCategoryIDPrefix = apikitid.ComposePrefix(VocProduction, VocProcess, VocItem, VocCategory)
	ProductionRunIDPrefix                 = apikitid.ComposePrefix(VocProduction, VocRun)
	ProductionStepIDPrefix                = apikitid.ComposePrefix(VocProduction, VocStep)
	ProductionStepTransformIDPrefix       = apikitid.ComposePrefix(VocProduction, VocStep, VocTransform)
	ProductionShiftIDPrefix               = apikitid.ComposePrefix(VocProduction, VocShift)

	// Production-schedule prefix values
	ProductionScheduleIDPrefix                = apikitid.ComposePrefix(VocProduction, VocSchedule)
	ProductionScheduleLineIDPrefix            = apikitid.ComposePrefix(VocProduction, VocSchedule, VocLine)
	ProductionScheduleLineOrderIDPrefix       = apikitid.ComposePrefix(VocProduction, VocSchedule, VocLine, VocOrder)
	ProductionScheduleItemPolicyIDPrefix      = apikitid.ComposePrefix(VocProduction, VocSchedule, VocItem, VocPolicy)
	ProductionScheduleFinishedPolicyIDPrefix  = apikitid.ComposePrefix(VocProduction, VocSchedule, VocFinished, VocPolicy)
	ProductionScheduleFinishingLineIDPrefix   = apikitid.ComposePrefix(VocProduction, VocSchedule, VocFinished, VocLine)
	ProductionScheduleDeviationIDPrefix       = apikitid.ComposePrefix(VocProduction, VocSchedule, VocDeviation)
	ScheduleDeviationTypeIDPrefix             = apikitid.ComposePrefix(VocProduction, VocSchedule, VocDeviation, VocType)
	ProductionScheduleDerivedLineIDPrefix     = apikitid.ComposePrefix(VocProduction, VocSchedule, VocDerived)
	AccountProductionScheduleSettingIDPrefix  = apikitid.ComposePrefix(VocAccount, VocProduction, VocSchedule, VocSetting)
	ProductionScheduleResourceSettingIDPrefix = apikitid.ComposePrefix(VocProduction, VocSchedule, VocResource, VocSetting)
	ProductionScheduleItemSettingIDPrefix     = apikitid.ComposePrefix(VocProduction, VocSchedule, VocItem, VocSetting)

	// Property-related prefix values
	PropertyIDPrefix        = apikitid.ComposePrefix(VocProperty)
	SysPropertyIDPrefix     = apikitid.ComposePrefix(VocSystem, VocProperty)
	SysPropertyTypeIDPrefix = apikitid.ComposePrefix(VocSystem, VocProperty, VocType)

	// Quantity-related prefix values
	QuantityIDPrefix = apikitid.ComposePrefix(VocQuantity)

	// Rate-related prefix values
	RateIDPrefix = apikitid.ComposePrefix(VocRate)

	// Role-related prefix values
	RoleIDPrefix           = apikitid.ComposePrefix(VocRole)
	RolePermissionIDPrefix = apikitid.ComposePrefix(VocRole, VocPermission)
	RoleTypeIDPrefix       = apikitid.ComposePrefix(VocRole, VocType)

	// Order-related prefix values
	OrderIDPrefix       = apikitid.ComposePrefix(VocOrder)
	OrderLineIDPrefix   = apikitid.ComposePrefix(VocOrder, VocLine)
	OrderStatusIDPrefix = apikitid.ComposePrefix(VocOrder, VocStatus)
	OrderTypeIDPrefix   = apikitid.ComposePrefix(VocOrder, VocType)

	// Territory-related prefix values
	TerritoryIDPrefix = apikitid.ComposePrefix(VocTerritory)

	// Target-related prefix values
	TargetIDPrefix = apikitid.ComposePrefix(VocTarget)

	// Sandbox-related prefix values
	SandboxAccountIDPrefix = apikitid.ComposePrefix(VocSandbox, VocAccount)

	// Scanning Station-related prefix values
	ScanningStationIDPrefix     = apikitid.ComposePrefix(VocScanning, VocStation)
	ScanningStationTypeIDPrefix = apikitid.ComposePrefix(VocScanning, VocStation, VocType)

	// Session-related prefix values
	SessionIDPrefix = apikitid.ComposePrefix(VocSession)

	// Shipment-related prefix values
	ShipmentIDPrefix         = apikitid.ComposePrefix(VocShipment)
	ShipmentLineIDPrefix     = apikitid.ComposePrefix(VocShipment, VocLine)
	ShipmentStatusIDPrefix   = apikitid.ComposePrefix(VocShipment, VocStatus)
	ShippingCaseIDPrefix     = apikitid.ComposePrefix(VocShipment, VocCase)
	ShippingTermIDPrefix     = apikitid.ComposePrefix(VocShipment, VocTerm)
	FreeShippingRuleIDPrefix = apikitid.ComposePrefix(VocFreight, VocShipment)

	// Unit-related prefix values
	UnitIDPrefix            = apikitid.ComposePrefix(VocUnit)
	UnitDimensionIDPrefix   = apikitid.ComposePrefix(VocUnit, VocDimension)
	UnitGroupIDPrefix       = apikitid.ComposePrefix(VocUnit, VocGroup)
	UnitGroupsUnitsIDPrefix = apikitid.ComposePrefix(VocUnit, VocGroup, VocUnit)
	UnitConversionIDPrefix  = apikitid.ComposePrefix(VocUnit, VocConversion)

	// User-related prefix values
	UserIDPrefix        = apikitid.ComposePrefix(VocUser)
	UserAccountIDPrefix = apikitid.ComposePrefix(VocUser, VocAccount)
	UserStatusIDPrefix  = apikitid.ComposePrefix(VocUser, VocStatus)
	UserTypeIDPrefix    = apikitid.ComposePrefix(VocUser, VocType)

	// Verification-related prefix values
	VerificationTokenIDPrefix = apikitid.ComposePrefix(VocVerification, VocToken)

	// Visibility-related prefix values
	VisibilityTypeIDPrefix = apikitid.ComposePrefix(VocVisibility, VocType)

	// Adjustment-related prefix values
	AdjustmentTypeIDPrefix = apikitid.ComposePrefix(VocAdjustment, VocType)

	// Change Log-related prefix values
	ChangeLogIDPrefix = apikitid.ComposePrefix(VocChange, VocLog)

	// Location-related prefix values
	LocationIDPrefix = apikitid.ComposePrefix(VocLocation)

	// Email-related prefix values
	EmailLogIDPrefix       = apikitid.ComposePrefix(VocEmail, VocLog)
	EmailRecipientIDPrefix = apikitid.ComposePrefix(VocEmail, VocRecipient)

	// AccountFollowupIDPrefix identifies the personal follow-up a new registrant receives after signing up.
	AccountFollowupIDPrefix = apikitid.ComposePrefix(VocAccount, VocFollowup)
	OrderEmailIDPrefix      = apikitid.ComposePrefix(VocOrder, VocEmail)
	EmailDomainIDPrefix     = apikitid.ComposePrefix(VocEmail, VocDomain)
	EmailInboxIDPrefix      = apikitid.ComposePrefix(VocEmail, VocInbox)
	EmailMessageIDPrefix    = apikitid.ComposePrefix(VocEmail, VocMessage)
	EmailSenderIDPrefix     = apikitid.ComposePrefix(VocEmail, VocSender)

	// Inventory-related prefix values
	InventoryChangeLogIDPrefix  = apikitid.ComposePrefix(VocInventory, VocChange, VocLog)
	InventoryLogIDPrefix        = apikitid.ComposePrefix(VocInventory, VocLog)
	InventoryReceiptIDPrefix    = apikitid.ComposePrefix(VocInventory, VocRecipient)
	InventoryIssueIDPrefix      = apikitid.ComposePrefix(VocInventory, VocIssue)
	InventoryAllocationIDPrefix = apikitid.ComposePrefix(VocInventory, VocAllocation)

	// Portal-related prefix values
	PortalDomainIDPrefix = apikitid.ComposePrefix(VocPortal, VocDomain)

	// Registration-related prefix values
	RegistrationFlowIDPrefix = apikitid.ComposePrefix(VocRegistration, VocFlow)

	PortalRegistrationSessionIDPrefix = apikitid.ComposePrefix(VocPortal, VocRegistration, VocSession)

	// Onboarding-related prefix values
	OnboardingStatusIDPrefix = apikitid.ComposePrefix(VocOnboarding, VocStatus)

	// Settlement-related prefix values
	SettlementIDPrefix = apikitid.ComposePrefix(VocSettlement)

	// Stripe-related prefix values
	StripeEventLogIDPrefix = apikitid.ComposePrefix(VocStripe, VocEvent, VocLog)

	// Transaction-related prefix values
	TransactionIDPrefix           = apikitid.ComposePrefix(VocTransaction)
	TransactionAllocationIDPrefix = apikitid.ComposePrefix(VocTransaction, VocAllocation)
	TransactionMethodIDPrefix     = apikitid.ComposePrefix(VocTransaction, VocMethod)
	TransactionTypeIDPrefix       = apikitid.ComposePrefix(VocTransaction, VocType)

	// Request-related prefix values
	RequestIDPrefix = apikitid.ComposePrefix(VocRequest)

	// Audit-related prefix values
	AuditEventIDPrefix = apikitid.ComposePrefix(VocAudit, VocEvent)

	// Plan-related prefix values
	PricingPlanIDPrefix = apikitid.ComposePrefix(VocPlan)
	PlanChangeIDPrefix  = apikitid.ComposePrefix(VocPlan, VocChange)

	// Enterprise Inquiry-related prefix values
	EnterpriseInquiryIDPrefix = apikitid.ComposePrefix(VocEnterprise, VocInquiry)

	// Idempotency-related prefix values
	IdempotencyKeyIDPrefix        = apikitid.ComposePrefix(VocIdempotency, VocKey)
	ServiceIdempotencyKeyIDPrefix = apikitid.ComposePrefix(VocService, VocIdempotency, VocKey)

	// Lot-related prefix values
	LotIDPrefix = apikitid.ComposePrefix(VocLot)

	// Receiving-related prefix values
	ReceivingOrderIDPrefix     = apikitid.ComposePrefix(VocReceiving, VocOrder)
	ReceivingOrderLineIDPrefix = apikitid.ComposePrefix(VocReceiving, VocOrder, VocLine)

	// Messaging-related prefix values
	MessageIDPrefix                 = apikitid.ComposePrefix(VocMessage)
	AsyncJobIDPrefix                = apikitid.ComposePrefix(VocJob)
	ConversationIDPrefix            = apikitid.ComposePrefix(VocConversation)
	ConversationParticipantIDPrefix = apikitid.ComposePrefix(VocConversation, VocPart)
	MessagingGroupIDPrefix          = apikitid.ComposePrefix(VocConversation, VocGroup)
	MessagingGroupMemberIDPrefix    = apikitid.ComposePrefix(VocConversation, VocGroup, VocPart)
	MessageAttachmentIDPrefix       = apikitid.ComposePrefix(VocMessage, VocAttachment)
	MessageReceiptIDPrefix          = apikitid.ComposePrefix(VocMessage, VocReceiving)
	MessageBlockIDPrefix            = apikitid.ComposePrefix(VocMessage, VocBlock)
	MessageReportIDPrefix           = apikitid.ComposePrefix(VocMessage, VocReport)
	NotificationIDPrefix            = apikitid.ComposePrefix(VocNotification)
	NotificationPreferenceIDPrefix  = apikitid.ComposePrefix(VocNotification, VocPreference)
	ScheduledMessageIDPrefix        = apikitid.ComposePrefix(VocSchedule, VocMessage)
	AnnouncementIDPrefix            = apikitid.ComposePrefix(VocAnnouncement)
	AnnouncementReceiptIDPrefix     = apikitid.ComposePrefix(VocAnnouncement, VocReceiving)
	SupportRouteIDPrefix            = apikitid.ComposePrefix(VocSupport, VocRoute)
	ReplyDraftIDPrefix              = apikitid.ComposePrefix(VocReply, VocDraft)
	ConversationLinkIDPrefix        = apikitid.ComposePrefix(VocConversation, VocLink)

	// forge.1 resources, registered up front so the units that build them never edit this list. A resource renamed from an older table (work center from department, supplier item from supplier material, device from scanning station) keeps its rows' existing IDs; only new rows take the new prefix. Bills and supplier credits are invoices and credit notes read by the other party, so they share those IDs.
	JobIDPrefix                 = apikitid.ComposePrefix(VocProduction, VocJob)
	WorkOrderIDPrefix           = apikitid.ComposePrefix(VocWork, VocOrder)
	ManufacturingMethodIDPrefix = apikitid.ComposePrefix(VocProduction, VocMethod)
	MethodOperationIDPrefix     = apikitid.ComposePrefix(VocMethod, VocOperation)
	MethodMaterialIDPrefix      = apikitid.ComposePrefix(VocMethod, VocMaterial)
	PriceListIDPrefix           = apikitid.ComposePrefix(VocPrice, VocList)
	PriceListRuleIDPrefix       = apikitid.ComposePrefix(VocPrice, VocList, VocRule)
	CreditNoteIDPrefix          = apikitid.ComposePrefix(VocCredit, VocNote)
	PaymentIDPrefix             = apikitid.ComposePrefix(VocPayment)
	ApplicationIDPrefix         = apikitid.ComposePrefix(VocApplication)
	SupplierItemIDPrefix        = apikitid.ComposePrefix(VocSupplier, VocItem)
	InventoryMovementIDPrefix   = apikitid.ComposePrefix(VocInventory, VocMovement)
	LotSizeDefaultIDPrefix      = apikitid.ComposePrefix(VocLot, VocSize)
	BatchScanIDPrefix           = apikitid.ComposePrefix(VocBatch, VocScanning)
	WorkCenterIDPrefix          = apikitid.ComposePrefix(VocWork, VocCenter)
	DeviceIDPrefix              = apikitid.ComposePrefix(VocDevice)
	DevicePairingIDPrefix       = apikitid.ComposePrefix(VocDevice, VocPairing)
	NoteIDPrefix                = apikitid.ComposePrefix(VocNote)
	SubscriptionIDPrefix        = apikitid.ComposePrefix(VocSubscription)
	AgentIDPrefix               = apikitid.ComposePrefix(VocAgent)
	AgentVersionIDPrefix        = apikitid.ComposePrefix(VocAgent, VocVersion)
	ToolIDPrefix                = apikitid.ComposePrefix(VocTool)
	ToolCallIDPrefix            = apikitid.ComposePrefix(VocTool, VocCall)
	AgentSignUpIDPrefix         = apikitid.ComposePrefix(VocAgent, VocRegistration)
	RoleTemplateIDPrefix        = apikitid.ComposePrefix(VocRole, VocTemplate)
	DemandAdjustmentIDPrefix    = apikitid.ComposePrefix(VocDemand, VocAdjustment)
)
