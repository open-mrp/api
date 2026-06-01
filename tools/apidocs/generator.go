package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
)

func endpointSpecField(e apiendpoint.APIEndpointer) reflect.Value {
	val := reflect.ValueOf(e)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	specField := val.FieldByName("APIEndpoint")
	if !specField.IsValid() {
		specField = val
	}

	return specField
}

func endpointRequestHasJSONFields(reqType reflect.Type) bool {
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}
	if reqType.Kind() != reflect.Struct {
		return false
	}

	for _, f := range flattenStructFields(reqType) {
		jsonTag := f.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			return true
		}
	}

	return false
}

func buildOpenAPISpec(groups []apiendpoint.APIEndpointGroup, publicOnly bool, version string) (OpenAPI, map[string][]string, error) {
	docReader := NewDocReader()
	spec := OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Augno API",
			Description: "API for Augno services",
			Version:     version,
		},
		Servers: []Server{
			{URL: "https://api.augno.com", Description: "Production server"},
			{URL: "http://localhost:8081", Description: "Local development server"},
		},
		Paths: make(map[string]map[string]Operation),
		Components: Components{
			Schemas: make(map[string]Schema),
			SecuritySchemes: map[string]SecuritySchemeSpec{
				"BearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
			},
		},
		Tags: []Tag{},
		// BearerAuth for API keys; empty alternative allows cookie-based user sessions without Authorization.
		Security: []map[string][]string{
			{"BearerAuth": {}},
			{},
		},
	}

	tagNames := make(map[string]bool)

	for _, group := range groups {
		groupHasEndpoints := false
		groupPathMap := make(map[string]map[string]Operation)

		for _, e := range group.Endpoints {
			specField := endpointSpecField(e)

			isPublic := specField.FieldByName("Public").Bool()
			if publicOnly && !isPublic {
				continue
			}

			isPreview := specField.FieldByName("Preview").Bool()

			groupHasEndpoints = true
			title := specField.FieldByName("Title").String()
			var description string
			if epTypeVal := specField.FieldByName("EndpointType"); epTypeVal.IsValid() && !epTypeVal.IsNil() {
				epType := epTypeVal.Interface().(reflect.Type)
				description = docReader.GetTypeDoc(epType).Doc
			}
			method := strings.ToUpper(strings.TrimSpace(specField.FieldByName("Method").String()))
			route := strings.TrimSpace(specField.FieldByName("Route").String())

			if method == "" || route == "" {
				panic(fmt.Errorf("Error: encountered endpoint with empty method or route: %s", title))
			}

			if groupPathMap[route] == nil {
				groupPathMap[route] = make(map[string]Operation)
			}

			operationID := strings.ReplaceAll(strings.ToLower(title), " ", "-")
			operation := Operation{
				Summary:     title,
				Description: description,
				OperationID: operationID,
				Tags:        []string{group.Title},
				Responses:   make(map[string]Response),
				Parameters:  []Parameter{},
				XPreview:    isPreview,
			}

			// Handle Parameters and Request Body
			origReqType := e.GetRequestType()
			reqType := origReqType
			if reqType.Kind() == reflect.Pointer {
				reqType = reqType.Elem()
			}
			{
				hasJSONFields := endpointRequestHasJSONFields(reqType)
				if reqType.Kind() == reflect.Struct {
					for _, f := range flattenStructFields(reqType) {
						// Handle Parameters
						if header := f.Tag.Get("header"); header != "" {
							desc := fmt.Sprintf("Header parameter: %s for %s", header, title)
							if header == "Authorization" {
								desc = fmt.Sprintf("The authentication token (Bearer or Basic scheme) for %s", title)
							}
							operation.Parameters = append(operation.Parameters, Parameter{
								Name:        header,
								In:          "header",
								Description: desc,
								Required:    strings.Contains(f.Tag.Get("validate"), "required"),
								Schema:      generateSchema(f.Type, &spec.Components, docReader),
							})
						}
						if query := f.Tag.Get("query"); query != "" {
							paramSchema := generateSchema(f.Type, &spec.Components, docReader)
							desc := getFieldDoc(reqType, f, docReader)
							if desc == "" {
								desc = fmt.Sprintf("Query parameter: %s for %s", query, title)
							}

							// Normalize array query parameter names to use [] suffix so Stainless
							// sees a consistent brackets format across the spec.
							paramName := query
							if paramSchema.Type == "array" && !strings.HasSuffix(paramName, "[]") {
								paramName = paramName + "[]"
							}

							operation.Parameters = append(operation.Parameters, Parameter{
								Name:        paramName,
								In:          "query",
								Description: desc,
								Required:    strings.Contains(f.Tag.Get("validate"), "required"),
								Schema:      paramSchema,
								Example:     queryParameterExample(reqType, f, paramSchema),
							})
						}
						if cookie := f.Tag.Get("cookie"); cookie != "" {
							desc := fmt.Sprintf("Cookie parameter: %s for %s", cookie, title)
							if cookie == "__Secure-augno.refresh-token" {
								desc = fmt.Sprintf("The Secure refresh token cookie for %s", title)
							}
							operation.Parameters = append(operation.Parameters, Parameter{
								Name:        cookie,
								In:          "cookie",
								Description: desc,
								Required:    strings.Contains(f.Tag.Get("validate"), "required"),
								Schema:      generateSchema(f.Type, &spec.Components, docReader),
							})
						}
						if pathParam := f.Tag.Get("path"); pathParam != "" {
							desc := getFieldDoc(reqType, f, docReader)
							if desc == "" {
								desc = fmt.Sprintf("Path parameter: %s for %s", pathParam, title)
							}
							paramSchema := generateSchema(f.Type, &spec.Components, docReader)
							operation.Parameters = append(operation.Parameters, Parameter{
								Name:        pathParam,
								In:          "path",
								Description: desc,
								Required:    true,
								Schema:      paramSchema,
								Example:     pathParameterExample(reqType, f, pathParam, route, paramSchema),
							})
						}
					}
				}

				// Add include[] parameter if endpoint has IncludeConfig.
				// If the request struct already added an include[] parameter
				// (via a query:"include" field), replace it with the richer
				// IncludeConfig version that carries enum values and a proper
				// description.
				includeConfigField := specField.FieldByName("IncludeConfig")
				if includeConfigField.IsValid() && !includeConfigField.IsNil() {
					includeConfig := includeConfigField.Interface().(*apiendpoint.IncludeConfig)
					allowedKeys := includeConfig.AllowedKeys()
					enumValues := make([]any, len(allowedKeys))
					for i, k := range allowedKeys {
						enumValues[i] = k
					}
					includeParam := Parameter{
						Name:        "include[]",
						In:          "query",
						Description: "Sub-objects to expand in the response. When omitted, sub-objects are returned as `null`.",
						Required:    false,
						Schema: Schema{
							Type:  "array",
							Items: &Schema{Type: "string", Enum: enumValues},
						},
						Example: []any{allowedKeys[0]},
					}

					replaced := false
					for i, p := range operation.Parameters {
						if p.Name == "include[]" && p.In == "query" {
							operation.Parameters[i] = includeParam
							replaced = true
							break
						}
					}
					if !replaced {
						operation.Parameters = append(operation.Parameters, includeParam)
					}
				}

				schemaName := getCleanTypeName(reqType)
				if hasJSONFields && schemaName != "" && schemaName != "EmptyResource" {
					// GET and DELETE requests should not have a request body
					if method != "DELETE" && method != "GET" {
						if _, ok := spec.Components.Schemas[schemaName]; !ok {
							spec.Components.Schemas[schemaName] = generateSchema(origReqType, &spec.Components, docReader)
						}

						operation.RequestBody = &RequestBody{
							Description: "The request body for " + title,
							Content: map[string]MediaConfig{
								"application/json": {
									Schema:  Schema{Ref: "#/components/schemas/" + schemaName},
									Example: spec.Components.Schemas[schemaName].Example,
								},
							},
						}
					}
				}
			}

			successStatusCode := int(specField.FieldByName("SuccessStatusCode").Int())
			if successStatusCode == 0 {
				successStatusCode = http.StatusOK
			}
			successStatusCodeStr := fmt.Sprintf("%d", successStatusCode)

			// Handle Response
			respType := e.GetResponseType()
			schemaName := getCleanTypeName(respType)
			if schemaName != "" && schemaName != "EmptyResource" {
				if _, ok := spec.Components.Schemas[schemaName]; !ok {
					spec.Components.Schemas[schemaName] = generateSchema(respType, &spec.Components, docReader, route)
				}
				operation.Responses[successStatusCodeStr] = Response{
					Description: "Successful response for " + title,
					Content: map[string]MediaConfig{
						"application/json": {
							Schema:  Schema{Ref: "#/components/schemas/" + schemaName},
							Example: spec.Components.Schemas[schemaName].Example,
						},
					},
				}
			} else {
				// For EmptyResource or other cases, we still return an empty object in JSON
				operation.Responses[successStatusCodeStr] = Response{
					Description: "Successful response for " + title,
					Content: map[string]MediaConfig{
						"application/json": {
							Schema: Schema{
								Type:                  "object",
								XStainlessEmptyObject: true,
							},
							Example: map[string]any{},
						},
					},
				}
			}
			// Add default error response referencing APIErrorResponse
			operation.Responses["4XX"] = Response{
				Description: "Error response",
				Content: map[string]MediaConfig{
					"application/json": {
						Schema: Schema{Ref: "#/components/schemas/APIErrorResponse"},
					},
				},
			}

			groupPathMap[route][strings.ToLower(method)] = operation
		}

		if groupHasEndpoints {
			if !tagNames[group.Title] {
				spec.Tags = append(spec.Tags, Tag{Name: group.Title, Description: group.Description})
				tagNames[group.Title] = true
			}
			for route, methods := range groupPathMap {
				if spec.Paths[route] == nil {
					spec.Paths[route] = make(map[string]Operation)
				}
				for method, operation := range methods {
					spec.Paths[route][method] = operation
				}
			}
		}
	}

	// Register error response schema for documentation
	apiErrorType := reflect.TypeFor[apierror.APIErrorResponse]()
	if _, ok := spec.Components.Schemas["APIErrorResponse"]; !ok {
		spec.Components.Schemas["APIErrorResponse"] = generateSchema(apiErrorType, &spec.Components, docReader)
	}

	// Collect property orders before marshaling (PropertyOrder is json:"-")
	propertyOrders := collectPropertyOrders(spec.Components.Schemas)

	return spec, propertyOrders, nil
}

func buildOpenAPIDocument(groups []apiendpoint.APIEndpointGroup, publicOnly bool, transforms []Transform, version string) (map[string]any, error) {
	spec, propertyOrders, err := buildOpenAPISpec(groups, publicOnly, version)
	if err != nil {
		return nil, err
	}

	// Marshal to generic map for transforms
	b, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec for transforms: %w", err)
	}
	var data any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal spec for transforms: %w", err)
	}

	// Apply transforms
	data = applyTransforms(data, transforms)

	// Reorder schema properties and examples to match struct field declaration order
	applyPropertyOrders(data, propertyOrders)
	applyExampleOrders(data, propertyOrders)

	root, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("open api document root is %T, want object", data)
	}
	return root, nil
}

// formatOpenAPIJSON marshals an OpenAPI document the same way generate() writes specs/.
// encoding/json sorts map keys (paths, methods, schema names), so endpoint registration
// order does not affect bytes on disk—only route paths and schema content do.
func formatOpenAPIJSON(data map[string]any) ([]byte, error) {
	compact, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal spec: %w", err)
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, compact, "", "  "); err != nil {
		return nil, fmt.Errorf("indent spec: %w", err)
	}
	return indented.Bytes(), nil
}

func generate(groups []apiendpoint.APIEndpointGroup, outputPath string, publicOnly bool, transforms []Transform, version string) {
	data, err := buildOpenAPIDocument(groups, publicOnly, transforms, version)
	if err != nil {
		log.Fatalf("Error building OpenAPI document: %v", err)
	}

	output, err := formatOpenAPIJSON(data)
	if err != nil {
		log.Fatalf("Error formatting spec: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		log.Fatalf("Error creating directory for spec: %v", err)
	}

	err = os.WriteFile(outputPath, output, 0600)
	if err != nil {
		log.Fatalf("Error writing spec to %s: %v", outputPath, err)
	}

	totalEndpoints := 0
	if paths, ok := data["paths"].(map[string]any); ok {
		for _, methods := range paths {
			if mm, ok := methods.(map[string]any); ok {
				totalEndpoints += len(mm)
			}
		}
	}

	specType := "internal"
	if publicOnly {
		specType = "public"
	}
	logInfof("OpenAPI spec generated in %s (%d %s endpoints)\n", outputPath, totalEndpoints, specType)
}

func getCleanTypeName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}

	name := t.Name()
	if name == "" {
		return ""
	}

	// Handle generic types like List[T]
	// Replace [ with _ and remove ]
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "")

	// Handle package paths if they are included in Name() for some reason
	// (they seem to be for generic instantiations)
	if strings.Contains(name, "/") || strings.Contains(name, ".") {
		segments := strings.Split(name, "_")
		for i, seg := range segments {
			if strings.Contains(seg, "/") {
				parts := strings.Split(seg, "/")
				seg = parts[len(parts)-1]
			}
			if strings.Contains(seg, ".") {
				parts := strings.Split(seg, ".")
				seg = parts[len(parts)-1]
			}
			segments[i] = seg
		}
		name = strings.Join(segments, "_")
	}

	return name
}

func generateSchema(t reflect.Type, components *Components, docReader *DocReader, route ...string) Schema {
	origT := t
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Handle time.Time as a string with date-time format
	if t.Kind() == reflect.Struct && t.PkgPath() == "time" && t.Name() == "Time" {
		return Schema{Type: "string", Format: "date-time"}
	}

	// json.RawMessage represents arbitrary JSON (object/array/primitives).
	// For documentation purposes, model it as a generic JSON object.
	// Without this special-case, we'd treat it like a Go slice ([]byte) and
	// incorrectly render it as `type: array` in the OpenAPI schema.
	if t.Kind() == reflect.Slice && t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return Schema{Type: "object"}
	}

	switch t.Kind() {
	case reflect.String:
		schema := Schema{Type: "string"}
		if enumValues := getEnumValuesForStringType(t); len(enumValues) > 0 {
			schema.Enum = enumValues
		}
		return schema
	case reflect.Int, reflect.Int32, reflect.Int64:
		return Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return Schema{Type: "number"}
	case reflect.Bool:
		return Schema{Type: "boolean"}
	case reflect.Slice:
		elemType := t.Elem()
		if elemType.Kind() == reflect.Pointer {
			elemType = elemType.Elem()
		}
		itemSchema := generateSchema(elemType, components, docReader)
		return Schema{Type: "array", Items: &itemSchema}
	}

	var example any
	typeName := getCleanTypeName(origT)

	// Special handling for List types: populate list body examples even when the schema is
	// first registered without a route (e.g. nested expandable List fields).
	if strings.HasPrefix(typeName, "List_") {
		// Extract the item type from List[T] by finding the "data" field
		var itemExample any
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				jsonTag := f.Tag.Get("json")
				if jsonTag == "data" || strings.HasPrefix(jsonTag, "data,") {
					// Found the data field, get its element type
					elemType := f.Type
					if elemType.Kind() == reflect.Slice {
						elemType = elemType.Elem()
						if elemType.Kind() == reflect.Pointer {
							elemType = elemType.Elem()
						}
						// First try to get example from already-generated schema
						itemTypeName := getCleanTypeName(elemType)
						if itemTypeName != "" {
							if existingSchema, ok := components.Schemas[itemTypeName]; ok && existingSchema.Example != nil {
								itemExample = existingSchema.Example
							}
						}
						// If not found, try to get it from DocumentedType
						if itemExample == nil && reflect.PointerTo(elemType).Implements(reflect.TypeFor[contracts.DocumentedType]()) {
							v := reflect.New(elemType).Interface().(contracts.DocumentedType)
							func() {
								defer func() { recover() }()
								itemExample = v.SchemaExample()
							}()
						}
					}
					break
				}
			}
		}
		// Create a List example with the route as the URL
		dataArray := []any{}
		var nextCursor *string
		if itemExample != nil {
			// Convert to map if it's not already
			if itemMap, ok := itemExample.(map[string]any); ok {
				dataArray = []any{itemMap}
				if id, ok := itemMap["id"].(string); ok {
					nextCursor = &id
				}
			} else {
				dataArray = []any{itemExample}
			}
		}

		example = map[string]any{
			"object": "list",
			"page_info": map[string]any{
				"next_page_url":     nextCursor,
				"previous_page_url": nil,
				"has_next_page":     true,
				"has_prev_page":     false,
			},
			"data": dataArray,
		}
	} else if reflect.PointerTo(t).Implements(reflect.TypeFor[contracts.DocumentedType]()) {
		v := reflect.New(t).Interface().(contracts.DocumentedType)
		func() {
			defer func() { recover() }()
			example = v.SchemaExample()
		}()
	}

	// If a documented example encodes `null` for a field (represented as nil in Go),
	// we should mark that field as nullable in the OpenAPI schema even when the
	// Go type isn't a pointer. This keeps the UI's `nullable` pill aligned with
	// real example output.
	var exampleMap map[string]any
	if m, ok := example.(map[string]any); ok {
		exampleMap = m
	}

	typeDoc := docReader.GetTypeDoc(t)
	description := typeDoc.Doc

	schema := Schema{
		Type:        "object",
		Properties:  make(map[string]Schema),
		Description: description,
		Example:     example,
	}

	if t.Kind() != reflect.Struct {
		return schema
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")

		if f.Anonymous && jsonTag == "" {
			embeddedType := f.Type
			if embeddedType.Kind() == reflect.Pointer {
				embeddedType = embeddedType.Elem()
			}
			if embeddedType.Kind() == reflect.Struct {
				embeddedName := getCleanTypeName(embeddedType)
				if embeddedName != "" && embeddedName != "EmptyResource" {
					if _, ok := components.Schemas[embeddedName]; !ok {
						components.Schemas[embeddedName] = Schema{}
						components.Schemas[embeddedName] = generateSchema(embeddedType, components, docReader)
					}
					schema.AllOf = append(schema.AllOf, Schema{Ref: "#/components/schemas/" + embeddedName})
				}
			}
			continue
		}

		if jsonTag == "-" || jsonTag == "" {
			continue
		}
		parts := strings.Split(jsonTag, ",")
		name := parts[0]

		hasRequiredInJSON := false
		hasOmitempty := false
		for _, part := range parts[1:] {
			if part == "required" {
				hasRequiredInJSON = true
			}
			if part == "omitempty" || part == "omitzero" {
				hasOmitempty = true
			}
		}

		validateTag := f.Tag.Get("validate")
		hasRequiredInValidate := strings.Contains(validateTag, "required")

		// patch.Field[T] and patch.Nullable[T] unwrap to the inner type for OpenAPI.
		isPatchField := patch.IsFieldType(f.Type)
		isNullableInput := patch.IsNullableType(f.Type)
		fieldType := f.Type
		if isPatchField {
			if innerType := patch.FieldElemType(f.Type); innerType != nil {
				fieldType = innerType
			}
		} else if isNullableInput {
			if innerType := patch.NullableElemType(f.Type); innerType != nil {
				fieldType = innerType
			}
		}

		isOptionalPointer := f.Type.Kind() == reflect.Pointer && hasOmitempty
		var isRequired bool
		if isPatchField || isNullableInput {
			isRequired = hasRequiredInJSON || hasRequiredInValidate
		} else if f.Type.Kind() == reflect.Slice && hasOmitempty {
			// Slices use omitempty semantics like pointers; empty slice is omitted from JSON.
			isRequired = hasRequiredInJSON || hasRequiredInValidate
		} else {
			isRequired = hasRequiredInJSON || hasRequiredInValidate || !(f.Type.Kind() == reflect.Pointer && hasOmitempty)
		}

		if isRequired {
			schema.Required = append(schema.Required, name)
		}

		fieldSchema := Schema{
			Description: typeDoc.Fields[f.Name],
		}
		switch {
		case isPatchField:
			fieldSchema.Nullable = true
			fieldSchema.XNullableClear = true
		case isNullableInput:
			fieldSchema.Nullable = true
		case f.Type.Kind() == reflect.Pointer && !hasOmitempty:
			// Response-style nullable fields: present in JSON as value or null.
			fieldSchema.Nullable = true
		case isOptionalPointer:
			// Optional input pointers: omit to leave unchanged; explicit null is rejected at runtime.
			fieldSchema.Nullable = false
		}

		// Add Stainless pagination annotations for List types
		if strings.HasPrefix(typeName, "List_") && name == "data" {
			fieldSchema.XStainlessPaginationProperty = map[string]string{"purpose": "items"}
		}

		if enumTag := f.Tag.Get("enum"); enumTag != "" {
			enums := strings.Split(enumTag, ",")
			for _, e := range enums {
				fieldSchema.Enum = append(fieldSchema.Enum, strings.TrimSpace(e))
			}
		}

		for _, part := range strings.Split(validateTag, ",") {
			if strings.HasPrefix(part, "enum=") {
				enumValue := strings.TrimPrefix(part, "enum=")
				fieldSchema.Enum = []any{enumValue}
				break
			}
		}

		if f.Tag.Get("expandable") == "true" {
			fieldSchema.XExpandable = true
		}

		if f.Tag.Get("readOnly") == "true" {
			fieldSchema.ReadOnly = true
		}

		if defaultVal := f.Tag.Get("default"); defaultVal != "" {
			fieldSchema.Default = defaultVal
		}

		if formatVal := f.Tag.Get("format"); formatVal != "" {
			fieldSchema.Format = formatVal
		}

		if !isPatchField && !isNullableInput {
			fieldType = f.Type
		}
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.String:
			fieldSchema.Type = "string"
			if len(fieldSchema.Enum) == 0 {
				if enumValues := getEnumValuesForStringType(fieldType); len(enumValues) > 0 {
					fieldSchema.Enum = enumValues
				}
			}
		case reflect.Int, reflect.Int32, reflect.Int64:
			fieldSchema.Type = "integer"
		case reflect.Float32, reflect.Float64:
			fieldSchema.Type = "number"
		case reflect.Bool:
			fieldSchema.Type = "boolean"
		case reflect.Struct:
			if fieldType.PkgPath() == "time" && fieldType.Name() == "Time" {
				fieldSchema.Type = "string"
				fieldSchema.Format = "date-time"
			} else if name := getCleanTypeName(fieldType); name != "" {
				schemaName := name
				if _, ok := components.Schemas[schemaName]; !ok {
					components.Schemas[schemaName] = Schema{}
					components.Schemas[schemaName] = generateSchema(fieldType, components, docReader)
				}
				// In OpenAPI 3.0.x, $ref cannot have sibling properties.
				// Always use allOf to wrap the reference so description can be a sibling of allOf.
				fieldSchema.AllOf = []Schema{{Ref: "#/components/schemas/" + schemaName}}
				if ex := components.Schemas[schemaName].Example; ex != nil {
					fieldSchema.Example = ex
				}
			} else {
				fieldSchema = generateSchema(fieldType, components, docReader)
			}
		case reflect.Slice:
			// Treat json.RawMessage as a JSON object for docs. RawMessage is a
			// Go slice ([]byte), so without this check we'd render it as an array.
			if fieldType.PkgPath() == "encoding/json" && fieldType.Name() == "RawMessage" {
				fieldSchema.Type = "object"
				// Absent values are nil RawMessage, which encode as JSON null.
				fieldSchema.Nullable = true
				const jsonValueHint = " Encoded as a JSON value (object, array, string, number, boolean, or null), not a JSON-encoded string."
				desc := fieldSchema.Description
				if desc == "" {
					fieldSchema.Description = strings.TrimSpace("Arbitrary JSON." + jsonValueHint)
				} else if !strings.Contains(desc, "JSON-encoded string") {
					fieldSchema.Description = desc + jsonValueHint
				}
				break
			}

			fieldSchema.Type = "array"
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}

			if elemType.Kind() == reflect.Struct && getCleanTypeName(elemType) != "" {
				schemaName := getCleanTypeName(elemType)
				if _, ok := components.Schemas[schemaName]; !ok {
					components.Schemas[schemaName] = Schema{}
					components.Schemas[schemaName] = generateSchema(elemType, components, docReader)
				}
				fieldSchema.Items = &Schema{Ref: "#/components/schemas/" + schemaName}
			} else {
				elemSchema := generateSchema(elemType, components, docReader)
				fieldSchema.Items = &elemSchema
			}
		case reflect.Map:
			fieldSchema.Type = "object"
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}
			elemSchema := generateSchema(elemType, components, docReader)
			fieldSchema.AdditionalProperties = &elemSchema
		}

		// In OpenAPI 3.0.x, a nullable enum must include null in the values list.
		if fieldSchema.Nullable && len(fieldSchema.Enum) > 0 {
			fieldSchema.Enum = append(fieldSchema.Enum, nil)
		}

		// If the schema example provides `null` for this property, mark it nullable.
		// This is independent of the Go type so we don't miss nullable badges when
		// the API chooses to return null even for non-pointer fields.
		if exampleMap != nil {
			if val, exists := exampleMap[name]; exists && isJSONNullish(val) {
				fieldSchema.Nullable = true
				// Keep nullable-enum null-inclusion consistent with the updated flag.
				// Only append nil if not already present (the block above may have added it).
				if len(fieldSchema.Enum) > 0 && !enumContainsNil(fieldSchema.Enum) {
					fieldSchema.Enum = append(fieldSchema.Enum, nil)
				}
			}
		}

		schema.Properties[name] = fieldSchema
		schema.PropertyOrder = append(schema.PropertyOrder, name)
	}

	// Hardening: Some request examples may be produced by marshaling the full Go
	// struct (including fields that are marked as `path` or otherwise not part
	// of the JSON payload). Filter object examples down to schema properties so
	// `requestBody.example` never includes invalid keys.
	schema.Example = fillOptionalBooleanExampleDefaults(
		filterExampleToSchemaProperties(schema.Example, schema),
		schema,
	)

	if len(schema.Properties) == 0 && schema.AdditionalProperties == nil && len(schema.AllOf) == 0 {
		schema.XStainlessEmptyObject = true
	}

	return schema
}

// fillOptionalBooleanExampleDefaults adds `false` for optional, non-nullable boolean
// properties missing from an object example. STLC uses `null` in generated SDK tests
// for example-absent optional params; without an explicit boolean example value those
// tests fail TypeScript lint (`null` is not assignable to `boolean | undefined`).
func fillOptionalBooleanExampleDefaults(example any, schema Schema) any {
	if schema.Type != "object" || len(schema.Properties) == 0 {
		return example
	}

	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}

	ex, ok := example.(map[string]any)
	if !ok {
		if example == nil {
			ex = make(map[string]any)
		} else {
			return example
		}
	}

	filled := make(map[string]any, len(ex)+len(schema.Properties))
	for k, v := range ex {
		filled[k] = v
	}

	for name, prop := range schema.Properties {
		if prop.Type != "boolean" || prop.Nullable {
			continue
		}
		if _, isRequired := required[name]; isRequired {
			continue
		}
		if _, exists := filled[name]; exists {
			continue
		}
		filled[name] = false
	}

	return filled
}

func filterExampleToSchemaProperties(example any, schema Schema) any {
	if example == nil {
		return example
	}

	// Only filter map-like object examples when the schema has known properties.
	if schema.Type == "object" && (len(schema.Properties) > 0 || schema.AdditionalProperties != nil) {
		m, ok := example.(map[string]any)
		if !ok {
			return example
		}
		filtered := make(map[string]any, len(m))
		for k, v := range m {
			if propSchema, ok := schema.Properties[k]; ok {
				filtered[k] = filterExampleToSchemaProperties(v, propSchema)
				continue
			}
			// If additionalProperties are allowed, keep unknown keys but still
			// recursively filter nested structure when possible.
			if schema.AdditionalProperties != nil {
				filtered[k] = filterExampleToSchemaProperties(v, *schema.AdditionalProperties)
			}
		}
		return filtered
	}

	// Recursively filter arrays if we have an item schema.
	if schema.Type == "array" && schema.Items != nil {
		if arr, ok := example.([]any); ok {
			for i := range arr {
				arr[i] = filterExampleToSchemaProperties(arr[i], *schema.Items)
			}
			return arr
		}
	}

	return example
}

func isJSONNullish(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func getEnumValuesForStringType(t reflect.Type) []any {
	if t.Kind() != reflect.String || t.Name() == "" || t.Name() == "string" {
		return nil
	}

	pkgPath := t.PkgPath()
	if pkgPath == "" {
		return nil
	}

	ptrType := reflect.PointerTo(t)
	method, ok := ptrType.MethodByName("EnumValues")
	if !ok {
		return nil
	}

	if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 {
		return nil
	}

	outType := method.Type.Out(0)
	if outType.Kind() != reflect.Slice || outType.Elem().Kind() != reflect.String {
		return nil
	}

	zeroVal := reflect.New(t)
	results := method.Func.Call([]reflect.Value{zeroVal})
	if len(results) != 1 {
		return nil
	}

	slice := results[0]
	enumValues := make([]any, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		enumValues[i] = slice.Index(i).String()
	}

	return enumValues
}

func parameterExample(s Schema) any {
	if s.Type == "array" && s.Items != nil {
		if len(s.Items.Enum) > 0 {
			return []any{s.Items.Enum[0]}
		}
		return []any{}
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}
	switch s.Type {
	case "string":
		return "example"
	case "integer":
		return 100
	case "boolean":
		return true
	}
	return nil
}

// collectPropertyOrders extracts the tracked property insertion order from
// each component schema before marshaling (PropertyOrder has json:"-").
func collectPropertyOrders(schemas map[string]Schema) map[string][]string {
	orders := make(map[string][]string)
	for name, schema := range schemas {
		if len(schema.PropertyOrder) > 0 {
			orders[name] = schema.PropertyOrder
		}
	}
	return orders
}

// applyPropertyOrders walks the generic spec tree and replaces each schema's
// properties map with an orderedJSONMap so the final JSON output preserves
// struct field declaration order.
func applyPropertyOrders(data any, orders map[string][]string) {
	root, ok := data.(map[string]any)
	if !ok {
		return
	}
	components, ok := root["components"].(map[string]any)
	if !ok {
		return
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return
	}
	for name, order := range orders {
		schemaRaw, ok := schemas[name].(map[string]any)
		if !ok {
			continue
		}
		props, ok := schemaRaw["properties"].(map[string]any)
		if !ok {
			continue
		}
		schemaRaw["properties"] = orderedJSONMap{order: order, values: props}
	}
}

// applyExampleOrders walks the generic spec tree and wraps example maps with
// orderedJSONMap so examples match struct field declaration order.
func applyExampleOrders(data any, orders map[string][]string) {
	root, ok := data.(map[string]any)
	if !ok {
		return
	}
	components, _ := root["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		return
	}

	// Order examples in component schemas
	for name := range orders {
		schemaRaw, ok := schemas[name].(map[string]any)
		if !ok {
			continue
		}
		if example, ok := schemaRaw["example"].(map[string]any); ok {
			schemaRaw["example"] = orderExampleForSchema(example, name, schemas, orders)
		}
	}

	// Order field-level examples in schema properties
	for name := range orders {
		schemaRaw, ok := schemas[name].(map[string]any)
		if !ok {
			continue
		}
		var props map[string]any
		switch p := schemaRaw["properties"].(type) {
		case orderedJSONMap:
			props = p.values
		case map[string]any:
			props = p
		}
		for _, propRaw := range props {
			propMap, ok := propRaw.(map[string]any)
			if !ok {
				continue
			}
			if ex, ok := propMap["example"].(map[string]any); ok {
				if refName := resolveRefName(propMap); refName != "" && orders[refName] != nil {
					propMap["example"] = orderExampleForSchema(ex, refName, schemas, orders)
				}
			}
		}
	}

	// Order examples in paths (request bodies and responses)
	paths, _ := root["paths"].(map[string]any)
	for _, methods := range paths {
		methodMap, _ := methods.(map[string]any)
		for _, op := range methodMap {
			opMap, _ := op.(map[string]any)
			if rb, ok := opMap["requestBody"].(map[string]any); ok {
				orderMediaExamples(rb["content"], schemas, orders)
			}
			if responses, ok := opMap["responses"].(map[string]any); ok {
				for _, resp := range responses {
					if respMap, ok := resp.(map[string]any); ok {
						orderMediaExamples(respMap["content"], schemas, orders)
					}
				}
			}
		}
	}
}

// orderMediaExamples finds examples in media type content and wraps them
// with orderedJSONMap based on the referenced schema's property order.
func orderMediaExamples(content any, schemas map[string]any, orders map[string][]string) {
	contentMap, ok := content.(map[string]any)
	if !ok {
		return
	}
	for _, media := range contentMap {
		mediaMap, ok := media.(map[string]any)
		if !ok {
			continue
		}
		example, ok := mediaMap["example"].(map[string]any)
		if !ok {
			continue
		}
		schemaName := resolveRefName(mediaMap["schema"])
		if schemaName == "" || orders[schemaName] == nil {
			continue
		}
		mediaMap["example"] = orderExampleForSchema(example, schemaName, schemas, orders)
	}
}

// orderExampleForSchema recursively wraps example maps with orderedJSONMap
// to match the struct field order defined in the schema.
func orderExampleForSchema(example any, schemaName string, allSchemas map[string]any, orders map[string][]string) any {
	m, ok := example.(map[string]any)
	if !ok {
		if arr, ok := example.([]any); ok {
			for i, item := range arr {
				arr[i] = orderExampleForSchema(item, schemaName, allSchemas, orders)
			}
		}
		return example
	}

	order := orders[schemaName]
	if len(order) == 0 {
		return example
	}

	schema, _ := allSchemas[schemaName].(map[string]any)
	if schema == nil {
		return example
	}

	// Get properties map (may already be orderedJSONMap from applyPropertyOrders)
	var props map[string]any
	switch p := schema["properties"].(type) {
	case orderedJSONMap:
		props = p.values
	case map[string]any:
		props = p
	}

	// Recursively order nested objects and arrays
	for key, val := range m {
		propSchema, _ := props[key].(map[string]any)
		if propSchema == nil {
			continue
		}
		switch v := val.(type) {
		case map[string]any:
			if refName := resolveRefName(propSchema); refName != "" {
				m[key] = orderExampleForSchema(v, refName, allSchemas, orders)
			}
		case []any:
			items, _ := propSchema["items"].(map[string]any)
			if items != nil {
				if itemRef := resolveRefName(items); itemRef != "" {
					for i, item := range v {
						v[i] = orderExampleForSchema(item, itemRef, allSchemas, orders)
					}
				}
			}
		}
	}

	return orderedJSONMap{order: order, values: m}
}

// resolveRefName extracts the schema name from a $ref or allOf containing a $ref.
func resolveRefName(schema any) string {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return ""
	}
	if ref, ok := schemaMap["$ref"].(string); ok {
		return refLastSegment(ref)
	}
	if allOf, ok := schemaMap["allOf"].([]any); ok {
		for _, item := range allOf {
			if itemMap, ok := item.(map[string]any); ok {
				if ref, ok := itemMap["$ref"].(string); ok {
					return refLastSegment(ref)
				}
			}
		}
	}
	return ""
}

// refLastSegment returns the last path segment of a $ref string.
func refLastSegment(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// getFieldDoc returns the doc comment for a struct field, handling promoted fields from embedded structs.
func getFieldDoc(structType reflect.Type, field reflect.StructField, docReader *DocReader) string {
	declaringType := structType
	if len(field.Index) > 1 {
		for _, idx := range field.Index[:len(field.Index)-1] {
			ft := declaringType.Field(idx).Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			declaringType = ft
		}
	}
	typeDoc := docReader.GetTypeDoc(declaringType)
	return typeDoc.Fields[field.Name]
}

// flattenStructFields returns all fields of a struct type, recursively expanding
// anonymous (embedded) struct fields so that promoted fields are included directly.
// Each returned field's Index is rewritten to be the full path from t, so downstream
// helpers (e.g. getFieldDoc) can resolve the declaring type of promoted fields.
func flattenStructFields(t reflect.Type) []reflect.StructField {
	return flattenStructFieldsWithPrefix(t, nil)
}

func flattenStructFieldsWithPrefix(t reflect.Type, prefix []int) []reflect.StructField {
	var fields []reflect.StructField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		idx := make([]int, 0, len(prefix)+1)
		idx = append(idx, prefix...)
		idx = append(idx, i)
		if f.Anonymous {
			embedded := f.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				fields = append(fields, flattenStructFieldsWithPrefix(embedded, idx)...)
				continue
			}
		}
		f.Index = idx
		fields = append(fields, f)
	}
	return fields
}

// enumContainsNil returns true if the enum slice already contains a nil entry.
func enumContainsNil(enum []any) bool {
	for _, v := range enum {
		if v == nil {
			return true
		}
	}
	return false
}
