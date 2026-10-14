//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A preview.9 caller still creates conversations with type and participant_account_user_ids, anchors them with
// topic_resource_*, and reads topic as an entity and unread; a one-member group stays a group.
func TestVersionCompatMessaging_ConversationsOnPreview9(t *testing.T) {
	t.Parallel()
	preview := chatUserClient(t).WithAPIVersion(preview9Version)

	status, raw, err := preview.Post(conversationsPath+"?include=topic,participants", map[string]any{
		"type":                         "group",
		"title":                        uniqueName("e2e-compat-group"),
		"participant_account_user_ids": []string{SeedAccountUser2ID},
		"topic_resource_type":          "sales_order",
		"topic_resource_id":            SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	conv := parseJSON(raw)
	convID := jsonField(conv, "id")
	assert.Equal(t, "group", jsonField(conv, "type"), "a preview.9 group with one member stays a group")
	assert.Equal(t, "0", jsonField(conv, "unread"))
	assert.NotContains(t, conv, "unread_count")
	assert.NotContains(t, conv, "related")
	topic := jsonObject(conv, "topic")
	require.NotNil(t, topic)
	assert.Equal(t, "entity", jsonField(topic, "object"))
	assert.Equal(t, "sales_order", jsonField(topic, "type"))
	assert.Equal(t, SeedSalesOrderID, jsonField(topic, "id"))

	status, raw, err = preview.Post(conversationsPath, map[string]any{
		"type":                         "direct_message",
		"participant_account_user_ids": []string{SeedAccountUser2ID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	assert.Equal(t, "direct_message", jsonField(parseJSON(raw), "type"))

	// The old list filters still work.
	list, _, err := preview.GetList(conversationsPath, url.Values{"topic_resource_type": {"sales_order"}, "topic_resource_id": {SeedSalesOrderID}, "status": {"active"}})
	require.NoError(t, err)
	var found bool
	for _, row := range list.Data {
		found = found || jsonField(parseJSON(row), "id") == convID
	}
	assert.True(t, found, "the record's discussions list the conversation")

	// A linked message reads resource; a link reads resource without an include.
	status, raw, err = preview.Post(conversationsPath+"/"+convID+"/messages?include=resource", map[string]any{
		"body": uniqueName("see order"), "client_message_id": newIdempotencyKey(),
		"link_resource_type": "sales_order", "link_resource_id": SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	msg := parseJSON(raw)
	assert.Equal(t, SeedSalesOrderID, jsonField(jsonObject(msg, "resource"), "id"))
	assert.Equal(t, "false", jsonField(msg, "agent_run_failed"))
	assert.NotContains(t, msg, "related")

	status, raw, err = preview.Post(conversationsPath+"/"+convID+"/links", map[string]any{
		"resource_type": "invoice", "resource_id": "inv_" + uniqueName("compat"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	link := parseJSON(raw)
	assert.Equal(t, "invoice", jsonField(jsonObject(link, "resource"), "type"))
	assert.NotContains(t, link, "record")
}

// A preview.9 caller still sets the email sender with PUT and reads email_domain_id, domain and domain_status, and
// reads an email domain's dkim_tokens.
func TestVersionCompatMessaging_EmailSenderOnPreview9(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	preview := acct.owner.WithAPIVersion(preview9Version)

	status, raw, err := preview.Post(covMessagingEmailDomainsPath, map[string]any{"domain": covMessagingEmailDomainsUniqueDomain("e2e-compat")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	dom := parseJSON(raw)
	domainID := jsonField(dom, "id")
	assert.NotEmpty(t, jsonStringSlice(dom, "dkim_tokens"))
	assert.NotContains(t, dom, "dns_records")
	status, raw, err = preview.Post(covMessagingEmailDomainsPath+"/"+domainID+"/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	status, raw, err = preview.Put(emailSenderPath, map[string]any{"email_domain_id": domainID, "local_part": "orders", "from_name": "Acme"})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	sender := parseJSON(raw)
	assert.Equal(t, domainID, jsonField(sender, "email_domain_id"))
	assert.Equal(t, jsonField(dom, "domain"), jsonField(sender, "domain"))
	assert.Equal(t, "verified", jsonField(sender, "domain_status"))
	assert.Equal(t, "Acme", jsonField(sender, "from_name"))
	assert.NotContains(t, sender, "email_domain")

	// A PUT replaces the sender: a field it leaves out is cleared.
	status, raw, err = preview.Put(emailSenderPath, map[string]any{"email_domain_id": domainID, "local_part": "billing"})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertNilField(t, parseJSON(raw), "from_name")

	status, raw, err = preview.Delete(emailSenderPath)
	require.NoError(t, err)
	assert.Contains(t, []int{200, 204}, status, string(raw))
}
