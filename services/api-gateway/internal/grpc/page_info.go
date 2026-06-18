package grpc

import (
	"context"
	"net/url"

	"github.com/augno/api/shared/appctx"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

// ProtoPageInfo is an interface satisfied by all proto-generated PageInfo types across the core, auth, platform, billing, and agent proto packages.
type ProtoPageInfo interface {
	GetNextCursor() string
	GetPrevCursor() string
	GetHasNextPage() bool
	GetHasPrevPage() bool
}

// MapProtoPageInfo converts any proto PageInfo into an API resource PageInfo, building relative pagination URLs from the current request URL stored in ctx.
func MapProtoPageInfo(ctx context.Context, pi ProtoPageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	requestURL, _ := appctx.GetRequestURL(ctx)
	return apiresource.PageInfo{
		NextPageURL:     buildPageURL(requestURL, pi.GetNextCursor()),
		PreviousPageURL: buildPageURL(requestURL, pi.GetPrevCursor()),
		HasNextPage:     pi.GetHasNextPage(),
		HasPrevPage:     pi.GetHasPrevPage(),
	}
}

// MapProtoPageInfoForPath is like MapProtoPageInfo but uses an explicit canonical path instead of the current request URL. Use this for sub-object lists embedded in a parent response where the correct pagination URL must point to the sub-resource's own list endpoint. The caller is responsible for expanding any path parameters (e.g. "/v1/catalog/properties/"+propertyID+"/attributes"). Only the cursor query parameter is appended; the parent request's query string is intentionally excluded.
func MapProtoPageInfoForPath(canonicalPath string, pi ProtoPageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextPageURL:     buildCursorURL(canonicalPath, pi.GetNextCursor()),
		PreviousPageURL: buildCursorURL(canonicalPath, pi.GetPrevCursor()),
		HasNextPage:     pi.GetHasNextPage(),
		HasPrevPage:     pi.GetHasPrevPage(),
	}
}

// buildPageURL constructs a relative pagination URL by setting the cursor query parameter on the original request URL. Returns nil when cursor is empty.
func buildPageURL(requestURL *url.URL, cursor string) *string {
	if cursor == "" || requestURL == nil {
		return nil
	}
	q := requestURL.Query()
	q.Set("cursor", cursor)
	result := requestURL.Path + "?" + q.Encode()
	return &result
}

// buildCursorURL constructs a minimal relative pagination URL from a canonical path and cursor value. Returns nil when cursor is empty.
func buildCursorURL(canonicalPath string, cursor string) *string {
	if cursor == "" || canonicalPath == "" {
		return nil
	}
	result := canonicalPath + "?cursor=" + url.QueryEscape(cursor)
	return &result
}
