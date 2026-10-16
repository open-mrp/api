package main

import (
	"strings"

	"github.com/open-mrp/apikit/openapi"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

// routeSegmentToSampleID maps the route path segment immediately before a generic {id} parameter to a documentation sample ID from apiresource.
var routeSegmentToSampleID = map[string]string{
	"machine-downtime-events":      apiresource.SampleMachineDowntimeEventID,
	"demand-overrides":             apiresource.SampleDemandOverrideID,
	"notes":                        apiresource.SampleNoteID,
	"manufacturing-methods":        apiresource.SampleManufacturingMethodID,
	"operations":                   apiresource.SampleMethodOperationID,
	"production-schedules":         apiresource.SampleProductionScheduleID,
	"announcements":                apiresource.SampleAnnouncementID,
	"notifications":                apiresource.SampleNotificationID,
	"async-jobs":                   apiresource.SampleAsyncJobID,
	"units":                        apiresource.SampleUnitID,
	"unit-groups":                  apiresource.SampleUnitGroupID,
	"properties":                   apiresource.SamplePropertyID,
	"attributes":                   apiresource.SampleAttributeID,
	"items":                        apiresource.SampleItemID,
	"item-categories":              apiresource.SampleItemCategoryID,
	"products":                     apiresource.SampleProductID,
	"product-lines":                apiresource.SampleProductLineID,
	"product-types":                apiresource.SampleProductTypeID,
	"materials":                    apiresource.SampleMaterialID,
	"departments":                  apiresource.SampleWorkCenterID,
	"work-centers":                 apiresource.SampleWorkCenterID,
	"machines":                     apiresource.SampleMachineID,
	"api-keys":                     apiresource.SampleAPIKeyID,
	"addresses":                    apiresource.SampleCRUDAddressID,
	"account-groups":               apiresource.SampleAccountGroupID,
	"accounts":                     apiresource.SampleAccountID,
	"sales-orders":                 apiresource.SampleSalesOrderID,
	"picks":                        apiresource.SamplePickID,
	"shipments":                    apiresource.SampleShipmentID,
	"invoices":                     apiresource.SampleInvoiceID,
	"carriers":                     apiresource.SampleCarrierID,
	"sandboxes":                    apiresource.SampleSandboxID,
	"users":                        apiresource.SampleUserID,
	"roles":                        apiresource.SampleRoleID,
	"payment-terms":                apiresource.SamplePaymentTermID,
	"shipping-terms":               apiresource.SampleShippingTermID,
	"priorities":                   apiresource.SamplePriorityID,
	"adjustment-types":             apiresource.SampleAdjustmentTypeID,
	"account-statuses":             apiresource.SampleAccountStatusID,
	"location-types":               apiresource.SampleLocationTypeID,
	"batches":                      apiresource.SampleBatchID,
	"supplier-materials":           apiresource.SampleSupplierMaterialID,
	"parts":                        apiresource.SamplePartID,
	"customers":                    apiresource.SampleCustomerID,
	"suppliers":                    apiresource.SampleSupplierID,
	"order-discounts":              apiresource.SampleOrderDiscountID,
	"shipping-cases":               apiresource.SampleShippingCaseID,
	"purchase-orders":              apiresource.SamplePurchaseOrderID,
	"receiving-orders":             apiresource.SampleReceivingOrderID,
	"production-runs":              apiresource.SampleProductionRunID,
	"production-steps":             apiresource.SampleProductionStepID,
	"production-flows":             apiresource.SampleProductionID,
	"territories":                  apiresource.SampleTerritoryID,
	"scanning-stations":            apiresource.SampleScanningStationID,
	"permission-groups":            apiresource.SamplePermissionGroupID,
	"account-integrations":         apiresource.SampleAccountIntegrationID,
	"sync":                         apiresource.SampleHubspotSyncJobID,
	"account-prices":               apiresource.SampleAccountPriceID,
	"account-users":                apiresource.SampleAccountUserID,
	"agents":                       apiresource.SampleAgentDefinitionID,
	"agent-runs":                   apiresource.SampleAgentRunID,
	"memories":                     apiresource.SampleAgentMemoryID,
	"agent-memories":               apiresource.SampleAgentMemoryID,
	"audit-events":                 apiresource.SampleAuditEventID,
	"request-logs":                 apiresource.SampleRequestLogID,
	"email-logs":                   apiresource.SampleEmailLogID,
	"settlements":                  apiresource.SampleSettlementID,
	"transactions":                 apiresource.SampleTransactionDetailID,
	"transaction-allocations":      apiresource.SampleAllocationEntryID,
	"volume-discounts":             apiresource.SampleVolumeDiscountID,
	"registration-flows":           apiresource.SampleRegistrationFlowID,
	"deliveries":                   apiresource.SampleDeliveryID,
	"consumptions":                 apiresource.SampleConsumptionID,
	"enterprise-inquiries":         apiresource.SampleEnterpriseInquiryID,
	"sales-targets":                apiresource.SampleSalesTargetID,
	"service-levels":               apiresource.SampleServiceLevelID,
	"inventory-change-logs":        apiresource.SampleInventoryChangeLogID,
	"sales-order-statuses":         apiresource.SampleSalesOrderStatusID,
	"conversations":                apiresource.SampleConversationID,
	"messages":                     apiresource.SampleMessageID,
	"portal-domains":               apiresource.SamplePortalDomainID,
	"portal-registration-sessions": apiresource.SamplePortalRegistrationSessionID,
	"email-domains":                apiresource.SampleEmailDomainID,
	"email-inboxes":                apiresource.SampleEmailInboxID,
	"groups":                       apiresource.SampleMessagingGroupID,
}

// pathParamToSampleID maps non-generic path parameter names to sample IDs.
var pathParamToSampleID = map[string]string{
	"attribute_id":            apiresource.SampleAttributeID,
	"category_id":             apiresource.SampleItemCategoryID,
	"unit_group_id":           apiresource.SampleUnitGroupID,
	"property_id":             apiresource.SamplePropertyID,
	"production_step_id":      apiresource.SampleProductionStepID,
	"manufacturing_method_id": apiresource.SampleManufacturingMethodID,
	"child_account_id":        apiresource.SampleCustomerID,
	"receiving_order_id":      apiresource.SampleReceivingOrderID,
	"shipment_id":             apiresource.SampleShipmentID,
	"account_group_id":        apiresource.SampleAccountGroupID,
	"session_id":              apiresource.SampleRegistrationSessionID,
	"review_id":               apiresource.SampleHubspotCompanyReviewID,
	"carrier_id":              apiresource.SampleCarrierID,
	"pid":                     apiresource.SampleConversationParticipantID,
	"slug":                    apiresource.SampleAccountPortalSlug,
	"place_id":                "ChIJN1gggt_t2Z44AR4PVM_67p73Y",
	"member_id":               apiresource.SampleMessagingGroupMemberID,
}

// fieldNameSampleIDs maps request struct field names to sample IDs when they do not follow the Sample{FieldName} constant naming convention.
var fieldNameSampleIDs = map[string]string{
	"OperatingCalendarID":  apiresource.SampleOperatingCalendarID,
	"ClosureID":            apiresource.SampleOperatingCalendarClosureID,
	"APIKeyID":             apiresource.SampleAPIKeyID,
	"UserID":               apiresource.SampleUserID,
	"AccountID":            apiresource.SampleAccountID,
	"CustomerID":           apiresource.SampleCustomerID,
	"ProductID":            apiresource.SampleProductID,
	"PropertyID":           apiresource.SamplePropertyID,
	"AttributeID":          apiresource.SampleAttributeID,
	"AddressID":            apiresource.SampleCRUDAddressID,
	"RoleID":               apiresource.SampleRoleID,
	"BatchID":              apiresource.SampleBatchID,
	"CarrierID":            apiresource.SampleCarrierID,
	"InvoiceID":            apiresource.SampleInvoiceID,
	"ShipmentID":           apiresource.SampleShipmentID,
	"PickID":               apiresource.SamplePickID,
	"UnitID":               apiresource.SampleUnitID,
	"UnitGroupID":          apiresource.SampleUnitGroupID,
	"CategoryID":           apiresource.SampleItemCategoryID,
	"MaterialID":           apiresource.SampleMaterialID,
	"MachineID":            apiresource.SampleMachineID,
	"DepartmentID":         apiresource.SampleWorkCenterID,
	"WorkCenterID":         apiresource.SampleWorkCenterID,
	"LocationID":           apiresource.SampleLocationID,
	"SandboxID":            apiresource.SampleSandboxID,
	"AgentDefinitionID":    apiresource.SampleAgentDefinitionID,
	"AgentRunID":           apiresource.SampleAgentRunID,
	"ConsumptionID":        apiresource.SampleConsumptionID,
	"ProductionStepID":     apiresource.SampleProductionStepID,
	"ReceivingOrderID":     apiresource.SampleReceivingOrderID,
	"SettlementID":         apiresource.SampleSettlementID,
	"AllocationID":         apiresource.SampleAllocationEntryID,
	"AccountGroupID":       apiresource.SampleAccountGroupID,
	"AccountUserID":        apiresource.SampleAccountUserID,
	"AccountPriceID":       apiresource.SampleAccountPriceID,
	"AccountIntegrationID": apiresource.SampleAccountIntegrationID,
	// JobID is deliberately absent: both /core/jobs/{id} and the HubSpot sync job use it,
	// so the route segment ("jobs" vs "sync") is what tells the two apart.
	"ReviewID":           apiresource.SampleHubspotCompanyReviewID,
	"AccountStatusID":    apiresource.SampleAccountStatusID,
	"ProductLineID":      apiresource.SampleProductLineID,
	"ProductTypeID":      apiresource.SampleProductTypeID,
	"PaymentTermID":      apiresource.SamplePaymentTermID,
	"ShippingTermID":     apiresource.SampleShippingTermID,
	"PriorityID":         apiresource.SamplePriorityID,
	"AdjustmentTypeID":   apiresource.SampleAdjustmentTypeID,
	"OrderDiscountID":    apiresource.SampleOrderDiscountID,
	"VolumeDiscountID":   apiresource.SampleVolumeDiscountID,
	"RegistrationFlowID": apiresource.SampleRegistrationFlowID,
	"ScanningStationID":  apiresource.SampleScanningStationID,
	"TerritoryID":        apiresource.SampleTerritoryID,
	"DeliveryID":         apiresource.SampleDeliveryID,
	"PermissionGroupID":  apiresource.SamplePermissionGroupID,
	"ChildAccountID":     apiresource.SampleCustomerID,
	"AssociatedUnitID":   apiresource.SampleUnitID,
	"LineID":             apiresource.SampleSalesOrderLineID,
	"ConversationID":     apiresource.SampleConversationID,
	"MessageID":          apiresource.SampleMessageID,
	"ParticipantID":      apiresource.SampleConversationParticipantID,
	"PlaceID":            "ChIJN1gggt_t2Z44AR4PVM_67p73Y",
	"ID":                 "", // resolved from route segment when path param is "id"
}

func sampleIDFromFieldName(fieldName string) string {
	if v, ok := fieldNameSampleIDs[fieldName]; ok {
		return v
	}
	return ""
}

func sampleIDFromPathParam(pathParamName, route string) string {
	if id, ok := pathParamToSampleID[pathParamName]; ok {
		return id
	}
	if pathParamName != "id" {
		return ""
	}
	// System properties (/v1/settings/properties/{id}) collide on the "properties" segment with item-category properties (/v1/catalog/item-categories/{id}/properties/{id}), which a single-segment map can't distinguish — disambiguate by full route.
	if strings.HasPrefix(route, "/v1/settings/properties/") {
		return apiresource.SampleSysPropertyID
	}
	segment := openapi.RouteSegmentBefore(route, "{id}")
	if segment == "" {
		return ""
	}
	return routeSegmentToSampleID[segment]
}
