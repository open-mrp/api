package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/field"

	invoiceep "github.com/augno/api/services/api-gateway/endpoints/invoices"
)

func TestMain(m *testing.M) {
	quietOutput = true
	os.Exit(m.Run())
}

type NestedStruct struct {
	// NestedField doc
	NestedField string `json:"nested_field"`
}

type DocumentedStruct struct {
	Value string `json:"value"`
}

func (d DocumentedStruct) SchemaExample() any {
	return DocumentedStruct{Value: "example"}
}

type PointersStruct struct {
	Name   *string       `json:"name"`
	Nested *NestedStruct `json:"nested"`
}

type OptionalPointerStruct struct {
	Optional *string `json:"optional,omitempty"`
}

// OptionalSliceStruct exercises slice + omitempty (must not appear in schema Required).
type OptionalSliceStruct struct {
	OptionalIDs []string `json:"optional_ids,omitempty"`
	Name        string   `json:"name" validate:"required"`
}

type ResponseNullableStruct struct {
	Description *string `json:"description"`
}

type patchQuantityInput struct {
	Value  float64 `json:"value"`
	UnitID string  `json:"unit_id"`
}

// PatchFieldStruct exercises field.Clearable[T] OpenAPI generation.
type PatchFieldStruct struct {
	Name        *string                             `json:"name,omitempty"`
	Description field.Clearable[string]             `json:"description,omitzero"`
	FlatRate    field.Clearable[patchQuantityInput] `json:"flat_rate,omitzero"`
	Tags        field.Clearable[[]string]           `json:"tags,omitzero"`
}

// OptionalInputStruct exercises field.Optional[T] OpenAPI generation.
type OptionalInputStruct struct {
	Optional    *string                `json:"optional,omitempty"`
	Description field.Optional[string] `json:"description,omitzero"`
	// A field.Optional[T] is always an optional input; even a stray
	// validate:"required" must not make it required in the schema.
	Mandatory field.Optional[string] `json:"mandatory,omitzero" validate:"required"`
}

type TestSchemaStruct struct {
	// Name doc
	Name string `json:"name" validate:"required"`
	// Items doc
	Items []string `json:"items"`
	// Nested doc
	Nested NestedStruct `json:"nested"`
	// Tags doc
	Tags map[string]string `json:"tags"`
	// Score doc
	Score float64 `json:"score"`
	// Status doc
	Status string `json:"status" enum:"active,inactive"`
	// CreatedAt doc
	CreatedAt string `json:"created_at" readOnly:"true"`
	// Count doc
	Count int `json:"count" default:"0"`
}

type TestRawMessageStruct struct {
	Raw json.RawMessage `json:"raw"`
}

type NullableFromExampleStruct struct {
	Raw json.RawMessage `json:"raw"`
}

func (*NullableFromExampleStruct) SchemaExample() any {
	return map[string]any{
		"raw": nil,
	}
}

// ExampleNullMismatchStruct documents a `null` example for a field whose Go type
// is non-nullable (a plain string). The example contradicts the type, which must
// fail schema generation loudly rather than be silently coerced to nullable.
type ExampleNullMismatchStruct struct {
	Name string `json:"name"`
}

func (*ExampleNullMismatchStruct) SchemaExample() any {
	return map[string]any{
		"name": nil,
	}
}

func TestGetCleanTypeName(t *testing.T) {
	t.Parallel()
	type GenericStruct[T any] struct {
		Data T
	}

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"Normal struct", TestSchemaStruct{}, "TestSchemaStruct"},
		{"Pointer to struct", &TestSchemaStruct{}, "TestSchemaStruct"},
		{"Int (not struct)", 1, ""},
		{"String (not struct)", "string", ""},
		{"Generic struct", GenericStruct[string]{}, "GenericStruct_string"},
		{"Generic struct with pointer", GenericStruct[*TestSchemaStruct]{}, "GenericStruct_TestSchemaStruct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCleanTypeName(reflect.TypeOf(tt.input))
			if result != tt.expected {
				t.Errorf("getCleanTypeName(%T) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateSchema(t *testing.T) {
	t.Parallel()
	reader := NewDocReader()
	components := &Components{Schemas: make(map[string]Schema)}
	testType := reflect.TypeOf(TestSchemaStruct{})

	schema := generateSchema(testType, components, reader)

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected property 'name' to exist")
	}

	if schema.Properties["name"].Type != "string" {
		t.Errorf("expected property 'name' type 'string', got '%s'", schema.Properties["name"].Type)
	}

	foundRequired := false
	for _, r := range schema.Required {
		if r == "name" {
			foundRequired = true
			break
		}
	}
	if !foundRequired {
		t.Error("expected 'name' to be in required fields")
	}

	if _, ok := schema.Properties["items"]; !ok {
		t.Error("expected property 'items' to exist")
	}

	if schema.Properties["items"].Type != "array" {
		t.Errorf("expected property 'items' type 'array', got '%s'", schema.Properties["items"].Type)
	}

	if _, ok := schema.Properties["tags"]; !ok {
		t.Error("expected property 'tags' to exist")
	}
	if schema.Properties["tags"].Type != "object" {
		t.Errorf("expected property 'tags' type 'object', got '%s'", schema.Properties["tags"].Type)
	}
	if schema.Properties["tags"].AdditionalProperties == nil {
		t.Error("expected 'tags' to have additionalProperties")
	}

	if schema.Properties["score"].Type != "number" {
		t.Errorf("expected 'score' type 'number', got '%s'", schema.Properties["score"].Type)
	}

	if len(schema.Properties["status"].Enum) != 2 {
		t.Errorf("expected 2 enum values for 'status', got %d", len(schema.Properties["status"].Enum))
	}

	if !schema.Properties["created_at"].ReadOnly {
		t.Error("expected 'created_at' to be readOnly")
	}

	if schema.Properties["count"].Default != "0" {
		t.Errorf("expected 'count' default '0', got '%v'", schema.Properties["count"].Default)
	}

	if _, ok := schema.Properties["nested"]; !ok {
		t.Error("expected property 'nested' to exist")
	}

	// Nested struct should have an AllOf with ref (we always use AllOf now to avoid $ref sibling issues)
	if len(schema.Properties["nested"].AllOf) == 0 || schema.Properties["nested"].AllOf[0].Ref == "" {
		t.Error("expected nested struct to have an AllOf with ref")
	}

	if _, ok := components.Schemas["NestedStruct"]; !ok {
		t.Error("expected 'NestedStruct' to be in components schemas")
	}

	// Test pointers
	ptrType := reflect.TypeOf(PointersStruct{})
	ptrSchema := generateSchema(ptrType, components, reader)
	if !ptrSchema.Properties["name"].Nullable {
		t.Error("expected pointer field 'name' to be nullable")
	}
	// Nullable struct references should use AllOf to avoid $ref sibling property violation in OpenAPI 3.0.x
	if !ptrSchema.Properties["nested"].Nullable {
		t.Error("expected pointer to struct 'nested' to be nullable")
	}
	if len(ptrSchema.Properties["nested"].AllOf) == 0 {
		t.Error("expected pointer to struct 'nested' to use AllOf for nullable reference")
	}
	if ptrSchema.Properties["nested"].AllOf[0].Ref != "#/components/schemas/NestedStruct" {
		t.Errorf("expected AllOf[0].Ref to be '#/components/schemas/NestedStruct', got '%s'", ptrSchema.Properties["nested"].AllOf[0].Ref)
	}

	// Optional input pointers (omitempty) are not nullable in OpenAPI.
	optType := reflect.TypeOf(OptionalPointerStruct{})
	optSchema := generateSchema(optType, components, reader)
	if optSchema.Properties["optional"].Nullable {
		t.Error("expected optional pointer field 'optional' to not be nullable")
	}

	optSliceType := reflect.TypeOf(OptionalSliceStruct{})
	optSliceSchema := generateSchema(optSliceType, components, reader)
	for _, req := range optSliceSchema.Required {
		if req == "optional_ids" {
			t.Error("expected slice field with omitempty 'optional_ids' to not be required in OpenAPI")
		}
	}
	var sawNameRequired bool
	for _, req := range optSliceSchema.Required {
		if req == "name" {
			sawNameRequired = true
		}
	}
	if !sawNameRequired {
		t.Error("expected validate-required field 'name' to be required")
	}

	// Response-style pointers without omitempty are nullable.
	respType := reflect.TypeOf(ResponseNullableStruct{})
	respSchema := generateSchema(respType, components, reader)
	if !respSchema.Properties["description"].Nullable {
		t.Error("expected response pointer field 'description' to be nullable")
	}

	// Test DocumentedType
	docType := reflect.TypeOf(DocumentedStruct{})
	docSchema := generateSchema(docType, components, reader)
	if docSchema.Example == nil {
		t.Error("expected DocumentedStruct to have an example")
	}
	example := docSchema.Example.(DocumentedStruct)
	if example.Value != "example" {
		t.Errorf("expected example value 'example', got '%s'", example.Value)
	}

	// json.RawMessage should be treated as a JSON object for docs, not a Go slice.
	rawType := reflect.TypeOf(TestRawMessageStruct{})
	rawSchema := generateSchema(rawType, components, reader)
	rawField, ok := rawSchema.Properties["raw"]
	if !ok {
		t.Fatal("expected property 'raw' to exist")
	}
	if rawField.Type != "object" {
		t.Errorf("expected property 'raw' type 'object', got '%s'", rawField.Type)
	}
	if !rawField.Nullable {
		t.Error("expected json.RawMessage field 'raw' to be nullable by default")
	}

	// A `null` in SchemaExample is permitted when it agrees with an already-nullable
	// Go type (here json.RawMessage, nullable by default); it is simply a no-op. A
	// null example on a non-nullable type instead fails generation loudly — see
	// TestGenerateSchema_ExampleNullOnNonNullableTypePanics.
	exampleNullType := reflect.TypeOf(NullableFromExampleStruct{})
	exampleNullSchema := generateSchema(exampleNullType, components, reader)
	exampleNullField, ok := exampleNullSchema.Properties["raw"]
	if !ok {
		t.Fatal("expected property 'raw' to exist on NullableFromExampleStruct")
	}
	if !exampleNullField.Nullable {
		t.Errorf("expected property 'raw' to be nullable, got %v", exampleNullField.Nullable)
	}

	patchType := reflect.TypeOf(PatchFieldStruct{})
	patchSchema := generateSchema(patchType, components, reader)

	for _, req := range patchSchema.Required {
		if req == "flat_rate" || req == "tags" || req == "description" {
			t.Errorf("field.Clearable property %q must not be required", req)
		}
	}

	desc, ok := patchSchema.Properties["description"]
	if !ok {
		t.Fatal("expected property 'description'")
	}
	if desc.Type != "string" {
		t.Errorf("expected description type string, got %q", desc.Type)
	}
	if !desc.Nullable {
		t.Error("expected clearable description to be nullable")
	}

	flat, ok := patchSchema.Properties["flat_rate"]
	if !ok {
		t.Fatal("expected property 'flat_rate'")
	}
	if !flat.Nullable {
		t.Error("expected clearable flat_rate to be nullable")
	}
	if len(flat.AllOf) == 0 || flat.AllOf[0].Ref != "#/components/schemas/patchQuantityInput" {
		t.Errorf("expected flat_rate to reference patchQuantityInput, got %+v", flat.AllOf)
	}

	tags, ok := patchSchema.Properties["tags"]
	if !ok {
		t.Fatal("expected property 'tags'")
	}
	if tags.Type != "array" || tags.Items == nil || tags.Items.Type != "string" {
		t.Errorf("expected tags type array of string, got %+v", tags)
	}
	if !tags.Nullable {
		t.Error("expected clearable tags to be nullable")
	}

	if _, ok := components.Schemas["Field"]; ok {
		t.Error("field.Clearable must not appear as a component schema name")
	}
	if _, ok := components.Schemas["patch_Field_string"]; ok {
		t.Error("field.Clearable must not leak into component schema names")
	}

	optionalType := reflect.TypeOf(OptionalInputStruct{})
	optionalSchema := generateSchema(optionalType, components, reader)

	opt, ok := optionalSchema.Properties["optional"]
	if !ok {
		t.Fatal("expected property 'optional'")
	}
	if opt.Nullable {
		t.Error("expected optional *string with omitempty to not be nullable")
	}

	descOptional, ok := optionalSchema.Properties["description"]
	if !ok {
		t.Fatal("expected property 'description'")
	}
	if descOptional.Type != "string" {
		t.Errorf("expected description type string, got %q", descOptional.Type)
	}
	// field.Optional[T] rejects an explicit null at runtime, so it is an optional
	// (omittable) input but NOT nullable on the wire.
	if descOptional.Nullable {
		t.Error("expected field.Optional[string] to not be nullable (it rejects explicit null)")
	}

	// A field.Optional[T] is always optional: a validate:"required" tag must not
	// promote it into the schema's required list.
	for _, req := range optionalSchema.Required {
		if req == "mandatory" {
			t.Error("expected field.Optional[string] with validate:\"required\" to not be required")
		}
	}
}

type TestRequest struct {
	ID string `json:"id" path:"id"`
}

type TestResponse struct {
	Message string `json:"message"`
}

type MockEndpoint struct {
	apiendpoint.APIEndpoint[TestRequest, TestResponse]
}

func (e *MockEndpoint) GetHandler() http.HandlerFunc {
	return nil
}

type retrieveAPIKeyMockEndpoint struct {
	apiendpoint.APIEndpoint[retrieveAPIKeyPathRequest, TestResponse]
}

func (e *retrieveAPIKeyMockEndpoint) GetHandler() http.HandlerFunc {
	return nil
}

// TestGenerateSchema_ExampleNullOnNonNullableTypePanics asserts the type is the
// single source of truth for nullability: a SchemaExample that encodes `null` for
// a non-nullable Go field is a contradiction and must fail generation loudly.
func TestGenerateSchema_ExampleNullOnNonNullableTypePanics(t *testing.T) {
	t.Parallel()
	reader := NewDocReader()
	components := &Components{Schemas: make(map[string]Schema)}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected generateSchema to panic when SchemaExample encodes null for a non-nullable field")
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, "non-nullable") {
			t.Errorf("expected panic to mention the non-nullable mismatch, got: %v", r)
		}
	}()

	generateSchema(reflect.TypeOf(ExampleNullMismatchStruct{}), components, reader)
}

func TestGenerate_FullAssembly(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "apidocs-assembly-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputPath := filepath.Join(tempDir, "internal_spec.json")

	groups := []apiendpoint.APIEndpointGroup{
		{
			Title: "Test Group",
			Endpoints: []apiendpoint.APIEndpointer{
				&MockEndpoint{
					APIEndpoint: apiendpoint.APIEndpoint[TestRequest, TestResponse]{
						Title:  "Public Endpoint",
						Method: "GET",
						Route:  "/public",
						Public: true,
					},
				},
				&MockEndpoint{
					APIEndpoint: apiendpoint.APIEndpoint[TestRequest, TestResponse]{
						Title:  "Private Endpoint",
						Method: "POST",
						Route:  "/private",
						Public: false,
					},
				},
			},
		},
	}

	// Generate internal spec (should include both)
	if err := generate(groups, outputPath, false, nil, "1.0.0"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read generated spec: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to unmarshal spec: %v", err)
	}

	components := spec["components"].(map[string]any)
	schemesAny, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatal("expected components.securitySchemes in spec")
	}
	if _, ok := schemesAny["BearerAuth"]; !ok {
		t.Error("expected BearerAuth security scheme")
	}
	if _, ok := schemesAny["AugnoApiKey"]; ok {
		t.Error("did not expect AugnoApiKey security scheme")
	}
	if sec, ok := spec["security"].([]any); !ok || len(sec) != 2 {
		t.Fatalf("expected top-level security with 2 alternatives, got %v", spec["security"])
	}

	paths := spec["paths"].(map[string]any)
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}

	// Verify error responses are NOT attached to individual endpoints
	publicPath := paths["/public"].(map[string]any)
	getOp := publicPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)

	errorCodes := []string{"400", "401", "403", "404", "409", "429", "500"}
	for _, code := range errorCodes {
		if _, ok := responses[code]; ok {
			t.Errorf("expected error response %s to NOT be attached to endpoint (errors defined once in components)", code)
		}
	}

	// Verify APIErrorResponse schema is still present in components for documentation
	schemas := components["schemas"].(map[string]any)
	if _, ok := schemas["APIErrorResponse"]; !ok {
		t.Error("expected APIErrorResponse schema to be present in components")
	}

	// Generate public spec (should only include public)
	publicOutputPath := filepath.Join(tempDir, "public_spec.json")
	if err := generate(groups, publicOutputPath, true, nil, "1.0.0"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	publicData, _ := os.ReadFile(publicOutputPath)
	var publicSpec map[string]any
	json.Unmarshal(publicData, &publicSpec)

	publicPaths := publicSpec["paths"].(map[string]any)
	if len(publicPaths) != 1 {
		t.Errorf("expected 1 path in public spec, got %d", len(publicPaths))
	}
	if _, ok := publicPaths["/public"]; !ok {
		t.Error("expected '/public' path to be present in public spec")
	}
}

func TestGenerateCreatesDirectory(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "apidocs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputDir := filepath.Join(tempDir, "missing-dir")
	outputPath := filepath.Join(outputDir, "spec.json")

	groups := []apiendpoint.APIEndpointGroup{}

	// This should create 'missing-dir' and write 'spec.json'
	if err := generate(groups, outputPath, true, nil, "1.0.0"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("expected file %s to be created, but it was not", outputPath)
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("expected directory %s to be created, but it was not", outputDir)
	}
}

func TestGetEnumValuesForStringType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		inputType      reflect.Type
		expectedValues []string
		expectNil      bool
	}{
		{
			name:           "AccountMode returns enum values",
			inputType:      reflect.TypeOf(constants.AccountMode("")),
			expectedValues: []string{"prod", "test"},
			expectNil:      false,
		},
		{
			name:      "plain string returns nil",
			inputType: reflect.TypeOf(""),
			expectNil: true,
		},
		{
			name:      "int returns nil",
			inputType: reflect.TypeOf(0),
			expectNil: true,
		},
		{
			name:      "struct returns nil",
			inputType: reflect.TypeOf(TestSchemaStruct{}),
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnumValuesForStringType(tt.inputType)

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("expected non-nil result, got nil")
				return
			}

			if len(result) != len(tt.expectedValues) {
				t.Errorf("expected %d values, got %d", len(tt.expectedValues), len(result))
				return
			}

			for i, expected := range tt.expectedValues {
				actual, ok := result[i].(string)
				if !ok {
					t.Errorf("expected string at index %d, got %T", i, result[i])
					continue
				}
				if actual != expected {
					t.Errorf("expected value %q at index %d, got %q", expected, i, actual)
				}
			}
		})
	}
}

type StructWithAccountMode struct {
	Mode constants.AccountMode `json:"mode"`
}

func TestGenerateSchema_EnumTypeField(t *testing.T) {
	t.Parallel()
	reader := NewDocReader()
	components := &Components{Schemas: make(map[string]Schema)}
	testType := reflect.TypeOf(StructWithAccountMode{})

	schema := generateSchema(testType, components, reader)

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}

	modeSchema, ok := schema.Properties["mode"]
	if !ok {
		t.Fatal("expected property 'mode' to exist")
	}

	if modeSchema.Type != "string" {
		t.Errorf("expected 'mode' type 'string', got '%s'", modeSchema.Type)
	}

	if len(modeSchema.Enum) != 2 {
		t.Errorf("expected 2 enum values for 'mode', got %d", len(modeSchema.Enum))
	}

	expectedEnums := map[string]bool{"prod": false, "test": false}
	for _, e := range modeSchema.Enum {
		s, ok := e.(string)
		if !ok {
			t.Errorf("expected enum value to be string, got %T", e)
			continue
		}
		if _, exists := expectedEnums[s]; !exists {
			t.Errorf("unexpected enum value %q", s)
		}
		expectedEnums[s] = true
	}

	for val, found := range expectedEnums {
		if !found {
			t.Errorf("expected enum value %q not found", val)
		}
	}
}

// listDocCommentItem is a minimal element type for apiresource.List[T] schema tests.
type listDocCommentItem struct {
	ID string `json:"id"`
}

func TestGenerateSchema_ListAndPageInfoUseDocComments(t *testing.T) {
	t.Parallel()
	reader := NewDocReader()
	components := &Components{Schemas: make(map[string]Schema)}

	listType := reflect.TypeOf(apiresource.List[listDocCommentItem]{})
	listSchema := generateSchema(listType, components, reader)

	if want := "A single page of resources, together with the metadata needed to page through the rest of the result set."; listSchema.Description != want {
		t.Errorf("List schema description = %q; want %q", listSchema.Description, want)
	}

	if got := listSchema.Properties["object"].Description; got != "Resource type identifier." {
		t.Errorf("object description = %q; want %q", got, "Resource type identifier.")
	}
	if got := listSchema.Properties["page_info"].Description; got != "Pagination metadata." {
		t.Errorf("page_info description = %q; want %q", got, "Pagination metadata.")
	}
	if got := listSchema.Properties["data"].Description; got != "Resources in this page." {
		t.Errorf("data description = %q; want %q", got, "Resources in this page.")
	}

	if purpose := listSchema.Properties["data"].XStainlessPaginationProperty["purpose"]; purpose != "items" {
		t.Errorf("data x-stainless-pagination-property purpose = %q; want items", purpose)
	}

	pageInfoSchema, ok := components.Schemas["PageInfo"]
	if !ok {
		t.Fatal("expected PageInfo in components.Schemas")
	}
	wantPageInfo := "PageInfo describes where the current page sits within a paginated result set and how to move to the adjacent pages.\n\n" +
		"Page a list by following the URLs below rather than assembling cursors yourself. For a top-level list endpoint the URL repeats the original request's query string with only the cursor swapped, so following it preserves the same filters, search term, and page size."
	if want := wantPageInfo; pageInfoSchema.Description != want {
		t.Errorf("PageInfo schema description = %q; want %q", pageInfoSchema.Description, want)
	}
	if got := pageInfoSchema.Properties["next_page_url"].Description; got != "Relative URL that fetches the next page of results." {
		t.Errorf("next_page_url description = %q", got)
	}
	if got := pageInfoSchema.Properties["previous_page_url"].Description; got != "Relative URL that fetches the previous page of results." {
		t.Errorf("previous_page_url description = %q", got)
	}
	if got := pageInfoSchema.Properties["has_next_page"].Description; got != "Whether more results exist after this page." {
		t.Errorf("has_next_page description = %q", got)
	}
	if got := pageInfoSchema.Properties["has_prev_page"].Description; got != "Whether results exist before this page." {
		t.Errorf("has_prev_page description = %q", got)
	}
}

type retrieveItemPathRequest struct {
	ItemID string `path:"id" validate:"required"`
}

func (*retrieveItemPathRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&retrieveItemPathRequest{
		ItemID: apiresource.SampleItemID,
	})
}

type retrieveAPIKeyPathRequest struct {
	APIKeyID string `path:"id" validate:"required"`
}

func TestPathParameterExample(t *testing.T) {
	t.Parallel()

	itemReqType := reflect.TypeOf(retrieveItemPathRequest{})
	itemField, _ := itemReqType.FieldByName("ItemID")
	itemExample := pathParameterExample(itemReqType, itemField, "id", "/v1/catalog/items/{id}", Schema{Type: "string"})
	if itemExample != apiresource.SampleItemID {
		t.Errorf("SchemaExample path param: got %v, want %s", itemExample, apiresource.SampleItemID)
	}

	keyReqType := reflect.TypeOf(retrieveAPIKeyPathRequest{})
	keyField, _ := keyReqType.FieldByName("APIKeyID")
	keyExample := pathParameterExample(keyReqType, keyField, "id", "/v1/auth/api-keys/{id}", Schema{Type: "string"})
	if keyExample != apiresource.SampleAPIKeyID {
		t.Errorf("field name path param: got %v, want %s", keyExample, apiresource.SampleAPIKeyID)
	}

	attrField := reflect.StructField{Name: "AttributeID", Type: reflect.TypeOf("")}
	attrExample := pathParameterExample(reflect.TypeOf(struct{}{}), attrField, "attribute_id", "/v1/catalog/items/{id}/attributes/{attribute_id}", Schema{Type: "string"})
	if attrExample != apiresource.SampleAttributeID {
		t.Errorf("named path param: got %v, want %s", attrExample, apiresource.SampleAttributeID)
	}
}

func TestGenerate_PathParameterExamples(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "apidocs-path-param-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputPath := filepath.Join(tempDir, "spec.json")
	groups := []apiendpoint.APIEndpointGroup{
		{
			Title: "API Keys",
			Endpoints: []apiendpoint.APIEndpointer{
				&retrieveAPIKeyMockEndpoint{
					APIEndpoint: apiendpoint.APIEndpoint[retrieveAPIKeyPathRequest, TestResponse]{
						Title:  "Retrieve API Key",
						Method: "GET",
						Route:  "/v1/auth/api-keys/{id}",
						Public: true,
					},
				},
			},
		},
	}
	if err := generate(groups, outputPath, false, nil, "1.0.0"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read spec: %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to unmarshal spec: %v", err)
	}

	paths := spec["paths"].(map[string]any)
	op := paths["/v1/auth/api-keys/{id}"].(map[string]any)["get"].(map[string]any)
	params := op["parameters"].([]any)
	var idParam map[string]any
	for _, p := range params {
		pm := p.(map[string]any)
		if pm["in"] == "path" && pm["name"] == "id" {
			idParam = pm
			break
		}
	}
	if idParam == nil {
		t.Fatal("expected path parameter id")
	}
	if idParam["example"] != apiresource.SampleAPIKeyID {
		t.Errorf("path param example = %v, want %s", idParam["example"], apiresource.SampleAPIKeyID)
	}
}

// TestOpenAPIGenerationDeterministic ensures consecutive generations produce identical
// spec bytes. CI compares generated release specs to a pre-upload snapshot of S3 openapi.json with cmp;
// this test guards that the generator itself is stable. Path/method maps are sorted by encoding/json;
// schema properties use struct field order (PropertyOrder / orderedJSONMap).
func TestOpenAPIGenerationDeterministic(t *testing.T) {
	t.Parallel()

	groups := openAPIEndpointGroups()
	const version = "determinism-test"

	for _, tc := range []struct {
		name       string
		publicOnly bool
	}{
		{name: "internal", publicOnly: false},
		{name: "public", publicOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc1, err := buildOpenAPIDocument(groups, tc.publicOnly, nil, version)
			if err != nil {
				t.Fatalf("first buildOpenAPIDocument: %v", err)
			}
			out1, err := formatOpenAPIJSON(doc1)
			if err != nil {
				t.Fatalf("first formatOpenAPIJSON: %v", err)
			}

			doc2, err := buildOpenAPIDocument(groups, tc.publicOnly, nil, version)
			if err != nil {
				t.Fatalf("second buildOpenAPIDocument: %v", err)
			}
			out2, err := formatOpenAPIJSON(doc2)
			if err != nil {
				t.Fatalf("second formatOpenAPIJSON: %v", err)
			}

			if !bytes.Equal(out1, out2) {
				t.Fatalf("%s spec differs between consecutive generations (%d vs %d bytes)", tc.name, len(out1), len(out2))
			}
		})
	}
}

func TestUpdateInvoiceRequestExampleMatchesNullableFlags(t *testing.T) {
	t.Parallel()
	components := &Components{Schemas: make(map[string]Schema)}
	schema := generateSchema(reflect.TypeFor[invoiceep.UpdateInvoiceRequest](), components, NewDocReader())
	assertExampleNullFieldsMarkedNullable(t, schema.Example, schema.Properties)
	assertOptionalBooleanExampleDefaults(t, schema.Example, schema)

	doc, err := buildOpenAPIDocument(openAPIEndpointGroups(), false, nil, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	componentsMap, _ := doc["components"].(map[string]any)
	schemas, _ := componentsMap["schemas"].(map[string]any)
	raw, _ := schemas["UpdateInvoiceRequest"].(map[string]any)
	props := schemaPropertiesFromDoc(raw["properties"])
	assertExampleNullFieldsMarkedNullable(t, raw["example"], props)
	assertOptionalBooleanExampleDefaults(t, raw["example"], schemaFromDoc(raw))
}

func schemaPropertiesFromDoc(raw any) map[string]Schema {
	out := make(map[string]Schema)
	switch p := raw.(type) {
	case map[string]any:
		for name, propRaw := range p {
			propMap, _ := propRaw.(map[string]any)
			out[name] = Schema{Nullable: propMap["nullable"] == true}
		}
	case orderedJSONMap:
		for name, propRaw := range p.values {
			propMap, _ := propRaw.(map[string]any)
			out[name] = Schema{Nullable: propMap["nullable"] == true}
		}
	}
	return out
}

func assertExampleNullFieldsMarkedNullable(t *testing.T, example any, properties map[string]Schema) {
	t.Helper()
	ex, ok := example.(map[string]any)
	if !ok {
		if ojm, ok := example.(orderedJSONMap); ok {
			ex = ojm.values
		} else {
			t.Fatalf("example type %T", example)
		}
	}
	for name, val := range ex {
		if isJSONNullish(val) && !properties[name].Nullable {
			t.Errorf("example has null for %q but property is not nullable", name)
		}
	}
}

func schemaFromDoc(raw map[string]any) Schema {
	schema := Schema{Type: "object", Properties: schemaPropertiesFromDoc(raw["properties"])}
	if req, ok := raw["required"].([]any); ok {
		for _, r := range req {
			if name, ok := r.(string); ok {
				schema.Required = append(schema.Required, name)
			}
		}
	}
	return schema
}

func assertOptionalBooleanExampleDefaults(t *testing.T, example any, schema Schema) {
	t.Helper()
	ex, ok := example.(map[string]any)
	if !ok {
		if ojm, ok := example.(orderedJSONMap); ok {
			ex = ojm.values
		} else {
			t.Fatalf("example type %T", example)
		}
	}

	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}

	for name, prop := range schema.Properties {
		if prop.Type != "boolean" || prop.Nullable {
			continue
		}
		if _, isRequired := required[name]; isRequired {
			continue
		}
		val, exists := ex[name]
		if !exists {
			t.Errorf("expected optional boolean %q to appear in example", name)
			continue
		}
		if isJSONNullish(val) {
			t.Errorf("expected optional boolean %q example to avoid null", name)
		}
	}
}
