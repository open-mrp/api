//go:build e2e

package api_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dashboardOperationIDs are the operations the dashboard calls in the domains it moved from Express to Go.
var dashboardOperationIDs = []string{
	"activate-account-user", "add-child-account", "add-item-attribute", "admin-update-shipment-tracking",
	"admin-update-shipping-case-tracking", "analyze-customer-pricing", "analyze-delivery-performance",
	"analyze-demand-forecast", "analyze-inventory-receipts", "analyze-materials", "analyze-oee", "analyze-oee-trend",
	"analyze-open-batches", "analyze-open-orders-breakdown", "analyze-open-orders-summary", "analyze-production-costs",
	"analyze-quarterly-orders", "analyze-realized-margins", "analyze-sales-breakdown", "analyze-sales-invoices",
	"analyze-sales-summary", "analyze-schedule-attainment", "analyze-weeks-of-sales", "bulk-delete-customers",
	"bulk-reconcile-items", "change-item-category", "create-account-group-product-line-access", "create-account-user",
	"create-address", "create-customer", "create-customer-product-line-access", "create-supplier",
	"create-supplier-material", "create-territory", "delete-account-group-product-line-access", "delete-address",
	"delete-customer", "delete-customer-product-line-access", "delete-shipment", "delete-supplier",
	"delete-supplier-material", "delete-territory", "disable-account-user", "email-receivables-for-customer",
	"email-record", "get-account-favicon-url", "get-frequently-ordered-products", "get-item-costs", "get-item-trends",
	"get-shipping-case-label-url", "list-account-group-product-line-access", "list-account-transactions",
	"list-account-users", "list-address-suggestions", "list-addresses", "list-child-accounts", "list-customer-invoices",
	"list-customer-notification-recipients", "list-customer-product-line-access", "list-customers", "list-inventories",
	"list-inventory-change-logs", "list-invoices", "list-items", "list-new-customers", "list-open-order-lines",
	"list-open-orders", "list-picks", "list-receivables", "list-receivables-by-customer", "list-receiving-orders",
	"list-sales-lines", "list-sales-targets", "list-shipments", "list-supplier-materials", "list-suppliers",
	"list-territories", "merge-customers", "pack-pick", "pick-all-lines", "pick-pick-line", "rate-shop",
	"receive-receiving-order", "receive-receiving-order-line", "register-customer", "remove-account-user",
	"remove-child-account", "remove-item-attribute", "retrieve-account", "retrieve-account-by-slug",
	"retrieve-account-group-product-line-access", "retrieve-account-user", "retrieve-address-details",
	"retrieve-customer", "retrieve-customer-lead-time", "retrieve-customer-product-line-access",
	"retrieve-inventory-change-log", "retrieve-invoice", "retrieve-item", "retrieve-item-inventory",
	"retrieve-item-lot-default", "retrieve-pick", "retrieve-portal-profile", "retrieve-receiving-order",
	"retrieve-shipment", "retrieve-supplier", "retrieve-territory", "ship-shipment", "stock-receiving-order",
	"update-account", "update-account-group-product-line-access", "update-account-user", "update-address",
	"update-customer", "update-customer-notification-recipients", "update-customer-product-line-access",
	"update-invoice", "update-item-inventory", "update-pick-line", "update-receiving-order-line", "update-shipment",
	"update-shipping-case", "update-supplier", "update-supplier-material", "update-territory", "update-user",
	"upload-account-favicon", "upload-account-logo", "upload-user-photo", "upsert-sales-target", "validate-address",
	"void-pick", "void-pick-line", "void-receiving-order", "void-receiving-order-line", "void-shipment",
}

// dashboardPublicOperations answer without credentials.
var dashboardPublicOperations = map[string]string{
	"retrieve-account-by-slug": "the portal's sign-in page shows the seller's branding before anyone signs in",
	"validate-address":         "address entry on the registration form runs before sign-in",
	"list-address-suggestions": "address entry on the registration form runs before sign-in",
	"retrieve-address-details": "address entry on the registration form runs before sign-in",
}

// dashboardValidBodies pass validation, for operations that decide on credentials only once the body is valid.
var dashboardValidBodies = map[string]map[string]any{
	"register-customer": {"account_slug": "e2e-sweep-no-such-seller"},
	"validate-address":  {"address_line_1": "1 Main St", "city": "Los Angeles", "state": "CA", "postal_code": "90001", "country": "US"},
}

type dashboardOperation struct {
	method, path, operationID string
	// permissions is the documented set; empty when the operation documents none.
	permissions []string
	// allOf is set when the caller must hold every permission rather than any one of them.
	allOf bool
}

var dashboardPermissionPattern = regexp.MustCompile("`([a-z_]+:[a-z]+)`")

// loadDashboardOperations reads the in-scope operations and their documented permissions from the spec.
func loadDashboardOperations(t *testing.T) []dashboardOperation {
	t.Helper()
	data, err := os.ReadFile(findSpecPath())
	require.NoError(t, err)
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Description string `json:"description"`
		} `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(data, &spec))

	byID := map[string]dashboardOperation{}
	for path, methods := range spec.Paths {
		for method, op := range methods {
			perms, allOf := documentedPermissions(op.Description)
			op := dashboardOperation{method: strings.ToUpper(method), path: path, operationID: op.OperationID,
				permissions: perms, allOf: allOf}
			byID[op.operationID] = op
		}
	}
	ops := make([]dashboardOperation, 0, len(dashboardOperationIDs))
	for _, id := range dashboardOperationIDs {
		op, ok := byID[id]
		require.True(t, ok, "operation %s is not in the spec", id)
		ops = append(ops, op)
	}
	return ops
}

// documentedPermissions reads the permission sentence: a comma list is any-of, one joined with "and" is all-of.
func documentedPermissions(description string) (perms []string, allOf bool) {
	for _, line := range strings.Split(description, "\n") {
		if strings.HasPrefix(line, "This endpoint requires the permission") {
			for _, m := range dashboardPermissionPattern.FindAllStringSubmatch(line, -1) {
				perms = append(perms, m[1])
			}
			return perms, strings.Contains(line, "` and `")
		}
	}
	return nil, false
}

// dashboardUnknownPath names nothing that exists, bar the caller's own account or user where a route refuses others.
func dashboardUnknownPath(op dashboardOperation) string {
	path := op.path
	for _, param := range pathParamNames(path) {
		var value string
		switch {
		case strings.HasPrefix(path, "/v1/identity/accounts/") && param == "id",
			strings.HasPrefix(path, "/v1/sales/accounts/") && param == "account_id":
			value = SeedAccountID
		case strings.HasPrefix(path, "/v1/identity/users/") && param == "id":
			value = SeedUserID
		default:
			value = unknownIDLike(op, param)
		}
		path = strings.ReplaceAll(path, "{"+param+"}", value)
	}
	return path
}

func pathParamNames(path string) []string {
	var names []string
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			names = append(names, strings.Trim(part, "{}"))
		}
	}
	return names
}

// unknownIDLike keeps the seed id's prefix; the rest is unique per operation, so record locks never contend.
func unknownIDLike(op dashboardOperation, param string) string {
	seed := ""
	if param == "id" {
		bestLen := 0
		for prefix, val := range pathSpecificIDSeeds {
			if strings.HasPrefix(op.path, prefix) && len(prefix) > bestLen {
				seed, bestLen = val, len(prefix)
			}
		}
	} else if val, ok := pathSpecificParamSeed(op.path, param); ok {
		seed = val
	} else {
		seed = pathParamSeeds[param]
	}
	prefix, rest, found := strings.Cut(seed, "_")
	if !found {
		prefix, rest = "zz", "000000000000"
	}
	sum := sha256.Sum256([]byte(op.operationID + "/" + param))
	suffix := hex.EncodeToString(sum[:])
	for len(suffix) < len(rest) {
		suffix += "0"
	}
	return prefix + "_" + suffix[:len(rest)]
}

// dashboardRequestBody is refused before anything is written, except a report's, which gets a window to run.
func dashboardRequestBody(op dashboardOperation) map[string]any {
	if body, ok := dashboardValidBodies[op.operationID]; ok {
		return body
	}
	if strings.HasPrefix(op.path, "/v1/core/analytics/") && op.method == http.MethodPut {
		end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
		return map[string]any{"starts_at": rfc3339(end.AddDate(0, -1, 0)), "ends_at": rfc3339(end)}
	}
	return map[string]any{}
}

func dashboardCall(t *testing.T, client *Client, op dashboardOperation) (int, []byte) {
	t.Helper()
	path := dashboardUnknownPath(op)
	var (
		status int
		body   []byte
		err    error
	)
	switch op.method {
	case http.MethodGet:
		status, body, err = client.GetListRaw(path, nil)
	case http.MethodPost:
		status, body, err = client.Post(path, dashboardRequestBody(op), newIdempotencyKey())
	case http.MethodPatch:
		status, body, err = client.Patch(path, dashboardRequestBody(op), newIdempotencyKey())
	case http.MethodPut:
		status, body, err = client.Put(path, dashboardRequestBody(op))
	case http.MethodDelete:
		status, body, err = client.Delete(path)
	default:
		t.Fatalf("unsupported method %s", op.method)
	}
	require.NoError(t, err, "%s %s", op.method, path)
	return status, body
}

// dashboardAnonymousCall sends the request with no Authorization header at all, as a signed-out browser would.
func dashboardAnonymousCall(t *testing.T, op dashboardOperation) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if op.method != http.MethodGet && op.method != http.MethodDelete {
		raw, err := json.Marshal(dashboardRequestBody(op))
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(op.method, apiClient.baseURL+dashboardUnknownPath(op), reader)
	require.NoError(t, err)
	req.Header.Set("OpenMRP-Account", SeedAccountID)
	req.Header.Set("OpenMRP-Version", apiClient.apiVersion)
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", newIdempotencyKey())
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

func TestDashboardEndpoints_RequireCredentials(t *testing.T) {
	t.Parallel()

	for _, op := range loadDashboardOperations(t) {
		t.Run(op.operationID, func(t *testing.T) {
			t.Parallel()
			status, body := dashboardAnonymousCall(t, op)
			if reason, public := dashboardPublicOperations[op.operationID]; public {
				assert.NotEqual(t, http.StatusUnauthorized, status, "public (%s): %s", reason, string(body))
				assert.Less(t, status, 500, string(body))
				return
			}
			requireStatus(t, http.StatusUnauthorized, status, body)
			requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
		})
	}
}

// dashboardAdminOnlyOperations also require the caller's role to be the account's administrator.
var dashboardAdminOnlyOperations = map[string]string{
	"admin-update-shipment-tracking":      "re-routing a shipment that already went out overrides the ordinary update",
	"admin-update-shipping-case-tracking": "re-routing a case that already went out overrides the ordinary update",
}

// A role holding a sibling permission outside the documented set is refused; one holding a documented one is treated like the admin.
func TestDashboardEndpoints_EnforceDocumentedPermissions(t *testing.T) {
	t.Parallel()
	ops := loadDashboardOperations(t)

	clients := map[string]*Client{}
	roleClient := func(perm string) *Client {
		if c, ok := clients[perm]; ok {
			return c
		}
		c := customRoleClient(t, perm)
		clients[perm] = c
		return c
	}
	type plan struct {
		op      dashboardOperation
		refused *Client
		holders map[string]*Client
		// all holds every documented permission, for an all-of set.
		all *Client
	}
	var plans []plan
	for _, op := range ops {
		if len(op.permissions) == 0 {
			continue
		}
		p := plan{op: op, refused: roleClient(dashboardRefusedPermission(op.permissions)), holders: map[string]*Client{}}
		for _, perm := range op.permissions {
			p.holders[perm] = roleClient(perm)
		}
		if op.allOf {
			p.all = customRoleClient(t, op.permissions...)
		}
		plans = append(plans, p)
	}
	require.NotEmpty(t, plans)

	for _, p := range plans {
		t.Run(p.op.operationID, func(t *testing.T) {
			t.Parallel()
			status, body := dashboardCall(t, p.refused, p.op)
			requireStatus(t, http.StatusForbidden, status, body)
			requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

			adminStatus, adminBody := dashboardCall(t, apiClient, p.op)
			require.Less(t, adminStatus, 500, string(adminBody))

			if p.op.allOf {
				for _, perm := range sortedKeys(p.holders) {
					status, body := dashboardCall(t, p.holders[perm], p.op)
					requireStatus(t, http.StatusForbidden, status, body)
					requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
				}
				status, body := dashboardCall(t, p.all, p.op)
				assert.Equal(t, adminStatus, status, "a role holding all of %v is treated unlike the admin (%d): %s", p.op.permissions, adminStatus, string(body))
				return
			}

			// Of an any-of set, one must suffice for the caller's own account; the rest may apply only to counterparties.
			got := map[string]int{}
			for _, perm := range sortedKeys(p.holders) {
				status, body := dashboardCall(t, p.holders[perm], p.op)
				require.Less(t, status, 500, "%s with %s: %s", p.op.operationID, perm, string(body))
				got[perm] = status
				if reason, adminOnly := dashboardAdminOnlyOperations[p.op.operationID]; adminOnly {
					requireStatus(t, http.StatusForbidden, status, body)
					assert.Contains(t, string(body), "administrator", reason)
					continue
				}
				if len(p.op.permissions) == 1 {
					assert.Equal(t, adminStatus, status, "a role holding %s is treated unlike the admin (%d): %s", perm, adminStatus, string(body))
				}
			}
			if _, adminOnly := dashboardAdminOnlyOperations[p.op.operationID]; !adminOnly && len(p.op.permissions) > 1 {
				assert.Contains(t, statusValues(got), adminStatus, "no single documented permission is treated like the admin (%d): %v", adminStatus, got)
			}
		})
	}
}

// dashboardRefusedPermission prefers a sibling of the set's domain, so a gate that checks only the domain shows up.
func dashboardRefusedPermission(allowed []string) string {
	in := map[string]bool{}
	for _, p := range allowed {
		in[p] = true
	}
	domain := strings.SplitN(allowed[0], ":", 2)[0]
	for _, action := range []string{"read", "update", "create", "delete"} {
		if candidate := domain + ":" + action; !in[candidate] {
			return candidate
		}
	}
	for _, candidate := range []string{"units:read", "carriers:read"} {
		if !in[candidate] {
			return candidate
		}
	}
	panic(fmt.Sprintf("no permission outside %v", allowed))
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func statusValues(m map[string]int) []int {
	out := make([]int, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
