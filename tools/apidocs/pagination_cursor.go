package main

import (
	"strings"
	"time"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/pagination"
)

// listElementTypesUsingInt64Cursor are list item types whose backends paginate with
// pagination.Cursor (internal_id int64) rather than pagination.StringCursor.
var listElementTypesUsingInt64Cursor = map[string]struct{}{
	"APIKey":              {},
	"Sandbox":             {},
	"PricingPlan":         {},
	"RegistrationSession": {},
}

func documentationNextCursor(itemTypeName string, itemMap map[string]any) string {
	occurredAt := documentationOccurredAt(itemMap)
	if occurredAt.IsZero() {
		occurredAt = apiresource.SampleAnalyticsPeriodStart
	}

	if _, ok := listElementTypesUsingInt64Cursor[itemTypeName]; ok {
		return pagination.EncodeDocumentationCursor(occurredAt, apiresource.SamplePaginationInternalID)
	}

	id, _ := itemMap["id"].(string)
	if id == "" {
		return ""
	}
	return pagination.EncodeDocumentationStringCursor(occurredAt, id)
}

func documentationOccurredAt(itemMap map[string]any) time.Time {
	for _, key := range []string{"created_at", "occurred_at"} {
		if v, ok := itemMap[key]; ok {
			if t := parseDocumentationTime(v); !t.IsZero() {
				return t
			}
		}
	}
	return time.Time{}
}

func parseDocumentationTime(v any) time.Time {
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	case time.Time:
		return val
	}
	return time.Time{}
}

func isSignedPaginationCursor(cursor string) bool {
	return strings.Count(cursor, ".") == 1
}
