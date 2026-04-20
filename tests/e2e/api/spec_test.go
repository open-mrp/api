//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ListEndpointSpec describes a single list endpoint extracted from the OpenAPI spec.
type ListEndpointSpec struct {
	Path        string
	OperationID string
	Params      []string // query parameter names
	PathParams  []string // path parameter names (e.g., "property_id")
}

// HasParam returns true if the endpoint accepts the given query parameter.
func (e *ListEndpointSpec) HasParam(name string) bool {
	for _, p := range e.Params {
		if p == name {
			return true
		}
	}
	return false
}

// ResolvePath replaces path parameters with seed data values.
// Returns the resolved path and true if all params could be resolved, or empty string and false otherwise.
func (e *ListEndpointSpec) ResolvePath() (string, bool) {
	if len(e.PathParams) == 0 {
		return e.Path, true
	}

	resolved := e.Path
	for _, param := range e.PathParams {
		var seedVal string
		if param == "id" {
			// Generic {id} — resolve based on longest matching path prefix.
			found := false
			bestLen := 0
			for prefix, val := range pathSpecificIDSeeds {
				if strings.HasPrefix(e.Path, prefix) && len(prefix) > bestLen {
					seedVal = val
					bestLen = len(prefix)
					found = true
				}
			}
			if !found {
				return "", false
			}
		} else {
			val, ok := pathParamSeeds[param]
			if !ok {
				return "", false
			}
			seedVal = val
		}
		if seedVal == "" {
			return "", false
		}
		resolved = strings.ReplaceAll(resolved, "{"+param+"}", seedVal)
	}
	return resolved, true
}

// openAPISpec is a minimal representation of the OpenAPI spec for test discovery.
type openAPISpec struct {
	Paths      map[string]map[string]openAPIOperation `json:"paths"`
	Components *openAPIComponents                     `json:"components"`
}

type openAPIComponents struct {
	Schemas map[string]openAPISchema `json:"schemas"`
}

type openAPISchema struct {
	Type                 string                   `json:"type"`
	Properties           map[string]openAPISchema `json:"properties"`
	Required             []string                 `json:"required"`
	Enum                 []any                    `json:"enum"`
	Items                *openAPISchema           `json:"items"`
	Ref                  string                   `json:"$ref"`
	AllOf                []openAPISchema          `json:"allOf"`
	OneOf                []openAPISchema          `json:"oneOf"`
	AnyOf                []openAPISchema          `json:"anyOf"`
	Format               string                   `json:"format"`
	Nullable             bool                     `json:"nullable"`
	AdditionalProperties *openAPISchema           `json:"additionalProperties,omitempty"`
	XNullableClear       bool                     `json:"x-nullable-clear,omitempty"`
}

type openAPIOperation struct {
	OperationID string                     `json:"operationId"`
	Parameters  []openAPIParam             `json:"parameters"`
	Responses   map[string]openAPIResponse `json:"responses"`
	RequestBody *openAPIRequestBody        `json:"requestBody,omitempty"`
}

type openAPIRequestBody struct {
	Content map[string]openAPIMediaType `json:"content"`
}

type openAPIResponse struct {
	Content map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema"`
}

type openAPIParam struct {
	Name   string         `json:"name"`
	In     string         `json:"in"`
	Schema *openAPISchema `json:"schema,omitempty"`
}

// fullOpenAPISpec holds the fully-parsed spec for schema validation.
var fullOpenAPISpec *openAPISpec

// LoadFullSpec loads and caches the complete OpenAPI spec (including schemas).
func LoadFullSpec() (*openAPISpec, error) {
	if fullOpenAPISpec != nil {
		return fullOpenAPISpec, nil
	}

	specPath := findSpecPath()
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec at %s: %w", specPath, err)
	}

	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	fullOpenAPISpec = &spec
	return &spec, nil
}

// ResolveSchemaRef resolves a $ref like "#/components/schemas/Customer" to the actual schema.
func (spec *openAPISpec) ResolveSchemaRef(ref string) (*openAPISchema, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, false
	}
	name := strings.TrimPrefix(ref, prefix)
	if spec.Components == nil {
		return nil, false
	}
	schema, ok := spec.Components.Schemas[name]
	if !ok {
		return nil, false
	}
	return &schema, true
}

// GetResponseSchema finds the response schema for a given path, method, and status code.
func (spec *openAPISpec) GetResponseSchema(path, method, statusCode string) (*openAPISchema, bool) {
	methods, ok := spec.Paths[path]
	if !ok {
		return nil, false
	}
	op, ok := methods[method]
	if !ok {
		return nil, false
	}
	resp, ok := op.Responses[statusCode]
	if !ok {
		return nil, false
	}
	mt, ok := resp.Content["application/json"]
	if !ok {
		return nil, false
	}
	schema := mt.Schema
	if schema.Ref != "" {
		return spec.ResolveSchemaRef(schema.Ref)
	}
	return &schema, true
}

// CollectSchemaFields returns all field names defined in a schema (resolving refs and allOf).
func (spec *openAPISpec) CollectSchemaFields(schema *openAPISchema) map[string]bool {
	fields := make(map[string]bool)
	spec.collectFields(schema, fields, 0)
	return fields
}

func (spec *openAPISpec) collectFields(schema *openAPISchema, fields map[string]bool, depth int) {
	if depth > 10 {
		return
	}
	if schema.Ref != "" {
		resolved, ok := spec.ResolveSchemaRef(schema.Ref)
		if ok {
			spec.collectFields(resolved, fields, depth+1)
		}
		return
	}
	for name := range schema.Properties {
		fields[name] = true
	}
	for _, s := range schema.AllOf {
		spec.collectFields(&s, fields, depth+1)
	}
	for _, s := range schema.OneOf {
		spec.collectFields(&s, fields, depth+1)
	}
	for _, s := range schema.AnyOf {
		spec.collectFields(&s, fields, depth+1)
	}
}

// excludedPaths contains endpoint path prefixes that are excluded from test
// discovery because they cannot be tested (no backing table, require different
// auth, etc.). These endpoints will not appear in test output at all.
var excludedPaths = []string{
	"/v1/operations/purchase-orders",            // no table in schema
	"/v1/auth/registration-sessions",            // requires Stripe + cookie auth
	"/v1/identity/me/tenancy/customer-accounts", // requires customer actor auth
}

// excludedPaginationPaths are endpoints that return data but don't support
// cursor-based pagination (hardcoded static lists, non-paginated service
// responses, admin-only endpoints, etc.).
var excludedPaginationPaths = []string{
	"/v1/finance/transaction-types",   // hardcoded static list, not cursor-paginated
	"/v1/finance/transaction-methods", // hardcoded static list, not cursor-paginated
	"/v1/core/request-logs",           // requires internal admin role
	"/v1/ai/agents",                   // returns all definitions, no cursor pagination
	"/v1/ai/tool-groups",              // returns all tool groups, no cursor pagination
	"/v1/ai/tools",                    // returns all tool definitions, no cursor pagination
	"/v1/catalog/catalog/",            // returns all products, no cursor pagination
}

// isExcludedPath returns true if the path matches any excluded prefix.
func isExcludedPath(path string) bool {
	for _, prefix := range excludedPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// excludedPaginationOperations are operationIDs that should be excluded from
// pagination tests because they use offset pagination or no pagination at all.
var excludedPaginationOperations = map[string]bool{
	"list-sales-targets": true, // uses offset pagination, not cursor
}

// excludedListOperations are operationIDs omitted from list endpoint tests.
// These are stale OpenAPI entries removed from runtime routing.
var excludedListOperations = map[string]bool{
	"list-carrier-options":     true,
	"list-address-suggestions": true, // requires mandatory `input` query param
}

// excludedUpdateOperations are operationIDs omitted from update endpoint tests.
// These are stale OpenAPI entries removed from runtime routing.
var excludedUpdateOperations = map[string]bool{
	"update-carrier-option": true,
}

// isExcludedFromPagination returns true if the path or operationID should be
// excluded from pagination tests.
func isExcludedFromPagination(path, operationID string) bool {
	if excludedPaginationOperations[operationID] {
		return true
	}
	for _, prefix := range excludedPaginationPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// LoadListEndpoints parses the OpenAPI spec and returns all list endpoints.
func LoadListEndpoints() ([]ListEndpointSpec, error) {
	specPath := findSpecPath()

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec at %s: %w", specPath, err)
	}

	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	var endpoints []ListEndpointSpec
	for path, methods := range spec.Paths {
		if isExcludedPath(path) {
			continue
		}

		op, ok := methods["get"]
		if !ok {
			continue
		}
		if !strings.Contains(strings.ToLower(op.OperationID), "list") {
			continue
		}
		if excludedListOperations[op.OperationID] {
			continue
		}

		ep := ListEndpointSpec{
			Path:        path,
			OperationID: op.OperationID,
		}

		for _, p := range op.Parameters {
			if p.In == "query" {
				ep.Params = append(ep.Params, p.Name)
			}
			if p.In == "path" {
				ep.PathParams = append(ep.PathParams, p.Name)
			}
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// UpdateEndpointSpec describes a single update (PATCH) endpoint extracted from the OpenAPI spec.
type UpdateEndpointSpec struct {
	Path                string
	OperationID         string
	PathParams          []string
	NullableClearFields []string // field names with x-nullable-clear in the request body schema
}

// ResolvePath replaces path parameters with seed data values.
func (e *UpdateEndpointSpec) ResolvePath() (string, bool) {
	if len(e.PathParams) == 0 {
		return e.Path, true
	}

	resolved := e.Path
	for _, param := range e.PathParams {
		var seedVal string
		if param == "id" {
			// Use longest prefix match to avoid shorter prefixes
			// incorrectly matching nested resource paths.
			var bestPrefix string
			for prefix, val := range pathSpecificIDSeeds {
				if strings.HasPrefix(e.Path, prefix) && len(prefix) > len(bestPrefix) {
					bestPrefix = prefix
					seedVal = val
				}
			}
			if bestPrefix == "" {
				return "", false
			}
		} else {
			val, ok := pathParamSeeds[param]
			if !ok {
				return "", false
			}
			seedVal = val
		}
		if seedVal == "" {
			return "", false
		}
		resolved = strings.ReplaceAll(resolved, "{"+param+"}", seedVal)
	}
	return resolved, true
}

// LoadUpdateEndpoints parses the OpenAPI spec and returns all PATCH endpoints.
func LoadUpdateEndpoints() ([]UpdateEndpointSpec, error) {
	specPath := findSpecPath()

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec at %s: %w", specPath, err)
	}

	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	var endpoints []UpdateEndpointSpec
	for path, methods := range spec.Paths {
		if isExcludedPath(path) {
			continue
		}

		op, ok := methods["patch"]
		if !ok {
			continue
		}
		if excludedUpdateOperations[op.OperationID] {
			continue
		}

		ep := UpdateEndpointSpec{
			Path:        path,
			OperationID: op.OperationID,
		}

		for _, p := range op.Parameters {
			if p.In == "path" {
				ep.PathParams = append(ep.PathParams, p.Name)
			}
		}

		// Extract x-nullable-clear fields from the request body schema.
		if op.RequestBody != nil {
			if mt, ok := op.RequestBody.Content["application/json"]; ok {
				reqSchema := mt.Schema
				if reqSchema.Ref != "" {
					if resolved, ok := spec.ResolveSchemaRef(reqSchema.Ref); ok {
						reqSchema = *resolved
					}
				}
				for name, prop := range reqSchema.Properties {
					if prop.XNullableClear {
						ep.NullableClearFields = append(ep.NullableClearFields, name)
					}
				}
			}
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// findSpecPath locates the OpenAPI spec relative to this source file.
func findSpecPath() string {
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Join(repoRoot, "specs", "internal_openapi_spec.json")
}
