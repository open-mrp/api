package apiendpoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandRoute_SubstitutesPlaceholders(t *testing.T) {
	got := ExpandRoute("/v1/catalog/properties/{property_id}/attributes", map[string]string{
		"property_id": "prp_123",
	})
	assert.Equal(t, "/v1/catalog/properties/prp_123/attributes", got)
}

func TestExpandRoute_MultiplePlaceholders(t *testing.T) {
	got := ExpandRoute("/v1/catalog/properties/{property_id}/attributes/{id}", map[string]string{
		"property_id": "prp_1",
		"id":          "attr_9",
	})
	assert.Equal(t, "/v1/catalog/properties/prp_1/attributes/attr_9", got)
}

func TestExpandRoute_EscapesSegment(t *testing.T) {
	got := ExpandRoute("/v1/r/{id}", map[string]string{"id": "a/b"})
	assert.Equal(t, "/v1/r/a%2Fb", got)
}

func TestExpandRoute_EmptyTemplate(t *testing.T) {
	assert.Equal(t, "", ExpandRoute("", map[string]string{"x": "y"}))
}

func TestExpandRoute_NoParamsLeavesTemplate(t *testing.T) {
	const tmpl = "/v1/catalog/properties/{property_id}/attributes"
	assert.Equal(t, tmpl, ExpandRoute(tmpl, nil))
}
