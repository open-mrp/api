package main

import (
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
)

func generate(groups []apiendpoint.APIEndpointGroup, outputPath string, publicOnly bool, transforms []Transform, version string) {
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
			{URL: "http://localhost:8080", Description: "Local development server"},
		},
		Paths: make(map[string]map[string]Operation),
		Components: Components{
			Schemas: make(map[string]Schema),
		},
		Tags: []Tag{},
	}

	tagNames := make(map[string]bool)
	apiErrorResponseRegistered := false

	for _, group := range groups {
		groupHasEndpoints := false
		groupPathMap := make(map[string]map[string]Operation)

		for _, e := range group.Endpoints {
			val := reflect.ValueOf(e)
			if val.Kind() == reflect.Pointer {
				val = val.Elem()
			}

			specField := val.FieldByName("APIEndpoint")
			if !specField.IsValid() {
				continue
			}

			isPublic := specField.FieldByName("IsPublic").Bool()
			if publicOnly && !isPublic {
				continue
			}

			groupHasEndpoints = true
			title := specField.FieldByName("Title").String()
			description := specField.FieldByName("Description").String()
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
			}

			// Handle Parameters and Request Body
			reqVal := specField.FieldByName("Request")
			reqType := reqVal.Type()
			if reqType.Kind() != reflect.Interface {
				origReqType := reqType
				if reqType.Kind() == reflect.Pointer {
					reqType = reqType.Elem()
				}

				hasJSONFields := false
				if reqType.Kind() == reflect.Struct {
					for i := 0; i < reqType.NumField(); i++ {
						f := reqType.Field(i)

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
							operation.Parameters = append(operation.Parameters, Parameter{
								Name:        query,
								In:          "query",
								Description: fmt.Sprintf("Query parameter: %s for %s", query, title),
								Required:    strings.Contains(f.Tag.Get("validate"), "required"),
								Schema:      generateSchema(f.Type, &spec.Components, docReader),
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
							operation.Parameters = append(operation.Parameters, Parameter{
								Name:        pathParam,
								In:          "path",
								Description: fmt.Sprintf("Path parameter: %s for %s", pathParam, title),
								Required:    true,
								Schema:      generateSchema(f.Type, &spec.Components, docReader),
							})
						}

						// Check if it has JSON fields for the request body
						jsonTag := f.Tag.Get("json")
						if jsonTag != "" && jsonTag != "-" {
							hasJSONFields = true
						}
					}
				}

				schemaName := getTypeName(reqType)
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
			respVal := specField.FieldByName("Response")
			respType := respVal.Type()
			schemaName := getTypeName(respType)
			if schemaName != "" && schemaName != "EmptyResource" {
				if _, ok := spec.Components.Schemas[schemaName]; !ok {
					spec.Components.Schemas[schemaName] = generateSchema(respType, &spec.Components, docReader)
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

			// Add default error responses
			errorStatusCodes := []int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusConflict,
				http.StatusTooManyRequests,
				http.StatusInternalServerError,
			}
			for _, code := range errorStatusCodes {
				codeStr := fmt.Sprintf("%d", code)
				if _, ok := operation.Responses[codeStr]; !ok {
					if !apiErrorResponseRegistered {
						apiErrorType := reflect.TypeFor[contracts.APIErrorResponse]()
						spec.Components.Schemas["APIErrorResponse"] = generateSchema(apiErrorType, &spec.Components, docReader)
						apiErrorResponseRegistered = true
					}
					operation.Responses[codeStr] = Response{
						Description: fmt.Sprintf("%s response for %s", http.StatusText(code), title),
						Content: map[string]MediaConfig{
							"application/json": {
								Schema:  Schema{Ref: "#/components/schemas/APIErrorResponse"},
								Example: spec.Components.Schemas["APIErrorResponse"].Example,
							},
						},
					}
				}
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

	// Marshal to generic map for transforms
	b, err := json.Marshal(spec)
	if err != nil {
		log.Fatalf("Error marshaling spec for transforms: %v", err)
	}
	var data any
	if err := json.Unmarshal(b, &data); err != nil {
		log.Fatalf("Error unmarshaling spec for transforms: %v", err)
	}

	// Apply transforms
	data = applyTransforms(data, transforms)

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling spec: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		log.Fatalf("Error creating directory for spec: %v", err)
	}

	err = os.WriteFile(outputPath, output, 0600)
	if err != nil {
		log.Fatalf("Error writing spec to %s: %v", outputPath, err)
	}

	log.Printf("OpenAPI spec generated in %s\n", outputPath)
}

func getTypeName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	return t.Name()
}

func generateSchema(t reflect.Type, components *Components, docReader *DocReader) Schema {
	origT := t
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return Schema{Type: "string"}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return Schema{Type: "number"}
	case reflect.Bool:
		return Schema{Type: "boolean"}
	}

	var example any
	if origT.Implements(reflect.TypeFor[contracts.DocumentedType]()) {
		v := reflect.New(t).Interface().(contracts.DocumentedType)
		func() {
			defer func() { recover() }()
			example = v.SchemaExample()
		}()
	}

	typeDoc := docReader.GetTypeDoc(t)

	schema := Schema{
		Type:        "object",
		Properties:  make(map[string]Schema),
		Description: typeDoc.Doc,
		Example:     example,
	}

	if t.Kind() != reflect.Struct {
		return schema
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
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
			if part == "omitempty" {
				hasOmitempty = true
			}
		}

		validateTag := f.Tag.Get("validate")
		hasRequiredInValidate := strings.Contains(validateTag, "required")

		isRequired := hasRequiredInJSON || hasRequiredInValidate || !hasOmitempty

		if isRequired {
			schema.Required = append(schema.Required, name)
		}

		fieldSchema := Schema{
			Description: typeDoc.Fields[f.Name],
			Example:     f.Tag.Get("example"),
			Nullable:    f.Type.Kind() == reflect.Pointer,
		}

		if enumTag := f.Tag.Get("enum"); enumTag != "" {
			enums := strings.Split(enumTag, ",")
			for _, e := range enums {
				fieldSchema.Enum = append(fieldSchema.Enum, strings.TrimSpace(e))
			}
		}

		if fieldSchema.Example == "" {
			fieldSchema.Example = nil
		}

		if f.Tag.Get("readOnly") == "true" {
			fieldSchema.ReadOnly = true
		}

		if defaultVal := f.Tag.Get("default"); defaultVal != "" {
			fieldSchema.Default = defaultVal
		}

		fieldType := f.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.String:
			fieldSchema.Type = "string"
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
			} else if fieldType.Name() != "" {
				schemaName := fieldType.Name()
				if _, ok := components.Schemas[schemaName]; !ok {
					components.Schemas[schemaName] = Schema{}
					components.Schemas[schemaName] = generateSchema(fieldType, components, docReader)
				}
				fieldSchema.Ref = "#/components/schemas/" + schemaName
			} else {
				fieldSchema = generateSchema(fieldType, components, docReader)
			}
		case reflect.Slice:
			fieldSchema.Type = "array"
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}

			if elemType.Kind() == reflect.Struct && elemType.Name() != "" {
				schemaName := elemType.Name()
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

		schema.Properties[name] = fieldSchema
	}

	if len(schema.Properties) == 0 && schema.AdditionalProperties == nil {
		schema.XStainlessEmptyObject = true
	}

	return schema
}
