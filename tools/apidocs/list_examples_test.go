package main

import (
	"reflect"
	"strings"
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/pagination"
)

func TestBuildListSchemaExample_WithoutRouteUsesNullPageURLs(t *testing.T) {
	t.Parallel()
	components := &Components{Schemas: make(map[string]Schema)}
	reader := NewDocReader()
	components.Schemas["Material"] = generateSchema(reflect.TypeOf(apiresource.Material{}), components, reader)

	listType := reflect.TypeOf(apiresource.List[apiresource.Material]{})
	ex := buildListSchemaExample(components, reader, listType, "", nil)

	pageInfo, ok := ex["page_info"].(map[string]any)
	if !ok {
		t.Fatalf("page_info type = %T", ex["page_info"])
	}
	if pageInfo["next_page_url"] != nil {
		t.Errorf("next_page_url = %v, want nil for nested list schemas", pageInfo["next_page_url"])
	}
	if pageInfo["has_next_page"] != false {
		t.Errorf("has_next_page = %v, want false", pageInfo["has_next_page"])
	}
}

func TestBuildListSchemaExample_PaginatedListURL(t *testing.T) {
	t.Parallel()
	components := &Components{Schemas: make(map[string]Schema)}
	reader := NewDocReader()
	components.Schemas["Material"] = generateSchema(reflect.TypeOf(apiresource.Material{}), components, reader)

	listType := reflect.TypeOf(apiresource.List[apiresource.Material]{})
	reqType := reflect.TypeOf(apiresource.PaginationRequest{})
	ex := buildListSchemaExample(components, reader, listType, "/v1/catalog/materials", reqType)

	pageInfo, ok := ex["page_info"].(map[string]any)
	if !ok {
		t.Fatalf("page_info type = %T", ex["page_info"])
	}
	nextURL, ok := pageInfo["next_page_url"].(string)
	if !ok || nextURL == "" {
		t.Fatalf("next_page_url = %v, want relative URL string", pageInfo["next_page_url"])
	}
	if !strings.HasPrefix(nextURL, "/v1/catalog/materials?") {
		t.Errorf("next_page_url = %q, want path prefix /v1/catalog/materials?", nextURL)
	}
	wantCursor := pagination.EncodeDocumentationStringCursor(
		apiresource.SampleAnalyticsPeriodStart,
		apiresource.SampleMaterialID,
	)
	if wantCursor == apiresource.SampleMaterialID {
		t.Fatal("cursor must be a signed pagination token, not a bare type ID")
	}
	if !isSignedPaginationCursor(wantCursor) {
		t.Fatalf("cursor = %q, want payload.signature form", wantCursor)
	}
	wantURL := buildDocumentationPageURL("/v1/catalog/materials", wantCursor)
	if nextURL != wantURL {
		t.Errorf("next_page_url = %q, want %q", nextURL, wantURL)
	}
	if pageInfo["has_next_page"] != true {
		t.Errorf("has_next_page = %v, want true", pageInfo["has_next_page"])
	}
}

func TestExpandDocumentationRoute_SubstitutesPathParams(t *testing.T) {
	t.Parallel()
	reqType := reflect.TypeOf(retrieveItemPathRequest{})
	got := expandDocumentationRoute("/v1/catalog/items/{id}/attributes", reqType)
	want := "/v1/catalog/items/" + apiresource.SampleItemID + "/attributes"
	if got != want {
		t.Errorf("expandDocumentationRoute() = %q, want %q", got, want)
	}
}
