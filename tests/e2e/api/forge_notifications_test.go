//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forge.1 notification preferences (one object), account and role notification defaults, record subscriptions, and
// @mentions in notes.

const (
	subscriptionsPath        = "/v1/messaging/subscriptions"
	notificationDefaultsPath = "/v1/settings/notification-defaults"
)

func getPreferences(t *testing.T, c *Client) map[string]any {
	t.Helper()
	resp, err := c.GetFull(preferencesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	m := parseJSON(resp.Body)
	assert.Equal(t, "notification_preferences", jsonField(m, "object"))
	return m
}

func patchPreferences(t *testing.T, c *Client, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := c.Patch(preferencesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

func setting(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func channelsOf(s map[string]any) []any {
	v, _ := s["channels"].([]any)
	return v
}

// ── Preferences ────────────────────────────────────────────────────

func TestForgePreferences_BuiltInDefaultsOnAFreshAccount(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	m := getPreferences(t, acct.owner)
	assert.Equal(t, []any{"in_app"}, channelsOf(setting(m, "default")), "in-app only until something else is set")
	assert.Equal(t, "instant", jsonField(setting(m, "default"), "digest"))
	assert.Equal(t, map[string]any{}, m["types"], "no per-type settings yet")
	assert.Equal(t, []any{"created", "assigned", "mentioned"}, m["auto_subscribe"], "editing doesn't subscribe by default")
	assertNilField(t, m, "paused_until")
	AssertResponseValid(t, m)
}

func TestForgePreferences_PatchMergesTypesAndNullRemoves(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	owner := acct.owner

	m := patchPreferences(t, owner, map[string]any{
		"types": map[string]any{
			"order.updated": map[string]any{"channels": []string{"in_app", "email"}},
			"chat.message":  map[string]any{"channels": []string{}, "digest": "daily"},
		},
	})
	types := setting(m, "types")
	assert.Equal(t, []any{"in_app", "email"}, channelsOf(setting(types, "order.updated")))
	assert.Equal(t, "instant", jsonField(setting(types, "order.updated"), "digest"), "digest defaults to instant")
	assert.Equal(t, []any{}, channelsOf(setting(types, "chat.message")), "an empty list turns a type off")
	assert.Equal(t, "daily", jsonField(setting(types, "chat.message"), "digest"))

	// Types left out are unchanged; a type sent as null goes back to the default.
	m = patchPreferences(t, owner, map[string]any{"types": map[string]any{"chat.message": nil}})
	types = setting(m, "types")
	assert.NotContains(t, types, "chat.message")
	assert.Contains(t, types, "order.updated")

	// The default, auto_subscribe and paused_until; GET reads the same.
	m = patchPreferences(t, owner, map[string]any{
		"default":        map[string]any{"channels": []string{"in_app", "push"}, "digest": "hourly"},
		"auto_subscribe": []string{"created", "edited"},
		"paused_until":   "2099-01-01T00:00:00Z",
	})
	assert.Equal(t, []any{"in_app", "push"}, channelsOf(setting(m, "default")))
	assert.Equal(t, "hourly", jsonField(setting(m, "default"), "digest"))
	assert.Equal(t, []any{"created", "edited"}, m["auto_subscribe"])
	assert.Equal(t, "2099-01-01T00:00:00Z", jsonField(m, "paused_until"))
	assert.Equal(t, m, getPreferences(t, owner))

	// null clears them back to the inherited values.
	m = patchPreferences(t, owner, map[string]any{"default": nil, "auto_subscribe": nil, "paused_until": nil})
	assert.Equal(t, []any{"in_app"}, channelsOf(setting(m, "default")))
	assert.Equal(t, []any{"created", "assigned", "mentioned"}, m["auto_subscribe"])
	assertNilField(t, m, "paused_until")
	assert.Contains(t, setting(m, "types"), "order.updated", "untouched")
}

func TestForgePreferences_Validation(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"required type can't drop email", map[string]any{"types": map[string]any{"security.sign_in": map[string]any{"channels": []string{}}}},
			"invalid_format", "types.security.sign_in.channels"},
		{"unknown type", map[string]any{"types": map[string]any{"bogus.type": map[string]any{"channels": []string{"in_app"}}}},
			"invalid_format", "types.bogus.type"},
		{"unknown channel", map[string]any{"default": map[string]any{"channels": []string{"sms"}}}, "invalid_format", "default.channels"},
		{"channels missing", map[string]any{"default": map[string]any{"digest": "daily"}}, "missing_field", "default.channels"},
		{"unknown auto_subscribe", map[string]any{"auto_subscribe": []string{"viewed"}}, "invalid_format", "auto_subscribe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := acct.owner.Patch(preferencesPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
			errObj := requireErrorResponse(t, body, tc.code, "invalid_request_error")
			assertErrorParam(t, errObj, tc.param)
		})
	}

	// A required type that keeps email is fine.
	m := patchPreferences(t, acct.owner, map[string]any{"types": map[string]any{"security.sign_in": map[string]any{"channels": []string{"email"}}}})
	assert.Equal(t, []any{"email"}, channelsOf(setting(setting(m, "types"), "security.sign_in")))
}

func TestForgePreferences_APIKeyCantChangeThem(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(preferencesPath, map[string]any{"auto_subscribe": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	// It reads the account's starting settings.
	m := getPreferences(t, apiClient)
	assert.Equal(t, map[string]any{}, m["types"])
}

func TestForgePreferences_PatchIsIdempotent(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	key := newIdempotencyKey()
	body := map[string]any{"types": map[string]any{"order.updated": map[string]any{"channels": []string{"email"}}}}
	s1, b1, err := acct.owner.Patch(preferencesPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)
	s2, b2, err := acct.owner.Patch(preferencesPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)
	assert.JSONEq(t, string(b1), string(b2))
}

// preview.9's per-category settings and forge's object are the same rows.
func TestForgePreferences_Preview9CategoryRowsShowAsTypes(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	upsertPreference(t, acct.owner, map[string]any{
		"category": "chat.mention", "in_app_enabled": true, "email_enabled": true, "push_enabled": false, "digest": "instant",
	})
	m := getPreferences(t, acct.owner)
	assert.Equal(t, []any{"in_app", "email"}, channelsOf(setting(setting(m, "types"), "chat.mention")))

	patchPreferences(t, acct.owner, map[string]any{"types": map[string]any{"chat.mention": map[string]any{"channels": []string{"push"}}}})
	list, _, err := asPreview9(acct.owner).GetList(preferencesPath, nil)
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	var row map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &row))
	assert.Equal(t, "chat.mention", jsonField(row, "category"))
	assert.Equal(t, false, row["in_app_enabled"])
	assert.Equal(t, false, row["email_enabled"])
	assert.Equal(t, true, row["push_enabled"])
}

// ── Account and role notification defaults ─────────────────────────

func TestForgeNotificationDefaults_AccountDefaultsReachPeopleWithoutTheirOwn(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	owner := acct.owner

	resp, err := owner.GetFull(notificationDefaultsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	d := parseJSON(resp.Body)
	assert.Equal(t, "notification_defaults", jsonField(d, "object"))
	assertNilField(t, d, "default")
	assert.Equal(t, map[string]any{}, d["types"])
	assertNilField(t, d, "auto_subscribe")
	AssertResponseValid(t, d)

	status, body, err := owner.Patch(notificationDefaultsPath, map[string]any{
		"default":        map[string]any{"channels": []string{"in_app", "email"}},
		"types":          map[string]any{"chat.message": map[string]any{"channels": []string{}}},
		"auto_subscribe": []string{"assigned"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	d = parseJSON(body)
	assert.Equal(t, []any{"in_app", "email"}, channelsOf(setting(d, "default")))
	assert.Equal(t, []any{}, channelsOf(setting(setting(d, "types"), "chat.message")))
	assert.Equal(t, []any{"assigned"}, d["auto_subscribe"])

	// The owner hasn't set their own, so they get the account's default and auto_subscribe.
	m := getPreferences(t, owner)
	assert.Equal(t, []any{"in_app", "email"}, channelsOf(setting(m, "default")))
	assert.Equal(t, []any{"assigned"}, m["auto_subscribe"])

	// Removing the account's settings goes back to OpenMRP's.
	status, body, err = owner.Patch(notificationDefaultsPath, map[string]any{"default": nil, "types": map[string]any{"chat.message": nil}, "auto_subscribe": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	d = parseJSON(body)
	assertNilField(t, d, "default")
	assert.Equal(t, map[string]any{}, d["types"])
	assertNilField(t, d, "auto_subscribe")
}

func TestForgeNotificationDefaults_RequiresAccountPermission(t *testing.T) {
	t.Parallel()
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.Patch(notificationDefaultsPath, map[string]any{"auto_subscribe": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
}

func TestForgeRoles_NotificationDefaults(t *testing.T) {
	t.Parallel()
	create := func(body map[string]any) map[string]any {
		resp, err := apiClient.PostFull(rolesPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, resp.StatusCode, resp.Body)
		role := parseJSON(resp.Body)
		t.Cleanup(func() { _, _, _ = apiClient.Delete(rolesPath + "/" + jsonField(role, "id")) })
		return role
	}

	// A role from a template starts from the template's (none yet).
	fromTemplate := create(map[string]any{"name": uniqueName("e2e-role-tpl"), "template": "warehouse"})
	nd := setting(fromTemplate, "notification_defaults")
	require.NotNil(t, nd, "notification_defaults is always present")
	assertNilField(t, nd, "default")
	assert.Equal(t, map[string]any{}, nd["types"])
	assertNilField(t, nd, "auto_subscribe")

	role := create(map[string]any{
		"name": uniqueName("e2e-role-nd"),
		"notification_defaults": map[string]any{
			"types":          map[string]any{"order.updated": map[string]any{"channels": []string{"email"}}},
			"auto_subscribe": []string{"assigned", "mentioned"},
		},
	})
	nd = setting(role, "notification_defaults")
	assert.Equal(t, []any{"email"}, channelsOf(setting(setting(nd, "types"), "order.updated")))
	assert.Equal(t, []any{"assigned", "mentioned"}, nd["auto_subscribe"])
	id := jsonField(role, "id")

	// PATCH merges: a new type is added, the old one kept, and auto_subscribe cleared.
	status, body, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{
		"notification_defaults": map[string]any{
			"default":        map[string]any{"channels": []string{"in_app"}},
			"types":          map[string]any{"chat.message": map[string]any{"channels": []string{}}},
			"auto_subscribe": nil,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	nd = setting(parseJSON(body), "notification_defaults")
	assert.Equal(t, []any{"in_app"}, channelsOf(setting(nd, "default")))
	assert.Contains(t, setting(nd, "types"), "order.updated")
	assert.Contains(t, setting(nd, "types"), "chat.message")
	assertNilField(t, nd, "auto_subscribe")

	// Required types keep email.
	status, body, err = apiClient.Patch(rolesPath+"/"+id, map[string]any{
		"notification_defaults": map[string]any{"types": map[string]any{"security.device_paired": map[string]any{"channels": []string{"in_app"}}}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "notification_defaults.types.security.device_paired.channels")
}

// ── Subscriptions ──────────────────────────────────────────────────

func followRecord(t *testing.T, c *Client, recordType, recordID string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(subscriptionsPath, map[string]any{"record": map[string]any{"type": recordType, "id": recordID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

func TestForgeSubscriptions_FollowListUnsubscribeSubscribe(t *testing.T) {
	t.Parallel()
	dane := notifUserClient(t)
	user2 := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	orderID, number := createActivityOrder(t, dane, SeedAccountUserID)

	sub := followRecord(t, user2, "sales_order", orderID)
	id := jsonField(sub, "id")
	assertIDFormat(t, id, "sk")
	assert.Equal(t, "subscription", jsonField(sub, "object"))
	assert.Equal(t, "manual", jsonField(sub, "reason"))
	assert.Equal(t, "subscribed", jsonField(sub, "status"))
	rec := jsonObject(sub, "record")
	assert.Equal(t, "record", jsonField(rec, "object"))
	assert.Equal(t, "sales_order", jsonField(rec, "type"))
	assert.Equal(t, orderID, jsonField(rec, "id"))
	assert.Equal(t, number, jsonField(rec, "number"))
	assertValidTimestamp(t, jsonField(sub, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(sub, "updated_at"), "updated_at")
	AssertResponseValid(t, sub)

	// Following again is the same subscription.
	again := followRecord(t, user2, "sales_order", orderID)
	assert.Equal(t, id, jsonField(again, "id"))

	// Retrieve, and list by record.
	resp, err := user2.GetFull(subscriptionsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, id, jsonField(parseJSON(resp.Body), "id"))
	list, _, err := user2.GetList(subscriptionsPath, url.Values{"record_ids[]": {orderID}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, id, DataItemField(list.Data[0], "id"))
	list, _, err = user2.GetList(subscriptionsPath, url.Values{"record_ids[]": {orderID}, "record_types[]": {"customer"}})
	require.NoError(t, err)
	assert.Empty(t, list.Data, "record_types[] narrows")

	// Someone else can't see it.
	resp, err = dane.GetFull(subscriptionsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	// Unsubscribe, twice; then the statuses[] filter.
	for range 2 {
		status, body, err := user2.Post(subscriptionsPath+"/"+id+"/actions/unsubscribe", nil, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		m := parseJSON(body)
		assert.Equal(t, "unsubscribed", jsonField(m, "status"))
		assert.Equal(t, "manual", jsonField(m, "reason"))
	}
	list, _, err = user2.GetList(subscriptionsPath, url.Values{"record_ids[]": {orderID}, "statuses[]": {"subscribed"}})
	require.NoError(t, err)
	assert.Empty(t, list.Data)
	list, _, err = user2.GetList(subscriptionsPath, url.Values{"record_ids[]": {orderID}, "statuses[]": {"unsubscribed"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)

	// Following an unsubscribed record subscribes again, as does the subscribe action.
	assert.Equal(t, "subscribed", jsonField(followRecord(t, user2, "sales_order", orderID), "status"))
	status, body, err := user2.Post(subscriptionsPath+"/"+id+"/actions/unsubscribe", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = user2.Post(subscriptionsPath+"/"+id+"/actions/subscribe", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "subscribed", jsonField(parseJSON(body), "status"))
}

func TestForgeSubscriptions_FollowACustomer(t *testing.T) {
	t.Parallel()
	user2 := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	customerID := setupOrderCustomer(t)
	sub := followRecord(t, user2, "customer", customerID)
	assert.Equal(t, "customer", jsonField(jsonObject(sub, "record"), "type"))
	status, body, err := user2.Post(subscriptionsPath+"/"+jsonField(sub, "id")+"/actions/unsubscribe", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestForgeSubscriptions_Validation(t *testing.T) {
	t.Parallel()
	user2 := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)

	cases := []struct {
		name   string
		record map[string]any
		code   string
		param  string
	}{
		{"unfollowable type", map[string]any{"type": "item", "id": SeedProductID}, "invalid_format", "record.type"},
		{"missing order", map[string]any{"type": "sales_order", "id": "or_doesnotexist0000000000000"}, "invalid_format", "record.id"},
		{"missing id", map[string]any{"type": "sales_order"}, "missing_field", "record.id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := user2.Post(subscriptionsPath, map[string]any{"record": tc.record}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), tc.param)
		})
	}

	status, body, err := user2.Post(subscriptionsPath+"/sk_doesnotexist00/actions/unsubscribe", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestForgeSubscriptions_APIKeysHaveNone(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(subscriptionsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Empty(t, list.Data)

	s, body, err := apiClient.Post(subscriptionsPath, map[string]any{"record": map[string]any{"type": "sales_order", "id": SeedSalesOrderID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, s, body)
}

func TestForgeSubscriptions_CreateIsIdempotent(t *testing.T) {
	t.Parallel()
	user2 := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	orderID, _ := createActivityOrder(t, notifUserClient(t), SeedAccountUserID)
	key := newIdempotencyKey()
	body := map[string]any{"record": map[string]any{"type": "sales_order", "id": orderID}}
	r1, err := user2.PostFull(subscriptionsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, r1.StatusCode, r1.Body)
	r2, err := user2.PostFull(subscriptionsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, r2.StatusCode, r2.Body)
	assert.Equal(t, jsonField(parseJSON(r1.Body), "id"), jsonField(parseJSON(r2.Body), "id"))
}

// ── @mentions in notes ─────────────────────────────────────────────

func TestForgeNoteMentions_NotifyAndSubscribe(t *testing.T) {
	t.Parallel()
	dane := notifUserClient(t)
	user2 := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	orderID, _ := createActivityOrder(t, dane, SeedAccountUserID)
	body := uniqueName("e2e-mention") + " please check the freight"

	resp, err := dane.PostFull(notesPath, map[string]any{
		"record":   map[string]any{"type": "sales_order", "id": orderID},
		"body":     body,
		"mentions": []string{SeedAccountUser2ID, SeedAccountUserID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	noteID := jsonField(parseJSON(resp.Body), "id")
	t.Cleanup(func() { _, _, _ = dane.Delete(notesPath + "/" + noteID) })

	var n map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := user2.GetList(notificationsPath, url.Values{"types[]": {"note.mention"}, "limit": {"100"}, "include": {"related.record"}})
		if err != nil {
			return err
		}
		for _, raw := range list.Data {
			var m map[string]any
			if json.Unmarshal(raw, &m) == nil && jsonField(m, "body") == body {
				n = m
				return nil
			}
		}
		return fmt.Errorf("mention not delivered yet")
	})
	assert.Equal(t, "note.mention", jsonField(n, "type"))
	assert.Equal(t, "high", jsonField(n, "priority"))
	assert.Contains(t, jsonField(n, "title"), "mentioned you in a note")
	assert.Equal(t, orderID, jsonField(jsonObject(jsonObject(n, "related"), "record"), "id"))

	sub := waitForSubscription(t, user2, orderID, "subscribed")
	assert.Equal(t, "mentioned", jsonField(sub, "reason"))

	// The author isn't told about their own mention.
	list, _, err := dane.GetList(notificationsPath, url.Values{"types[]": {"note.mention"}, "limit": {"100"}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		assert.NotEqual(t, body, jsonField(m, "body"))
	}
}

func TestForgeNoteMentions_OnlyAccountMembers(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(notesPath, map[string]any{
		"body":     uniqueName("e2e-mention-outsider"),
		"mentions": []string{"acus_notinthisaccount0000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "mentions[0]")
}

// ── Chat follows the new settings ──────────────────────────────────

func TestForgePreferences_ChatMessageTypeOffSuppressesTheBell(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	m := patchPreferences(t, acct.owner, map[string]any{"types": map[string]any{"chat.message": map[string]any{"channels": []string{}}}})
	assert.Equal(t, []any{}, channelsOf(setting(setting(m, "types"), "chat.message")))
	// The bell suppression itself is covered with preview.9's per-category row in TestChatBell_InAppDisabledSuppressesBell;
	// both write the same row.
	list, _, err := asPreview9(acct.owner).GetList(preferencesPath, nil)
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, "false", DataItemField(list.Data[0], "in_app_enabled"))
}
