package id

import (
	"fmt"
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
		{"AgentAlertIDPrefix", AgentAlertIDPrefix},
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
		{"InventoryChangeLogIDPrefix", InventoryChangeLogIDPrefix},
		{"InventoryLogIDPrefix", InventoryLogIDPrefix},
		{"InventoryAllocationIDPrefix", InventoryAllocationIDPrefix},
		{"RegistrationFlowIDPrefix", RegistrationFlowIDPrefix},
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
		{"InventoryReceiptIDPrefix", InventoryReceiptIDPrefix},
		{"InventoryIssueIDPrefix", InventoryIssueIDPrefix},
		{"FreeShippingRuleIDPrefix", FreeShippingRuleIDPrefix},
		{"PricingPlanIDPrefix", PricingPlanIDPrefix},
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
		VocBranding: true, VocCache: true, VocChange: true, VocColor: true,
		VocCommission: true, VocConsumption: true, VocCarrier: true, VocCase: true,
		VocContact: true, VocCustomer: true, VocConversion: true, VocCategory: true,
		VocDC: true, VocDelivery: true, VocDimension: true, VocDiscount: true, VocDefinition: true,
		VocDepartment: true, VocDocument: true, VocEDI: true, VocEmail: true,
		VocEnterprise: true, VocError: true, VocEvent: true, VocFormula: true,
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
		VocSupplier: true, VocSeverity: true, VocSystem: true, VocSize: true,
		VocTerritory: true, VocTarget: true, VocToken: true, VocTool: true,
		VocTerm: true, VocTransform: true, VocTier: true, VocTransaction: true,
		VocType: true, VocUnit: true, VocUser: true, VocVisibility: true,
		VocVerification: true,
		VocIssue:        true,
		VocReceiving:    true,
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
		{"AgentAlertIDPrefix", AgentAlertIDPrefix},
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
		{"InventoryChangeLogIDPrefix", InventoryChangeLogIDPrefix},
		{"InventoryLogIDPrefix", InventoryLogIDPrefix},
		{"InventoryAllocationIDPrefix", InventoryAllocationIDPrefix},
		{"RegistrationFlowIDPrefix", RegistrationFlowIDPrefix},
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
		{"InventoryReceiptIDPrefix", InventoryReceiptIDPrefix},
		{"InventoryIssueIDPrefix", InventoryIssueIDPrefix},
		{"FreeShippingRuleIDPrefix", FreeShippingRuleIDPrefix},
		{"PricingPlanIDPrefix", PricingPlanIDPrefix},
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
		{"VocTier", VocTier},
		{"VocTransaction", VocTransaction},
		{"VocType", VocType},
		{"VocUnit", VocUnit},
		{"VocUser", VocUser},
		{"VocVisibility", VocVisibility},
		{"VocVerification", VocVerification},
		{"VocIssue", VocIssue},
		{"VocReceiving", VocReceiving},
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
