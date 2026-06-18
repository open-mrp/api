package constants

// stringPtrEnum converts an optional enum value (which is represented as a pointer to a named string type) into an optional string pointer.
//
// This is used by StringPtr() methods across all enum-like constant types.
func stringPtrEnum[T ~string](v *T) *string {
	if v == nil {
		return nil
	}

	s := string(*v)
	return &s
}

func (m *AccountGroupType) StringPtr() *string { return stringPtrEnum(m) }
func (m *AccountIntegrationStatus) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *AccountRelationNotificationType) StringPtr() *string {
	return stringPtrEnum(m)
}
func (c *AccountTypeCode) StringPtr() *string { return stringPtrEnum(c) }
func (s *ScanningStationType) StringPtr() *string {
	return stringPtrEnum(s)
}

func (m *AdjustmentType) StringPtr() *string { return stringPtrEnum(m) }
func (m *DashboardPath) StringPtr() *string  { return stringPtrEnum(m) }
func (m *AgentTriggerType) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *UnitType) StringPtr() *string { return stringPtrEnum(m) }
func (m *CarrierBillingType) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *AccountMode) StringPtr() *string { return stringPtrEnum(m) }
func (m *CarrierCode) StringPtr() *string { return stringPtrEnum(m) }
func (m *IntegrationCode) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *CommissionPolicy) StringPtr() *string { return stringPtrEnum(m) }
func (m *RegistrationStep) StringPtr() *string { return stringPtrEnum(m) }
func (m *AccountStatusCode) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *ItemCategoryType) StringPtr() *string { return stringPtrEnum(m) }
func (s *AgentAlertSeverity) StringPtr() *string {
	return stringPtrEnum(s)
}
func (m *AccountUserStatus) StringPtr() *string { return stringPtrEnum(m) }
func (s *SubscriptionStatus) StringPtr() *string {
	return stringPtrEnum(s)
}
func (m *StripeConnectionStatus) StringPtr() *string { return stringPtrEnum(m) }
func (m *FreightPolicy) StringPtr() *string          { return stringPtrEnum(m) }
func (m *RoleType) StringPtr() *string               { return stringPtrEnum(m) }
func (m *TransactionType) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *TransactionMethod) StringPtr() *string {
	return stringPtrEnum(m)
}
func (t *EmailTemplate) StringPtr() *string { return stringPtrEnum(t) }
func (m *PlatformMode) StringPtr() *string  { return stringPtrEnum(m) }
func (p *PlanCode) StringPtr() *string      { return stringPtrEnum(p) }
func (p *PublicPlanCode) StringPtr() *string {
	return stringPtrEnum(p)
}
func (m *Color) StringPtr() *string { return stringPtrEnum(m) }
func (p *Protocol) StringPtr() *string {
	return stringPtrEnum(p)
}
func (m *AgentDefinitionType) StringPtr() *string { return stringPtrEnum(m) }
func (m *ShippingTermType) StringPtr() *string    { return stringPtrEnum(m) }
func (s *ShipmentStatus) StringPtr() *string      { return stringPtrEnum(s) }
func (s *AgentAlertStatus) StringPtr() *string    { return stringPtrEnum(s) }
func (m *SalesOrderStatusCode) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *SalesOrderStatusChange) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *SalesOrderPaymentStatus) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *InvoicePaymentStatus) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *AcknowledgmentStatus) StringPtr() *string {
	return stringPtrEnum(m)
}
func (f *SubassemblyFilter) StringPtr() *string   { return stringPtrEnum(f) }
func (m *OrderDiscountType) StringPtr() *string   { return stringPtrEnum(m) }
func (m *SandboxMode) StringPtr() *string         { return stringPtrEnum(m) }
func (m *ObjectType) StringPtr() *string          { return stringPtrEnum(m) }
func (m *SysPropertyTypeCode) StringPtr() *string { return stringPtrEnum(m) }
func (s *AgentActionStatus) StringPtr() *string   { return stringPtrEnum(s) }
func (s *APIKeyStatus) StringPtr() *string        { return stringPtrEnum(s) }
func (m *AgentAccountStatus) StringPtr() *string  { return stringPtrEnum(m) }
func (s *ToolSlug) StringPtr() *string            { return stringPtrEnum(s) }

func (m *AuditAction) StringPtr() *string               { return stringPtrEnum(m) }
func (m *DeletedRecordResourceType) StringPtr() *string { return stringPtrEnum(m) }
func (m *AgentRunStatus) StringPtr() *string            { return stringPtrEnum(m) }
func (m *DeliveryStatus) StringPtr() *string            { return stringPtrEnum(m) }
func (m *ItemTypeCode) StringPtr() *string              { return stringPtrEnum(m) }
func (m *InventoryActionType) StringPtr() *string       { return stringPtrEnum(m) }
func (o *InventoryUpdateOperation) StringPtr() *string  { return stringPtrEnum(o) }
func (m *PriorityCode) StringPtr() *string              { return stringPtrEnum(m) }
func (s *PaymentTermStatus) StringPtr() *string         { return stringPtrEnum(s) }
func (s *SupplierMaterialStatus) StringPtr() *string    { return stringPtrEnum(s) }
func (m *OwnerType) StringPtr() *string                 { return stringPtrEnum(m) }
func (m *CustomerPortalVisibility) StringPtr() *string  { return stringPtrEnum(m) }
func (s *LocationTypeCode) StringPtr() *string          { return stringPtrEnum(s) }
func (c *LabelSizeCode) StringPtr() *string             { return stringPtrEnum(c) }
func (c *LabelTypeCode) StringPtr() *string             { return stringPtrEnum(c) }
func (c *ServiceLevelCode) StringPtr() *string          { return stringPtrEnum(c) }
func (o *OperatorRequirement) StringPtr() *string       { return stringPtrEnum(o) }
func (m *AddressType) StringPtr() *string               { return stringPtrEnum(m) }
func (m *AddressValidationStatus) StringPtr() *string   { return stringPtrEnum(m) }
func (m *CustomerParentAccountStatus) StringPtr() *string {
	return stringPtrEnum(m)
}
func (m *CustomerRelationshipType) StringPtr() *string { return stringPtrEnum(m) }
func (m *EDIStatus) StringPtr() *string                { return stringPtrEnum(m) }
func (m *EmailSendStatus) StringPtr() *string          { return stringPtrEnum(m) }
func (m *RemovedResourceScope) StringPtr() *string     { return stringPtrEnum(m) }
