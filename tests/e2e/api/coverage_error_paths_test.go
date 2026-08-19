//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Error paths for endpoints the suite only ever exercised on the happy path.
//
// The shared question is whether an endpoint can tell "nothing here" from "not yours" — a read
// that answers for any ID it is handed will quietly serve another tenant's, and a delete that
// reports success without deleting anything tells a retrying caller the wrong thing.

// unknownIDCase is one endpoint probed with an ID that does not exist.
type unknownIDCase struct {
	method string
	path   string
	body   map[string]any
	// want is the status the endpoint must answer with. Most are 404; a few are 403 because the
	// account itself is the thing being reached for, and access is refused before existence is
	// considered.
	want int
}

func runUnknownIDCases(t *testing.T, cases map[string]unknownIDCase) {
	t.Helper()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				status int
				body   []byte
				err    error
			)
			switch tc.method {
			case "GET":
				status, body, err = apiClient.GetListRaw(tc.path, nil)
			case "DELETE":
				status, body, err = apiClient.Delete(tc.path)
			case "POST":
				status, body, err = apiClient.Post(tc.path, tc.body, newIdempotencyKey())
			case "PUT":
				status, body, err = apiClient.PutRaw(tc.path, nil, tc.body)
			case "PATCH":
				status, body, err = apiClient.Patch(tc.path, tc.body, newIdempotencyKey())
			default:
				t.Fatalf("unhandled method %q", tc.method)
			}

			require.NoError(t, err)
			require.Less(t, status, 500, "%s %s must not 5xx: %s", tc.method, tc.path, string(body))
			assert.Equal(t, tc.want, status, "%s %s: %s", tc.method, tc.path, string(body))
		})
	}
}

// ──────────────────────────────────────────────
// Reads
// ──────────────────────────────────────────────

func TestUnknownID_Reads(t *testing.T) {
	t.Parallel()

	runUnknownIDCases(t, map[string]unknownIDCase{
		"registration flow":            {method: "GET", path: "/v1/sales/registration-flows/rgfl_doesnotexist", want: 404},
		"customer product line access": {method: "GET", path: "/v1/sales/product-line-access/customers/ac_doesnotexist00000", want: 404},
		"group product line access":    {method: "GET", path: "/v1/sales/product-line-access/account-groups/acgr_doesnotexist", want: 404},
		"territory":                    {method: "GET", path: "/v1/sales/accounts/" + SeedAccountID + "/territories/terr_doesnotexist", want: 404},
		"supplier material":            {method: "GET", path: "/v1/operations/suppliers/" + SeedSupplierAccountID + "/materials/spmt_doesnotexist", want: 404},
		"shipment line":                {method: "GET", path: "/v1/operations/shipments/shp_doesnotexist00/lines/shpl_doesnotexist", want: 404},
		"production":                   {method: "GET", path: "/v1/operations/production-steps/prs_doesnotexist000/productions/pd_doesnotexist000", want: 404},
		"conversation links":           {method: "GET", path: "/v1/messaging/conversations/cnv_doesnotexist0/links", want: 404},
	})
}

// A flow graph and a list of past orders are both empty for an ID that does not exist, which is
// indistinguishable from a real item nobody has made or a real customer who has ordered nothing.
func TestUnknownID_DerivedReadsDoNotAnswerForAnyID(t *testing.T) {
	t.Parallel()

	runUnknownIDCases(t, map[string]unknownIDCase{
		"production flow by item":     {method: "GET", path: "/v1/operations/production-flows/by-item/it_doesnotexist00000", want: 404},
		"frequently ordered products": {method: "GET", path: "/v1/sales/customers/ac_doesnotexist00000/frequently-ordered-products", want: 404},
	})
}

// An item that exists but is bought rather than made has a genuinely empty flow, and that must
// stay a 200 — otherwise the fix above would turn a real answer into an error.
func TestProductionFlow_APurchasedMaterialHasAnEmptyFlow(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList("/v1/operations/production-flows/by-item/"+SeedPurchasedItemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.Empty(t, list.Data, "nothing produces a purchased material")
}

// ──────────────────────────────────────────────
// Deletes
// ──────────────────────────────────────────────

func TestUnknownID_Deletes(t *testing.T) {
	t.Parallel()

	runUnknownIDCases(t, map[string]unknownIDCase{
		"transaction":                  {method: "DELETE", path: "/v1/finance/transactions/tx_doesnotexist00000", want: 404},
		"email domain":                 {method: "DELETE", path: "/v1/messaging/email-domains/emdm_doesnotexist", want: 404},
		"department":                   {method: "DELETE", path: "/v1/operations/departments/dept_doesnotexist0", want: 404},
		"machine":                      {method: "DELETE", path: "/v1/operations/machines/mch_doesnotexist000", want: 404},
		"production step":              {method: "DELETE", path: "/v1/operations/production-steps/prs_doesnotexist000", want: 404},
		"registration flow":            {method: "DELETE", path: "/v1/sales/registration-flows/rgfl_doesnotexist", want: 404},
		"customer product line access": {method: "DELETE", path: "/v1/sales/product-line-access/customers/ac_doesnotexist00000", want: 404},
		"territory":                    {method: "DELETE", path: "/v1/sales/accounts/" + SeedAccountID + "/territories/terr_doesnotexist", want: 404},
		"calendar closure":             {method: "DELETE", path: "/v1/operations/operating-calendars/ocal_doesnotexist/closures/oclo_doesnotexist", want: 404},
		"production schedule line":     {method: "DELETE", path: "/v1/operations/production-schedules/prsc_doesnotexist/lines/prsl_doesnotexist", want: 404},
		// The child account is reached as an account, so access is refused before its existence
		// is considered — a 404 here would confirm which account IDs are real.
		"child account": {method: "DELETE", path: "/v1/identity/child-accounts/ac_doesnotexist00000", want: 403},
	})
}

// ──────────────────────────────────────────────
// Actions
// ──────────────────────────────────────────────

func TestUnknownID_SalesOrderActions(t *testing.T) {
	t.Parallel()

	runUnknownIDCases(t, map[string]unknownIDCase{
		"quote freight": {method: "POST", path: salesOrdersPath + "/or_doesnotexist0000/actions/quote-freight", want: 404},
		"unissue":       {method: "PUT", path: salesOrdersPath + "/or_doesnotexist0000/actions/unissue", want: 404},
	})
}

// Chat is a user-facing surface, so an API key is turned away before any conversation is looked
// up — which is also why these cannot leak whether a conversation ID exists.
func TestConversationActions_RefuseANonUserCaller(t *testing.T) {
	t.Parallel()

	runUnknownIDCases(t, map[string]unknownIDCase{
		"archive":    {method: "POST", path: "/v1/messaging/conversations/cnv_doesnotexist0/actions/archive", want: 403},
		"unarchive":  {method: "POST", path: "/v1/messaging/conversations/cnv_doesnotexist0/actions/unarchive", want: 403},
		"unmute":     {method: "POST", path: "/v1/messaging/conversations/cnv_doesnotexist0/actions/unmute", want: 403},
		"messages":   {method: "GET", path: "/v1/messaging/conversations/cnv_doesnotexist0/messages", want: 403},
		"upload url": {method: "POST", path: "/v1/messaging/conversations/cnv_doesnotexist0/attachments/actions/upload-url", body: map[string]any{"filename": "a.png", "content_type": "image/png"}, want: 403},
	})
}

// Every registration action is authorized against the person registering, so an API key cannot
// drive somebody else's signup to completion.
func TestRegistrationSession_ActionsRequireTheRegisteringUser(t *testing.T) {
	t.Parallel()

	runUnknownIDCases(t, map[string]unknownIDCase{
		"update":   {method: "PATCH", path: registrationSessionsPath + "/rgfw_doesnotexist", body: map[string]any{"step": "user_details"}, want: 401},
		"complete": {method: "POST", path: registrationSessionsPath + "/rgfw_doesnotexist/accounts", want: 401},
	})
}

// ──────────────────────────────────────────────
// Bulk delete
// ──────────────────────────────────────────────

// A bulk delete names many resources at once, so it has to be clear about the ones it could not
// find rather than silently deleting the subset it recognized.
func TestPurchaseOrders_BulkDeleteWithAnUnknownID(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(purchaseOrdersPath+"/actions/bulk-delete", map[string]any{
		"purchase_order_ids": []string{"po_doesnotexist000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 404}, status, "an unknown purchase order must not delete quietly: %s", string(body))
}

// Deleting an empty list deletes nothing and says so. Pinned because the alternative — treating
// it as a validation error — would break a caller that filters a selection down to nothing and
// sends the result, which is a reasonable thing for a UI to do.
func TestPurchaseOrders_BulkDeleteOfNothingIsANoOp(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(purchaseOrdersPath+"/actions/bulk-delete", map[string]any{
		"purchase_order_ids": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 200, status, "an empty bulk delete must succeed without deleting: %s", string(body))
}

// ──────────────────────────────────────────────
// Branding reads
// ──────────────────────────────────────────────

// The favicon read mirrors the logo: it backs the customer portal, so it answers for any account
// and reports no URL rather than an error when there is nothing to show.
func TestAccountFavicon_UnknownAccountReportsNoURL(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/identity/accounts/ac_doesnotexist00000/favicon", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["url"], "an unknown account has no favicon: %s", string(body))
}

// Uploading is a different matter: branding belongs to one account and only that account may
// change it.
func TestAccountFavicon_UploadToAnotherAccountIsRefused(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.PutBytes("/v1/identity/accounts/ac_doesnotexist00000/favicon", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 403, status, "only an account may set its own favicon: %s", string(body))
}

// ──────────────────────────────────────────────
// List parameters
// ──────────────────────────────────────────────

// Paging parameters live on an embedded struct shared by every list endpoint, so a caller who
// gets one wrong has to be told which query parameter they sent — not the Go field it maps to.
func TestListParameters_AreRejectedByTheirQueryParameterName(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		params url.Values
		param  string
	}{
		"limit below the floor": {url.Values{"limit": {"0"}}, "limit"},
		"limit above the cap":   {url.Values{"limit": {"99999"}}, "limit"},
		"unusable cursor":       {url.Values{"cursor": {"not-a-cursor"}}, "cursor"},
		"unknown parameter":     {url.Values{"sort": {"nonsense"}}, "sort"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(purchaseOrdersPath, tc.params)
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "%s must be rejected: %s", name, string(body))
			// Either the message quotes the parameter or the error's `param` carries it — both
			// let a client point at the offending input; naming neither does not.
			assert.Contains(t, string(body), tc.param,
				"the error must identify the parameter the caller sent: %s", string(body))
			assert.NotContains(t, string(body), "'Limit'", "a Go field name must never reach the caller: %s", string(body))
		})
	}
}
