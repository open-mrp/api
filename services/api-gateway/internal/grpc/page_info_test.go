package grpc_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
)

// mockProtoPageInfo is a test implementation of ProtoPageInfo.
type mockProtoPageInfo struct {
	nextCursor  string
	prevCursor  string
	hasNextPage bool
	hasPrevPage bool
}

func (m *mockProtoPageInfo) GetNextCursor() string { return m.nextCursor }
func (m *mockProtoPageInfo) GetPrevCursor() string { return m.prevCursor }
func (m *mockProtoPageInfo) GetHasNextPage() bool  { return m.hasNextPage }
func (m *mockProtoPageInfo) GetHasPrevPage() bool  { return m.hasPrevPage }

func ctxWithURL(rawURL string) context.Context {
	u, _ := url.Parse(rawURL)
	return appctx.WithRequestURL(context.Background(), u)
}

func TestMapProtoPageInfo_BuildsNextPageURL(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/priorities?limit=10&q=high")
	pi := &mockProtoPageInfo{
		nextCursor:  "eyJpZCI6IjEwMCJ9",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfo(ctx, pi)
	require.NotNil(t, result.NextPageURL)
	assert.Contains(t, *result.NextPageURL, "/v1/sales/priorities")
	assert.Contains(t, *result.NextPageURL, "cursor=eyJpZCI6IjEwMCJ9")
	assert.Contains(t, *result.NextPageURL, "limit=10")
	assert.Contains(t, *result.NextPageURL, "q=high")
	assert.True(t, result.HasNextPage)
}

func TestMapProtoPageInfo_BuildsPreviousPageURL(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/priorities?limit=10&cursor=abc")
	pi := &mockProtoPageInfo{
		prevCursor:  "eyJpZCI6IjEifQ",
		hasPrevPage: true,
	}
	result := grpcutil.MapProtoPageInfo(ctx, pi)
	require.NotNil(t, result.PreviousPageURL)
	assert.Contains(t, *result.PreviousPageURL, "cursor=eyJpZCI6IjEifQ")
	assert.True(t, result.HasPrevPage)
}

func TestMapProtoPageInfo_NilWhenNoCursor(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/priorities?limit=10")
	pi := &mockProtoPageInfo{
		hasNextPage: false,
		hasPrevPage: false,
	}
	result := grpcutil.MapProtoPageInfo(ctx, pi)
	assert.Nil(t, result.NextPageURL)
	assert.Nil(t, result.PreviousPageURL)
}

func TestMapProtoPageInfo_ReplacesExistingCursorParam(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/priorities?limit=10&cursor=oldcursor")
	pi := &mockProtoPageInfo{
		nextCursor:  "newcursor",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfo(ctx, pi)
	require.NotNil(t, result.NextPageURL)
	// Should have the new cursor, not the old one
	assert.Contains(t, *result.NextPageURL, "cursor=newcursor")
	assert.NotContains(t, *result.NextPageURL, "cursor=oldcursor")
}

func TestMapProtoPageInfo_NilProtoReturnsEmptyPageInfo(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/priorities?limit=10")
	result := grpcutil.MapProtoPageInfo(ctx, nil)
	assert.Equal(t, apiresource.PageInfo{}, result)
}

func TestMapProtoPageInfo_NilRequestURLReturnsNilURLs(t *testing.T) {
	// No URL in context
	pi := &mockProtoPageInfo{
		nextCursor:  "somecursor",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfo(context.Background(), pi)
	assert.Nil(t, result.NextPageURL)
	assert.True(t, result.HasNextPage)
}

func TestMapProtoPageInfo_URLIsRelativePath(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/priorities?limit=5")
	pi := &mockProtoPageInfo{
		nextCursor:  "cursor123",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfo(ctx, pi)
	require.NotNil(t, result.NextPageURL)
	// URL must be relative (path + query only, no scheme/host)
	assert.True(t, (*result.NextPageURL)[0] == '/', "URL should start with /")
	assert.NotContains(t, *result.NextPageURL, "http://")
}

func TestMapProtoPageInfo_PreservesAllQueryParams(t *testing.T) {
	ctx := ctxWithURL("http://api.example.com/v1/sales/addresses?limit=25&type=drop_ship&q=main&include%5B%5D=contact")
	pi := &mockProtoPageInfo{
		nextCursor:  "abc",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfo(ctx, pi)
	require.NotNil(t, result.NextPageURL)
	assert.Contains(t, *result.NextPageURL, "limit=25")
	assert.Contains(t, *result.NextPageURL, "type=drop_ship")
	assert.Contains(t, *result.NextPageURL, "q=main")
	assert.Contains(t, *result.NextPageURL, "cursor=abc")
}

// Tests for MapProtoPageInfoForPath — used for sub-object lists embedded in a parent
// response where pagination URLs must point to the sub-resource's own endpoint.

func TestMapProtoPageInfoForPath_BuildsNextPageURL(t *testing.T) {
	pi := &mockProtoPageInfo{
		nextCursor:  "eyJpZCI6IjUwIn0",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfoForPath("/v1/catalog/properties/prop_123/attributes", pi)
	require.NotNil(t, result.NextPageURL)
	assert.Equal(t, "/v1/catalog/properties/prop_123/attributes?cursor=eyJpZCI6IjUwIn0", *result.NextPageURL)
	assert.True(t, result.HasNextPage)
}

func TestMapProtoPageInfoForPath_BuildsPreviousPageURL(t *testing.T) {
	pi := &mockProtoPageInfo{
		prevCursor:  "eyJpZCI6IjEifQ",
		hasPrevPage: true,
	}
	result := grpcutil.MapProtoPageInfoForPath("/v1/catalog/properties/prop_123/attributes", pi)
	require.NotNil(t, result.PreviousPageURL)
	assert.Equal(t, "/v1/catalog/properties/prop_123/attributes?cursor=eyJpZCI6IjEifQ", *result.PreviousPageURL)
	assert.True(t, result.HasPrevPage)
}

func TestMapProtoPageInfoForPath_DoesNotInheritParentQueryParams(t *testing.T) {
	// The canonical path is explicit — parent request params (include[], limit, q, etc.) must
	// NOT bleed into sub-resource pagination URLs.
	pi := &mockProtoPageInfo{
		nextCursor:  "cursor_abc",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfoForPath("/v1/catalog/properties/prop_123/attributes", pi)
	require.NotNil(t, result.NextPageURL)
	assert.NotContains(t, *result.NextPageURL, "include")
	assert.NotContains(t, *result.NextPageURL, "limit")
	assert.NotContains(t, *result.NextPageURL, "q=")
	assert.Contains(t, *result.NextPageURL, "cursor=cursor_abc")
}

func TestMapProtoPageInfoForPath_NilProtoReturnsEmptyPageInfo(t *testing.T) {
	result := grpcutil.MapProtoPageInfoForPath("/v1/catalog/properties/prop_123/attributes", nil)
	assert.Equal(t, apiresource.PageInfo{}, result)
}

func TestMapProtoPageInfoForPath_NilWhenNoCursor(t *testing.T) {
	pi := &mockProtoPageInfo{
		hasNextPage: false,
		hasPrevPage: false,
	}
	result := grpcutil.MapProtoPageInfoForPath("/v1/catalog/properties/prop_123/attributes", pi)
	assert.Nil(t, result.NextPageURL)
	assert.Nil(t, result.PreviousPageURL)
}

func TestMapProtoPageInfoForPath_URLIsRelativePath(t *testing.T) {
	pi := &mockProtoPageInfo{
		nextCursor:  "cursor123",
		hasNextPage: true,
	}
	result := grpcutil.MapProtoPageInfoForPath("/v1/catalog/properties/prop_123/attributes", pi)
	require.NotNil(t, result.NextPageURL)
	assert.True(t, (*result.NextPageURL)[0] == '/', "URL should be a relative path starting with /")
	assert.NotContains(t, *result.NextPageURL, "http://")
}
