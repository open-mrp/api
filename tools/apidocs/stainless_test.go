package main

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"gopkg.in/yaml.v3"
)

type stainlessTestRequest struct {
	Name string `json:"name"`
}

type stainlessTestResponse struct {
	ID string `json:"id"`
}

func TestNormalizeIdentifier(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"APIKey":            "api_key",
		"List_AgentAction":  "list_agent_action",
		"fetch-doc-api-key": "fetch_doc_api_key",
		"OAuthResponse":     "oauth_response",
	}

	for input, want := range cases {
		if got := normalizeIdentifier(input); got != want {
			t.Fatalf("normalizeIdentifier(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestCollectEndpointSDKMetadataUsesCRUDDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	group := apiendpoint.APIEndpointGroup{
		Title: "Test",
		Endpoints: []apiendpoint.APIEndpointer{
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodGet,
				Route:             "/v1/auth/api-keys",
				SuccessStatusCode: http.StatusOK,
				Public:            true,
			},
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodPost,
				Route:             "/v1/auth/api-keys",
				SuccessStatusCode: http.StatusCreated,
				Public:            true,
			},
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodGet,
				Route:             "/v1/auth/api-keys/{id}",
				SuccessStatusCode: http.StatusOK,
				Public:            true,
			},
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodPost,
				Route:             "/v1/auth/api-keys/{id}/actions/rotate",
				SuccessStatusCode: http.StatusCreated,
				Public:            true,
			},
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodPut,
				Route:             "/v1/auth/access-tokens",
				SuccessStatusCode: http.StatusOK,
				Public:            true,
			},
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodGet,
				Route:             "/v1/identity/me/tenancy/customer-accounts/{vendor_account_id}",
				SuccessStatusCode: http.StatusOK,
				Public:            true,
			},
			&apiendpoint.APIEndpoint[*stainlessTestRequest, *stainlessTestResponse]{
				Method:            http.MethodPut,
				Route:             "/v1/catalog/items/{id}/category/{category_id}",
				SDKMethodKey:      "update",
				SuccessStatusCode: http.StatusOK,
				Public:            true,
			},
		},
	}

	spec, _, err := buildOpenAPISpec([]apiendpoint.APIEndpointGroup{group}, false, "1.0.0")
	if err != nil {
		t.Fatalf("buildOpenAPISpec: %v", err)
	}

	metas, err := collectEndpointSDKMetadata([]apiendpoint.APIEndpointGroup{group}, false, spec.Components.Schemas)
	if err != nil {
		t.Fatalf("collectEndpointSDKMetadata: %v", err)
	}

	assertMeta := func(route, method string, wantPath []string, wantKey string) {
		t.Helper()
		for _, meta := range metas {
			if meta.route == route && meta.method == method {
				if !reflect.DeepEqual(meta.resourcePath, wantPath) {
					t.Fatalf("%s %s resourcePath = %v; want %v", method, route, meta.resourcePath, wantPath)
				}
				if meta.methodKey != wantKey {
					t.Fatalf("%s %s methodKey = %q; want %q", method, route, meta.methodKey, wantKey)
				}
				return
			}
		}
		t.Fatalf("missing meta for %s %s", method, route)
	}

	assertMeta("/v1/auth/api-keys", http.MethodGet, []string{"auth", "api_keys"}, "list")
	assertMeta("/v1/auth/api-keys", http.MethodPost, []string{"auth", "api_keys"}, "create")
	assertMeta("/v1/auth/api-keys/{id}", http.MethodGet, []string{"auth", "api_keys"}, "retrieve")
	assertMeta("/v1/auth/api-keys/{id}/actions/rotate", http.MethodPost, []string{"auth", "api_keys", "actions"}, "rotate")
	assertMeta("/v1/auth/access-tokens", http.MethodPut, []string{"auth"}, "update_access_tokens")
	assertMeta("/v1/identity/me/tenancy/customer-accounts/{vendor_account_id}", http.MethodGet, []string{"identity", "me", "tenancy"}, "retrieve_customer_accounts")
	assertMeta("/v1/catalog/items/{id}/category/{category_id}", http.MethodPut, []string{"catalog", "items"}, "update")
}

func TestGenerateStainlessConfigBuildsAPIKeyMethodsForPublicAndInternal(t *testing.T) {
	t.Parallel()

	groups := openAPIEndpointGroups()
	version := "1.0.0"

	internalSpec, _, err := buildOpenAPISpec(groups, false, version)
	if err != nil {
		t.Fatalf("buildOpenAPISpec internal: %v", err)
	}
	internalMetas, err := collectEndpointSDKMetadata(groups, false, internalSpec.Components.Schemas)
	if err != nil {
		t.Fatalf("collectEndpointSDKMetadata internal: %v", err)
	}

	publicSpec, _, err := buildOpenAPISpec(groups, true, version)
	if err != nil {
		t.Fatalf("buildOpenAPISpec public: %v", err)
	}
	publicMetas, err := collectEndpointSDKMetadata(groups, true, publicSpec.Components.Schemas)
	if err != nil {
		t.Fatalf("collectEndpointSDKMetadata public: %v", err)
	}

	internalRoot := newStainlessNode()
	for _, meta := range internalMetas {
		node := internalRoot
		for _, segment := range meta.resourcePath {
			node = node.child(segment)
		}
		if err := node.setMethod(meta.methodKey, meta.method+" "+meta.route); err != nil {
			t.Fatalf("internal setMethod: %v", err)
		}
	}

	publicRoot := newStainlessNode()
	for _, meta := range publicMetas {
		node := publicRoot
		for _, segment := range meta.resourcePath {
			node = node.child(segment)
		}
		if err := node.setMethod(meta.methodKey, meta.method+" "+meta.route); err != nil {
			t.Fatalf("public setMethod: %v", err)
		}
	}

	internalAPIKeys := internalRoot.subresources["auth"].subresources["api_keys"]
	if internalAPIKeys == nil {
		t.Fatal("missing internal auth.api_keys resource")
	}
	if got := internalAPIKeys.methods["list"]; got == "" {
		t.Fatal("internal auth.api_keys missing list method")
	}
	if got := internalAPIKeys.methods["create"]; got == "" {
		t.Fatal("internal auth.api_keys missing create method")
	}
	if got := internalAPIKeys.methods["retrieve"]; got == "" {
		t.Fatal("internal auth.api_keys missing retrieve method")
	}

	internalActions := internalAPIKeys.subresources["actions"]
	if internalActions == nil || internalActions.methods["fetch_doc_api_key"] == "" {
		t.Fatal("internal auth.api_keys.actions missing fetch_doc_api_key")
	}
	if internalActions.methods["rotate"] == "" {
		t.Fatal("internal auth.api_keys.actions missing rotate")
	}

	publicAPIKeys := publicRoot.subresources["auth"].subresources["api_keys"]
	if publicAPIKeys == nil {
		t.Fatal("missing public auth.api_keys resource")
	}
	if got := publicAPIKeys.methods["list"]; got == "" {
		t.Fatal("public auth.api_keys missing list method")
	}
	if got := publicAPIKeys.methods["create"]; got == "" {
		t.Fatal("public auth.api_keys missing create method")
	}

	publicActions := publicAPIKeys.subresources["actions"]
	if publicActions == nil {
		t.Fatal("missing public auth.api_keys.actions resource")
	}
	if publicActions.methods["fetch_doc_api_key"] != "" {
		t.Fatal("public auth.api_keys.actions unexpectedly includes fetch_doc_api_key")
	}
}

func TestDedupeModelsClaimsFirstOccurrenceParentBeforeChild(t *testing.T) {
	t.Parallel()

	root := newStainlessNode()
	auth := root.child("auth")
	apiKeys := auth.child("api_keys")
	apiKeys.addModels([]string{"Account", "PageInfo", "APIKey"})
	apiKeys.child("actions").addModels([]string{"Account", "RotateAPIKeyRequest"})

	sandboxes := root.child("core").child("sandboxes")
	sandboxes.addModels([]string{"Account", "PageInfo", "Sandbox"})

	dedupeModels(root)

	// Parent claims shared aliases before the child and before later resources.
	if _, ok := apiKeys.models["account"]; !ok {
		t.Fatal("api_keys should retain account")
	}
	if _, ok := apiKeys.models["page_info"]; !ok {
		t.Fatal("api_keys should retain page_info")
	}
	if _, ok := apiKeys.subresources["actions"].models["account"]; ok {
		t.Fatal("actions should have dropped duplicate account")
	}
	if _, ok := apiKeys.subresources["actions"].models["rotate_api_key_request"]; !ok {
		t.Fatal("actions should retain its unique rotate_api_key_request")
	}
	if _, ok := sandboxes.models["account"]; ok {
		t.Fatal("sandboxes should have dropped duplicate account")
	}
	if _, ok := sandboxes.models["page_info"]; ok {
		t.Fatal("sandboxes should have dropped duplicate page_info")
	}
	if _, ok := sandboxes.models["sandbox"]; !ok {
		t.Fatal("sandboxes should retain its unique sandbox")
	}

	// Every alias must be declared exactly once across the whole tree.
	counts := make(map[string]int)
	var walk func(n *stainlessNode)
	walk = func(n *stainlessNode) {
		for alias := range n.models {
			counts[alias]++
		}
		for _, name := range n.subresourceOrder {
			walk(n.subresources[name])
		}
	}
	walk(root)
	for alias, count := range counts {
		if count != 1 {
			t.Fatalf("alias %q declared %d times; want exactly 1", alias, count)
		}
	}
}

func TestRewriteStainlessResourcesReplacesOnlyResourcesNode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "stainless.yml")
	input := []byte("edition: \"2026-02-23\"\norganization:\n  name: openmrp\nresources:\n  old:\n    methods:\n      list: get /old\n")
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := newStainlessNode()
	auth := root.child("auth")
	if err := auth.setMethod("update_access_tokens", "put /v1/auth/access-tokens"); err != nil {
		t.Fatalf("setMethod: %v", err)
	}

	if err := rewriteStainlessResources(path, root); err != nil {
		t.Fatalf("rewriteStainlessResources: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if got := doc["edition"]; got != "2026-02-23" {
		t.Fatalf("edition = %v; want 2026-02-23", got)
	}

	resources, ok := doc["resources"].(map[string]any)
	if !ok {
		t.Fatalf("resources type = %T; want map[string]any", doc["resources"])
	}
	if _, ok := resources["old"]; ok {
		t.Fatal("old resources entry still present after rewrite")
	}
	if _, ok := resources["auth"]; !ok {
		t.Fatal("auth resources entry missing after rewrite")
	}
}

func TestSyncVersionHeaderValueUpdatesOnlyVersionHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "stainless.yml")
	input := []byte("edition: \"2026-02-23\"\n" +
		"client_settings:\n" +
		"  default_headers:\n" +
		"    OpenMRP-Version:\n" +
		"      value: \"0.0.0-stale\"\n" +
		"      version_header: true\n" +
		"    X-Static:\n" +
		"      value: \"keep-me\"\n")
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := syncVersionHeaderValue(path, "1.2.3"); err != nil {
		t.Fatalf("syncVersionHeaderValue: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	headers := doc["client_settings"].(map[string]any)["default_headers"].(map[string]any)
	if got := headers["OpenMRP-Version"].(map[string]any)["value"]; got != "1.2.3" {
		t.Fatalf("OpenMRP-Version value = %v; want 1.2.3", got)
	}
	if got := headers["OpenMRP-Version"].(map[string]any)["version_header"]; got != true {
		t.Fatalf("version_header marker = %v; want true (must be preserved)", got)
	}
	// A header without the version_header marker must be left untouched.
	if got := headers["X-Static"].(map[string]any)["value"]; got != "keep-me" {
		t.Fatalf("X-Static value = %v; want keep-me", got)
	}
	if got := doc["edition"]; got != "2026-02-23" {
		t.Fatalf("edition = %v; want 2026-02-23", got)
	}
}

func TestSyncVersionHeaderValueNoOpWithoutHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "stainless.yml")
	input := []byte("edition: \"2026-02-23\"\nclient_settings:\n  omit_stainless_headers: true\n")
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := syncVersionHeaderValue(path, "1.2.3"); err != nil {
		t.Fatalf("syncVersionHeaderValue (no header configured) returned error: %v", err)
	}
}
