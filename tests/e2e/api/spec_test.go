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
			// A named param can mean different things per endpoint — {line_id} is a
			// sales-order line on one route and a schedule line on another — so a
			// path-specific seed wins over the global one.
			val, ok := pathSpecificParamSeed(e.Path, param)
			if !ok {
				val, ok = pathParamSeeds[param]
			}
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
var publicOpenAPISpec *openAPISpec

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

// LoadPublicSpec loads and caches the public OpenAPI spec.
func LoadPublicSpec() (*openAPISpec, error) {
	if publicOpenAPISpec != nil {
		return publicOpenAPISpec, nil
	}

	data, err := os.ReadFile(findPublicSpecPath())
	if err != nil {
		return nil, fmt.Errorf("reading public OpenAPI spec: %w", err)
	}

	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing public OpenAPI spec: %w", err)
	}

	publicOpenAPISpec = &spec
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
	"/v1/operations/purchase-orders",                   // no table in schema
	"/v1/auth/registration-sessions",                   // requires Stripe + cookie auth
	"/v1/identity/me/tenancy/customer-accounts",        // requires customer actor auth
	"/v1/sales/customers/{id}/notification-recipients", // derived from the customer's account-user notification prefs; the API-key harness seeds none. Covered by the dedicated notification-recipients test.
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
	// Active announcements are per-account_user; the API-key harness has no feed data to
	// paginate. Covered by messaging_announcements_test.go.
	"list-announcements": true,
	// Chat is participant-scoped and requires an account_user (API keys have none).
	// Covered by messaging_chat_test.go.
	"list-conversations": true,
	"list-messages":      true,
	"list-blocks":        true,
	// Notification preferences are per-account_user. Covered by messaging_preferences_test.go.
	"list-notification-preferences": true,
	// Scheduled messages are per-account_user. Covered by messaging_scheduled_test.go.
	"list-scheduled-messages": true,
}

// excludedListOperations are operationIDs omitted from list endpoint tests.
// These are stale OpenAPI entries removed from runtime routing.
var excludedListOperations = map[string]bool{
	"list-carrier-options":     true,
	"list-address-suggestions": true, // requires mandatory `input` query param
	// Per-user notification feed: scoped to the caller's account_user, which the API-key
	// harness has none of, so the generic tests (which assume API-key-visible seed data)
	// don't apply. Covered end-to-end by messaging_notifications_test.go.
	"list-notifications": true,
	// Active announcements: depend on broadcast sends + per-user receipts the API-key harness
	// can't produce, so the generic tests have no seed data. Covered by messaging_announcements_test.go.
	"list-announcements": true,
	// Chat is participant-scoped and requires an account_user. Covered by messaging_chat_test.go.
	"list-conversations": true,
	"list-messages":      true,
	"list-blocks":        true,
	// Notification preferences are per-account_user. Covered by messaging_preferences_test.go.
	"list-notification-preferences": true,
	// Scheduled messages are per-account_user. Covered by messaging_scheduled_test.go.
	"list-scheduled-messages": true,
	// External customer-service cases are participant/admin-scoped and need a real customer-support
	// case (audience=customer) the API-key harness can't provision; by-record requires mandatory
	// resource_type/resource_id query params. Covered end-to-end by messaging_external_cases_test.go.
	"list-inbox":                   true,
	"list-links":                   true,
	"list-reply-drafts":            true,
	"list-conversations-by-record": true,
	// Messaging groups (reusable rosters) are account_user-scoped; the API-key harness returns 403.
	// Covered by messaging_groups_test.go.
	"list-messaging-groups": true,
	// The messageable-contacts directory returns the caller's full (non-paginated) set and is
	// account_user-shaped; the generic list/pagination assertions don't apply. Covered by
	// messaging_contacts_test.go.
	"list-messaging-contacts": true,
	// A portal domain is a single-slot-per-account resource that every dedicated test creates and
	// then deletes (via t.Cleanup) to keep the slot free, so there is no standing row for the generic
	// list-schema test to read — it always sees an empty list. Covered end-to-end, including the
	// resource shape, by cov_settings_portal-domains_test.go.
	"list-portal-domains": true,
}

// excludedUpdateOperations are operationIDs omitted from update endpoint tests.
// These are stale OpenAPI entries removed from runtime routing.
var excludedUpdateOperations = map[string]bool{
	"update-carrier-option": true,
	// Chat edits are participant-scoped (account_user required). Covered by messaging_chat tests.
	"update-conversation": true,
	"edit-message":        true,
	// Legal hold needs a real conversation id and internal-actor permission. Covered by
	// messaging_redaction_test.go.
	"set-legal-hold": true,
	// Reply drafts need a real external case + draft id. Covered by messaging_external_cases_test.go.
	"update-reply-draft": true,
	// Messaging groups (reusable rosters) are account_user-scoped and need a real group id the
	// API-key harness can't seed. Covered by messaging_groups_test.go.
	"update-messaging-group": true,
	// Portal registration sessions are scoped to an authenticated buyer (cookie auth) and a
	// seller slug; the API-key harness has no buyer persona to own a session, so {id} can't be
	// resolved. Covered end-to-end by crud_portal_registration_sessions_test.go.
	"update-portal-registration-session": true,
}

// excludedCreateOperations are operationIDs omitted from POST body write tests.
var excludedCreateOperations = map[string]bool{
	// Chat creates require an active account_user (API keys have none) and reference real
	// participant/conversation ids. Covered end-to-end by messaging_chat_test.go.
	"create-conversation": true,
	"send-message":        true,
	"add-participant":     true,
	"block-user":          true,
	// Conversation action POSTs are participant-scoped and need real conversation/participant
	// ids the API-key harness can't resolve. Covered by messaging_chat tests.
	"mark-conversation-read":       true,
	"mute-conversation":            true,
	"update-participant-role":      true,
	"create-attachment-upload-url": true,
	"schedule-message":             true,
	"add-agent-participant":        true,
	// Redaction needs a real conversation id and internal-actor permission. Covered by
	// messaging_redaction_test.go.
	"redact-conversation": true,
	// Reporting needs a real conversation id the API-key harness can't resolve. Covered by
	// messaging_reports_test.go.
	"report-conversation": true,
	// External customer-service case actions need a real customer-support case (audience=customer) +
	// participant/admin context the API-key harness can't provision. Covered by
	// messaging_external_cases_test.go.
	"reply-to-customer":            true,
	"set-case-status":              true,
	"assign-case":                  true,
	"link-record":                  true,
	"create-reply-draft":           true,
	"approve-and-send-reply-draft": true,
	// Legal hold is a POST action needing a real conversation id + internal-actor permission the
	// API-key harness can't resolve. Covered by messaging_redaction_test.go.
	"set-legal-hold": true,
	// Adding a roster member needs a real messaging-group id the API-key harness can't seed, and is
	// account_user-scoped. Covered by messaging_groups_test.go.
	"add-messaging-group-member": true,
}

// excludedPutOperations are operationIDs omitted from PUT body write tests.
var excludedPutOperations = map[string]bool{
	// Upsert requires an account_user actor. Covered by messaging_preferences_test.go.
	"upsert-notification-preference": true,
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

// BodyEndpointSpec describes a POST or PUT endpoint with a JSON request body.
type BodyEndpointSpec struct {
	Path        string
	OperationID string
	PathParams  []string
}

// ResolvePath replaces path parameters with seed data values.
func (e *BodyEndpointSpec) ResolvePath() (string, bool) {
	return resolveEndpointPath(e.Path, e.PathParams)
}

// UpdateEndpointSpec describes a single update (PATCH) endpoint extracted from the OpenAPI spec.
type UpdateEndpointSpec struct {
	Path                string
	OperationID         string
	PathParams          []string
	NullableClearFields []string // request-body field names that are nullable (null clears them)
}

// ResolvePath replaces path parameters with seed data values.
func (e *UpdateEndpointSpec) ResolvePath() (string, bool) {
	return resolveEndpointPath(e.Path, e.PathParams)
}

func resolveEndpointPath(path string, pathParams []string) (string, bool) {
	if len(pathParams) == 0 {
		return path, true
	}

	resolved := path
	for _, param := range pathParams {
		var seedVal string
		if param == "id" {
			// Use longest prefix match to avoid shorter prefixes
			// incorrectly matching nested resource paths.
			var bestPrefix string
			for prefix, val := range pathSpecificIDSeeds {
				if strings.HasPrefix(path, prefix) && len(prefix) > len(bestPrefix) {
					bestPrefix = prefix
					seedVal = val
				}
			}
			if bestPrefix == "" {
				return "", false
			}
		} else {
			// A named param can mean different things per endpoint — {line_id} is a
			// sales-order line on one route and a schedule line on another — so a
			// path-specific seed wins over the global one.
			val, ok := pathSpecificParamSeed(path, param)
			if !ok {
				val, ok = pathParamSeeds[param]
			}
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

		// Collect nullable request-body fields. In a request body, a nullable
		// field is a clearable PATCH field: sending null clears it.
		if op.RequestBody != nil {
			if mt, ok := op.RequestBody.Content["application/json"]; ok {
				reqSchema := mt.Schema
				if reqSchema.Ref != "" {
					if resolved, ok := spec.ResolveSchemaRef(reqSchema.Ref); ok {
						reqSchema = *resolved
					}
				}
				for name, prop := range reqSchema.Properties {
					if prop.Nullable {
						ep.NullableClearFields = append(ep.NullableClearFields, name)
					}
				}
			}
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// LoadBodyEndpoints parses the OpenAPI spec and returns POST or PUT endpoints with JSON bodies.
func LoadBodyEndpoints(httpMethod string, excluded map[string]bool) ([]BodyEndpointSpec, error) {
	specPath := findSpecPath()

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec at %s: %w", specPath, err)
	}

	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	var endpoints []BodyEndpointSpec
	for path, methods := range spec.Paths {
		if isExcludedPath(path) {
			continue
		}

		op, ok := methods[httpMethod]
		if !ok {
			continue
		}
		if excluded[op.OperationID] {
			continue
		}
		if op.RequestBody == nil {
			continue
		}
		if _, ok := op.RequestBody.Content["application/json"]; !ok {
			continue
		}

		ep := BodyEndpointSpec{
			Path:        path,
			OperationID: op.OperationID,
		}

		for _, p := range op.Parameters {
			if p.In == "path" {
				ep.PathParams = append(ep.PathParams, p.Name)
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

func findPublicSpecPath() string {
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Join(repoRoot, "specs", "public_openapi_spec.json")
}
