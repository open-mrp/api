package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/patch"
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

// PatchFieldStruct exercises patch.Field[T] OpenAPI generation.
type PatchFieldStruct struct {
	Name        *string                          `json:"name,omitempty"`
	Description *patch.Field[string]             `json:"description"`
	FlatRate    *patch.Field[patchQuantityInput] `json:"flat_rate"`
	Tags        *patch.Field[[]string]           `json:"tags,omitempty"`
}

// NullableInputStruct exercises patch.Nullable[T] OpenAPI generation.
type NullableInputStruct struct {
	Optional    *string                `json:"optional,omitempty"`
	Description patch.Nullable[string] `json:"description,omitzero"`
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
	if respSchema.Properties["description"].XNullableClear {
		t.Error("expected response pointer without x-nullable-clear")
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

	// If a documented example encodes `null`, we should mark the field nullable
	// even when the Go type isn't a pointer.
	exampleNullType := reflect.TypeOf(NullableFromExampleStruct{})
	exampleNullSchema := generateSchema(exampleNullType, components, reader)
	exampleNullField, ok := exampleNullSchema.Properties["raw"]
	if !ok {
		t.Fatal("expected property 'raw' to exist on NullableFromExampleStruct")
	}
	if exampleNullField.Nullable != true {
		t.Errorf("expected property 'raw' to be nullable, got %v", exampleNullField.Nullable)
	}

	patchType := reflect.TypeOf(PatchFieldStruct{})
	patchSchema := generateSchema(patchType, components, reader)

	for _, req := range patchSchema.Required {
		if req == "flat_rate" || req == "tags" || req == "description" {
			t.Errorf("patch.Field property %q must not be required", req)
		}
	}

	desc, ok := patchSchema.Properties["description"]
	if !ok {
		t.Fatal("expected property 'description'")
	}
	if desc.Type != "string" {
		t.Errorf("expected description type string, got %q", desc.Type)
	}
	if !desc.Nullable || !desc.XNullableClear {
		t.Error("expected description to be nullable with x-nullable-clear")
	}

	flat, ok := patchSchema.Properties["flat_rate"]
	if !ok {
		t.Fatal("expected property 'flat_rate'")
	}
	if !flat.Nullable || !flat.XNullableClear {
		t.Error("expected flat_rate to be nullable with x-nullable-clear")
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
	if !tags.Nullable || !tags.XNullableClear {
		t.Error("expected tags to be nullable with x-nullable-clear")
	}

	if _, ok := components.Schemas["Field"]; ok {
		t.Error("patch.Field must not appear as a component schema name")
	}
	if _, ok := components.Schemas["patch_Field_string"]; ok {
		t.Error("patch.Field must not leak into component schema names")
	}

	nullableType := reflect.TypeOf(NullableInputStruct{})
	nullableSchema := generateSchema(nullableType, components, reader)

	opt, ok := nullableSchema.Properties["optional"]
	if !ok {
		t.Fatal("expected property 'optional'")
	}
	if opt.Nullable {
		t.Error("expected optional *string with omitempty to not be nullable")
	}

	descNullable, ok := nullableSchema.Properties["description"]
	if !ok {
		t.Fatal("expected property 'description'")
	}
	if descNullable.Type != "string" {
		t.Errorf("expected description type string, got %q", descNullable.Type)
	}
	if !descNullable.Nullable {
		t.Error("expected Nullable[string] to be nullable")
	}
	if descNullable.XNullableClear {
		t.Error("expected Nullable[string] without x-nullable-clear")
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
	generate(groups, outputPath, false, nil, "1.0.0")

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read generated spec: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to unmarshal spec: %v", err)
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
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	if _, ok := schemas["APIErrorResponse"]; !ok {
		t.Error("expected APIErrorResponse schema to be present in components")
	}

	// Generate public spec (should only include public)
	publicOutputPath := filepath.Join(tempDir, "public_spec.json")
	generate(groups, publicOutputPath, true, nil, "1.0.0")

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
	generate(groups, outputPath, true, nil, "1.0.0")

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
	listSchema := generateSchema(listType, components, reader, "/v1/example")

	if want := "List represents a paginated list of resources."; listSchema.Description != want {
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
	if want := "PageInfo contains URL-based pagination metadata."; pageInfoSchema.Description != want {
		t.Errorf("PageInfo schema description = %q; want %q", pageInfoSchema.Description, want)
	}
	if got := pageInfoSchema.Properties["next_page_url"].Description; got != "URL to fetch the next page, `null` if no more pages." {
		t.Errorf("next_page_url description = %q", got)
	}
	if got := pageInfoSchema.Properties["previous_page_url"].Description; got != "URL to fetch the previous page, `null` if on the first page." {
		t.Errorf("previous_page_url description = %q", got)
	}
	if got := pageInfoSchema.Properties["has_next_page"].Description; got != "Whether more results exist after this page." {
		t.Errorf("has_next_page description = %q", got)
	}
	if got := pageInfoSchema.Properties["has_prev_page"].Description; got != "Whether results exist before this page." {
		t.Errorf("has_prev_page description = %q", got)
	}
}
