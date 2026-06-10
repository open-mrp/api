package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/contracts"
)

// noAccountHeaderGroups are groups where the Augno-Account-ID header should be omitted.
var noAccountHeaderGroups = map[string]bool{
	"Auth":                  true,
	"Health":                true,
	"Registration Sessions": true,
	"Registration Flows":    true,
}

// pathParamPattern matches {param} segments in route paths.
var pathParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

// routeParamToVariable maps specific route path parameter names to HTTPie variable names.
var routeParamToVariable = map[string]string{
	"id":               "id", // handled contextually
	"sales_order_id":   "sales-order-id",
	"order_id":         "sales-order-id",
	"pick_id":          "pick-id",
	"shipment_id":      "shipment-id",
	"invoice_id":       "invoice-id",
	"item_id":          "item-id",
	"product_id":       "product-id",
	"product_line_id":  "product-line-id",
	"unit_id":          "unit-id",
	"unit_group_id":    "unit-group-id",
	"property_id":      "property-id",
	"attribute_id":     "attribute-id",
	"material_id":      "material-id",
	"department_id":    "department-id",
	"machine_id":       "machine-id",
	"account_group_id": "account-group-id",
	"address_id":       "address-id",
	"carrier_id":       "carrier-id",
	"sandbox_id":       "sandbox-id",
	"user_id":          "user-id",
	"role_id":          "admin-role-id",
	"key_id":           "key-id",
	"account_id":       "act-id",
	"customer_id":      "customer-account-id",
	"place_id":         "place-id",
	"category_id":      "item-category-id",
	"batch_id":         "batch-id",
	"location_id":      "location-id",
}

// routeSegmentToIDVariable infers a variable name from the route path for generic {id} params.
// e.g. /v1/core/units/{id} → "unit-id"
var routeSegmentToIDVariable = map[string]string{
	"units":              "unit-id",
	"unit-groups":        "unit-group-id",
	"properties":         "property-id",
	"sys-properties":     "property-id",
	"attributes":         "attribute-id",
	"items":              "item-id",
	"item-categories":    "item-category-id",
	"products":           "product-id",
	"product-lines":      "product-line-id",
	"product-types":      "product-type-id",
	"materials":          "material-id",
	"departments":        "department-id",
	"machines":           "machine-id",
	"api-keys":           "key-id",
	"addresses":          "address-id",
	"account-groups":     "account-group-id",
	"accounts":           "act-id",
	"sales-orders":       "sales-order-id",
	"picks":              "pick-id",
	"shipments":          "shipment-id",
	"invoices":           "invoice-id",
	"carriers":           "carrier-id",
	"sandboxes":          "sandbox-id",
	"users":              "user-id",
	"roles":              "admin-role-id",
	"payment-terms":      "payment-term-id",
	"shipping-terms":     "shipping-term-id",
	"priorities":         "priority-id",
	"adjustment-types":   "adjustment-type-id",
	"account-statuses":   "account-status-id",
	"locations":          "location-id",
	"batches":            "batch-id",
	"supplier-materials": "material-id",
	"parts":              "part-id",
	"customers":          "customer-account-id",
	"suppliers":          "customer-account-id",
	"order-discounts":    "sales-order-id",
	"shipping-cases":     "shipment-id",
	"purchase-orders":    "sales-order-id",
	"receiving-orders":   "sales-order-id",
	"production-runs":    "sales-order-id",
	"production-steps":   "sales-order-id",
	"production-flows":   "sales-order-id",
	"territories":        "sales-order-id",
	"scanning-stations":  "sales-order-id",
	"permission-groups":  "admin-role-id",
}

// collectionColors assigns rotating colors to collections for visual variety.
var collectionColors = []string{"blue", "green", "purple", "orange", "red", "aqua", "yellow"}

// authGroupTitles are endpoint groups that should use type "none" auth at the collection level.
var authGroupTitles = map[string]bool{
	"Auth":                  true,
	"Registration Sessions": true,
	"Registration Flows":    true,
}

func generateHTTPieWorkspace(groups []apiendpoint.APIEndpointGroup, outputPath string) (err error) {
	// Schema generation reports invariant violations by panicking deep inside
	// the recursive builder; surface those as returned errors instead of
	// crashing the process (main() owns the exit).
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("generating HTTPie workspace %s: %v\n%s", outputPath, r, debug.Stack())
		}
	}()

	workspace := HTTPieWorkspace{
		Meta: HTTPieMeta{
			Format:      "httpie",
			Version:     "1.0.0",
			ContentType: "workspace",
			Schema:      "https://schema.httpie.io/1.0.0.json",
			Docs:        "https://httpie.io/r/help/export-from-httpie",
			Source:      "Augno API Generator",
		},
		Entry: HTTPieEntry{
			Name: "Augno API",
			Icon: HTTPieIcon{
				Name:  "default",
				Color: "gray",
			},
			Collections:  buildCollections(groups),
			Environments: buildEnvironments(),
			Drafts:       []any{},
		},
	}

	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling HTTPie workspace: %w", err)
	}

	// Ensure trailing newline
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("creating directory for HTTPie workspace: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("writing HTTPie workspace to %s: %w", outputPath, err)
	}

	logInfof("Generated HTTPie workspace at %s", outputPath)
	return nil
}

func buildCollections(groups []apiendpoint.APIEndpointGroup) []HTTPieCollection {
	var collections []HTTPieCollection

	for i, group := range groups {
		requests := buildRequests(group)
		if len(requests) == 0 {
			continue
		}

		color := collectionColors[i%len(collectionColors)]

		var auth HTTPieAuth
		if authGroupTitles[group.Title] {
			auth = HTTPieAuth{Type: "none"}
		} else {
			auth = HTTPieAuth{
				Type:   "bearer",
				Target: "headers",
				Credentials: &HTTPieCredentials{ // #nosec G101 -- template placeholder, not a real credential
					Username: "",
					Password: "{{api-key}}",
				},
			}
		}

		collections = append(collections, HTTPieCollection{
			Name: group.Title,
			Icon: HTTPieIcon{
				Name:  "default",
				Color: color,
			},
			Auth:     auth,
			Requests: requests,
		})
	}

	return collections
}

func buildRequests(group apiendpoint.APIEndpointGroup) []HTTPieRequest {
	var requests []HTTPieRequest

	for _, e := range group.Endpoints {
		val := reflect.ValueOf(e)
		if val.Kind() == reflect.Pointer {
			val = val.Elem()
		}

		specField := val.FieldByName("APIEndpoint")
		if !specField.IsValid() {
			specField = val
		}

		title := specField.FieldByName("Title").String()
		method := strings.ToUpper(strings.TrimSpace(specField.FieldByName("Method").String()))
		route := strings.TrimSpace(specField.FieldByName("Route").String())

		if method == "" || route == "" {
			continue
		}

		url := buildURL(route)
		headers := buildHeaders(group.Title, method)
		queryParams := buildQueryParams(specField)
		body := buildBody(method, e.GetRequestType())

		requests = append(requests, HTTPieRequest{
			Name:        title,
			URL:         url,
			Method:      method,
			Headers:     headers,
			QueryParams: queryParams,
			PathParams:  []HTTPieParam{},
			Auth:        HTTPieAuth{Type: "inherited"},
			Body:        body,
		})
	}

	return requests
}

func buildURL(route string) string {
	// First, replace path parameters while route still uses single braces
	result := pathParamPattern.ReplaceAllStringFunc(route, func(match string) string {
		paramName := match[1 : len(match)-1] // strip { }

		// Check direct mapping first
		if varName, ok := routeParamToVariable[paramName]; ok {
			if varName == "id" {
				// For generic {id}, infer from surrounding route segments
				return "<<" + inferIDVariable(route, match) + ">>"
			}
			return "<<" + varName + ">>"
		}

		// Default: convert snake_case param to kebab-case variable
		return "<<" + strings.ReplaceAll(paramName, "_", "-") + ">>"
	})

	// Now replace the route prefix with host/version variables
	if strings.HasPrefix(result, "/v1/") {
		result = "<<host>>/<<version>>/" + result[4:]
	} else if strings.HasPrefix(result, "/") {
		result = "<<host>>" + result
	}

	// Convert << >> placeholders to {{ }}
	result = strings.ReplaceAll(result, "<<", "{{")
	result = strings.ReplaceAll(result, ">>", "}}")

	return result
}

// inferIDVariable determines the variable name for a generic {id} parameter
// based on the route path segment preceding it.
func inferIDVariable(route string, match string) string {
	// Find the segment before {id}
	idx := strings.Index(route, match)
	if idx <= 0 {
		return "id"
	}

	before := route[:idx]
	if strings.HasSuffix(before, "/") {
		before = before[:len(before)-1]
	}
	lastSlash := strings.LastIndex(before, "/")
	if lastSlash < 0 {
		return "id"
	}
	segment := before[lastSlash+1:]

	if varName, ok := routeSegmentToIDVariable[segment]; ok {
		return varName
	}

	// Fallback: singularize the segment and append -id
	singular := segment
	if strings.HasSuffix(singular, "ies") {
		singular = singular[:len(singular)-3] + "y"
	} else if strings.HasSuffix(singular, "ses") {
		singular = singular[:len(singular)-2]
	} else if strings.HasSuffix(singular, "s") {
		singular = singular[:len(singular)-1]
	}
	return singular + "-id"
}

func buildHeaders(groupTitle string, method string) []HTTPieHeader {
	var headers []HTTPieHeader

	if !noAccountHeaderGroups[groupTitle] {
		headers = append(headers, HTTPieHeader{
			Name:    "Augno-Account-ID",
			Value:   "{{act-id}}",
			Enabled: true,
		})
	}

	headers = append(headers, HTTPieHeader{
		Name:    "Augno-Version",
		Value:   "{{api-version}}",
		Enabled: true,
	})

	if method == http.MethodPost || method == http.MethodPatch {
		headers = append(headers, HTTPieHeader{
			Name:    "Idempotency-Key",
			Value:   "test-idempotency-key",
			Enabled: false,
		})
	}

	return headers
}

func buildQueryParams(specField reflect.Value) []HTTPieParam {
	params := []HTTPieParam{}

	// Add include[] params if endpoint has IncludeConfig
	includeConfigField := specField.FieldByName("IncludeConfig")
	if includeConfigField.IsValid() && !includeConfigField.IsNil() {
		includeConfig := includeConfigField.Interface().(*apiendpoint.IncludeConfig)
		for _, key := range includeConfig.AllowedKeys() {
			params = append(params, HTTPieParam{
				Name:    "include[]",
				Value:   key,
				Enabled: false,
			})
		}
	}

	return params
}

func buildBody(method string, reqType reflect.Type) HTTPieBody {
	emptyBody := HTTPieBody{
		Type:    "none",
		File:    HTTPieFile{Name: ""},
		Text:    HTTPieText{Value: "", Format: "application/json"},
		Form:    HTTPieForm{IsMultipart: false, Fields: []any{}},
		GraphQL: HTTPieGraphQL{Query: "", Variables: ""},
	}

	if method == http.MethodGet || method == http.MethodDelete {
		return emptyBody
	}

	// Try to get example body from SchemaExample
	bodyJSON := getExampleBody(reqType)
	if bodyJSON == "" {
		// Try reflection-based body generation
		bodyJSON = getReflectionBody(reqType)
	}

	if bodyJSON == "" {
		return emptyBody
	}

	// Replace known IDs with variable references
	bodyJSON = replaceIDsWithVariables(bodyJSON)

	return HTTPieBody{
		Type:    "text",
		File:    HTTPieFile{Name: ""},
		Text:    HTTPieText{Value: bodyJSON, Format: "application/json"},
		Form:    HTTPieForm{IsMultipart: false, Fields: []any{}},
		GraphQL: HTTPieGraphQL{Query: "", Variables: ""},
	}
}

func getExampleBody(reqType reflect.Type) string {
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}
	if reqType.Kind() != reflect.Struct {
		return ""
	}

	// Check if the type implements DocumentedType (SchemaExample)
	ptrType := reflect.PointerTo(reqType)
	if !ptrType.Implements(reflect.TypeFor[contracts.DocumentedType]()) {
		return ""
	}

	v := reflect.New(reqType).Interface().(contracts.DocumentedType)
	example := v.SchemaExample()
	if example == nil {
		return ""
	}

	// Marshal the example to get the full JSON
	data, err := json.Marshal(example)
	if err != nil {
		return ""
	}

	// Filter to only json-tagged fields (exclude path, query, header fields)
	return filterToJSONFields(reqType, data)
}

// filterToJSONFields removes fields from the example that aren't JSON body fields.
func filterToJSONFields(reqType reflect.Type, data []byte) string {
	var fullMap map[string]any
	if err := json.Unmarshal(data, &fullMap); err != nil {
		return ""
	}

	// Collect only JSON body field names
	jsonFields := map[string]bool{}
	for i := 0; i < reqType.NumField(); i++ {
		f := reqType.Field(i)
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		// Skip path, query, header, cookie fields
		if f.Tag.Get("path") != "" || f.Tag.Get("query") != "" || f.Tag.Get("header") != "" || f.Tag.Get("cookie") != "" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		jsonFields[name] = true
	}

	if len(jsonFields) == 0 {
		return ""
	}

	filtered := map[string]any{}
	for k, v := range fullMap {
		if jsonFields[k] {
			filtered[k] = v
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	result, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return ""
	}
	return string(result)
}

func getReflectionBody(reqType reflect.Type) string {
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}
	if reqType.Kind() != reflect.Struct {
		return ""
	}

	body := map[string]any{}
	for i := 0; i < reqType.NumField(); i++ {
		f := reqType.Field(i)
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		// Skip non-body fields
		if f.Tag.Get("path") != "" || f.Tag.Get("query") != "" || f.Tag.Get("header") != "" || f.Tag.Get("cookie") != "" {
			continue
		}

		name := strings.Split(jsonTag, ",")[0]
		body[name] = reflectionFieldValue(f.Type)
	}

	if len(body) == 0 {
		return ""
	}

	result, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return ""
	}
	return string(result)
}

func reflectionFieldValue(t reflect.Type) any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Bool:
		return false
	case reflect.Slice:
		return []any{}
	default:
		return ""
	}
}

func replaceIDsWithVariables(body string) string {
	// Sort prefixes by length descending so longer prefixes match first
	type prefixMapping struct {
		prefix   string
		variable string
	}
	var mappings []prefixMapping
	for prefix, variable := range idPrefixToVariable {
		mappings = append(mappings, prefixMapping{prefix, variable})
	}
	// Sort by prefix length descending
	for i := 0; i < len(mappings); i++ {
		for j := i + 1; j < len(mappings); j++ {
			if len(mappings[j].prefix) > len(mappings[i].prefix) {
				mappings[i], mappings[j] = mappings[j], mappings[i]
			}
		}
	}

	// For each known seed value, replace with its variable reference
	for _, v := range v2Variables {
		if v.Value == "" || v.Name == "host" || v.Name == "version" || v.Name == "api-version" || v.Name == "api-key" {
			continue
		}
		// Replace exact seed value occurrences in JSON string values
		body = strings.ReplaceAll(body, fmt.Sprintf(`"%s"`, v.Value), fmt.Sprintf(`"{{%s}}"`, v.Name))
	}

	return body
}
