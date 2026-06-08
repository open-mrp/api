package constants

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// enumTypeInfo holds parsed information about a string enum type.
type enumTypeInfo struct {
	Name      string
	Constants []string
}

// parseEnumTypes scans all non-test .go files in the constants package directory
// and returns every `type X string` declaration that has 2+ associated constants.
func parseEnumTypes(t *testing.T) []enumTypeInfo {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", entry.Name(), err)
		}
		files = append(files, f)
	}

	// Collect all type X string declarations.
	stringTypes := map[string]bool{}
	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				ident, ok := typeSpec.Type.(*ast.Ident)
				if ok && ident.Name == "string" {
					stringTypes[typeSpec.Name.Name] = true
				}
			}
		}
	}

	// Collect constants for each string type.
	typeConstants := map[string][]string{}
	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				if valueSpec.Type == nil {
					continue
				}
				ident, ok := valueSpec.Type.(*ast.Ident)
				if !ok {
					continue
				}
				typeName := ident.Name
				if !stringTypes[typeName] {
					continue
				}
				for _, name := range valueSpec.Names {
					typeConstants[typeName] = append(typeConstants[typeName], name.Name)
				}
			}
		}
	}

	// Filter to types with 2+ constants.
	var result []enumTypeInfo
	for typeName, consts := range typeConstants {
		if len(consts) >= 2 {
			result = append(result, enumTypeInfo{Name: typeName, Constants: consts})
		}
	}
	return result
}

// typeRegistry maps type names to a zero-value instance that can be used for
// reflection-based method calls.
var typeRegistry = map[string]any{
	"ActorType":                       ActorType(""),
	"RecordType":                      RecordType(""),
	"HTTPMethod":                      HTTPMethod(""),
	"CommissionPolicy":                CommissionPolicy(""),
	"AccountRelationNotificationType": AccountRelationNotificationType(""),
	"FreightPolicy":                   FreightPolicy(""),
	"AuditAction":                     AuditAction(""),
	"AccountGroupType":                AccountGroupType(""),
	"AccountMode":                     AccountMode(""),
	"AccountStatusCode":               AccountStatusCode(""),
	"AccountTypeCode":                 AccountTypeCode(""),
	"AccountUserStatus":               AccountUserStatus(""),
	"AgentAccountStatus":              AgentAccountStatus(""),
	"AgentActionStatus":               AgentActionStatus(""),
	"AgentAlertSeverity":              AgentAlertSeverity(""),
	"AgentAlertStatus":                AgentAlertStatus(""),
	"AgentDefinitionType":             AgentDefinitionType(""),
	"AgentTriggerType":                AgentTriggerType(""),
	"APIKeyStatus":                    APIKeyStatus(""),
	"DashboardPath":                   DashboardPath(""),
	"DeletedRecordResourceType":       DeletedRecordResourceType(""),
	"EmailTemplate":                   EmailTemplate(""),
	"IntegrationCode":                 IntegrationCode(""),
	"ObjectType":                      ObjectType(""),
	"PlanCode":                        PlanCode(""),
	"PlatformMode":                    PlatformMode(""),
	"Protocol":                        Protocol(""),
	"PublicPlanCode":                  PublicPlanCode(""),
	"RegistrationStep":                RegistrationStep(""),
	"RoleType":                        RoleType(""),
	"SandboxMode":                     SandboxMode(""),
	"ShipmentStatus":                  ShipmentStatus(""),
	"ShippingTermType":                ShippingTermType(""),
	"SubscriptionStatus":              SubscriptionStatus(""),
	"SubassemblyFilter":               SubassemblyFilter(""),
	"SupplierMaterialStatus":          SupplierMaterialStatus(""),
	"SysPropertyTypeCode":             SysPropertyTypeCode(""),
	"UnitType":                        UnitType(""),
	"CarrierCode":                     CarrierCode(""),
	"ScanningStationType":             ScanningStationType(""),
	"ItemCategoryType":                ItemCategoryType(""),
	"CarrierBillingType":              CarrierBillingType(""),
	"AddressType":                     AddressType(""),
	"AddressValidationStatus":         AddressValidationStatus(""),
	"CustomerParentAccountStatus":     CustomerParentAccountStatus(""),
	"CustomerRelationshipType":        CustomerRelationshipType(""),
	"EDIStatus":                       EDIStatus(""),
	"EmailSendStatus":                 EmailSendStatus(""),
	"RemovedResourceScope":            RemovedResourceScope(""),
	"SalesOrderStatusCode":            SalesOrderStatusCode(""),
	"SalesOrderStatusChange":          SalesOrderStatusChange(""),
	"SalesOrderPaymentStatus":         SalesOrderPaymentStatus(""),
	"InvoicePaymentStatus":            InvoicePaymentStatus(""),
	"AcknowledgmentStatus":            AcknowledgmentStatus(""),
	"OrderDiscountType":               OrderDiscountType(""),
	"AdjustmentType":                  AdjustmentType(""),
	"Color":                           Color(""),
	"TransactionMethod":               TransactionMethod(""),
	"TransactionType":                 TransactionType(""),
	"AgentRunStatus":                  AgentRunStatus(""),
	"DeliveryStatus":                  DeliveryStatus(""),
	"ItemTypeCode":                    ItemTypeCode(""),
	"InventoryActionType":             InventoryActionType(""),
	"InventoryUpdateOperation":        InventoryUpdateOperation(""),
	"ToolSlug":                        ToolSlug(""),
	"PriorityCode":                    PriorityCode(""),
	"PaymentTermStatus":               PaymentTermStatus(""),
	"OwnerType":                       OwnerType(""),
	"CustomerPortalVisibility":        CustomerPortalVisibility(""),
	"LocationTypeCode":                LocationTypeCode(""),
	"LabelSizeCode":                   LabelSizeCode(""),
	"LabelTypeCode":                   LabelTypeCode(""),
	"ProductTypeCode":                 ProductTypeCode(""),
	"OperatorRequirement":             OperatorRequirement(""),
}

func TestAdherence_AllEnumTypesHaveIsValid(t *testing.T) {
	t.Parallel()
	enumTypes := parseEnumTypes(t)
	if len(enumTypes) == 0 {
		t.Fatal("no enum types found — parser may be broken")
	}

	for _, et := range enumTypes {
		t.Run(et.Name, func(t *testing.T) {
			instance, ok := typeRegistry[et.Name]
			if !ok {
				t.Fatalf("type %s not registered in typeRegistry — add it to constants_adherence_test.go", et.Name)
			}

			v := reflect.ValueOf(instance)
			method := v.MethodByName("IsValid")
			if !method.IsValid() {
				t.Errorf("type %s is missing IsValid() method", et.Name)
			}
		})
	}
}

func TestAdherence_AllEnumTypesHaveEnumValues(t *testing.T) {
	t.Parallel()
	enumTypes := parseEnumTypes(t)

	for _, et := range enumTypes {
		t.Run(et.Name, func(t *testing.T) {
			instance, ok := typeRegistry[et.Name]
			if !ok {
				t.Fatalf("type %s not registered in typeRegistry — add it to constants_adherence_test.go", et.Name)
			}

			v := reflect.ValueOf(instance)
			method := v.MethodByName("EnumValues")
			if !method.IsValid() {
				t.Errorf("type %s is missing EnumValues() method", et.Name)
			}
		})
	}
}

func TestAdherence_IsValidMatchesEnumValues(t *testing.T) {
	t.Parallel()
	enumTypes := parseEnumTypes(t)

	for _, et := range enumTypes {
		t.Run(et.Name, func(t *testing.T) {
			instance, ok := typeRegistry[et.Name]
			if !ok {
				t.Skipf("type %s not registered in typeRegistry", et.Name)
			}

			v := reflect.ValueOf(instance)

			enumValuesMethod := v.MethodByName("EnumValues")
			if !enumValuesMethod.IsValid() {
				t.Skipf("type %s missing EnumValues()", et.Name)
			}

			results := enumValuesMethod.Call(nil)
			values := results[0].Interface().([]string)

			// EnumValues length should match the number of constants.
			if len(values) != len(et.Constants) {
				t.Errorf("type %s: EnumValues() returned %d values but has %d constants",
					et.Name, len(values), len(et.Constants))
			}

			// Every value from EnumValues should pass IsValid.
			for _, val := range values {
				typedVal := reflect.New(v.Type()).Elem()
				typedVal.SetString(val)
				isValidMethod := typedVal.MethodByName("IsValid")
				if !isValidMethod.IsValid() {
					t.Skipf("type %s missing IsValid()", et.Name)
				}
				result := isValidMethod.Call(nil)
				if !result[0].Bool() {
					t.Errorf("type %s: IsValid() returned false for EnumValues() value %q", et.Name, val)
				}
			}
		})
	}
}

func TestAdherence_AllEnumTypesHaveStringPtr(t *testing.T) {
	t.Parallel()
	enumTypes := parseEnumTypes(t)

	for _, et := range enumTypes {
		t.Run(et.Name, func(t *testing.T) {
			instance, ok := typeRegistry[et.Name]
			if !ok {
				t.Fatalf("type %s not registered in typeRegistry — add it to constants_adherence_test.go", et.Name)
			}

			v := reflect.ValueOf(instance)
			ptr := reflect.New(v.Type()) // *T so it matches the pointer receiver.

			method := ptr.MethodByName("StringPtr")
			if !method.IsValid() {
				t.Errorf("type %s is missing StringPtr() method", et.Name)
			}
		})
	}
}

func TestAdherence_StringPtrMatchesEnumValues(t *testing.T) {
	t.Parallel()
	enumTypes := parseEnumTypes(t)

	for _, et := range enumTypes {
		t.Run(et.Name, func(t *testing.T) {
			instance, ok := typeRegistry[et.Name]
			if !ok {
				t.Fatalf("type %s not registered in typeRegistry — add it to constants_adherence_test.go", et.Name)
			}

			v := reflect.ValueOf(instance)

			enumValuesMethod := v.MethodByName("EnumValues")
			if !enumValuesMethod.IsValid() {
				t.Skipf("type %s missing EnumValues()", et.Name)
			}

			results := enumValuesMethod.Call(nil)
			values := results[0].Interface().([]string)

			// Nil pointer should produce nil *string.
			ptrType := reflect.New(v.Type()).Type() // *T
			nilPtr := reflect.Zero(ptrType)
			stringPtrNilMethod := nilPtr.MethodByName("StringPtr")
			if !stringPtrNilMethod.IsValid() {
				t.Skipf("type %s missing StringPtr()", et.Name)
			}

			gotNil := stringPtrNilMethod.Call(nil)[0]
			if !gotNil.IsNil() {
				t.Errorf("type %s: StringPtr() expected nil for nil receiver, got %v", et.Name, gotNil)
			}

			// Each EnumValues() string should map to the correct *string.
			for _, val := range values {
				typedVal := reflect.New(v.Type()).Elem()
				typedVal.SetString(val)

				ptrVal := typedVal.Addr()
				stringPtrMethod := ptrVal.MethodByName("StringPtr")
				if !stringPtrMethod.IsValid() {
					t.Skipf("type %s missing StringPtr() on pointer receiver", et.Name)
				}

				got := stringPtrMethod.Call(nil)[0]
				if got.IsNil() {
					t.Errorf("type %s: StringPtr() returned nil for %q", et.Name, val)
					continue
				}
				if got.Elem().String() != val {
					t.Errorf("type %s: StringPtr() expected %q, got %q", et.Name, val, got.Elem().String())
				}
			}
		})
	}
}

func TestAdherence_TypeRegistryCoversAllEnumTypes(t *testing.T) {
	t.Parallel()
	enumTypes := parseEnumTypes(t)

	for _, et := range enumTypes {
		t.Run(et.Name, func(t *testing.T) {
			if _, ok := typeRegistry[et.Name]; !ok {
				t.Errorf("type %s has %d constants but is not in typeRegistry — add it to constants_adherence_test.go",
					et.Name, len(et.Constants))
			}
		})
	}
}

func TestAdherence_NoExtraFilesWithoutTest(t *testing.T) {
	t.Parallel(
	// Verify that every non-test .go file in the constants directory is parseable.
	)

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		_, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Errorf("failed to parse %s: %v", entry.Name(), err)
		}
	}
}
