package constants

// HTTPMethod represents an HTTP request method.
type HTTPMethod string

const (
	// HTTPMethodGet represents the GET method.
	HTTPMethodGet HTTPMethod = "GET"
	// HTTPMethodPost represents the POST method.
	HTTPMethodPost HTTPMethod = "POST"
	// HTTPMethodPut represents the PUT method.
	HTTPMethodPut HTTPMethod = "PUT"
	// HTTPMethodPatch represents the PATCH method.
	HTTPMethodPatch HTTPMethod = "PATCH"
	// HTTPMethodDelete represents the DELETE method.
	HTTPMethodDelete HTTPMethod = "DELETE"
	// HTTPMethodHead represents the HEAD method.
	HTTPMethodHead HTTPMethod = "HEAD"
	// HTTPMethodOptions represents the OPTIONS method.
	HTTPMethodOptions HTTPMethod = "OPTIONS"
)

func (m HTTPMethod) IsValid() bool {
	switch m {
	case HTTPMethodGet, HTTPMethodPost, HTTPMethodPut, HTTPMethodPatch, HTTPMethodDelete, HTTPMethodHead, HTTPMethodOptions:
		return true
	default:
		return false
	}
}

func (m HTTPMethod) EnumValues() []string {
	return []string{
		string(HTTPMethodGet),
		string(HTTPMethodPost),
		string(HTTPMethodPut),
		string(HTTPMethodPatch),
		string(HTTPMethodDelete),
		string(HTTPMethodHead),
		string(HTTPMethodOptions),
	}
}

func (m *HTTPMethod) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
