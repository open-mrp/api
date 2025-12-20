package main

// OpenAPI structure
type OpenAPI struct {
	OpenAPI    string                          `json:"openapi"`
	Info       Info                            `json:"info"`
	Servers    []Server                        `json:"servers,omitempty"`
	Paths      map[string]map[string]Operation `json:"paths"`
	Components Components                      `json:"components"`
	Tags       []Tag                           `json:"tags,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Operation structure
type Operation struct {
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Schema      Schema `json:"schema"`
}

type RequestBody struct {
	Description string                 `json:"description,omitempty"`
	Content     map[string]MediaConfig `json:"content"`
	Required    bool                   `json:"required,omitempty"`
}

type Response struct {
	Description string                 `json:"description"`
	Content     map[string]MediaConfig `json:"content,omitempty"`
}

type MediaConfig struct {
	Schema  Schema `json:"schema"`
	Example any    `json:"example,omitempty"`
}

type Schema struct {
	Ref                   string            `json:"$ref,omitempty"`
	Type                  string            `json:"type,omitempty"`
	Items                 *Schema           `json:"items,omitempty"`
	Properties            map[string]Schema `json:"properties,omitempty"`
	Description           string            `json:"description,omitempty"`
	Example               any               `json:"example,omitempty"`
	Format                string            `json:"format,omitempty"`
	Required              []string          `json:"required,omitempty"`
	Enum                  []any             `json:"enum,omitempty"`
	OneOf                 []Schema          `json:"oneOf,omitempty"`
	AnyOf                 []Schema          `json:"anyOf,omitempty"`
	AllOf                 []Schema          `json:"allOf,omitempty"`
	ReadOnly              bool              `json:"readOnly,omitempty"`
	Default               any               `json:"default,omitempty"`
	AdditionalProperties  *Schema           `json:"additionalProperties,omitempty"`
	XStainlessEmptyObject bool              `json:"x-stainless-empty-object,omitempty"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}
