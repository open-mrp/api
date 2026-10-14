package apiresource

import "github.com/open-mrp/apikit/object"

// PageInfo is apikit's page_info: another page exists exactly when its URL is present.
type PageInfo = object.PageInfo

// List is apikit's paginated list.
type List[T any] = object.List[T]

// NewList creates a new List of resources. A nil slice becomes an empty list.
func NewList[T any](data []T, pageInfo PageInfo) *List[T] {
	return object.NewList(data, pageInfo)
}
