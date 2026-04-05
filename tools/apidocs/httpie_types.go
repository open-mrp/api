package main

// HTTPieWorkspace is the top-level structure for an HTTPie Desktop workspace export.
type HTTPieWorkspace struct {
	Meta  HTTPieMeta  `json:"meta"`
	Entry HTTPieEntry `json:"entry"`
}

type HTTPieMeta struct {
	Format      string `json:"format"`
	Version     string `json:"version"`
	ContentType string `json:"contentType"`
	Schema      string `json:"schema"`
	Docs        string `json:"docs"`
	Source      string `json:"source"`
}

type HTTPieEntry struct {
	Name         string              `json:"name"`
	Icon         HTTPieIcon          `json:"icon"`
	Collections  []HTTPieCollection  `json:"collections"`
	Environments []HTTPieEnvironment `json:"environments"`
	Drafts       []any               `json:"drafts"`
}

type HTTPieIcon struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type HTTPieCollection struct {
	Name     string          `json:"name"`
	Icon     HTTPieIcon      `json:"icon"`
	Auth     HTTPieAuth      `json:"auth"`
	Requests []HTTPieRequest `json:"requests"`
}

type HTTPieAuth struct {
	Type        string             `json:"type"`
	Target      string             `json:"target,omitempty"`
	Credentials *HTTPieCredentials `json:"credentials,omitempty"`
}

type HTTPieCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"` // #nosec G117 -- HTTPie auth schema field, not a real credential
}

type HTTPieRequest struct {
	Name        string         `json:"name"`
	URL         string         `json:"url"`
	Method      string         `json:"method"`
	Headers     []HTTPieHeader `json:"headers"`
	QueryParams []HTTPieParam  `json:"queryParams"`
	PathParams  []HTTPieParam  `json:"pathParams"`
	Auth        HTTPieAuth     `json:"auth"`
	Body        HTTPieBody     `json:"body"`
}

type HTTPieHeader struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type HTTPieParam struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type HTTPieBody struct {
	Type    string        `json:"type"`
	File    HTTPieFile    `json:"file"`
	Text    HTTPieText    `json:"text"`
	Form    HTTPieForm    `json:"form"`
	GraphQL HTTPieGraphQL `json:"graphql"`
}

type HTTPieFile struct {
	Name string `json:"name"`
}

type HTTPieText struct {
	Value  string `json:"value"`
	Format string `json:"format"`
}

type HTTPieForm struct {
	IsMultipart bool  `json:"isMultipart"`
	Fields      []any `json:"fields"`
}

type HTTPieGraphQL struct {
	Query     string `json:"query"`
	Variables string `json:"variables"`
}

type HTTPieEnvironment struct {
	Name        string           `json:"name"`
	Color       string           `json:"color"`
	IsDefault   bool             `json:"isDefault"`
	IsLocalOnly bool             `json:"isLocalOnly"`
	Variables   []HTTPieVariable `json:"variables"`
}

type HTTPieVariable struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}
