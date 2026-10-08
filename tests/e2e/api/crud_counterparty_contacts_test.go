//go:build e2e

package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A seller manages the users of a customer or supplier account it owns (its contacts) through the account users
// endpoints, acting in that account with OpenMRP-Account. These are the calls the dashboard's customer and supplier
// contact pages make.

const (
	contactsSuppliersPath = "/v1/operations/suppliers"
	contactsRolesPath     = "/v1/identity/roles"
	globalCustomerRoleID  = "rl_7vafmsquekgt"
)

// counterpartyKind is a kind of account a seller keeps contacts for.
type counterpartyKind struct {
	name string
	// permDomain is the permission domain that governs this kind's contacts.
	permDomain string
	// otherDomain is the other kind's domain, which must not reach this kind's contacts.
	otherDomain string
	create      func(t *testing.T) string
	// welcomed reports whether a new contact is emailed a password; suppliers have no portal to sign in to.
	welcomed bool
}

var counterpartyKinds = []counterpartyKind{
	{name: "customer", permDomain: "customers", otherDomain: "suppliers", create: createContactsCustomer, welcomed: true},
	{name: "supplier", permDomain: "suppliers", otherDomain: "customers", create: createContactsSupplier, welcomed: false},
}

func createContactsCustomer(t *testing.T) string {
	t.Helper()
	customerID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-contacts-cust"))), "id")
	require.NotEmpty(t, customerID)
	return customerID
}

func createContactsSupplier(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.Post(contactsSuppliersPath, map[string]any{
		"name":   uniqueName("e2e-contacts-supp"),
		"number": uniqueName("SUP"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	supplierID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, supplierID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(contactsSuppliersPath + "/" + supplierID) })
	return supplierID
}

func contactEmail(prefix string) (name, email string) {
	name = uniqueName(prefix)
	return name, name + "@e2e-test.openmrp.ai"
}

// createContact adds a contact to the account c acts in and returns it as created, with its user included.
func createContact(t *testing.T, c *Client, name, email string, prefs map[string]bool) map[string]any {
	t.Helper()
	body := map[string]any{"name": name, "email": email}
	if prefs != nil {
		body["preferences"] = preferenceToggles(prefs)
	}
	status, resp, err := c.Post(accountUsersPath+"?include=user", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	created := parseJSON(resp)
	require.NotEmpty(t, jsonField(created, "id"))
	return created
}

func preferenceToggles(prefs map[string]bool) []map[string]any {
	codes := make([]string, 0, len(prefs))
	for code := range prefs {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	toggles := make([]map[string]any, 0, len(prefs))
	for _, code := range codes {
		toggles = append(toggles, map[string]any{"notification_type": code, "enabled": prefs[code]})
	}
	return toggles
}

// notificationTypes reads a contact's notification_types, failing when it is null rather than a list.
func notificationTypes(t *testing.T, contact map[string]any) []string {
	t.Helper()
	raw, present := contact["notification_types"]
	require.True(t, present, "notification_types must be present: %v", contact)
	require.NotNil(t, raw, "a counterparty's contact carries a list, not null: %v", contact)
	items, ok := raw.([]any)
	require.True(t, ok, "notification_types is a list: %v", raw)
	types := make([]string, 0, len(items))
	for _, item := range items {
		types = append(types, item.(string))
	}
	sort.Strings(types)
	return types
}

func getContact(t *testing.T, c *Client, contactID string) map[string]any {
	t.Helper()
	status, body, err := c.GetListRaw(accountUsersPath+"/"+contactID, url.Values{"include": {"user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func patchContact(t *testing.T, c *Client, contactID string, body map[string]any) map[string]any {
	t.Helper()
	status, resp, err := c.Patch(accountUsersPath+"/"+contactID+"?include=user", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	return parseJSON(resp)
}

// listContacts reads every page of the account c acts in, following next_page_url.
func listContacts(t *testing.T, c *Client, params url.Values) []map[string]any {
	t.Helper()
	merged := url.Values{"include": {"user"}}
	for k, v := range params {
		merged[k] = v
	}
	list, status, err := c.GetList(accountUsersPath, merged)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	var contacts []map[string]any
	for page := 0; ; page++ {
		require.Less(t, page, 50, "listing must end")
		for _, item := range list.Data {
			contacts = append(contacts, parseJSON(item))
		}
		if !list.PageInfo.HasNextPage {
			return contacts
		}
		require.NotNil(t, list.PageInfo.NextPageURL)
		list, _, err = c.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
}

func contactIDs(contacts []map[string]any) []string {
	ids := make([]string, 0, len(contacts))
	for _, c := range contacts {
		ids = append(ids, jsonField(c, "id"))
	}
	return ids
}

func findContact(contacts []map[string]any, contactID string) map[string]any {
	for _, c := range contacts {
		if jsonField(c, "id") == contactID {
			return c
		}
	}
	return nil
}

func removeContact(t *testing.T, c *Client, contactID string) (int, []byte) {
	t.Helper()
	status, body, err := c.Put(accountUsersPath+"/"+contactID+"/actions/remove", nil)
	require.NoError(t, err)
	return status, body
}

// storedPreferenceCount counts a contact's stored preference rows, so duplicates show even when the API dedupes them.
func storedPreferenceCount(t *testing.T, contactID string) int {
	t.Helper()
	var n int
	require.NoError(t, authDB(t).QueryRow(
		`SELECT COUNT(*) FROM account_relation_notification_preference WHERE recipient_account_user_id = ?`, contactID,
	).Scan(&n))
	return n
}

func queuedEmailCount(t *testing.T, marker string) int {
	t.Helper()
	var n int
	require.NoError(t, authDB(t).QueryRow(`
		SELECT COUNT(*) FROM message_outbox
		WHERE id > ? AND routing_key = 'notification.cmd.send_email'
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?`,
		outboxFloor(t), "%"+marker+"%").Scan(&n))
	return n
}

// sellerPortalLoginLink is where the seed seller's portal users sign in: its verified custom domain when it has one,
// else its slug on the portal host.
func sellerPortalLoginLink(t *testing.T) string {
	t.Helper()
	var domain string
	err := authDB(t).QueryRow(`SELECT domain FROM portal_domain WHERE account_id = ? AND status = 'verified'`, SeedAccountID).Scan(&domain)
	if err == nil {
		return "https://" + domain + "/auth/login"
	}
	require.True(t, errors.Is(err, sql.ErrNoRows), "reading the portal domain: %v", err)
	var slug string
	require.NoError(t, authDB(t).QueryRow(`SELECT slug FROM account_portal WHERE owner_account_id = ?`, SeedAccountID).Scan(&slug))
	return envOr("E2E_PORTAL_URL", "http://localhost:4300") + "/" + slug + "/auth/login"
}

// contactsRoleClient is an API key on the seed seller whose role holds exactly perms, acting in accountID.
func contactsRoleClient(t *testing.T, accountID string, perms ...string) *Client {
	t.Helper()
	status, body, err := apiClient.Post(contactsRolesPath, map[string]any{
		"name":        uniqueName("e2e-contacts-role"),
		"permissions": perms,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	roleID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, roleID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(contactsRolesPath + "/" + roleID) })

	status, body, err = apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-contacts-key"),
		"role_id": roleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	keyID := jsonField(jsonObject(created, "api_key_info"), "id")
	require.NotEmpty(t, keyID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(apiKeysPath + "/" + keyID) })
	secret := jsonField(created, "api_key_secret")
	require.NotEmpty(t, secret)
	return apiClient.WithBearerToken(secret, accountID)
}

// sellerSession is the seeded user signed in to the seed seller, the identity the dashboard sends.
type sellerSession struct {
	token string
}

func newSellerSession(t *testing.T) sellerSession {
	t.Helper()
	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	token := cookieValue(resp.Header["Set-Cookie"], accessTokenCookie)
	require.NotEmpty(t, token)
	return sellerSession{token: token}
}

// do sends a request acting in targetAccountID: OpenMRP-Actor-Account names the account the user belongs to and
// OpenMRP-Account the one they work in.
func (s sellerSession) do(t *testing.T, targetAccountID, method, path string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, apiClient.baseURL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("OpenMRP-Actor-Account", SeedAccountID)
	req.Header.Set("OpenMRP-Account", targetAccountID)
	req.Header.Set("OpenMRP-Version", apiClient.apiVersion)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Idempotency-Key", newIdempotencyKey())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := apiClient.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, respBody
}

// --- Create ---

// A new contact comes back with the notification types its preferences enabled and its profile, and is stored under
// the account it was added to. A customer's contact gets the customer role and a welcome email whose link signs in to
// the seller's portal; a supplier's gets neither.
func TestCounterpartyContacts_Create(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			accountID := kind.create(t)
			c := apiClient.WithAccountID(accountID)
			name, email := contactEmail("e2e-contact-create")

			created := createContact(t, c, name, email, map[string]bool{
				"order_acknowledgement":     true,
				"invoice":                   false,
				"purchase_order_submission": true,
			})
			contactID := jsonField(created, "id")
			assertIDFormat(t, contactID, id.AccountUserIDPrefix)
			assertObjectField(t, created, "account_user")
			assert.Equal(t, "active", jsonField(created, "status"))
			assert.Equal(t, "false", jsonField(created, "is_commission_eligible"))
			assert.Equal(t, []string{"order_acknowledgement", "purchase_order_submission"}, notificationTypes(t, created))
			assertNilField(t, created, "role")
			assertNilField(t, created, "department")
			assertNilField(t, created, "last_used_at")
			assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
			assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
			user := jsonObject(created, "user")
			require.NotNil(t, user, "include=user returns the profile")
			assert.Equal(t, name, jsonField(user, "name"))
			assert.Equal(t, email, jsonField(user, "email"))
			assertObjectField(t, user, "user")

			var storedAccountID string
			var roleID sql.NullString
			require.NoError(t, authDB(t).QueryRow(`SELECT account_id, role_id FROM account_user WHERE id = ?`, contactID).Scan(&storedAccountID, &roleID))
			assert.Equal(t, accountID, storedAccountID, "the contact belongs to the %s's account", kind.name)
			if kind.welcomed {
				assert.Equal(t, globalCustomerRoleID, roleID.String, "a customer's contact gets the customer role")

				welcome := queuedSendEmail(t, email)
				assert.Equal(t, []any{email}, welcome["to"])
				assert.Equal(t, "new_user_welcome", welcome["template_id"])
				assert.Contains(t, welcome["subject"], "Welcome to the ")
				assert.Equal(t, sellerPortalLoginLink(t), emailParam(t, welcome, "LoginLink"))
				assert.NotEmpty(t, emailParam(t, welcome, "Password"))
			} else {
				assert.False(t, roleID.Valid, "a supplier's contact has no role")
				assert.Zero(t, queuedEmailCount(t, email), "a supplier's contact has no portal and is not emailed")
			}

			got := getContact(t, c, contactID)
			assert.Equal(t, []string{"order_acknowledgement", "purchase_order_submission"}, notificationTypes(t, got))
			assert.Equal(t, email, jsonField(jsonObject(got, "user"), "email"))
		})
	}
}

// Adding a contact who is already one of the account's users conflicts rather than linking them twice.
func TestCounterpartyContacts_CreateDuplicateConflicts(t *testing.T) {
	t.Parallel()
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			c := apiClient.WithAccountID(kind.create(t))
			name, email := contactEmail("e2e-contact-dup")
			createContact(t, c, name, email, nil)

			status, body, err := c.Post(accountUsersPath, map[string]any{"name": name, "email": email}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 409, status, body)
		})
	}
}

// --- List ---

// The list holds the account's active contacts with the notification types each receives, finds them by name, leaves
// removed ones out unless asked, and pages.
func TestCounterpartyContacts_List(t *testing.T) {
	t.Parallel()
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			c := apiClient.WithAccountID(kind.create(t))

			aName, aEmail := contactEmail("e2e-contact-list-a")
			a := jsonField(createContact(t, c, aName, aEmail, map[string]bool{"invoice": true}), "id")
			bName, bEmail := contactEmail("e2e-contact-list-b")
			b := jsonField(createContact(t, c, bName, bEmail, nil), "id")
			gName, gEmail := contactEmail("e2e-contact-list-gone")
			gone := jsonField(createContact(t, c, gName, gEmail, map[string]bool{"order_acknowledgement": true}), "id")
			status, body := removeContact(t, c, gone)
			requireStatus(t, 200, status, body)

			contacts := listContacts(t, c, nil)
			assert.ElementsMatch(t, []string{a, b}, contactIDs(contacts), "only the active contacts are listed")
			assert.Equal(t, []string{"invoice"}, notificationTypes(t, findContact(contacts, a)))
			assert.Empty(t, notificationTypes(t, findContact(contacts, b)), "a contact with nothing enabled has an empty list")
			assert.Equal(t, aEmail, jsonField(jsonObject(findContact(contacts, a), "user"), "email"))

			withRemoved := listContacts(t, c, url.Values{"removed_scope": {"included"}})
			assert.ElementsMatch(t, []string{a, b, gone}, contactIDs(withRemoved))
			assert.Equal(t, "removed", jsonField(findContact(withRemoved, gone), "status"))

			found := listContacts(t, c, url.Values{"q": {aName}})
			assert.Equal(t, []string{a}, contactIDs(found), "search finds the contact by name")
			assert.Empty(t, listContacts(t, c, url.Values{"q": {"zzzz-no-such-contact"}}))

			first, status, err := c.GetList(accountUsersPath, url.Values{"limit": {"1"}})
			require.NoError(t, err)
			require.Equal(t, 200, status)
			require.Len(t, first.Data, 1)
			require.True(t, first.PageInfo.HasNextPage)
			second, _, err := c.GetListFromPageURL(first.PageInfo.NextPageURL)
			require.NoError(t, err)
			require.Len(t, second.Data, 1)
			assert.ElementsMatch(t, []string{a, b}, []string{DataItemField(first.Data[0], "id"), DataItemField(second.Data[0], "id")})
		})
	}
}

// --- Update ---

// Renaming a contact updates their profile, and each notification type turns on and off on its own without touching
// the others.
func TestCounterpartyContacts_UpdateNameAndNotifications(t *testing.T) {
	t.Parallel()
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			c := apiClient.WithAccountID(kind.create(t))
			name, email := contactEmail("e2e-contact-upd")
			contactID := jsonField(createContact(t, c, name, email, nil), "id")

			renamed := uniqueName("e2e-contact-renamed")
			updated := patchContact(t, c, contactID, map[string]any{"name": renamed})
			assert.Equal(t, renamed, jsonField(jsonObject(updated, "user"), "name"))
			assert.Equal(t, email, jsonField(jsonObject(updated, "user"), "email"))
			assert.Empty(t, notificationTypes(t, updated), "renaming enables nothing")

			all := []string{"invoice", "order_acknowledgement", "purchase_order_submission"}
			for _, code := range all {
				on := patchContact(t, c, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{code: true})})
				assert.Equal(t, []string{code}, notificationTypes(t, on), "turning on %s", code)
				assert.Equal(t, []string{code}, notificationTypes(t, getContact(t, c, contactID)), "%s is stored", code)

				off := patchContact(t, c, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{code: false})})
				assert.Empty(t, notificationTypes(t, off), "turning off %s", code)
			}

			everything := patchContact(t, c, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{
				"invoice": true, "order_acknowledgement": true, "purchase_order_submission": true,
			})})
			assert.Equal(t, all, notificationTypes(t, everything))
			again := patchContact(t, c, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{"invoice": true})})
			assert.Equal(t, all, notificationTypes(t, again), "enabling an enabled type changes nothing")
			assert.Equal(t, 3, storedPreferenceCount(t, contactID), "an enabled type is stored once")

			oneOff := patchContact(t, c, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{"order_acknowledgement": false})})
			assert.Equal(t, []string{"invoice", "purchase_order_submission"}, notificationTypes(t, oneOff), "the other types are untouched")
			assert.Equal(t, renamed, jsonField(jsonObject(oneOff, "user"), "name"), "toggling leaves the name")
		})
	}
}

// Turning a notification on is a change to the contact, so it is audited with the types before and after even when
// nothing else changed. The event belongs to the customer's account, which the seller cannot list, so it is read from
// the store.
func TestCounterpartyContacts_NotificationToggleIsAudited(t *testing.T) {
	t.Parallel()
	accountID := createContactsCustomer(t)
	c := apiClient.WithAccountID(accountID)
	name, email := contactEmail("e2e-contact-audit")
	contactID := jsonField(createContact(t, c, name, email, nil), "id")
	patchContact(t, c, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{"invoice": true})})

	type fieldChange struct {
		Field    string
		OldValue any
		NewValue any
	}
	var audited fieldChange
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		rows, err := authDB(t).Query(`SELECT changes FROM audit_event WHERE resource_id = ? AND action = 'update' AND account_id = ?`, contactID, accountID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var raw []byte
			if err := rows.Scan(&raw); err != nil {
				return err
			}
			var changes []fieldChange
			if err := json.Unmarshal(raw, &changes); err != nil {
				return err
			}
			for _, change := range changes {
				if change.Field == "notification_types" {
					audited = change
					return nil
				}
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return fmt.Errorf("no notification_types change audited yet for %s", contactID)
	})
	assert.Equal(t, []any{}, audited.OldValue)
	assert.Equal(t, []any{"invoice"}, audited.NewValue)
}

// --- Remove ---

// Removing a contact takes them off the list; adding them again restores the same membership with the preferences
// asked for, stored once each.
func TestCounterpartyContacts_RemoveAndReadd(t *testing.T) {
	t.Parallel()
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			c := apiClient.WithAccountID(kind.create(t))
			name, email := contactEmail("e2e-contact-readd")
			contactID := jsonField(createContact(t, c, name, email, map[string]bool{"order_acknowledgement": true, "invoice": true}), "id")

			status, body := removeContact(t, c, contactID)
			requireStatus(t, 200, status, body)
			assert.Nil(t, findContact(listContacts(t, c, nil), contactID), "a removed contact is not listed")

			readded := createContact(t, c, name, email, map[string]bool{
				"order_acknowledgement": true, "invoice": false, "purchase_order_submission": true,
			})
			assert.Equal(t, contactID, jsonField(readded, "id"), "the removed membership is restored, not duplicated")
			assert.Equal(t, "active", jsonField(readded, "status"))
			assert.Equal(t, []string{"order_acknowledgement", "purchase_order_submission"}, notificationTypes(t, readded))
			assert.Equal(t, 2, storedPreferenceCount(t, contactID), "restoring does not store a preference twice")

			listed := findContact(listContacts(t, c, nil), contactID)
			require.NotNil(t, listed, "a restored contact is listed again")
			assert.Equal(t, []string{"order_acknowledgement", "purchase_order_submission"}, notificationTypes(t, listed))
		})
	}
}

// The dashboard acts as a signed-in user, whose membership is in the seller's account, not the customer's. Adding,
// editing, removing and restoring a contact must all work for that user.
func TestCounterpartyContacts_SignedInUserManagesContacts(t *testing.T) {
	t.Parallel()
	session := newSellerSession(t)
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			accountID := kind.create(t)
			name, email := contactEmail("e2e-contact-session")

			status, body := session.do(t, accountID, http.MethodPost, accountUsersPath+"?include=user", map[string]any{
				"name": name, "email": email,
				"preferences": preferenceToggles(map[string]bool{"invoice": true}),
			})
			requireStatus(t, 201, status, body)
			contactID := jsonField(parseJSON(body), "id")
			assert.Equal(t, []string{"invoice"}, notificationTypes(t, parseJSON(body)))

			status, body = session.do(t, accountID, http.MethodGet, accountUsersPath+"?include=user", nil)
			requireStatus(t, 200, status, body)
			assert.Contains(t, string(body), contactID)

			status, body = session.do(t, accountID, http.MethodPatch, accountUsersPath+"/"+contactID, map[string]any{
				"preferences": preferenceToggles(map[string]bool{"invoice": false, "order_acknowledgement": true}),
			})
			requireStatus(t, 200, status, body)
			assert.Equal(t, []string{"order_acknowledgement"}, notificationTypes(t, parseJSON(body)))

			status, body = session.do(t, accountID, http.MethodPut, accountUsersPath+"/"+contactID+"/actions/remove", nil)
			requireStatus(t, 200, status, body)
			assert.Equal(t, "removed", jsonField(getContact(t, apiClient.WithAccountID(accountID), contactID), "status"))

			status, body = session.do(t, accountID, http.MethodPut, accountUsersPath+"/"+contactID+"/actions/activate", nil)
			requireStatus(t, 200, status, body)
			assert.Equal(t, "active", jsonField(getContact(t, apiClient.WithAccountID(accountID), contactID), "status"))
		})
	}
}

// --- Permissions ---

// A kind's contacts follow that kind's permissions: reading (with profiles) takes read alone, adding takes create,
// editing takes update, and removing takes delete. The other kind's permissions grant nothing.
func TestCounterpartyContacts_Permissions(t *testing.T) {
	t.Parallel()
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			accountID := kind.create(t)
			admin := apiClient.WithAccountID(accountID)
			name, email := contactEmail("e2e-contact-perm")
			contactID := jsonField(createContact(t, admin, name, email, map[string]bool{"invoice": true}), "id")
			perm := func(action string) string { return kind.permDomain + ":" + action }

			// Read alone lists and retrieves, profiles included, without the seller's team permission.
			reader := contactsRoleClient(t, accountID, perm("read"))
			contacts := listContacts(t, reader, nil)
			require.NotNil(t, findContact(contacts, contactID))
			assert.Equal(t, email, jsonField(jsonObject(findContact(contacts, contactID), "user"), "email"))
			assert.Equal(t, []string{"invoice"}, notificationTypes(t, getContact(t, reader, contactID)))
			assertContactRefused(t, reader, contactID, "create", "update", "delete")

			creator := contactsRoleClient(t, accountID, perm("read"), perm("create"))
			cName, cEmail := contactEmail("e2e-contact-perm-new")
			created := createContact(t, creator, cName, cEmail, map[string]bool{"order_acknowledgement": true})
			assert.Equal(t, cEmail, jsonField(jsonObject(created, "user"), "email"))
			assertContactRefused(t, creator, contactID, "update", "delete")

			updater := contactsRoleClient(t, accountID, perm("read"), perm("update"))
			patchContact(t, updater, contactID, map[string]any{"preferences": preferenceToggles(map[string]bool{"invoice": false})})
			assertContactRefused(t, updater, contactID, "create", "delete")

			deleter := contactsRoleClient(t, accountID, perm("read"), perm("delete"))
			assertContactRefused(t, deleter, contactID, "create", "update")
			status, body := removeContact(t, deleter, contactID)
			requireStatus(t, 200, status, body)

			other := contactsRoleClient(t, accountID, kind.otherDomain+":read", kind.otherDomain+":create", kind.otherDomain+":update", kind.otherDomain+":delete")
			status, body, err := other.GetListRaw(accountUsersPath, nil)
			require.NoError(t, err)
			requireStatus(t, 403, status, body)
			assertContactRefused(t, other, jsonField(created, "id"), "create", "update", "delete")
		})
	}
}

// assertContactRefused checks that c may not take each action on the account's contacts, and that a refused removal
// leaves the contact in place.
func assertContactRefused(t *testing.T, c *Client, contactID string, actions ...string) {
	t.Helper()
	for _, action := range actions {
		var status int
		var body []byte
		switch action {
		case "create":
			_, email := contactEmail("e2e-contact-refused")
			var err error
			status, body, err = c.Post(accountUsersPath, map[string]any{"email": email}, newIdempotencyKey())
			require.NoError(t, err)
		case "update":
			var err error
			status, body, err = c.Patch(accountUsersPath+"/"+contactID, map[string]any{"name": uniqueName("e2e-refused")}, newIdempotencyKey())
			require.NoError(t, err)
		case "delete":
			status, body = removeContact(t, c, contactID)
		}
		requireStatus(t, 403, status, body)
		requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	}
}

// --- Tenant isolation ---

// Another seller can neither act in the account nor reach its contacts by ID.
func TestCounterpartyContacts_TenantIsolation(t *testing.T) {
	t.Parallel()
	for _, kind := range counterpartyKinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			accountID := kind.create(t)
			c := apiClient.WithAccountID(accountID)
			name, email := contactEmail("e2e-contact-iso")
			contactID := jsonField(createContact(t, c, name, email, map[string]bool{"invoice": true}), "id")

			intruder := getTenantBClient().WithAccountID(accountID)
			status, body, err := intruder.GetListRaw(accountUsersPath, nil)
			require.NoError(t, err)
			requireStatus(t, 403, status, body)
			status, body, err = intruder.Post(accountUsersPath, map[string]any{"email": uniqueName("e2e-intruder") + "@e2e-test.openmrp.ai"}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 403, status, body)

			own := getTenantBClient()
			status, body, err = own.GetListRaw(accountUsersPath+"/"+contactID, nil)
			require.NoError(t, err)
			requireStatus(t, 404, status, body)
			status, body, err = own.Patch(accountUsersPath+"/"+contactID, map[string]any{
				"name": "hijacked", "preferences": preferenceToggles(map[string]bool{"invoice": false}),
			}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 404, status, body)
			status, body = removeContact(t, own, contactID)
			requireStatus(t, 404, status, body)

			got := getContact(t, c, contactID)
			assert.Equal(t, "active", jsonField(got, "status"))
			assert.Equal(t, name, jsonField(jsonObject(got, "user"), "name"))
			assert.Equal(t, []string{"invoice"}, notificationTypes(t, got))
		})
	}
}

// --- Own team ---

// The seller's own users have no relation to carry notification types, so the field stays null, and preferences sent
// with a new team member are ignored. The welcome email links to the dashboard's sign-in page.
func TestCounterpartyContacts_OwnTeamHasNoNotificationTypes(t *testing.T) {
	t.Parallel()
	markOutbox(t)

	got := parseJSON(mustGet(t, accountUsersPath+"/"+SeedAccountUserID))
	assert.Contains(t, got, "notification_types")
	assertNilField(t, got, "notification_types")

	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data)
	for _, item := range list.Data {
		assertNilField(t, parseJSON(item), "notification_types")
	}

	name, email := contactEmail("e2e-own-team")
	created := createContact(t, apiClient, name, email, map[string]bool{"invoice": true})
	defer removeAccountUser(jsonField(created, "id"))
	assertNilField(t, created, "notification_types")
	assert.Zero(t, storedPreferenceCount(t, jsonField(created, "id")))

	welcome := queuedSendEmail(t, email)
	assert.Equal(t, "Welcome to OpenMRP", welcome["subject"])
	assert.Equal(t, envOr("E2E_FRONTEND_URL", "http://localhost:4200")+"/auth/login", emailParam(t, welcome, "LoginLink"))
}
