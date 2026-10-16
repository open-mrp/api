package versiontransforms

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

func TestPreview9_NotificationShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeNotification, `{"id":"nf_1","object":"notification","type":"order.updated","occurrences":3,
		"related":{"object":"notification_related","record":`+sampleRecordJSON+`},"resource":{"id":"so_1","object":"entity","type":"sales_order"}}`)
	assert.Equal(t, "order.updated", got["category"])
	assert.NotContains(t, got, "type")
	assert.Equal(t, float64(3), got["change_count"])
	assert.NotContains(t, got, "occurrences")
	assert.NotContains(t, got, "related")
	assert.Equal(t, "so_1", got["resource"].(map[string]any)["id"], "preview.9's resource link is kept")

	single := downgrade(t, constants.ObjectTypeNotification, `{"id":"nf_2","object":"notification","type":"chat.message","occurrences":1,"related":{"object":"notification_related","record":null},"resource":null}`)
	assert.Contains(t, single, "change_count")
	assert.Nil(t, single["change_count"], "a single event had no change count")
}

func TestPreview9_NotificationListFilters(t *testing.T) {
	t.Parallel()

	up := func(q url.Values) url.Values {
		return version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeNotification, notificationsRoute, q)
	}
	feed := up(url.Values{})
	assert.Equal(t, []string{"unseen", "seen", "read"}, feed["statuses[]"], "no status was the active feed")

	filtered := up(url.Values{"category": {"chat.mention"}, "status": {"dismissed"}})
	assert.Equal(t, []string{"chat.mention"}, filtered["types[]"])
	assert.Equal(t, []string{"dismissed"}, filtered["statuses[]"])
	assert.NotContains(t, filtered, "category")
	assert.NotContains(t, filtered, "status")
}

func TestPreview9_NotificationPreferencesRoute(t *testing.T) {
	t.Parallel()

	var seen string
	h := RoutePreview9NotificationPreferences(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.Method + " " + r.URL.Path }))
	for _, tc := range []struct {
		method, version, want string
	}{
		{http.MethodGet, apiversion.V1_0_Forge_Preview9.String(), "GET " + preview9PreferencesRoute},
		{http.MethodPut, apiversion.V1_0_Forge_Preview9.String(), "PUT " + preview9PreferencesRoute},
		{http.MethodGet, apiversion.V1_0_Forge.String(), "GET " + preferencesRoute},
		{http.MethodPatch, apiversion.V1_0_Forge_Preview9.String(), "PATCH " + preferencesRoute},
	} {
		req := httptest.NewRequest(tc.method, preferencesRoute, nil)
		req.Header.Set(apiversion.Header, tc.version)
		h.ServeHTTP(httptest.NewRecorder(), req)
		assert.Equal(t, tc.want, seen)
	}
}
