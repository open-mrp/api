package apiresource

import "github.com/augno/api/shared/constants"

// List represents a paginated list of resources
type List[T any] struct {
	// The object type, always "list"
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Whether there are more results available after this page
	HasMore bool `json:"has_more"`
	// Cursor for fetching the next page, null if on last page
	NextCursor *string `json:"next_cursor"`
	// The array of resources in this page
	Data []T `json:"data" validate:"required"`
}

func NewList[T any](data []T, hasMore bool, nextCursor *string) *List[T] {
	if data == nil {
		data = []T{}
	}
	return &List[T]{
		Object:     constants.ObjectTypeList,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Data:       data,
	}
}
