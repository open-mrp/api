package main

import (
	"fmt"
	"reflect"
	"strings"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
)

func queryParameterExample(reqType reflect.Type, field reflect.StructField, paramSchema Schema) any {
	if v := queryValueFromSchemaExample(reqType, field); v != nil && queryExampleSatisfiesEnum(v, paramSchema) {
		return v
	}
	queryTag := field.Tag.Get("query")
	if queryTag != "" {
		paramName := queryTag
		if paramSchema.Type == "array" && !strings.HasSuffix(paramName, "[]") {
			paramName = paramName + "[]"
		}
		if v := sampleQueryExampleForOpenAPIName(paramName); v != nil && queryExampleSatisfiesEnum(v, paramSchema) {
			return v
		}
	}
	return parameterExample(paramSchema)
}

// queryExampleSatisfiesEnum reports whether v is consistent with paramSchema's enum constraint, if it has one. A generic documented/hard-coded sample that is not a member of the parameter's enum (e.g. a `type` sample of "payment" applied to an endpoint whose `type` enum is [material_category, product_category]) must be rejected so the enum-aware fallback can substitute a valid value.
func queryExampleSatisfiesEnum(v any, paramSchema Schema) bool {
	if paramSchema.Type == "array" && paramSchema.Items != nil && len(paramSchema.Items.Enum) > 0 {
		arr, ok := v.([]any)
		if !ok {
			return false
		}
		for _, e := range arr {
			if !enumContains(paramSchema.Items.Enum, e) {
				return false
			}
		}
		return true
	}
	if len(paramSchema.Enum) > 0 {
		return enumContains(paramSchema.Enum, v)
	}
	return true
}

func enumContains(enum []any, v any) bool {
	for _, e := range enum {
		if fmt.Sprint(e) == fmt.Sprint(v) {
			return true
		}
	}
	return false
}

func queryValueFromSchemaExample(reqType reflect.Type, field reflect.StructField) any {
	queryTag := field.Tag.Get("query")
	if queryTag == "" {
		return nil
	}
	key := strings.TrimSuffix(queryTag, "[]")
	return lookupQueryInDocumentedTypes(reqType, key)
}

func lookupQueryInDocumentedTypes(reqType reflect.Type, queryKey string) any {
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}
	if reqType.Kind() != reflect.Struct {
		return nil
	}
	if v := tryQueryFromDocumentedType(reqType, queryKey); v != nil {
		return v
	}
	t := reqType
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.Anonymous {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}
		if v := lookupQueryInDocumentedTypes(ft, queryKey); v != nil {
			return v
		}
	}
	return nil
}

func tryQueryFromDocumentedType(t reflect.Type, queryKey string) any {
	ptr := reflect.PointerTo(t)
	if !ptr.Implements(reflect.TypeFor[contracts.DocumentedType]()) {
		return nil
	}
	ex := reflect.New(t).Interface().(contracts.DocumentedType).SchemaExample()
	if ex == nil {
		return nil
	}
	m, ok := ex.(map[string]any)
	if !ok {
		return nil
	}
	if v, ok := m[queryKey]; ok && !isEmptyQueryExampleValue(v) {
		return v
	}
	return nil
}

func isEmptyQueryExampleValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok && s == "" {
		return true
	}
	if arr, ok := v.([]any); ok && len(arr) == 0 {
		return true
	}
	return false
}

// sampleQueryExampleForOpenAPIName documents coherent defaults after optional PaginationRequest[] / SchemaExample lookup misses.
func sampleQueryExampleForOpenAPIName(openAPIParam string) any {
	base := strings.TrimSuffix(openAPIParam, "[]")
	switch base {
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
		return "sess_augno_docs_example"
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
	case "topic_resource_type":
		return string(constants.ObjectTypeSalesOrder)
	case "topic_resource_id":
		return apiresource.SampleSalesOrderID
	case "agent_definition_id":
		return apiresource.SampleAgentDefinitionID
	case "trend_type":
		return "sales_velocity"
	case "normalized_routes":
		return []any{"/v1/catalog/items"}
	case "hosts":
		return []any{"api.augno.com"}
	case "status_codes":
		return []any{float64(200)}
	case "status_code_classes":
		return []any{float64(5)}
	case "scope_codes":
		return []any{"item"}
	case "override_type_codes":
		return []any{"delta_units"}
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
		"attribute_ids":        apiresource.SampleAttributeID,
		"unit_group_ids":       apiresource.SampleUnitGroupID,
		"customer_group_ids":   apiresource.SampleAccountGroupID,
		"pricing_group_ids":    apiresource.SampleAccountGroupID,
		"sales_rep_ids":        apiresource.SampleAccountUserID,
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
