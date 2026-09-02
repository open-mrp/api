package apiendpoint

import (
	"reflect"
	"strings"
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

// resourceRegistry maps each ObjectType that has an include registration to a
// zero instance of the corresponding API resource struct. When you add a new
// resource type with expandable:"true" fields AND register it in
// include_definitions.go, add an entry here so the adherence test can verify
// every expandable field has a matching include registration.
//
// If this test fails, it means a resource field is tagged expandable:"true" but
// missing from RegisterIncludes in include_definitions.go — which will cause a
// startup panic if any endpoint tries to include that field.
var resourceRegistry = map[constants.ObjectType]any{
	constants.ObjectTypeAccount:          apiresource.Account{},
	constants.ObjectTypeAccountPrice:     apiresource.AccountPrice{},
	constants.ObjectTypeAccountUser:      apiresource.AccountUser{},
	constants.ObjectTypeAgentDefinition:  apiresource.AgentDefinition{},
	constants.ObjectTypeAgentRun:         apiresource.AgentRun{},
	constants.ObjectTypeAPIKey:           apiresource.APIKey{},
	constants.ObjectTypeAuditEvent:       apiresource.AuditEvent{},
	constants.ObjectTypeCarrier:          apiresource.Carrier{},
	constants.ObjectTypeConsumption:      apiresource.Consumption{},
	constants.ObjectTypeCustomer:         apiresource.Customer{},
	constants.ObjectTypeDepartment:       apiresource.Department{},
	constants.ObjectTypeEmailInbox:       apiresource.EmailInbox{},
	constants.ObjectTypeInvoice:          apiresource.Invoice{},
	constants.ObjectTypeItem:             apiresource.Item{},
	constants.ObjectTypeItemCategory:     apiresource.ItemCategory{},
	constants.ObjectTypeMachine:          apiresource.Machine{},
	constants.ObjectTypeMaterial:         apiresource.Material{},
	constants.ObjectTypeOwner:            apiresource.Owner{},
	constants.ObjectTypePart:             apiresource.Part{},
	constants.ObjectTypePick:             apiresource.Pick{},
	constants.ObjectTypeProduct:          apiresource.Product{},
	constants.ObjectTypeProductLine:      apiresource.ProductLine{},
	constants.ObjectTypeProductionRun:    apiresource.ProductionRun{},
	constants.ObjectTypeProperty:         apiresource.Property{},
	constants.ObjectTypePurchaseOrder:    apiresource.PurchaseOrder{},
	constants.ObjectTypeQuantity:         apiresource.Quantity{},
	constants.ObjectTypeRate:             apiresource.Rate{},
	constants.ObjectTypeRequestLog:       apiresource.RequestLog{},
	constants.ObjectTypeRole:             apiresource.Role{},
	constants.ObjectTypeSalesOrder:       apiresource.SalesOrder{},
	constants.ObjectTypeSandbox:          apiresource.Sandbox{},
	constants.ObjectTypeScanningStation:  apiresource.ScanningStation{},
	constants.ObjectTypeSettlement:       apiresource.Settlement{},
	constants.ObjectTypeShipment:         apiresource.Shipment{},
	constants.ObjectTypeShippingCase:     apiresource.ShippingCase{},
	constants.ObjectTypeShippingTerm:     apiresource.ShippingTerm{},
	constants.ObjectTypeSupplier:         apiresource.Supplier{},
	constants.ObjectTypeSupplierMaterial: apiresource.SupplierMaterial{},
	constants.ObjectTypeToolGroup:        apiresource.ToolGroup{},
	constants.ObjectTypeTransaction:      apiresource.TransactionDetail{},
	constants.ObjectTypeUnitGroup:        apiresource.UnitGroup{},
	constants.ObjectTypeVolumeDiscount:   apiresource.VolumeDiscount{},
}

// TestIncludeRegistry_AllExpandableFieldsRegistered verifies that every resource
// field tagged expandable:"true" has a corresponding entry in the include
// registry. This prevents startup panics when IncludesFor references a field
// that was never registered.
func TestIncludeRegistry_AllExpandableFieldsRegistered(t *testing.T) {
	t.Parallel()
	for objectType, resource := range resourceRegistry {
		rt := reflect.TypeOf(resource)
		if rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}

		oi := registry[objectType]
		if oi == nil {
			expandable := collectExpandableJSONNames(rt)
			if len(expandable) > 0 {
				t.Errorf("object type %q has expandable fields %v but no include registration in include_definitions.go", objectType, expandable)
			}
			continue
		}

		registeredKeys := make(map[string]bool)
		for _, field := range oi.Fields {
			registeredKeys[field.Key] = true
		}

		for field := range rt.Fields() {
			if field.Tag.Get("expandable") != "true" {
				continue
			}

			jsonName := field.Tag.Get("json")
			if idx := strings.Index(jsonName, ","); idx != -1 {
				jsonName = jsonName[:idx]
			}
			if jsonName == "" || jsonName == "-" {
				continue
			}

			if !registeredKeys[jsonName] {
				t.Errorf("object type %q: field %q (struct field %s) is tagged expandable:\"true\" but is not registered in include_definitions.go",
					objectType, jsonName, field.Name)
			}
		}
	}
}

// collectExpandableJSONNames returns the JSON names of all expandable fields.
func collectExpandableJSONNames(rt reflect.Type) []string {
	var names []string
	for field := range rt.Fields() {
		if field.Tag.Get("expandable") != "true" {
			continue
		}
		jsonName := field.Tag.Get("json")
		if idx := strings.Index(jsonName, ","); idx != -1 {
			jsonName = jsonName[:idx]
		}
		if jsonName != "" && jsonName != "-" {
			names = append(names, jsonName)
		}
	}
	return names
}
