package id

import "strings"

// IDPrefix is a 2-letter prefix for a type-specific ID. This will be used to compose the prefix for the ID. Since each word has a unique 2 letter vocabulary, we can ensure the type ID provides a machine-readable and human-readable identifier.
type IDPrefix string

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
	VocCase         = "cs"
	VocContact      = "ct"
	VocCustomer     = "cu"
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
	VocSeverity     = "sv"
	VocSystem       = "sy"
	VocSize         = "sz"
	VocTerritory    = "te"
	VocTarget       = "ta"
	VocToken        = "tk"
	VocTool         = "tl"
	VocTerm         = "tm"
	VocTransform    = "tf"
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
)

// composePrefix concatenates vocabulary words to form a prefix. This ensures consistency and makes prefix construction explicit.
func composePrefix(words ...string) IDPrefix {
	return IDPrefix(strings.Join(words, ""))
}

// IDPrefixes contains all the type ID prefix values used in the system. This serves as a centralized reference for all type ID prefix values. These constants can be used when creating new type ID structs or for validation.
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

	// Agent-related prefix values
	AgentDefinitionIDPrefix     = composePrefix(VocAgent, VocDefinition)
	AgentConfigIDPrefix         = composePrefix(VocAgent, VocAccount)
	AgentRunIDPrefix            = composePrefix(VocAgent, VocRun)
	AgentActionIDPrefix         = composePrefix(VocAgent, VocAction)
	AgentArtifactIDPrefix       = composePrefix(VocAgent, VocArtifact)
	AgentMemoryIDPrefix         = composePrefix(VocAgent, VocMemory)
	AgentTokenUsageIDPrefix     = composePrefix(VocAgent, VocToken)
	TokenPackPurchaseIDPrefix   = composePrefix(VocToken, VocPayment)
	AgentDefinitionToolIDPrefix = composePrefix(VocAgent, VocDefinition, VocTool)
	AgentAccountStatusIDPrefix  = composePrefix(VocAgent, VocAccount, VocStatus)
	AgentRunEventIDPrefix       = composePrefix(VocAgent, VocRun, VocEvent)
	ToolDefinitionIDPrefix      = composePrefix(VocTool, VocDefinition)

	// Address-related prefix values
	AddressIDPrefix = composePrefix(VocAddress)

	// API-related prefix values
	APILocationIDPrefix = composePrefix(VocAPI, VocLocation)
	APIKeyIDPrefix      = composePrefix(VocAPI, VocKey)
	DocAPIKeyIDPrefix   = composePrefix(VocDocument, VocAPI, VocKey)

	// Attribute-related prefix values
	AttributeIDPrefix = composePrefix(VocAttribute)

	// Batch-related prefix values
	BatchIDPrefix = composePrefix(VocBatch)

	// Carrier-related prefix values
	CarrierIDPrefix = composePrefix(VocCarrier)

	// Service level-related prefix values
	ServiceLevelIDPrefix = composePrefix(VocService, VocLevel)

	// Color-related prefix values
	ColorIDPrefix = composePrefix(VocColor)

	// Commission-related prefix values
	CommissionPolicyIDPrefix = composePrefix(VocCommission, VocStatus)

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

	// Delivery-related prefix values
	DeliveryIDPrefix     = composePrefix(VocDelivery)
	DeliveryLineIDPrefix = composePrefix(VocDelivery, VocLine)

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
	FreightPolicyIDPrefix = composePrefix(VocFreight, VocStatus)

	// Geolocation-related prefix values
	GeolocationIDPrefix = composePrefix(VocGeolocation)

	// Integration-related prefix values
	IntegrationIDPrefix = composePrefix(VocIntegration)

	// HubSpot sync (backfill/reconciliation) prefix values
	HubspotSyncJobIDPrefix       = composePrefix(VocIntegration, VocJob)
	HubspotSyncRecordIDPrefix    = composePrefix(VocIntegration, VocRecord)
	HubspotCompanyReviewIDPrefix = composePrefix(VocIntegration, VocReview)

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
	MachineIDPrefix               = composePrefix(VocMachine)
	MachineDowntimeEventIDPrefix  = composePrefix(VocMachine, VocDowntime)
	MachineDowntimeReasonIDPrefix = composePrefix(VocMachine, VocDowntime, VocType)

	// Demand-override prefix values
	DemandOverrideIDPrefix     = composePrefix(VocDemand, VocOverride)
	DemandOverrideTypeIDPrefix = composePrefix(VocDemand, VocOverride, VocType)

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
	ProductionShiftIDPrefix               = composePrefix(VocProduction, VocShift)

	// Production-schedule prefix values
	ProductionScheduleIDPrefix                = composePrefix(VocProduction, VocSchedule)
	ProductionScheduleLineIDPrefix            = composePrefix(VocProduction, VocSchedule, VocLine)
	ProductionScheduleItemPolicyIDPrefix      = composePrefix(VocProduction, VocSchedule, VocItem, VocPolicy)
	ProductionScheduleFinishedPolicyIDPrefix  = composePrefix(VocProduction, VocSchedule, VocFinished, VocPolicy)
	ProductionScheduleDeviationIDPrefix       = composePrefix(VocProduction, VocSchedule, VocDeviation)
	ScheduleDeviationTypeIDPrefix             = composePrefix(VocProduction, VocSchedule, VocDeviation, VocType)
	ProductionScheduleDerivedLineIDPrefix     = composePrefix(VocProduction, VocSchedule, VocDerived)
	AccountProductionScheduleSettingIDPrefix  = composePrefix(VocAccount, VocProduction, VocSchedule, VocSetting)
	ProductionScheduleResourceSettingIDPrefix = composePrefix(VocProduction, VocSchedule, VocResource, VocSetting)
	ProductionScheduleItemSettingIDPrefix     = composePrefix(VocProduction, VocSchedule, VocItem, VocSetting)

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
	SessionIDPrefix = composePrefix(VocSession)

	// Shipment-related prefix values
	ShipmentIDPrefix         = composePrefix(VocShipment)
	ShipmentLineIDPrefix     = composePrefix(VocShipment, VocLine)
	ShipmentStatusIDPrefix   = composePrefix(VocShipment, VocStatus)
	ShippingCaseIDPrefix     = composePrefix(VocShipment, VocCase)
	ShippingTermIDPrefix     = composePrefix(VocShipment, VocTerm)
	FreeShippingRuleIDPrefix = composePrefix(VocFreight, VocShipment)

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

	// Location-related prefix values
	LocationIDPrefix = composePrefix(VocLocation)

	// Email-related prefix values
	EmailLogIDPrefix       = composePrefix(VocEmail, VocLog)
	EmailRecipientIDPrefix = composePrefix(VocEmail, VocRecipient)
	OrderEmailIDPrefix     = composePrefix(VocOrder, VocEmail)
	EmailDomainIDPrefix    = composePrefix(VocEmail, VocDomain)
	EmailInboxIDPrefix     = composePrefix(VocEmail, VocInbox)
	EmailMessageIDPrefix   = composePrefix(VocEmail, VocMessage)

	// Inventory-related prefix values
	InventoryChangeLogIDPrefix  = composePrefix(VocInventory, VocChange, VocLog)
	InventoryLogIDPrefix        = composePrefix(VocInventory, VocLog)
	InventoryReceiptIDPrefix    = composePrefix(VocInventory, VocRecipient)
	InventoryIssueIDPrefix      = composePrefix(VocInventory, VocIssue)
	InventoryAllocationIDPrefix = composePrefix(VocInventory, VocAllocation)

	// Portal-related prefix values
	PortalDomainIDPrefix = composePrefix(VocPortal, VocDomain)

	// Registration-related prefix values
	RegistrationFlowIDPrefix = composePrefix(VocRegistration, VocFlow)

	PortalRegistrationSessionIDPrefix = composePrefix(VocPortal, VocRegistration, VocSession)

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

	// Audit-related prefix values
	AuditEventIDPrefix = composePrefix(VocAudit, VocEvent)

	// Plan-related prefix values
	PricingPlanIDPrefix = composePrefix(VocPlan)
	PlanChangeIDPrefix  = composePrefix(VocPlan, VocChange)

	// Enterprise Inquiry-related prefix values
	EnterpriseInquiryIDPrefix = composePrefix(VocEnterprise, VocInquiry)

	// Idempotency-related prefix values
	IdempotencyKeyIDPrefix        = composePrefix(VocIdempotency, VocKey)
	ServiceIdempotencyKeyIDPrefix = composePrefix(VocService, VocIdempotency, VocKey)

	// Lot-related prefix values
	LotIDPrefix = composePrefix(VocLot)

	// Receiving-related prefix values
	ReceivingOrderIDPrefix     = composePrefix(VocReceiving, VocOrder)
	ReceivingOrderLineIDPrefix = composePrefix(VocReceiving, VocOrder, VocLine)

	// Messaging-related prefix values
	MessageIDPrefix                 = composePrefix(VocMessage)
	JobIDPrefix                     = composePrefix(VocJob)
	ConversationIDPrefix            = composePrefix(VocConversation)
	ConversationParticipantIDPrefix = composePrefix(VocConversation, VocPart)
	MessagingGroupIDPrefix          = composePrefix(VocConversation, VocGroup)
	MessagingGroupMemberIDPrefix    = composePrefix(VocConversation, VocGroup, VocPart)
	MessageAttachmentIDPrefix       = composePrefix(VocMessage, VocAttachment)
	MessageReceiptIDPrefix          = composePrefix(VocMessage, VocReceiving)
	MessageBlockIDPrefix            = composePrefix(VocMessage, VocBlock)
	MessageReportIDPrefix           = composePrefix(VocMessage, VocReport)
	NotificationIDPrefix            = composePrefix(VocNotification)
	NotificationPreferenceIDPrefix  = composePrefix(VocNotification, VocPreference)
	ScheduledMessageIDPrefix        = composePrefix(VocSchedule, VocMessage)
	AnnouncementIDPrefix            = composePrefix(VocAnnouncement)
	AnnouncementReceiptIDPrefix     = composePrefix(VocAnnouncement, VocReceiving)
	SupportRouteIDPrefix            = composePrefix(VocSupport, VocRoute)
	ReplyDraftIDPrefix              = composePrefix(VocReply, VocDraft)
	ConversationLinkIDPrefix        = composePrefix(VocConversation, VocLink)
)
