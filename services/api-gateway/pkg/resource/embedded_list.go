package apiresource

import (
	"net/url"

	"github.com/open-mrp/apikit/object"
)

// EmbeddedListLimit is how many items an expanded list inside a resource holds.
const EmbeddedListLimit = object.EmbeddedListLimit

// NewEmbeddedList is an expanded list inside a resource: its first EmbeddedListLimit items, and when there are more (items holds more than the limit, or more is set), a next_page_url at the sub-resource's own list endpoint so nothing is silently cut off.
//
// Load at least EmbeddedListLimit+1 items so a longer list is noticed without counting it. listPath is the sub-resource's list route, such as /v1/operations/work-orders/wkor_1/jobs. cursorAfter, when given, is that endpoint's cursor for the page after an item, so the link continues after the embedded items; without it the link opens the list at its first page.
func NewEmbeddedList[T any](items []T, more bool, listPath string, cursorAfter func(last T) string) *List[T] {
	if len(items) > EmbeddedListLimit {
		items, more = items[:EmbeddedListLimit], true
	}
	var page PageInfo
	if more && listPath != "" {
		next := listPath
		if cursorAfter != nil && len(items) > 0 {
			next += "?" + url.Values{"cursor": {cursorAfter(items[len(items)-1])}}.Encode()
		}
		page.NextPageURL = &next
	}
	return NewList(items, page)
}
