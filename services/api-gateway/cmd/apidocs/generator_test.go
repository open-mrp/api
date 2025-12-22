package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
)

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

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{TestSchemaStruct{}, "TestSchemaStruct"},
		{&TestSchemaStruct{}, "TestSchemaStruct"},
		{1, ""},
		{"string", ""},
	}

	for _, tt := range tests {
		result := getTypeName(reflect.TypeOf(tt.input))
		if result != tt.expected {
			t.Errorf("getTypeName(%T) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateSchema(t *testing.T) {
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

	// Nested struct should have a ref if it has a name
	if schema.Properties["nested"].Ref == "" {
		t.Error("expected nested struct to have a ref")
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
	if ptrSchema.Properties["nested"].Ref == "" {
		t.Error("expected pointer to struct 'nested' to have a ref")
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

func (e *MockEndpoint) Materialize() apiendpoint.APIEndpointer {
	return e
}

func (e *MockEndpoint) GetHandler() http.HandlerFunc {
	return nil
}

func TestGenerate_FullAssembly(t *testing.T) {
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
						Title:    "Public Endpoint",
						Method:   "GET",
						Route:    "/public",
						IsPublic: true,
					},
				},
				&MockEndpoint{
					APIEndpoint: apiendpoint.APIEndpoint[TestRequest, TestResponse]{
						Title:    "Private Endpoint",
						Method:   "POST",
						Route:    "/private",
						IsPublic: false,
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

	// Verify error responses are attached
	publicPath := paths["/public"].(map[string]any)
	getOp := publicPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)

	errorCodes := []string{"400", "401", "403", "404", "409", "429", "500"}
	for _, code := range errorCodes {
		if _, ok := responses[code]; !ok {
			t.Errorf("expected error response %s to be attached to public endpoint", code)
		}
	}

	// Verify APIErrorResponse schema is present
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
