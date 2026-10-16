package versiontransforms

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
)

func TestPreview9_IntegrationStatusAndConnection(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeAccountIntegration, `{"id":"acig_1","object":"account_integration","provider":"stripe","status":"disabled","connection_status":"connected"}`)
	assert.Equal(t, "inactive", got["status"])
	assert.NotContains(t, got, "connection_status")

	list := downgrade(t, constants.ObjectTypeAccountIntegration, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"id":"acig_2","object":"account_integration","status":"active","connection_status":"unknown"}]}`)
	row := list["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "active", row["status"])
	assert.NotContains(t, row, "connection_status")
}

func TestPreview9_IntegrationWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeAccountIntegration, http.MethodPost, integrationsRoute, map[string]any{
		"name": "S", "provider": "shippo", "credentials": `{"api_key":"shippo_test_1"}`,
	})
	assert.Equal(t, map[string]any{"name": "S", "provider": "shippo", "credentials": map[string]any{"api_key": "shippo_test_1"}}, created)

	malformed := upgradeBody(constants.ObjectTypeAccountIntegration, http.MethodPost, integrationsRoute, map[string]any{"credentials": "not json"})
	assert.Equal(t, "not json", malformed["credentials"], "left for validation to refuse")

	updated := upgradeBody(constants.ObjectTypeAccountIntegration, http.MethodPatch, integrationRoute, map[string]any{"status": "inactive"})
	assert.Equal(t, map[string]any{"status": "disabled"}, updated)
}

func TestPreview9_PortalDomainRecordsBecomeAList(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypePortalDomain, `{"id":"podn_1","object":"portal_domain","dns_records":[
		{"type":"A","name":"acme.com","value":"76.76.21.21"},
		{"type":"TXT","name":"_vercel.acme.com","value":"vc-domain-verify=acme.com,abc"}]}`)
	records := got["dns_records"].(map[string]any)
	assert.Equal(t, "list", records["object"])
	assert.Equal(t, map[string]any{"next_page_url": nil, "previous_page_url": nil, "has_next_page": false, "has_prev_page": false}, records["page_info"])
	data := records["data"].([]any)
	assert.Equal(t, map[string]any{"object": "dns_record", "type": "A", "name": "acme.com", "value": "76.76.21.21", "reason": "routing"}, data[0])
	assert.Equal(t, "ownership", data[1].(map[string]any)["reason"])

	empty := downgrade(t, constants.ObjectTypePortalDomain, `{"id":"podn_2","object":"portal_domain","dns_records":[]}`)
	assert.Equal(t, []any{}, empty["dns_records"].(map[string]any)["data"])
}

func TestRoutePreview9Settings(t *testing.T) {
	t.Parallel()

	serve := func(method, path string, v string) string {
		var seen string
		h := RoutePreview9Settings(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.Method }))
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(apiversion.Header, v)
		h.ServeHTTP(httptest.NewRecorder(), req)
		return seen
	}

	preview := apiversion.V1_0_Forge_Preview9.String()
	assert.Equal(t, http.MethodPatch, serve(http.MethodPut, "/v1/settings/integrations/acig_1", preview))
	assert.Equal(t, http.MethodPut, serve(http.MethodPut, "/v1/settings/integrations/acig_1", apiversion.V1_0_Forge.String()), "forge callers are not rerouted")
	assert.Equal(t, http.MethodPut, serve(http.MethodPut, "/v1/settings/integrations/hubspot/sync", preview), "only a single integration's route")
	assert.Equal(t, http.MethodDelete, serve(http.MethodDelete, "/v1/settings/integrations/acig_1", preview))
}
