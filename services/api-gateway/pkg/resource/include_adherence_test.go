package apiresource

import (
	"reflect"
	"strings"
	"testing"

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
var includeAdherenceResources = map[constants.ObjectType]any{
	constants.ObjectTypeAccount:                   Account{},
	constants.ObjectTypeAccountPrice:              AccountPrice{},
	constants.ObjectTypeAccountUser:               AccountUser{},
	constants.ObjectTypeAgentDefinition:           AgentDefinition{},
	constants.ObjectTypeAgentRun:                  AgentRun{},
	constants.ObjectTypeAPIKey:                    APIKey{},
	constants.ObjectTypeAuditEvent:                AuditEvent{},
	constants.ObjectTypeCarrier:                   Carrier{},
	constants.ObjectTypeConsumption:               Consumption{},
	constants.ObjectTypeCustomer:                  Customer{},
	constants.ObjectTypeDepartment:                Department{},
	constants.ObjectTypeEmailInbox:                EmailInbox{},
	constants.ObjectTypeEmailSender:               EmailSender{},
	constants.ObjectTypeInvoice:                   Invoice{},
	constants.ObjectTypeItem:                      Item{},
	constants.ObjectTypeItemCategory:              ItemCategory{},
	constants.ObjectTypeMachine:                   Machine{},
	constants.ObjectTypeMachineItemScanSuggestion: MachineItemScanSuggestion{},
	constants.ObjectTypeMaterial:                  Material{},
	constants.ObjectTypeOwner:                     Owner{},
	constants.ObjectTypePart:                      Part{},
	constants.ObjectTypePick:                      Pick{},
	constants.ObjectTypeProduct:                   Product{},
	constants.ObjectTypeProductLine:               ProductLine{},
	constants.ObjectTypeProductionRun:             ProductionRun{},
	constants.ObjectTypeProperty:                  Property{},
	constants.ObjectTypePurchaseOrder:             PurchaseOrder{},
	constants.ObjectTypeRequestLog:                RequestLog{},
	constants.ObjectTypeRole:                      Role{},
	constants.ObjectTypeSalesOrder:                SalesOrder{},
	constants.ObjectTypeSandbox:                   Sandbox{},
	constants.ObjectTypeScanningStation:           ScanningStation{},
	constants.ObjectTypeSettlement:                Settlement{},
	constants.ObjectTypeShipment:                  Shipment{},
	constants.ObjectTypeShippingCase:              ShippingCase{},
	constants.ObjectTypeShippingTerm:              ShippingTerm{},
	constants.ObjectTypeSupplier:                  Supplier{},
	constants.ObjectTypeSupplierMaterial:          SupplierMaterial{},
	constants.ObjectTypeToolGroup:                 ToolGroup{},
	constants.ObjectTypeTransaction:               TransactionDetail{},
	constants.ObjectTypeUnitGroup:                 UnitGroup{},
	constants.ObjectTypeVolumeDiscount:            VolumeDiscount{},
}

// TestIncludeRegistry_AllExpandableFieldsRegistered verifies that every resource
// field tagged expandable:"true" has a corresponding entry in the include
// registry. This prevents startup panics when IncludesFor references a field
// that was never registered.
func TestIncludeRegistry_AllExpandableFieldsRegistered(t *testing.T) {
	t.Parallel()
	for objectType, resource := range includeAdherenceResources {
		rt := reflect.TypeOf(resource)
		if rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}

		oi := includeRegistry[objectType]
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
