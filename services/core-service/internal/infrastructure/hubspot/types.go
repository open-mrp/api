package hubspot

// objectInput is the request body for creating or updating a CRM object.
type objectInput struct {
	Properties map[string]string `json:"properties"`
}

// objectResponse is a single CRM object as returned by the API.
type objectResponse struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
}

// searchRequest is the body for the CRM search endpoints.
type searchRequest struct {
	FilterGroups []filterGroup `json:"filterGroups"`
	Properties   []string      `json:"properties,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	After        string        `json:"after,omitempty"`
}

type filterGroup struct {
	Filters []filter `json:"filters"`
}

type filter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

// searchResponse is the result of a CRM search.
type searchResponse struct {
	Total   int              `json:"total"`
	Results []objectResponse `json:"results"`
	Paging  *paging          `json:"paging,omitempty"`
}

// listResponse is the result of a paginated CRM list (GET).
type listResponse struct {
	Results []objectResponse `json:"results"`
	Paging  *paging          `json:"paging,omitempty"`
}

type paging struct {
	Next *pagingNext `json:"next,omitempty"`
}

type pagingNext struct {
	After string `json:"after"`
}

// apiErrorResponse is the standard HubSpot API error body.
type apiErrorResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Category string `json:"category"`
}

// propertyDefinition is the request body for creating a custom CRM property.
type propertyDefinition struct {
	Name           string `json:"name"`
	Label          string `json:"label"`
	Type           string `json:"type"`
	FieldType      string `json:"fieldType"`
	GroupName      string `json:"groupName"`
	HasUniqueValue bool   `json:"hasUniqueValue,omitempty"`
}
