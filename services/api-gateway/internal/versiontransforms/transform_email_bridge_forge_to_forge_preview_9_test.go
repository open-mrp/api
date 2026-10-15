package versiontransforms

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

func TestPreview9_EmailDomainRecordsBecomeTokensAndMailFrom(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeEmailDomain, `{"id":"emdom_1","object":"email_domain","domain":"acme.com","dns_records":[
		{"type":"CNAME","name":"abc._domainkey.acme.com","value":"abc.dkim.amazonses.com"},
		{"type":"CNAME","name":"def._domainkey.acme.com","value":"def.dkim.amazonses.com"},
		{"type":"MX","name":"mail.acme.com","value":"10 feedback-smtp.us-east-1.amazonses.com"},
		{"type":"TXT","name":"mail.acme.com","value":"v=spf1 include:amazonses.com ~all"}]}`)
	assert.Equal(t, []any{"abc", "def"}, got["dkim_tokens"])
	assert.Equal(t, "mail.acme.com", got["mail_from_domain"])
	assert.Equal(t, "10 feedback-smtp.us-east-1.amazonses.com", got["mail_from_mx_record"])
	assert.Equal(t, "v=spf1 include:amazonses.com ~all", got["mail_from_spf_record"])
	assert.NotContains(t, got, "dns_records")

	bare := downgrade(t, constants.ObjectTypeEmailDomain, `{"id":"emdom_2","object":"email_domain","dns_records":[]}`)
	assert.Equal(t, []any{}, bare["dkim_tokens"])
	assert.Nil(t, bare["mail_from_domain"])
}

func TestPreview9_EmailDomainInListAndInbox(t *testing.T) {
	t.Parallel()

	list := downgrade(t, constants.ObjectTypeEmailInbox, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"id":"eminb_1","object":"email_inbox","group":{"id":"cvgp_1","object":"messaging_group"},
		 "email_domain":{"id":"emdom_1","object":"email_domain","dns_records":[{"type":"CNAME","name":"t._domainkey.acme.com","value":"t.dkim.amazonses.com"}]}},
		{"id":"eminb_2","object":"email_inbox","group":null,"email_domain":null}]}`)
	rows := list["data"].([]any)
	first := rows[0].(map[string]any)
	assert.Equal(t, "cvgp_1", first["group_id"])
	assert.NotContains(t, first, "group")
	assert.Equal(t, []any{"t"}, first["email_domain"].(map[string]any)["dkim_tokens"])
	second := rows[1].(map[string]any)
	assert.Nil(t, second["group_id"])
}

func TestPreview9_EmailSenderFlattensItsDomain(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeEmailSender, `{"id":"emsx_1","object":"email_sender","local_part":"orders",
		"email_domain":{"id":"emdom_1","object":"email_domain","domain":"acme.com","status":"verified","dns_records":[]}}`)
	assert.Equal(t, "emdom_1", got["email_domain_id"])
	assert.Equal(t, "acme.com", got["domain"])
	assert.Equal(t, "verified", got["domain_status"])
	assert.NotContains(t, got, "email_domain")

	forced := version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeEmailSender)
	assert.Contains(t, forced, "email_domain")
	forced = version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeEmailInbox)
	assert.Contains(t, forced, "group")
}

func TestPreview9_EmailSenderPutClearsOmittedFields(t *testing.T) {
	t.Parallel()

	body := version.TransformEndpointRequest(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeEmailSender,
		http.MethodPatch, "/v1/messaging/email-sender", map[string]any{"email_domain_id": "emdom_1", "local_part": "orders", "from_name": "Acme"})
	assert.Equal(t, "Acme", body["from_name"])
	assert.Contains(t, body, "reply_to")
	assert.Nil(t, body["reply_to"])
}

func TestPreview9_EmailSenderPutRoutesToPatch(t *testing.T) {
	t.Parallel()

	var method string
	h := RoutePreview9Messaging(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { method = r.Method }))
	req := httptest.NewRequest(http.MethodPut, "/v1/messaging/email-sender", nil)
	req.Header.Set(apiversion.Header, apiversion.V1_0_Forge_Preview9.String())
	h.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, http.MethodPatch, method)

	forge := httptest.NewRequest(http.MethodPut, "/v1/messaging/email-sender", nil)
	forge.Header.Set(apiversion.Header, apiversion.V1_0_Forge.String())
	h.ServeHTTP(httptest.NewRecorder(), forge)
	assert.Equal(t, http.MethodPut, method, "forge callers are not rerouted")
}
