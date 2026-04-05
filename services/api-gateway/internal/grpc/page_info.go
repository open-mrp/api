package grpc

import (
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

// ProtoPageInfo is an interface satisfied by all proto-generated PageInfo types
// across the core, auth, platform, billing, and agent proto packages.
type ProtoPageInfo interface {
	GetNextCursor() string
	GetPrevCursor() string
	GetHasNextPage() bool
	GetHasPrevPage() bool
}

// MapProtoPageInfo converts any proto PageInfo into an API resource PageInfo.
func MapProtoPageInfo(pi ProtoPageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextCursor:  optionalString(pi.GetNextCursor()),
		PrevCursor:  optionalString(pi.GetPrevCursor()),
		HasNextPage: pi.GetHasNextPage(),
		HasPrevPage: pi.GetHasPrevPage(),
	}
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
