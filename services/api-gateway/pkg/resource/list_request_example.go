package apiresource

import "github.com/open-mrp/apikit/pagination"

// ListRequestSchemaExample documents the cursor, limit and q query parameters of a list request, for a request type's SchemaExample to extend.
func ListRequestSchemaExample() map[string]any {
	return map[string]any{
		"cursor": pagination.EncodeDocumentationStringCursor(SampleAnalyticsPeriodStart, SampleItemID),
		"limit":  int64(25),
		"q":      "6061",
	}
}
