package id

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// --- GenID tests ---

func TestGenID_DefaultLength(t *testing.T) {
	t.Parallel()
	id, apiErr := GenID(UserIDPrefix, nil)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}

	prefix := string(UserIDPrefix) + "_"
	if !strings.HasPrefix(id, prefix) {
		t.Errorf("expected prefix %q, got %q", prefix, id)
	}

	// Default length is 12, so total = len(prefix) + 12
	nanoIDPart := strings.TrimPrefix(id, prefix)
	if len(nanoIDPart) != int(IDLength12) {
		t.Errorf("expected nano ID length %d, got %d (%q)", IDLength12, len(nanoIDPart), nanoIDPart)
	}
}

func TestGenID_CustomLengths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		length IDLength
	}{
		{"IDLength12", IDLength12},
		{"IDLength19", IDLength19},
		{"IDLength22", IDLength22},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length := tt.length
			id, apiErr := GenID(AccountIDPrefix, &length)
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}

			prefix := string(AccountIDPrefix) + "_"
			nanoIDPart := strings.TrimPrefix(id, prefix)
			if len(nanoIDPart) != int(tt.length) {
				t.Errorf("expected nano ID length %d, got %d (%q)", tt.length, len(nanoIDPart), nanoIDPart)
			}
		})
	}
}

func TestGenID_UsesCorrectCharset(t *testing.T) {
	t.Parallel()
	for range 50 {
		id, apiErr := GenID(UserIDPrefix, nil)
		if apiErr != nil {
			t.Fatalf("unexpected error: %v", apiErr)
		}

		nanoIDPart := strings.TrimPrefix(id, string(UserIDPrefix)+"_")
		for _, c := range nanoIDPart {
			if !strings.ContainsRune(charset, c) {
				t.Errorf("character %c not in charset %q (id: %q)", c, charset, id)
			}
		}
	}
}

func TestGenID_Uniqueness(t *testing.T) {
	t.Parallel()
	const count = 1000
	seen := make(map[string]struct{}, count)

	for range count {
		id, apiErr := GenID(OrderIDPrefix, nil)
		if apiErr != nil {
			t.Fatalf("unexpected error: %v", apiErr)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestGenID_DifferentPrefixes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		prefix   IDPrefix
		expected string
	}{
		{UserIDPrefix, "us_"},
		{AccountIDPrefix, "ac_"},
		{OrderIDPrefix, "or_"},
		{OrganizationIDPrefix, "og_"},
		{APIKeyIDPrefix, "apke_"},
		{ProductionFormulaItemIDPrefix, "pnfmit_"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			id, apiErr := GenID(tt.prefix, nil)
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}
			if !strings.HasPrefix(id, tt.expected) {
				t.Errorf("expected prefix %q, got %q", tt.expected, id)
			}
		})
	}
}

// --- composePrefix tests ---

func TestComposePrefix_SingleWord(t *testing.T) {
	t.Parallel()
	prefix := composePrefix(VocUser)
	if prefix != "us" {
		t.Errorf("expected %q, got %q", "us", prefix)
	}
}

func TestComposePrefix_MultipleWords(t *testing.T) {
	t.Parallel()
	tests := []struct {
		words    []string
		expected IDPrefix
	}{
		{[]string{VocAccount, VocAddress}, "acad"},
		{[]string{VocAPI, VocKey}, "apke"},
		{[]string{VocProduction, VocFormula, VocItem}, "pnfmit"},
		{[]string{VocUnit, VocGroup, VocUnit}, "ungpun"},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			result := composePrefix(tt.words...)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// --- genNanoID tests ---

func TestGenNanoID_Length(t *testing.T) {
	t.Parallel()
	for _, length := range []IDLength{IDLength12, IDLength19, IDLength22} {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			id, apiErr := genNanoID(length)
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}
			if len(id) != int(length) {
				t.Errorf("expected length %d, got %d (%q)", length, len(id), id)
			}
		})
	}
}

func TestGenNanoID_Charset(t *testing.T) {
	t.Parallel()
	for range 50 {
		id, apiErr := genNanoID(IDLength22)
		if apiErr != nil {
			t.Fatalf("unexpected error: %v", apiErr)
		}
		for _, c := range id {
			if !strings.ContainsRune(charset, c) {
				t.Errorf("character %c not in charset %q (id: %q)", c, charset, id)
			}
		}
	}
}

// --- IDPrefix uniqueness test ---

func TestIDPrefixes_NoDuplicates(t *testing.T) {
	t.Parallel(
	// All prefix values that are actively used (non-deprecated).
	)

	prefixes := []struct {
		name  string
		value IDPrefix
	}{
		{"AccountIDPrefix", AccountIDPrefix},
		{"AccountAddressIDPrefix", AccountAddressIDPrefix},
		{"AccountBillingIDPrefix", AccountBillingIDPrefix},
		{"AccountBrandingIDPrefix", AccountBrandingIDPrefix},
		{"AccountPortalIDPrefix", AccountPortalIDPrefix},
		{"AccountIntegrationIDPrefix", AccountIntegrationIDPrefix},
		{"HubspotSyncJobIDPrefix", HubspotSyncJobIDPrefix},
		{"HubspotSyncRecordIDPrefix", HubspotSyncRecordIDPrefix},
		{"HubspotCompanyReviewIDPrefix", HubspotCompanyReviewIDPrefix},
		{"AccountGroupIDPrefix", AccountGroupIDPrefix},
		{"AccountGroupTypeIDPrefix", AccountGroupTypeIDPrefix},
		{"AccountTypeIDPrefix", AccountTypeIDPrefix},
		{"AccountRelationIDPrefix", AccountRelationIDPrefix},
		{"AccountStatusIDPrefix", AccountStatusIDPrefix},
		{"AccountPriceIDPrefix", AccountPriceIDPrefix},
		{"AccountPriceItemCategoryIDPrefix", AccountPriceItemCategoryIDPrefix},
		{"AccountGroupQuantityDiscountIDPrefix", AccountGroupQuantityDiscountIDPrefix},
		{"AccountPriceAttributeIDPrefix", AccountPriceAttributeIDPrefix},
		{"AccountRelationProductLineIDPrefix", AccountRelationProductLineIDPrefix},
		{"AccountRelationPriceGroupIDPrefix", AccountRelationPriceGroupIDPrefix},
		{"AccountRelationNotificationPreferenceIDPrefix", AccountRelationNotificationPreferenceIDPrefix},
		{"AccountRelationNotificationTypeIDPrefix", AccountRelationNotificationTypeIDPrefix},
		{"AccountGroupProductLineIDPrefix", AccountGroupProductLineIDPrefix},
		{"AccountPreferenceIDPrefix", AccountPreferenceIDPrefix},
		{"AccountUserIDPrefix", AccountUserIDPrefix},
		{"ActionTypeIDPrefix", ActionTypeIDPrefix},
		{"AgentDefinitionIDPrefix", AgentDefinitionIDPrefix},
		{"AgentConfigIDPrefix", AgentConfigIDPrefix},
		{"AgentRunIDPrefix", AgentRunIDPrefix},
		{"AgentActionIDPrefix", AgentActionIDPrefix},
		{"AgentArtifactIDPrefix", AgentArtifactIDPrefix},
		{"AgentMemoryIDPrefix", AgentMemoryIDPrefix},
		{"AgentTokenUsageIDPrefix", AgentTokenUsageIDPrefix},
		{"TokenPackPurchaseIDPrefix", TokenPackPurchaseIDPrefix},
		{"AgentDefinitionToolIDPrefix", AgentDefinitionToolIDPrefix},
		{"AgentAccountStatusIDPrefix", AgentAccountStatusIDPrefix},
		{"AgentRunEventIDPrefix", AgentRunEventIDPrefix},
		{"ToolDefinitionIDPrefix", ToolDefinitionIDPrefix},
		{"AddressIDPrefix", AddressIDPrefix},
		{"APILocationIDPrefix", APILocationIDPrefix},
		{"APIKeyIDPrefix", APIKeyIDPrefix},
		{"DocAPIKeyIDPrefix", DocAPIKeyIDPrefix},
		{"AttributeIDPrefix", AttributeIDPrefix},
		{"BatchIDPrefix", BatchIDPrefix},
		{"CarrierIDPrefix", CarrierIDPrefix},
		{"CarrierTransitEstimateIDPrefix", CarrierTransitEstimateIDPrefix},
		{"OperatingCalendarIDPrefix", OperatingCalendarIDPrefix},
		{"OperatingCalendarClosureIDPrefix", OperatingCalendarClosureIDPrefix},
		{"ServiceLevelIDPrefix", ServiceLevelIDPrefix},
		{"ColorIDPrefix", ColorIDPrefix},
		{"CommissionPolicyIDPrefix", CommissionPolicyIDPrefix},
		{"ConsumptionIDPrefix", ConsumptionIDPrefix},
		{"ContactIDPrefix", ContactIDPrefix},
		{"ContactTypeIDPrefix", ContactTypeIDPrefix},
		{"DeliveryIDPrefix", DeliveryIDPrefix},
		{"DeliveryLineIDPrefix", DeliveryLineIDPrefix},
		{"DepartmentIDPrefix", DepartmentIDPrefix},
		{"DiscountTypeIDPrefix", DiscountTypeIDPrefix},
		{"OrderDiscountIDPrefix", OrderDiscountIDPrefix},
		{"QuantityDiscountIDPrefix", QuantityDiscountIDPrefix},
		{"QuantityDiscountTierIDPrefix", QuantityDiscountTierIDPrefix},
		{"ErrorSeverityIDPrefix", ErrorSeverityIDPrefix},
		{"ErrorLogIDPrefix", ErrorLogIDPrefix},
		{"FreightPolicyIDPrefix", FreightPolicyIDPrefix},
		{"GeolocationIDPrefix", GeolocationIDPrefix},
		{"IntegrationIDPrefix", IntegrationIDPrefix},
		{"InvoiceIDPrefix", InvoiceIDPrefix},
		{"InvoiceLineIDPrefix", InvoiceLineIDPrefix},
		{"ItemIDPrefix", ItemIDPrefix},
		{"ItemCategoryIDPrefix", ItemCategoryIDPrefix},
		{"ItemCategoryTypeIDPrefix", ItemCategoryTypeIDPrefix},
		{"ItemTypeIDPrefix", ItemTypeIDPrefix},
		{"LabelSizeIDPrefix", LabelSizeIDPrefix},
		{"LabelTypeIDPrefix", LabelTypeIDPrefix},
		{"MachineIDPrefix", MachineIDPrefix},
		{"MaterialIDPrefix", MaterialIDPrefix},
		{"SupplierMaterialIDPrefix", SupplierMaterialIDPrefix},
		{"OrganizationIDPrefix", OrganizationIDPrefix},
		{"PartIDPrefix", PartIDPrefix},
		{"PaymentTermIDPrefix", PaymentTermIDPrefix},
		{"OrderPaymentIntentIDPrefix", OrderPaymentIntentIDPrefix},
		{"PermissionIDPrefix", PermissionIDPrefix},
		{"PermissionGroupIDPrefix", PermissionGroupIDPrefix},
		{"PickIDPrefix", PickIDPrefix},
		{"PickLineIDPrefix", PickLineIDPrefix},
		{"PickAssignmentIDPrefix", PickAssignmentIDPrefix},
		{"PreferenceDefinitionIDPrefix", PreferenceDefinitionIDPrefix},
		{"PriorityIDPrefix", PriorityIDPrefix},
		{"ProductIDPrefix", ProductIDPrefix},
		{"ProductLineIDPrefix", ProductLineIDPrefix},
		{"ProductLineTargetIDPrefix", ProductLineTargetIDPrefix},
		{"ProductTypeIDPrefix", ProductTypeIDPrefix},
		{"ProductImageIDPrefix", ProductImageIDPrefix},
		{"ProductionIDPrefix", ProductionIDPrefix},
		{"ProductionFormulaIDPrefix", ProductionFormulaIDPrefix},
		{"ProductionFormulaItemIDPrefix", ProductionFormulaItemIDPrefix},
		{"ProductionProcessIDPrefix", ProductionProcessIDPrefix},
		{"ProductionProcessItemCategoryIDPrefix", ProductionProcessItemCategoryIDPrefix},
		{"ProductionRunIDPrefix", ProductionRunIDPrefix},
		{"ProductionStepIDPrefix", ProductionStepIDPrefix},
		{"ProductionStepTransformIDPrefix", ProductionStepTransformIDPrefix},
		{"PropertyIDPrefix", PropertyIDPrefix},
		{"SysPropertyIDPrefix", SysPropertyIDPrefix},
		{"SysPropertyTypeIDPrefix", SysPropertyTypeIDPrefix},
		{"QuantityIDPrefix", QuantityIDPrefix},
		{"RateIDPrefix", RateIDPrefix},
		{"RoleIDPrefix", RoleIDPrefix},
		{"RolePermissionIDPrefix", RolePermissionIDPrefix},
		{"RoleTypeIDPrefix", RoleTypeIDPrefix},
		{"OrderIDPrefix", OrderIDPrefix},
		{"OrderLineIDPrefix", OrderLineIDPrefix},
		{"OrderStatusIDPrefix", OrderStatusIDPrefix},
		{"OrderTypeIDPrefix", OrderTypeIDPrefix},
		{"TerritoryIDPrefix", TerritoryIDPrefix},
		{"TargetIDPrefix", TargetIDPrefix},
		{"SandboxAccountIDPrefix", SandboxAccountIDPrefix},
		{"ScanningStationIDPrefix", ScanningStationIDPrefix},
		{"ScanningStationTypeIDPrefix", ScanningStationTypeIDPrefix},
		{"SessionIDPrefix", SessionIDPrefix},
		{"ShipmentIDPrefix", ShipmentIDPrefix},
		{"ShipmentLineIDPrefix", ShipmentLineIDPrefix},
		{"ShipmentStatusIDPrefix", ShipmentStatusIDPrefix},
		{"ShippingCaseIDPrefix", ShippingCaseIDPrefix},
		{"ShippingTermIDPrefix", ShippingTermIDPrefix},
		{"UnitIDPrefix", UnitIDPrefix},
		{"UnitDimensionIDPrefix", UnitDimensionIDPrefix},
		{"UnitGroupIDPrefix", UnitGroupIDPrefix},
		{"UnitGroupsUnitsIDPrefix", UnitGroupsUnitsIDPrefix},
		{"UnitConversionIDPrefix", UnitConversionIDPrefix},
		{"UserIDPrefix", UserIDPrefix},
		{"UserAccountIDPrefix", UserAccountIDPrefix},
		{"UserStatusIDPrefix", UserStatusIDPrefix},
		{"UserTypeIDPrefix", UserTypeIDPrefix},
		{"VerificationTokenIDPrefix", VerificationTokenIDPrefix},
		{"VisibilityTypeIDPrefix", VisibilityTypeIDPrefix},
		{"AdjustmentTypeIDPrefix", AdjustmentTypeIDPrefix},
		{"ChangeLogIDPrefix", ChangeLogIDPrefix},
		{"DCLocationIDPrefix", DCLocationIDPrefix},
		{"LocationIDPrefix", LocationIDPrefix},
		{"EmailLogIDPrefix", EmailLogIDPrefix},
		{"EmailRecipientIDPrefix", EmailRecipientIDPrefix},
		{"OrderEmailIDPrefix", OrderEmailIDPrefix},
		{"EmailDomainIDPrefix", EmailDomainIDPrefix},
		{"EmailInboxIDPrefix", EmailInboxIDPrefix},
		{"EmailMessageIDPrefix", EmailMessageIDPrefix},
		{"EmailSenderIDPrefix", EmailSenderIDPrefix},
		{"PortalDomainIDPrefix", PortalDomainIDPrefix},
		{"InventoryChangeLogIDPrefix", InventoryChangeLogIDPrefix},
		{"InventoryLogIDPrefix", InventoryLogIDPrefix},
		{"InventoryAllocationIDPrefix", InventoryAllocationIDPrefix},
		{"RegistrationFlowIDPrefix", RegistrationFlowIDPrefix},
		{"PortalRegistrationSessionIDPrefix", PortalRegistrationSessionIDPrefix},
		{"OnboardingStatusIDPrefix", OnboardingStatusIDPrefix},
		{"SettlementIDPrefix", SettlementIDPrefix},
		{"StripeEventLogIDPrefix", StripeEventLogIDPrefix},
		{"TransactionIDPrefix", TransactionIDPrefix},
		{"TransactionAllocationIDPrefix", TransactionAllocationIDPrefix},
		{"TransactionMethodIDPrefix", TransactionMethodIDPrefix},
		{"TransactionTypeIDPrefix", TransactionTypeIDPrefix},
		{"EDIRunIDPrefix", EDIRunIDPrefix},
		{"RequestIDPrefix", RequestIDPrefix},
		{"AuditEventIDPrefix", AuditEventIDPrefix},
		{"PlanChangeIDPrefix", PlanChangeIDPrefix},
		{"EnterpriseInquiryIDPrefix", EnterpriseInquiryIDPrefix},
		{"IdempotencyKeyIDPrefix", IdempotencyKeyIDPrefix},
		{"ServiceIdempotencyKeyIDPrefix", ServiceIdempotencyKeyIDPrefix},
		{"LotIDPrefix", LotIDPrefix},
		{"ReceivingOrderIDPrefix", ReceivingOrderIDPrefix},
		{"ReceivingOrderLineIDPrefix", ReceivingOrderLineIDPrefix},
		{"MessageIDPrefix", MessageIDPrefix},
		{"ConversationIDPrefix", ConversationIDPrefix},
		{"ConversationParticipantIDPrefix", ConversationParticipantIDPrefix},
		{"MessagingGroupIDPrefix", MessagingGroupIDPrefix},
		{"MessagingGroupMemberIDPrefix", MessagingGroupMemberIDPrefix},
		{"MessageAttachmentIDPrefix", MessageAttachmentIDPrefix},
		{"MessageReceiptIDPrefix", MessageReceiptIDPrefix},
		{"MessageBlockIDPrefix", MessageBlockIDPrefix},
		{"MessageReportIDPrefix", MessageReportIDPrefix},
		{"NotificationIDPrefix", NotificationIDPrefix},
		{"NotificationPreferenceIDPrefix", NotificationPreferenceIDPrefix},
		{"ScheduledMessageIDPrefix", ScheduledMessageIDPrefix},
		{"AnnouncementIDPrefix", AnnouncementIDPrefix},
		{"AnnouncementReceiptIDPrefix", AnnouncementReceiptIDPrefix},
		{"SupportRouteIDPrefix", SupportRouteIDPrefix},
		{"ReplyDraftIDPrefix", ReplyDraftIDPrefix},
		{"ConversationLinkIDPrefix", ConversationLinkIDPrefix},
		{"InventoryReceiptIDPrefix", InventoryReceiptIDPrefix},
		{"InventoryIssueIDPrefix", InventoryIssueIDPrefix},
		{"FreeShippingRuleIDPrefix", FreeShippingRuleIDPrefix},
		{"PricingPlanIDPrefix", PricingPlanIDPrefix},
		{"MachineDowntimeEventIDPrefix", MachineDowntimeEventIDPrefix},
		{"MachineDowntimeReasonIDPrefix", MachineDowntimeReasonIDPrefix},
		{"DemandOverrideIDPrefix", DemandOverrideIDPrefix},
		{"DemandOverrideTypeIDPrefix", DemandOverrideTypeIDPrefix},
		{"ProductionShiftIDPrefix", ProductionShiftIDPrefix},
		{"ProductionScheduleIDPrefix", ProductionScheduleIDPrefix},
		{"ProductionScheduleLineIDPrefix", ProductionScheduleLineIDPrefix},
		{"ProductionScheduleItemPolicyIDPrefix", ProductionScheduleItemPolicyIDPrefix},
		{"ProductionScheduleFinishedPolicyIDPrefix", ProductionScheduleFinishedPolicyIDPrefix},
		{"ProductionScheduleDeviationIDPrefix", ProductionScheduleDeviationIDPrefix},
		{"ScheduleDeviationTypeIDPrefix", ScheduleDeviationTypeIDPrefix},
		{"ProductionScheduleDerivedLineIDPrefix", ProductionScheduleDerivedLineIDPrefix},
		{"AccountProductionScheduleSettingIDPrefix", AccountProductionScheduleSettingIDPrefix},
		{"ProductionScheduleResourceSettingIDPrefix", ProductionScheduleResourceSettingIDPrefix},
		{"ProductionScheduleItemSettingIDPrefix", ProductionScheduleItemSettingIDPrefix},
	}

	seen := make(map[IDPrefix]string, len(prefixes))
	for _, p := range prefixes {
		if existing, exists := seen[p.value]; exists {
			t.Errorf("duplicate prefix value %q: %s and %s", p.value, existing, p.name)
		}
		seen[p.value] = p.name
	}
}

// --- IDPrefix vocabulary composition test ---

func TestIDPrefixes_ValidVocabularyComposition(t *testing.T) {
	t.Parallel(
	// All known 2-letter vocabulary codes.
	)

	knownVocabs := map[string]bool{
		VocAccount: true, VocAddress: true, VocAction: true, VocAgent: true,
		VocAPI: true, VocArtifact: true, VocAttribute: true, VocAdjustment: true,
		VocAudit:      true,
		VocAllocation: true, VocAssignment: true, VocBatch: true, VocBilling: true,
		VocBranding: true, VocCache: true, VocCalendar: true, VocChange: true, VocColor: true,
		VocClosure: true, VocOperating: true,
		VocCommission: true, VocConsumption: true, VocCarrier: true, VocCase: true,
		VocContact: true, VocCustomer: true, VocConversion: true, VocCategory: true,
		VocDC: true, VocDelivery: true, VocDimension: true, VocDiscount: true, VocDefinition: true,
		VocDepartment: true, VocDocument: true, VocDomain: true, VocEDI: true, VocEmail: true,
		VocEnterprise: true, VocError: true, VocEstimate: true, VocEvent: true, VocFormula: true,
		VocFreight: true, VocFlow: true, VocGeolocation: true, VocGroup: true,
		VocIdempotency: true, VocIntegration: true, VocImage: true, VocInventory: true,
		VocInquiry: true, VocIntent: true, VocItem: true, VocInvoice: true,
		VocInbox: true, VocKey: true, VocLabel: true, VocLevel: true, VocLocation: true,
		VocLog: true, VocLine: true, VocLot: true, VocMachine: true, VocMemory: true,
		VocMethod: true, VocMessage: true, VocMaterial: true, VocNotification: true,
		VocOrganization: true, VocOnboarding: true, VocOption: true, VocOrder: true,
		VocOutbox: true, VocProduct: true, VocPreference: true, VocPriority: true,
		VocPick: true, VocPortal: true, VocPlan: true, VocPermission: true,
		VocProduction: true, VocProperty: true, VocPrice: true, VocProcess: true,
		VocPart: true, VocPayment: true, VocQuantity: true, VocRecipient: true,
		VocRelation: true, VocRegistration: true, VocRole: true, VocRun: true,
		VocRequest: true, VocRate: true, VocSandbox: true, VocSession: true,
		VocShipment: true, VocSettlement: true, VocScanning: true, VocService: true,
		VocStripe: true, VocStep: true, VocStation: true, VocStatus: true,
		VocSupplier: true, VocSender: true, VocSeverity: true, VocSystem: true, VocSize: true,
		VocTerritory: true, VocTarget: true, VocToken: true, VocTool: true,
		VocTerm: true, VocTransform: true, VocTransit: true, VocTier: true, VocTransaction: true,
		VocType: true, VocUnit: true, VocUser: true, VocVisibility: true,
		VocVerification: true,
		VocIssue:        true,
		VocReceiving:    true,
		VocJob:          true,
		VocRecord:       true,
		VocReview:       true,
		VocConversation: true,
		VocSchedule:     true,
		VocAttachment:   true,
		VocAnnouncement: true,
		VocBlock:        true,
		VocReport:       true,
		VocSupport:      true,
		VocRoute:        true,
		VocReply:        true,
		VocDraft:        true,
		VocLink:         true,
		VocDemand:       true,
		VocDeviation:    true,
		VocDerived:      true,
		VocDowntime:     true,
		VocOverride:     true,
		VocResource:     true,
		VocSetting:      true,
		VocShift:        true,
		VocPolicy:       true,
		VocFinished:     true,
	}

	prefixes := []struct {
		name  string
		value IDPrefix
	}{
		{"AccountIDPrefix", AccountIDPrefix},
		{"AccountAddressIDPrefix", AccountAddressIDPrefix},
		{"AccountBillingIDPrefix", AccountBillingIDPrefix},
		{"AccountBrandingIDPrefix", AccountBrandingIDPrefix},
		{"AccountPortalIDPrefix", AccountPortalIDPrefix},
		{"AccountIntegrationIDPrefix", AccountIntegrationIDPrefix},
		{"HubspotSyncJobIDPrefix", HubspotSyncJobIDPrefix},
		{"HubspotSyncRecordIDPrefix", HubspotSyncRecordIDPrefix},
		{"HubspotCompanyReviewIDPrefix", HubspotCompanyReviewIDPrefix},
		{"AccountGroupIDPrefix", AccountGroupIDPrefix},
		{"AccountGroupTypeIDPrefix", AccountGroupTypeIDPrefix},
		{"AccountTypeIDPrefix", AccountTypeIDPrefix},
		{"AccountRelationIDPrefix", AccountRelationIDPrefix},
		{"AccountStatusIDPrefix", AccountStatusIDPrefix},
		{"AccountPriceIDPrefix", AccountPriceIDPrefix},
		{"AccountPriceItemCategoryIDPrefix", AccountPriceItemCategoryIDPrefix},
		{"AccountGroupQuantityDiscountIDPrefix", AccountGroupQuantityDiscountIDPrefix},
		{"AccountPriceAttributeIDPrefix", AccountPriceAttributeIDPrefix},
		{"AccountRelationProductLineIDPrefix", AccountRelationProductLineIDPrefix},
		{"AccountRelationPriceGroupIDPrefix", AccountRelationPriceGroupIDPrefix},
		{"AccountRelationNotificationPreferenceIDPrefix", AccountRelationNotificationPreferenceIDPrefix},
		{"AccountRelationNotificationTypeIDPrefix", AccountRelationNotificationTypeIDPrefix},
		{"AccountGroupProductLineIDPrefix", AccountGroupProductLineIDPrefix},
		{"AccountPreferenceIDPrefix", AccountPreferenceIDPrefix},
		{"AccountUserIDPrefix", AccountUserIDPrefix},
		{"ActionTypeIDPrefix", ActionTypeIDPrefix},
		{"AgentDefinitionIDPrefix", AgentDefinitionIDPrefix},
		{"AgentConfigIDPrefix", AgentConfigIDPrefix},
		{"AgentRunIDPrefix", AgentRunIDPrefix},
		{"AgentActionIDPrefix", AgentActionIDPrefix},
		{"AgentArtifactIDPrefix", AgentArtifactIDPrefix},
		{"AgentMemoryIDPrefix", AgentMemoryIDPrefix},
		{"AgentTokenUsageIDPrefix", AgentTokenUsageIDPrefix},
		{"TokenPackPurchaseIDPrefix", TokenPackPurchaseIDPrefix},
		{"AgentDefinitionToolIDPrefix", AgentDefinitionToolIDPrefix},
		{"AgentAccountStatusIDPrefix", AgentAccountStatusIDPrefix},
		{"AgentRunEventIDPrefix", AgentRunEventIDPrefix},
		{"ToolDefinitionIDPrefix", ToolDefinitionIDPrefix},
		{"AddressIDPrefix", AddressIDPrefix},
		{"APILocationIDPrefix", APILocationIDPrefix},
		{"APIKeyIDPrefix", APIKeyIDPrefix},
		{"DocAPIKeyIDPrefix", DocAPIKeyIDPrefix},
		{"AttributeIDPrefix", AttributeIDPrefix},
		{"BatchIDPrefix", BatchIDPrefix},
		{"CarrierIDPrefix", CarrierIDPrefix},
		{"CarrierTransitEstimateIDPrefix", CarrierTransitEstimateIDPrefix},
		{"OperatingCalendarIDPrefix", OperatingCalendarIDPrefix},
		{"OperatingCalendarClosureIDPrefix", OperatingCalendarClosureIDPrefix},
		{"ServiceLevelIDPrefix", ServiceLevelIDPrefix},
		{"ColorIDPrefix", ColorIDPrefix},
		{"CommissionPolicyIDPrefix", CommissionPolicyIDPrefix},
		{"ConsumptionIDPrefix", ConsumptionIDPrefix},
		{"ContactIDPrefix", ContactIDPrefix},
		{"ContactTypeIDPrefix", ContactTypeIDPrefix},
		{"DeliveryIDPrefix", DeliveryIDPrefix},
		{"DeliveryLineIDPrefix", DeliveryLineIDPrefix},
		{"DepartmentIDPrefix", DepartmentIDPrefix},
		{"DiscountTypeIDPrefix", DiscountTypeIDPrefix},
		{"OrderDiscountIDPrefix", OrderDiscountIDPrefix},
		{"QuantityDiscountIDPrefix", QuantityDiscountIDPrefix},
		{"QuantityDiscountTierIDPrefix", QuantityDiscountTierIDPrefix},
		{"ErrorSeverityIDPrefix", ErrorSeverityIDPrefix},
		{"ErrorLogIDPrefix", ErrorLogIDPrefix},
		{"FreightPolicyIDPrefix", FreightPolicyIDPrefix},
		{"GeolocationIDPrefix", GeolocationIDPrefix},
		{"IntegrationIDPrefix", IntegrationIDPrefix},
		{"InvoiceIDPrefix", InvoiceIDPrefix},
		{"InvoiceLineIDPrefix", InvoiceLineIDPrefix},
		{"ItemIDPrefix", ItemIDPrefix},
		{"ItemCategoryIDPrefix", ItemCategoryIDPrefix},
		{"ItemCategoryTypeIDPrefix", ItemCategoryTypeIDPrefix},
		{"ItemTypeIDPrefix", ItemTypeIDPrefix},
		{"LabelSizeIDPrefix", LabelSizeIDPrefix},
		{"LabelTypeIDPrefix", LabelTypeIDPrefix},
		{"MachineIDPrefix", MachineIDPrefix},
		{"MaterialIDPrefix", MaterialIDPrefix},
		{"SupplierMaterialIDPrefix", SupplierMaterialIDPrefix},
		{"OrganizationIDPrefix", OrganizationIDPrefix},
		{"PartIDPrefix", PartIDPrefix},
		{"PaymentTermIDPrefix", PaymentTermIDPrefix},
		{"OrderPaymentIntentIDPrefix", OrderPaymentIntentIDPrefix},
		{"PermissionIDPrefix", PermissionIDPrefix},
		{"PermissionGroupIDPrefix", PermissionGroupIDPrefix},
		{"PickIDPrefix", PickIDPrefix},
		{"PickLineIDPrefix", PickLineIDPrefix},
		{"PickAssignmentIDPrefix", PickAssignmentIDPrefix},
		{"PreferenceDefinitionIDPrefix", PreferenceDefinitionIDPrefix},
		{"PriorityIDPrefix", PriorityIDPrefix},
		{"ProductIDPrefix", ProductIDPrefix},
		{"ProductLineIDPrefix", ProductLineIDPrefix},
		{"ProductLineTargetIDPrefix", ProductLineTargetIDPrefix},
		{"ProductTypeIDPrefix", ProductTypeIDPrefix},
		{"ProductImageIDPrefix", ProductImageIDPrefix},
		{"ProductionIDPrefix", ProductionIDPrefix},
		{"ProductionFormulaIDPrefix", ProductionFormulaIDPrefix},
		{"ProductionFormulaItemIDPrefix", ProductionFormulaItemIDPrefix},
		{"ProductionProcessIDPrefix", ProductionProcessIDPrefix},
		{"ProductionProcessItemCategoryIDPrefix", ProductionProcessItemCategoryIDPrefix},
		{"ProductionRunIDPrefix", ProductionRunIDPrefix},
		{"ProductionStepIDPrefix", ProductionStepIDPrefix},
		{"ProductionStepTransformIDPrefix", ProductionStepTransformIDPrefix},
		{"PropertyIDPrefix", PropertyIDPrefix},
		{"SysPropertyIDPrefix", SysPropertyIDPrefix},
		{"SysPropertyTypeIDPrefix", SysPropertyTypeIDPrefix},
		{"QuantityIDPrefix", QuantityIDPrefix},
		{"RateIDPrefix", RateIDPrefix},
		{"RoleIDPrefix", RoleIDPrefix},
		{"RolePermissionIDPrefix", RolePermissionIDPrefix},
		{"RoleTypeIDPrefix", RoleTypeIDPrefix},
		{"OrderIDPrefix", OrderIDPrefix},
		{"OrderLineIDPrefix", OrderLineIDPrefix},
		{"OrderStatusIDPrefix", OrderStatusIDPrefix},
		{"OrderTypeIDPrefix", OrderTypeIDPrefix},
		{"TerritoryIDPrefix", TerritoryIDPrefix},
		{"TargetIDPrefix", TargetIDPrefix},
		{"SandboxAccountIDPrefix", SandboxAccountIDPrefix},
		{"ScanningStationIDPrefix", ScanningStationIDPrefix},
		{"ScanningStationTypeIDPrefix", ScanningStationTypeIDPrefix},
		{"SessionIDPrefix", SessionIDPrefix},
		{"ShipmentIDPrefix", ShipmentIDPrefix},
		{"ShipmentLineIDPrefix", ShipmentLineIDPrefix},
		{"ShipmentStatusIDPrefix", ShipmentStatusIDPrefix},
		{"ShippingCaseIDPrefix", ShippingCaseIDPrefix},
		{"ShippingTermIDPrefix", ShippingTermIDPrefix},
		{"UnitIDPrefix", UnitIDPrefix},
		{"UnitDimensionIDPrefix", UnitDimensionIDPrefix},
		{"UnitGroupIDPrefix", UnitGroupIDPrefix},
		{"UnitGroupsUnitsIDPrefix", UnitGroupsUnitsIDPrefix},
		{"UnitConversionIDPrefix", UnitConversionIDPrefix},
		{"UserIDPrefix", UserIDPrefix},
		{"UserAccountIDPrefix", UserAccountIDPrefix},
		{"UserStatusIDPrefix", UserStatusIDPrefix},
		{"UserTypeIDPrefix", UserTypeIDPrefix},
		{"VerificationTokenIDPrefix", VerificationTokenIDPrefix},
		{"VisibilityTypeIDPrefix", VisibilityTypeIDPrefix},
		{"AdjustmentTypeIDPrefix", AdjustmentTypeIDPrefix},
		{"ChangeLogIDPrefix", ChangeLogIDPrefix},
		{"DCLocationIDPrefix", DCLocationIDPrefix},
		{"LocationIDPrefix", LocationIDPrefix},
		{"EmailLogIDPrefix", EmailLogIDPrefix},
		{"EmailRecipientIDPrefix", EmailRecipientIDPrefix},
		{"OrderEmailIDPrefix", OrderEmailIDPrefix},
		{"EmailDomainIDPrefix", EmailDomainIDPrefix},
		{"EmailInboxIDPrefix", EmailInboxIDPrefix},
		{"EmailMessageIDPrefix", EmailMessageIDPrefix},
		{"EmailSenderIDPrefix", EmailSenderIDPrefix},
		{"PortalDomainIDPrefix", PortalDomainIDPrefix},
		{"InventoryChangeLogIDPrefix", InventoryChangeLogIDPrefix},
		{"InventoryLogIDPrefix", InventoryLogIDPrefix},
		{"InventoryAllocationIDPrefix", InventoryAllocationIDPrefix},
		{"RegistrationFlowIDPrefix", RegistrationFlowIDPrefix},
		{"PortalRegistrationSessionIDPrefix", PortalRegistrationSessionIDPrefix},
		{"OnboardingStatusIDPrefix", OnboardingStatusIDPrefix},
		{"SettlementIDPrefix", SettlementIDPrefix},
		{"StripeEventLogIDPrefix", StripeEventLogIDPrefix},
		{"TransactionIDPrefix", TransactionIDPrefix},
		{"TransactionAllocationIDPrefix", TransactionAllocationIDPrefix},
		{"TransactionMethodIDPrefix", TransactionMethodIDPrefix},
		{"TransactionTypeIDPrefix", TransactionTypeIDPrefix},
		{"EDIRunIDPrefix", EDIRunIDPrefix},
		{"RequestIDPrefix", RequestIDPrefix},
		{"AuditEventIDPrefix", AuditEventIDPrefix},
		{"PlanChangeIDPrefix", PlanChangeIDPrefix},
		{"EnterpriseInquiryIDPrefix", EnterpriseInquiryIDPrefix},
		{"IdempotencyKeyIDPrefix", IdempotencyKeyIDPrefix},
		{"ServiceIdempotencyKeyIDPrefix", ServiceIdempotencyKeyIDPrefix},
		{"LotIDPrefix", LotIDPrefix},
		{"ReceivingOrderIDPrefix", ReceivingOrderIDPrefix},
		{"ReceivingOrderLineIDPrefix", ReceivingOrderLineIDPrefix},
		{"MessageIDPrefix", MessageIDPrefix},
		{"ConversationIDPrefix", ConversationIDPrefix},
		{"ConversationParticipantIDPrefix", ConversationParticipantIDPrefix},
		{"MessagingGroupIDPrefix", MessagingGroupIDPrefix},
		{"MessagingGroupMemberIDPrefix", MessagingGroupMemberIDPrefix},
		{"MessageAttachmentIDPrefix", MessageAttachmentIDPrefix},
		{"MessageReceiptIDPrefix", MessageReceiptIDPrefix},
		{"MessageBlockIDPrefix", MessageBlockIDPrefix},
		{"MessageReportIDPrefix", MessageReportIDPrefix},
		{"NotificationIDPrefix", NotificationIDPrefix},
		{"NotificationPreferenceIDPrefix", NotificationPreferenceIDPrefix},
		{"ScheduledMessageIDPrefix", ScheduledMessageIDPrefix},
		{"AnnouncementIDPrefix", AnnouncementIDPrefix},
		{"AnnouncementReceiptIDPrefix", AnnouncementReceiptIDPrefix},
		{"SupportRouteIDPrefix", SupportRouteIDPrefix},
		{"ReplyDraftIDPrefix", ReplyDraftIDPrefix},
		{"ConversationLinkIDPrefix", ConversationLinkIDPrefix},
		{"InventoryReceiptIDPrefix", InventoryReceiptIDPrefix},
		{"InventoryIssueIDPrefix", InventoryIssueIDPrefix},
		{"FreeShippingRuleIDPrefix", FreeShippingRuleIDPrefix},
		{"PricingPlanIDPrefix", PricingPlanIDPrefix},
		{"MachineDowntimeEventIDPrefix", MachineDowntimeEventIDPrefix},
		{"MachineDowntimeReasonIDPrefix", MachineDowntimeReasonIDPrefix},
		{"DemandOverrideIDPrefix", DemandOverrideIDPrefix},
		{"DemandOverrideTypeIDPrefix", DemandOverrideTypeIDPrefix},
		{"ProductionShiftIDPrefix", ProductionShiftIDPrefix},
		{"ProductionScheduleIDPrefix", ProductionScheduleIDPrefix},
		{"ProductionScheduleLineIDPrefix", ProductionScheduleLineIDPrefix},
		{"ProductionScheduleItemPolicyIDPrefix", ProductionScheduleItemPolicyIDPrefix},
		{"ProductionScheduleFinishedPolicyIDPrefix", ProductionScheduleFinishedPolicyIDPrefix},
		{"ProductionScheduleDeviationIDPrefix", ProductionScheduleDeviationIDPrefix},
		{"ScheduleDeviationTypeIDPrefix", ScheduleDeviationTypeIDPrefix},
		{"ProductionScheduleDerivedLineIDPrefix", ProductionScheduleDerivedLineIDPrefix},
		{"AccountProductionScheduleSettingIDPrefix", AccountProductionScheduleSettingIDPrefix},
		{"ProductionScheduleResourceSettingIDPrefix", ProductionScheduleResourceSettingIDPrefix},
		{"ProductionScheduleItemSettingIDPrefix", ProductionScheduleItemSettingIDPrefix},
	}

	for _, p := range prefixes {
		t.Run(p.name, func(t *testing.T) {
			prefix := string(p.value)
			if len(prefix) == 0 {
				t.Fatalf("prefix is empty")
			}
			if len(prefix)%2 != 0 {
				t.Fatalf("prefix %q has odd length %d — must be composed of 2-char vocabulary codes", prefix, len(prefix))
			}
			for i := 0; i < len(prefix); i += 2 {
				chunk := prefix[i : i+2]
				if !knownVocabs[chunk] {
					t.Errorf("prefix %q contains unknown vocabulary code %q at position %d", prefix, chunk, i)
				}
			}
		})
	}
}

// --- IDPrefix format tests ---

func TestIDPrefixes_NonEmpty(t *testing.T) {
	t.Parallel()
	prefixes := []struct {
		name  string
		value IDPrefix
	}{
		{"UserIDPrefix", UserIDPrefix},
		{"AccountIDPrefix", AccountIDPrefix},
		{"OrderIDPrefix", OrderIDPrefix},
		{"APIKeyIDPrefix", APIKeyIDPrefix},
	}

	for _, p := range prefixes {
		t.Run(p.name, func(t *testing.T) {
			if p.value == "" {
				t.Errorf("prefix %s is empty", p.name)
			}
			// Prefixes should only contain lowercase letters from the vocabulary
			for _, c := range string(p.value) {
				if c < 'a' || c > 'z' {
					t.Errorf("prefix %s contains non-lowercase character %c (%q)", p.name, c, p.value)
				}
			}
		})
	}
}

// --- Vocabulary uniqueness test ---

func TestVocabulary_NoDuplicates(t *testing.T) {
	t.Parallel()
	vocabs := []struct {
		name  string
		value string
	}{
		{"VocAccount", VocAccount},
		{"VocAddress", VocAddress},
		{"VocAction", VocAction},
		{"VocAgent", VocAgent},
		{"VocAPI", VocAPI},
		{"VocArtifact", VocArtifact},
		{"VocAttribute", VocAttribute},
		{"VocAdjustment", VocAdjustment},
		{"VocAudit", VocAudit},
		{"VocAllocation", VocAllocation},
		{"VocAssignment", VocAssignment},
		{"VocBatch", VocBatch},
		{"VocBilling", VocBilling},
		{"VocBranding", VocBranding},
		{"VocCache", VocCache},
		{"VocChange", VocChange},
		{"VocColor", VocColor},
		{"VocCommission", VocCommission},
		{"VocConsumption", VocConsumption},
		{"VocCarrier", VocCarrier},
		{"VocCase", VocCase},
		{"VocContact", VocContact},
		{"VocCustomer", VocCustomer},
		{"VocConversion", VocConversion},
		{"VocCategory", VocCategory},
		{"VocDC", VocDC},
		{"VocDelivery", VocDelivery},
		{"VocDimension", VocDimension},
		{"VocDiscount", VocDiscount},
		{"VocDefinition", VocDefinition},
		{"VocDepartment", VocDepartment},
		{"VocDocument", VocDocument},
		{"VocEDI", VocEDI},
		{"VocEmail", VocEmail},
		{"VocEnterprise", VocEnterprise},
		{"VocError", VocError},
		{"VocEstimate", VocEstimate},
		{"VocEvent", VocEvent},
		{"VocFormula", VocFormula},
		{"VocFreight", VocFreight},
		{"VocFlow", VocFlow},
		{"VocGeolocation", VocGeolocation},
		{"VocGroup", VocGroup},
		{"VocIdempotency", VocIdempotency},
		{"VocIntegration", VocIntegration},
		{"VocImage", VocImage},
		{"VocInventory", VocInventory},
		{"VocInquiry", VocInquiry},
		{"VocIntent", VocIntent},
		{"VocItem", VocItem},
		{"VocInvoice", VocInvoice},
		{"VocInbox", VocInbox},
		{"VocKey", VocKey},
		{"VocLabel", VocLabel},
		{"VocLevel", VocLevel},
		{"VocLocation", VocLocation},
		{"VocLog", VocLog},
		{"VocLine", VocLine},
		{"VocLot", VocLot},
		{"VocMachine", VocMachine},
		{"VocMemory", VocMemory},
		{"VocMethod", VocMethod},
		{"VocMessage", VocMessage},
		{"VocMaterial", VocMaterial},
		{"VocNotification", VocNotification},
		{"VocOrganization", VocOrganization},
		{"VocOnboarding", VocOnboarding},
		{"VocOption", VocOption},
		{"VocOrder", VocOrder},
		{"VocOutbox", VocOutbox},
		{"VocProduct", VocProduct},
		{"VocPreference", VocPreference},
		{"VocPriority", VocPriority},
		{"VocPick", VocPick},
		{"VocPortal", VocPortal},
		{"VocPlan", VocPlan},
		{"VocPermission", VocPermission},
		{"VocProduction", VocProduction},
		{"VocProperty", VocProperty},
		{"VocPrice", VocPrice},
		{"VocProcess", VocProcess},
		{"VocPart", VocPart},
		{"VocPayment", VocPayment},
		{"VocQuantity", VocQuantity},
		{"VocRecipient", VocRecipient},
		{"VocRelation", VocRelation},
		{"VocRegistration", VocRegistration},
		{"VocRole", VocRole},
		{"VocRun", VocRun},
		{"VocRequest", VocRequest},
		{"VocRate", VocRate},
		{"VocSandbox", VocSandbox},
		{"VocSession", VocSession},
		{"VocShipment", VocShipment},
		{"VocSettlement", VocSettlement},
		{"VocScanning", VocScanning},
		{"VocService", VocService},
		{"VocStripe", VocStripe},
		{"VocStep", VocStep},
		{"VocStation", VocStation},
		{"VocStatus", VocStatus},
		{"VocSupplier", VocSupplier},
		{"VocSeverity", VocSeverity},
		{"VocSystem", VocSystem},
		{"VocSize", VocSize},
		{"VocTerritory", VocTerritory},
		{"VocTarget", VocTarget},
		{"VocToken", VocToken},
		{"VocTool", VocTool},
		{"VocTerm", VocTerm},
		{"VocTransform", VocTransform},
		{"VocTransit", VocTransit},
		{"VocTier", VocTier},
		{"VocTransaction", VocTransaction},
		{"VocType", VocType},
		{"VocUnit", VocUnit},
		{"VocUser", VocUser},
		{"VocVisibility", VocVisibility},
		{"VocVerification", VocVerification},
		{"VocIssue", VocIssue},
		{"VocReceiving", VocReceiving},
		{"VocConversation", VocConversation},
		{"VocSchedule", VocSchedule},
		{"VocAttachment", VocAttachment},
		{"VocAnnouncement", VocAnnouncement},
		{"VocBlock", VocBlock},
		{"VocReport", VocReport},
		{"VocDomain", VocDomain},
		{"VocDraft", VocDraft},
		{"VocJob", VocJob},
		{"VocLink", VocLink},
		{"VocRecord", VocRecord},
		{"VocReply", VocReply},
		{"VocReview", VocReview},
		{"VocRoute", VocRoute},
		{"VocSupport", VocSupport},
		{"VocDemand", VocDemand},
		{"VocDeviation", VocDeviation},
		{"VocDerived", VocDerived},
		{"VocDowntime", VocDowntime},
		{"VocOverride", VocOverride},
		{"VocResource", VocResource},
		{"VocSetting", VocSetting},
		{"VocShift", VocShift},
		{"VocPolicy", VocPolicy},
		{"VocFinished", VocFinished},
	}

	seen := make(map[string]string, len(vocabs))
	for _, v := range vocabs {
		if existing, exists := seen[v.value]; exists {
			t.Errorf("duplicate vocabulary code %q: %s and %s", v.value, existing, v.name)
		}
		seen[v.value] = v.name
	}

	// Each vocabulary code should be exactly 2 lowercase letters
	for _, v := range vocabs {
		if len(v.value) != 2 {
			t.Errorf("vocabulary %s has length %d, expected 2 (%q)", v.name, len(v.value), v.value)
		}
		for _, c := range v.value {
			if c < 'a' || c > 'z' {
				t.Errorf("vocabulary %s contains non-lowercase character %c (%q)", v.name, c, v.value)
			}
		}
	}
}

// --- GenID prefix validation tests ---

// GenID performs no validation on the prefix; these pin the permissive behavior so a change to it is a deliberate one.
func TestGenID_UnvalidatedPrefixes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		prefix         IDPrefix
		expectedPrefix string
	}{
		{"empty", IDPrefix(""), "_"},
		{"unregistered", IDPrefix("zz"), "zz_"},
		{"typo_of_real_prefix", IDPrefix("usr"), "usr_"},
		{"uppercase", IDPrefix("US"), "US_"},
		{"non_alphanumeric", IDPrefix("a-b"), "a-b_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, apiErr := GenID(tt.prefix, nil)
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}
			if !strings.HasPrefix(id, tt.expectedPrefix) {
				t.Errorf("expected prefix %q, got %q", tt.expectedPrefix, id)
			}
			nanoIDPart := strings.TrimPrefix(id, tt.expectedPrefix)
			if len(nanoIDPart) != int(IDLength12) {
				t.Errorf("expected nano ID length %d, got %d (%q)", IDLength12, len(nanoIDPart), nanoIDPart)
			}
		})
	}
}

// --- Source-derived completeness tests ---

// parsePrefixSource reads every Voc* constant and every composePrefix var straight out of id_prefix_values.go, so the checks below cover the full set rather than a hand-copied list that drifts behind the source.
func parsePrefixSource(t *testing.T) (vocab map[string]string, prefixes map[string]IDPrefix) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "id_prefix_values.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse id_prefix_values.go: %v", err)
	}

	vocab = make(map[string]string)
	prefixes = make(map[string]IDPrefix)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != len(valueSpec.Values) {
				continue
			}

			for i, name := range valueSpec.Names {
				switch genDecl.Tok {
				case token.CONST:
					if !strings.HasPrefix(name.Name, "Voc") {
						continue
					}
					lit, ok := valueSpec.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						t.Fatalf("%s is not declared as a string literal, so it escapes the vocabulary checks", name.Name)
					}
					value, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatalf("failed to unquote value of %s: %v", name.Name, err)
					}
					vocab[name.Name] = value
				case token.VAR:
					if !strings.HasSuffix(name.Name, "IDPrefix") {
						continue
					}
					// A prefix built any other way would never reach the uniqueness checks below.
					call, ok := valueSpec.Values[i].(*ast.CallExpr)
					if !ok {
						t.Fatalf("%s is not declared with composePrefix", name.Name)
					}
					fn, ok := call.Fun.(*ast.Ident)
					if !ok || fn.Name != "composePrefix" {
						t.Fatalf("%s is not declared with composePrefix", name.Name)
					}

					var words []string
					for _, arg := range call.Args {
						ident, ok := arg.(*ast.Ident)
						if !ok {
							t.Fatalf("%s: composePrefix argument is not a vocabulary identifier", name.Name)
						}
						words = append(words, ident.Name)
					}
					prefixes[name.Name] = composePrefixFromNames(t, vocab, name.Name, words)
				}
			}
		}
	}

	if len(vocab) == 0 {
		t.Fatal("no vocabulary constants found in id_prefix_values.go")
	}
	if len(prefixes) == 0 {
		t.Fatal("no composePrefix variables found in id_prefix_values.go")
	}

	return vocab, prefixes
}

func composePrefixFromNames(t *testing.T, vocab map[string]string, prefixName string, words []string) IDPrefix {
	t.Helper()

	values := make([]string, 0, len(words))
	for _, word := range words {
		value, ok := vocab[word]
		if !ok {
			t.Fatalf("%s references unknown vocabulary constant %s", prefixName, word)
		}
		values = append(values, value)
	}

	return composePrefix(values...)
}

func TestVocabulary_AllCodesUniqueAndWellFormed(t *testing.T) {
	t.Parallel()
	vocab, _ := parsePrefixSource(t)

	seen := make(map[string]string, len(vocab))
	for name, value := range vocab {
		if existing, exists := seen[value]; exists {
			t.Errorf("duplicate vocabulary code %q: %s and %s", value, existing, name)
		}
		seen[value] = name

		if len(value) != 2 {
			t.Errorf("vocabulary %s has length %d, expected 2 (%q)", name, len(value), value)
		}
		for _, c := range value {
			if c < 'a' || c > 'z' {
				t.Errorf("vocabulary %s contains non-lowercase character %c (%q)", name, c, value)
			}
		}
	}
}

func TestIDPrefixes_AllPrefixesUnique(t *testing.T) {
	t.Parallel()
	_, prefixes := parsePrefixSource(t)

	seen := make(map[IDPrefix]string, len(prefixes))
	for name, value := range prefixes {
		if existing, exists := seen[value]; exists {
			t.Errorf("duplicate prefix %q: %s and %s", value, existing, name)
		}
		seen[value] = name

		if value == "" {
			t.Errorf("prefix %s is empty", name)
		}
		if len(value)%2 != 0 {
			t.Errorf("prefix %s has length %d, expected a multiple of 2 (%q)", name, len(value), value)
		}
		for _, c := range string(value) {
			if c < 'a' || c > 'z' {
				t.Errorf("prefix %s contains non-lowercase character %c (%q)", name, c, value)
			}
		}
	}
}

// The hand-maintained lists in the tests above only prove uniqueness of what someone remembered to copy; this pins that the entities most at risk of a silent prefix collision are actually present in the source-derived set.
func TestIDPrefixes_SourceSetMatchesExportedValues(t *testing.T) {
	t.Parallel()
	_, prefixes := parsePrefixSource(t)

	expected := map[string]IDPrefix{
		"JobIDPrefix": JobIDPrefix,
		"ProductionScheduleFinishingLineIDPrefix": ProductionScheduleFinishingLineIDPrefix,
		"ProductionScheduleLineOrderIDPrefix":     ProductionScheduleLineOrderIDPrefix,
		"UserIDPrefix":                            UserIDPrefix,
		"AccountIDPrefix":                         AccountIDPrefix,
		"OrderIDPrefix":                           OrderIDPrefix,
		"APIKeyIDPrefix":                          APIKeyIDPrefix,
	}

	for name, want := range expected {
		got, ok := prefixes[name]
		if !ok {
			t.Errorf("%s not found in id_prefix_values.go", name)
			continue
		}
		if got != want {
			t.Errorf("%s: source-derived prefix %q does not match exported value %q", name, got, want)
		}
	}
}
