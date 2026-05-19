package main

import (
	"reflect"
	"strings"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/contracts"
)

// routeSegmentToSampleID maps the route path segment immediately before a generic
// {id} parameter to a documentation sample ID from apiresource.
var routeSegmentToSampleID = map[string]string{
	"units":                   apiresource.SampleUnitID,
	"unit-groups":             apiresource.SampleUnitGroupID,
	"properties":              apiresource.SamplePropertyID,
	"sys-properties":          apiresource.SampleSysPropertyID,
	"attributes":              apiresource.SampleAttributeID,
	"items":                   apiresource.SampleItemID,
	"item-categories":         apiresource.SampleItemCategoryID,
	"products":                apiresource.SampleProductID,
	"product-lines":           apiresource.SampleProductLineID,
	"product-types":           apiresource.SampleProductTypeID,
	"materials":               apiresource.SampleMaterialID,
	"departments":             apiresource.SampleDepartmentID,
	"machines":                apiresource.SampleMachineID,
	"api-keys":                apiresource.SampleAPIKeyID,
	"addresses":               apiresource.SampleCRUDAddressID,
	"account-groups":          apiresource.SampleAccountGroupID,
	"accounts":                apiresource.SampleAccountID,
	"sales-orders":            apiresource.SampleSalesOrderDetailID,
	"picks":                   apiresource.SamplePickDetailID,
	"shipments":               apiresource.SampleShipmentDetailID,
	"invoices":                apiresource.SampleInvoiceID,
	"carriers":                apiresource.SampleCarrierID,
	"sandboxes":               apiresource.SampleSandboxID,
	"users":                   apiresource.SampleUserID,
	"roles":                   apiresource.SampleRoleID,
	"payment-terms":           apiresource.SamplePaymentTermID,
	"shipping-terms":          apiresource.SampleShippingTermID,
	"priorities":              apiresource.SamplePriorityID,
	"adjustment-types":        apiresource.SampleAdjustmentTypeID,
	"account-statuses":        apiresource.SampleAccountStatusID,
	"locations":               apiresource.SampleLocationID,
	"batches":                 apiresource.SampleBatchID,
	"supplier-materials":      apiresource.SampleSupplierMaterialID,
	"parts":                   apiresource.SamplePartID,
	"customers":               apiresource.SampleCustomerID,
	"suppliers":               apiresource.SampleSupplierID,
	"order-discounts":         apiresource.SampleOrderDiscountID,
	"shipping-cases":          apiresource.SampleShippingCaseID,
	"purchase-orders":         apiresource.SamplePurchaseOrderDetailID,
	"receiving-orders":        apiresource.SampleReceivingOrderID,
	"production-runs":         apiresource.SampleProductionRunID,
	"production-steps":        apiresource.SampleProductionStepID,
	"production-flows":        apiresource.SampleProductionID,
	"territories":             apiresource.SampleTerritoryID,
	"scanning-stations":       apiresource.SampleScanningStationID,
	"permission-groups":       apiresource.SamplePermissionGroupID,
	"account-integrations":    apiresource.SampleAccountIntegrationID,
	"account-prices":          apiresource.SampleAccountPriceID,
	"account-users":           apiresource.SampleAccountUserID,
	"agents":                  apiresource.SampleAgentDefinitionID,
	"agent-alerts":            apiresource.SampleAgentAlertID,
	"agent-runs":              apiresource.SampleAgentRunID,
	"agent-memories":          apiresource.SampleAgentMemoryID,
	"audit-events":            apiresource.SampleAuditEventID,
	"request-logs":            apiresource.SampleRequestLogID,
	"email-logs":              apiresource.SampleEmailLogID,
	"settlements":             apiresource.SampleSettlementID,
	"transaction-allocations": apiresource.SampleAllocationEntryID,
	"volume-discounts":        apiresource.SampleVolumeDiscountID,
	"registration-flows":      apiresource.SampleRegistrationFlowID,
	"rates":                   apiresource.SampleRateID,
	"quantities":              apiresource.SampleQuantityID,
	"deliveries":              apiresource.SampleDeliveryID,
	"consumptions":            apiresource.SampleConsumptionID,
	"edi-dc-locations":        apiresource.SampleDCLocationID,
	"enterprise-inquiries":    apiresource.SampleEnterpriseInquiryID,
	"sales-targets":           apiresource.SampleSalesTargetID,
	"service-levels":          apiresource.SampleServiceLevelID,
	"inventory-change-logs":   apiresource.SampleInventoryChangeLogID,
	"sales-order-statuses":    apiresource.SampleSalesOrderStatusID,
}

// pathParamToSampleID maps non-generic path parameter names to sample IDs.
var pathParamToSampleID = map[string]string{
	"attribute_id":       apiresource.SampleAttributeID,
	"category_id":        apiresource.SampleItemCategoryID,
	"unit_group_id":      apiresource.SampleUnitGroupID,
	"property_id":        apiresource.SamplePropertyID,
	"production_step_id": apiresource.SampleProductionStepID,
	"child_account_id":   apiresource.SampleCustomerID,
	"receiving_order_id": apiresource.SampleReceivingOrderID,
	"shipment_id":        apiresource.SampleShipmentDetailID,
	"account_group_id":   apiresource.SampleAccountGroupID,
	"session_id":         apiresource.SampleRegistrationSessionID,
	"place_id":           "ChIJN1gggt_t2Z44AR4PVM_67p73Y",
}

// fieldNameSampleIDs maps request struct field names to sample IDs when they do
// not follow the Sample{FieldName} constant naming convention.
var fieldNameSampleIDs = map[string]string{
	"APIKeyID":             apiresource.SampleAPIKeyID,
	"ItemID":               apiresource.SampleItemID,
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
	"ShipmentID":           apiresource.SampleShipmentDetailID,
	"PickID":               apiresource.SamplePickDetailID,
	"UnitID":               apiresource.SampleUnitID,
	"UnitGroupID":          apiresource.SampleUnitGroupID,
	"CategoryID":           apiresource.SampleItemCategoryID,
	"MaterialID":           apiresource.SampleMaterialID,
	"MachineID":            apiresource.SampleMachineID,
	"DepartmentID":         apiresource.SampleDepartmentID,
	"LocationID":           apiresource.SampleLocationID,
	"SandboxID":            apiresource.SampleSandboxID,
	"AgentDefinitionID":    apiresource.SampleAgentDefinitionID,
	"AgentRunID":           apiresource.SampleAgentRunID,
	"AlertID":              apiresource.SampleAgentAlertID,
	"ConsumptionID":        apiresource.SampleConsumptionID,
	"ProductionStepID":     apiresource.SampleProductionStepID,
	"ReceivingOrderID":     apiresource.SampleReceivingOrderID,
	"SettlementID":         apiresource.SampleSettlementID,
	"AllocationID":         apiresource.SampleAllocationEntryID,
	"DCLocationID":         apiresource.SampleDCLocationID,
	"AccountGroupID":       apiresource.SampleAccountGroupID,
	"AccountUserID":        apiresource.SampleAccountUserID,
	"AccountPriceID":       apiresource.SampleAccountPriceID,
	"AccountIntegrationID": apiresource.SampleAccountIntegrationID,
	"AccountStatusID":      apiresource.SampleAccountStatusID,
	"ProductLineID":        apiresource.SampleProductLineID,
	"ProductTypeID":        apiresource.SampleProductTypeID,
	"PaymentTermID":        apiresource.SamplePaymentTermID,
	"ShippingTermID":       apiresource.SampleShippingTermID,
	"PriorityID":           apiresource.SamplePriorityID,
	"AdjustmentTypeID":     apiresource.SampleAdjustmentTypeID,
	"OrderDiscountID":      apiresource.SampleOrderDiscountID,
	"VolumeDiscountID":     apiresource.SampleVolumeDiscountID,
	"RegistrationFlowID":   apiresource.SampleRegistrationFlowID,
	"ScanningStationID":    apiresource.SampleScanningStationID,
	"TerritoryID":          apiresource.SampleTerritoryID,
	"DeliveryID":           apiresource.SampleDeliveryID,
	"RateID":               apiresource.SampleRateID,
	"QuantityID":           apiresource.SampleQuantityID,
	"PermissionGroupID":    apiresource.SamplePermissionGroupID,
	"ChildAccountID":       apiresource.SampleCustomerID,
	"LineID":               apiresource.SampleSalesOrderLineDetailID,
	"PlaceID":              "ChIJN1gggt_t2Z44AR4PVM_67p73Y",
	"ID":                   "", // resolved from route segment when path param is "id"
}

func pathParameterExample(reqType reflect.Type, field reflect.StructField, pathParamName, route string, paramSchema Schema) any {
	if v := pathValueFromSchemaExample(reqType, field); v != nil {
		return v
	}
	if v := sampleIDFromFieldName(field.Name); v != "" {
		return v
	}
	if v := sampleIDFromPathParam(pathParamName, route); v != "" {
		return v
	}
	return parameterExample(paramSchema)
}

func pathValueFromSchemaExample(reqType reflect.Type, field reflect.StructField) any {
	ptrType := reflect.PointerTo(reqType)
	if !ptrType.Implements(reflect.TypeFor[contracts.DocumentedType]()) {
		return nil
	}

	example := reflect.New(reqType).Interface().(contracts.DocumentedType).SchemaExample()
	if example == nil {
		return nil
	}

	v := reflect.ValueOf(example)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		fv := v.FieldByName(field.Name)
		if fv.IsValid() && fv.Kind() == reflect.String && fv.String() != "" {
			return fv.String()
		}
		return nil
	}

	m, ok := example.(map[string]any)
	if !ok {
		return nil
	}
	if val, ok := m[field.Name]; ok && val != nil {
		return val
	}
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		key := strings.Split(jsonTag, ",")[0]
		if key != "" && key != "-" {
			if val, ok := m[key]; ok && val != nil {
				return val
			}
		}
	}
	return nil
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
	segment := routeSegmentBeforeParam(route, "{id}")
	if segment == "" {
		return ""
	}
	return routeSegmentToSampleID[segment]
}

func routeSegmentBeforeParam(route, param string) string {
	idx := strings.Index(route, param)
	if idx <= 0 {
		return ""
	}
	prefix := strings.TrimSuffix(route[:idx], "/")
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		return prefix[i+1:]
	}
	return prefix
}
