package versiontransforms

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/constants"
)

func TestPreview9_SiteIsABuilding(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeLocation, `{"id":"lc_1","object":"location","type":"site","status":"active",`+
		`"children":{"object":"list","data":[{"id":"lc_2","object":"location","type":"bin"}]}}`)
	assert.Equal(t, "building", got["type"])
	child := got["children"].(map[string]any)["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "bin", child["type"])

	list := downgrade(t, constants.ObjectTypeLocation, `{"object":"list","data":[{"id":"lc_1","object":"location","type":"site"},{"id":"lc_3","object":"location","type":"rack"}]}`)
	rows := list["data"].([]any)
	assert.Equal(t, "building", rows[0].(map[string]any)["type"])
	assert.Equal(t, "rack", rows[1].(map[string]any)["type"])

	nested := downgrade(t, constants.ObjectTypeDepartment, `{"id":"dp_1","object":"department","location":{"id":"lc_1","object":"location","type":"site"}}`)
	assert.Equal(t, "building", nested["location"].(map[string]any)["type"])

	noLocation := downgrade(t, constants.ObjectTypeDepartment, `{"id":"dp_1","object":"department","location":null}`)
	assert.Nil(t, noLocation["location"])
}
