package main

import (
	"strings"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

// sampleQueryExampleForOpenAPIName documents coherent defaults after optional PaginationRequest[] / SchemaExample lookup misses.
func sampleQueryExampleForOpenAPIName(openAPIParam string) any {
	base := strings.TrimSuffix(openAPIParam, "[]")
	switch base {
	case "cursor":
		return pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleItemID)
	case "limit":
		return int64(25)
	case "starts_at":
		return apiresource.SampleFilterStartDateRFC3339
	case "ends_at":
		return apiresource.SampleFilterEndDateRFC3339
	case "ship_by_after", "ship_by_before":
		return apiresource.SampleFilterDateOnly
	case "from_date", "to_date":
		return apiresource.SampleFilterDateOnly
	case "cutoff_at":
		return apiresource.SampleFilterEndDateRFC3339
	case "as_of":
		return apiresource.SampleFilterEndDateRFC3339
	case "category":
		return "preference"
	case "entity_type":
		return "customer"
	case "transaction_type":
		return string(constants.TransactionTypePayment)
	case "type":
		return string(constants.TransactionTypePayment)
	case "q":
		return "6061"
	case "input":
		return "1 Ferry Building"
	case "session_token":
		return "sess_openmrp_docs_example"
	case "idempotency_key":
		return apiresource.SampleRequestLogID
	case "city":
		return "San Francisco"
	case "state":
		return "CA"
	case "postal_code":
		return "94103"
	case "status":
		return "shipped"
	case "recipient_account_id":
		return apiresource.SampleAccountID
	case "week_index":
		// Mid-horizon rather than zero: a zero example reads as "unset" in a filter and would not show that the parameter narrows anything.
		return 2
	case "item_id":
		return apiresource.SampleItemID
	case "supplier_id":
		return apiresource.SampleSupplierID
	case "assignee_resource_id":
		return apiresource.SampleAccountUserID
	case "resource_id":
		return apiresource.SampleSalesOrderID
	case "record_type":
		return string(constants.RecordTypeSalesOrder)
	case "record_id":
		return apiresource.SampleSalesOrderID
	case "agent_definition_id":
		return apiresource.SampleAgentDefinitionID
	case "trend_type":
		return "sales_velocity"
	case "normalized_routes":
		return []any{"/v1/catalog/items"}
	case "hosts":
		return []any{"api.openmrp.ai"}
	case "status_codes":
		return []any{float64(200)}
	case "status_code_classes":
		return []any{float64(5)}
	case "scope_codes":
		return []any{"item"}
	case "override_type_codes":
		return []any{"delta_units"}
	case "numbers":
		return []any{"INV-001"}
	case "reasons":
		return []any{"breakdown"}
	case "reason":
		return "rush_order"
	case "reason_note":
		return "Pulled forward for the Northwind rush order."
	}

	idSamples := map[string]string{
		"customer_ids":         apiresource.SampleCustomerID,
		"supplier_ids":         apiresource.SampleSupplierID,
		"department_ids":       apiresource.SampleDepartmentID,
		"product_line_ids":     apiresource.SampleProductLineID,
		"category_ids":         apiresource.SampleItemCategoryID,
		"item_category_ids":    apiresource.SampleItemCategoryID,
		"attribute_ids":        apiresource.SampleAttributeID,
		"unit_group_ids":       apiresource.SampleUnitGroupID,
		"customer_group_ids":   apiresource.SampleAccountGroupID,
		"pricing_group_ids":    apiresource.SampleAccountGroupID,
		"sales_rep_ids":        apiresource.SampleAccountUserID,
		"participant_ids":      apiresource.SampleAccountUserID,
		"shipping_term_ids":    apiresource.SampleShippingTermID,
		"payment_term_ids":     apiresource.SamplePaymentTermID,
		"carrier_ids":          apiresource.SampleCarrierID,
		"service_level_ids":    apiresource.SampleServiceLevelID,
		"account_ids":          apiresource.SampleAccountID,
		"actor_account_ids":    apiresource.SampleAccountID,
		"target_account_ids":   apiresource.SampleAccountID,
		"actor_ids":            apiresource.SampleUserID,
		"resource_ids":         apiresource.SampleAuditEventResourceID,
		"transaction_ids":      apiresource.SampleTransactionDetailID,
		"invoice_ids":          apiresource.SampleInvoiceID,
		"item_ids":             apiresource.SampleItemID,
		"machine_ids":          apiresource.SampleMachineID,
		"scope_ref_ids":        apiresource.SampleItemID,
		"root_resource_id":     apiresource.SampleAuditEventResourceID,
		"scanning_station_ids": apiresource.SampleScanningStationID,
		"sender_ids":           apiresource.SampleAccountUserID,
		"input_step_ids":       apiresource.SampleProductionStepID,
		"output_step_ids":      apiresource.SampleProductionStepID,
		"customer":             apiresource.SampleCustomerID,
		"relation_account_id":  apiresource.SampleCustomerID,
	}
	if sid, ok := idSamples[base]; ok {
		return []any{sid}
	}

	codeListSamples := map[string][]any{
		"types":               {string(constants.TransactionTypePayment)},
		"adjustment_types":    {apiresource.SampleAdjustmentTypeCode},
		"methods":             {string(constants.TransactionMethodCreditCard)},
		"action_type_codes":   {"scan"},
		"changed_by_user_ids": {apiresource.SampleUserID},
	}
	if v, ok := codeListSamples[base]; ok {
		return v
	}

	return nil
}
