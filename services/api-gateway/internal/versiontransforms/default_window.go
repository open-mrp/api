package versiontransforms

import "net/url"

// unboundedStart is earlier than any logged event, so passing it as starts_at reproduces a list that
// searched its whole history. preview.5 gave log lists a default window; older versions keep this.
const unboundedStart = "1970-01-01T00:00:00Z"

// searchWholeHistoryWithoutStart pins starts_at to the beginning of time on listRoute when the caller
// omitted it. Other routes of the same object type (a retrieve, an export) take no window and are left
// alone, since some reject parameters they do not know.
func searchWholeHistoryWithoutStart(route, listRoute string, query url.Values) url.Values {
	if route != listRoute || query.Get("starts_at") != "" {
		return query
	}
	query.Set("starts_at", unboundedStart)
	return query
}
