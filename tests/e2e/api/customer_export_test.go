//go:build e2e

package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const customerExportPath = customersPath + "/actions/export"

// storedCustomerExport reads what the accept phase recorded on the job: the account it exports and
// the filters the worker replays. The file itself is out of reach of the e2e harness.
func storedCustomerExport(t *testing.T, jobID string) (accountID string, slug string, filters map[string]any) {
	t.Helper()
	var raw []byte
	require.NoError(t, authDB(t).QueryRow("SELECT account_id, job_items FROM job WHERE job_id = ?", jobID).Scan(&accountID, &raw))
	var payload struct {
		Slug    string         `json:"slug"`
		Filters map[string]any `json:"filters"`
	}
	require.NoError(t, json.Unmarshal(raw, &payload), "job_items is not an export payload: %s", string(raw))
	return accountID, payload.Slug, payload.Filters
}

// --- Render ---

func TestCustomerExport_RendersThroughAJob(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-customer-export")
	createAndCleanup(t, customersPath, validCustomerBody(name))

	resp, err := apiClient.DoFull(http.MethodPost, customerExportPath, map[string]any{"q": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, resp.StatusCode, resp.Body)
	accepted := parseJSON(resp.Body)
	jobID := jsonField(accepted, "id")
	assert.Equal(t, "job", jsonField(accepted, "object"))
	assert.Equal(t, "export", jsonField(accepted, "type"))
	assert.Equal(t, "customer", jsonField(accepted, "resource_type"), "the job says what the export lists")
	assert.Equal(t, jobsPath+"/"+jobID, resp.Header.Get("Location"), "202 points at the job to poll")

	job := awaitExportJob(t, apiClient, jobID)
	assert.Equal(t, "customer", jsonField(job, "resource_type"))
	export := jsonObject(job, "export")
	require.NotNil(t, export, "a completed export links its file: %v", job)
	assert.Equal(t, "job_export", jsonField(export, "object"))
	url := jsonField(export, "url")
	assert.Contains(t, url, "/customers/", "the file is stored under the customer export")
	assert.Contains(t, url, "customers_export_", "the file is named for the customer export")
	assert.Contains(t, url, ".xlsx")
}

func TestCustomerExport_WithNoFiltersExportsTheWholeAccount(t *testing.T) {
	t.Parallel()
	job := completedExportJob(t, customerExportPath, nil)
	require.NotNil(t, jsonObject(job, "export"), "a completed export links its file: %v", job)
}

// --- Filters ---

// The dashboard sends the list's own filters; each must reach the worker and run there without failing the job.
func TestCustomerExport_EachListFilterRendersAFile(t *testing.T) {
	t.Parallel()
	for name, filters := range map[string]map[string]any{
		"q":                       {"q": "Global"},
		"customer_group_ids":      {"customer_group_ids": []string{SeedCustomerGroupID}},
		"pricing_group_ids":       {"pricing_group_ids": []string{SeedCustomerGroupID}},
		"sales_rep_ids":           {"sales_rep_ids": []string{SeedAccountUserID}},
		"status_codes":            {"status_codes": []string{"normal", "hold_shipment"}},
		"shipping_term_ids":       {"shipping_term_ids": []string{SeedShippingTermID}},
		"payment_term_ids":        {"payment_term_ids": []string{SeedPaymentTermID}},
		"commission_status_codes": {"commission_status_codes": []string{"commission_exempt"}},
		"freight_status_codes":    {"freight_status_codes": []string{"billed_freight"}},
		"carrier_ids":             {"carrier_ids": []string{SeedCarrierID}},
		"service_level_ids":       {"service_level_ids": []string{SeedServiceLevelID}},
		"parent_account_status":   {"parent_account_status": "parent"},
		"non_parent":              {"parent_account_status": "non_parent"},
		"address":                 {"city": "Los Angeles", "state": "CA", "postal_code": "90001"},
		"created window":          {"starts_at": rfc3339(time.Now().AddDate(-1, 0, 0)), "ends_at": rfc3339(time.Now())},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			job := completedExportJob(t, customerExportPath, filters)
			require.NotNil(t, jsonObject(job, "export"), "a completed export links its file: %v", job)
		})
	}
}

// The job records the list's filters exactly as the list takes them, so the worker selects the
// customers the list would; paging and includes are the export's own and are never stored.
func TestCustomerExport_RecordsTheListFiltersOnTheJob(t *testing.T) {
	t.Parallel()
	startsAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	endsAt := time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)
	job := completedExportJob(t, customerExportPath, map[string]any{
		"q":                       "harbor",
		"customer_group_ids":      []string{SeedCustomerGroupID},
		"pricing_group_ids":       []string{"acgp_e2eexportprice"},
		"sales_rep_ids":           []string{SeedAccountUserID},
		"status_codes":            []string{"normal"},
		"shipping_term_ids":       []string{SeedShippingTermID},
		"payment_term_ids":        []string{SeedPaymentTermID},
		"commission_status_codes": []string{"commission_exempt"},
		"freight_status_codes":    []string{"free_freight"},
		"carrier_ids":             []string{SeedCarrierID},
		"service_level_ids":       []string{SeedServiceLevelID},
		"parent_account_status":   "non_parent",
		"city":                    "Los Angeles",
		"state":                   "CA",
		"postal_code":             "90001",
		"starts_at":               rfc3339(startsAt),
		"ends_at":                 rfc3339(endsAt),
	})

	accountID, slug, filters := storedCustomerExport(t, jsonField(job, "id"))
	assert.Equal(t, SeedAccountID, accountID, "the export belongs to the caller's account")
	assert.Equal(t, "customers", slug)
	assert.Equal(t, map[string]any{
		"AccountID":             "",
		"Cursor":                nil,
		"Limit":                 float64(0),
		"Includes":              nil,
		"Query":                 "harbor",
		"CustomerGroupIDs":      []any{SeedCustomerGroupID},
		"PricingGroupIDs":       []any{"acgp_e2eexportprice"},
		"SalesRepIDs":           []any{SeedAccountUserID},
		"StatusCodes":           []any{"normal"},
		"ShippingTermIDs":       []any{SeedShippingTermID},
		"PaymentTermIDs":        []any{SeedPaymentTermID},
		"CommissionPolicyCodes": []any{"commission_exempt"},
		"FreightPolicyCodes":    []any{"free_freight"},
		"CarrierIDs":            []any{SeedCarrierID},
		"ServiceLevelIDs":       []any{SeedServiceLevelID},
		"IsParentAccount":       false,
		"City":                  "Los Angeles",
		"State":                 "CA",
		"PostalCode":            "90001",
		"StartDate":             startsAt.Format(time.RFC3339),
		"EndDate":               endsAt.Format(time.RFC3339),
	}, filters)
}

func TestCustomerExport_RejectsInvalidFilters(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		body  map[string]any
		param string
	}{
		"unknown status":         {map[string]any{"status_codes": []string{"bogus"}}, "status_codes"},
		"unknown commission":     {map[string]any{"commission_status_codes": []string{"bogus"}}, "commission_status_codes"},
		"unknown freight":        {map[string]any{"freight_status_codes": []string{"bogus"}}, "freight_status_codes"},
		"unknown parent status":  {map[string]any{"parent_account_status": "bogus"}, "parent_account_status"},
		"null search":            {map[string]any{"q": nil}, "q"},
		"search too long":        {map[string]any{"q": strings.Repeat("x", 501)}, "q"},
		"start is not a date":    {map[string]any{"starts_at": "yesterday"}, "starts_at"},
		"unknown field":          {map[string]any{bogusE2EJSONField: "x"}, ""},
		"ids are not a list":     {map[string]any{"customer_group_ids": SeedCustomerGroupID}, "customer_group_ids"},
		"parent status not null": {map[string]any{"parent_account_status": nil}, "parent_account_status"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(customerExportPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, http.StatusBadRequest, status, body)
			errObj := requireErrorResponse(t, body, "", "invalid_request_error")
			if tc.param != "" {
				assert.Contains(t, errObj["param"], tc.param, "the error names the offending filter: %s", string(body))
			}
		})
	}
}

// --- Idempotency ---

func TestCustomerExport_SameKeyReplaysTheSameJob(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()
	body := map[string]any{"status_codes": []string{"normal"}}

	status, first, err := apiClient.Post(customerExportPath, body, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, first)
	status, second, err := apiClient.Post(customerExportPath, body, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, second)

	assert.Equal(t, jsonField(parseJSON(first), "id"), jsonField(parseJSON(second), "id"), "a retried accept must not start a second export")
}

// --- Tenancy ---

// Each tenant's export is recorded against its own account, and neither can read the other's job or its file.
func TestCustomerExport_TenantsExportOnlyTheirOwnAccount(t *testing.T) {
	t.Parallel()
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	jobB := completedExportJobAs(t, tenantB, customerExportPath, map[string]any{"customer_group_ids": []string{SeedCustomerGroupID}})
	accountID, _, _ := storedCustomerExport(t, jsonField(jobB, "id"))
	assert.Equal(t, SeedTenantBAccountID, accountID, "tenant B's export reads tenant B's customers, whatever IDs it filters by")

	jobA := completedExportJob(t, customerExportPath, nil)
	accountID, _, _ = storedCustomerExport(t, jsonField(jobA, "id"))
	assert.Equal(t, SeedAccountID, accountID)

	resp, err := tenantB.GetFull(jobsPath+"/"+jsonField(jobA, "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "another tenant's export job must not resolve: %s", string(resp.Body))

	resp, err = apiClient.GetFull(jobsPath+"/"+jsonField(jobB, "id"), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "another tenant's export job must not resolve: %s", string(resp.Body))
}

// --- Auth ---

func TestCustomerExport_RequiresAuthentication(t *testing.T) {
	t.Parallel()
	anon := apiClient.WithBearerToken("", SeedAccountID)
	status, body, err := anon.Post(customerExportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, body)
}

// A role without customers:read (the scanning-station role) may not export customers.
func TestCustomerExport_RefusesACallerWithoutCustomersRead(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-customer-export-scanner"),
		"role_id": SeedScannerRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	created := parseJSON(body)
	keyID := jsonField(jsonObject(created, "api_key_info"), "id")
	require.NotEmpty(t, keyID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(apiKeysPath + "/" + keyID) })
	scanner := apiClient.WithBearerToken(jsonField(created, "api_key_secret"), SeedAccountID)

	status, body, err = scanner.Post(customerExportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, body)
	errObj := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	assert.Contains(t, errObj["message"], "customers:read")
}

// The customer portal reads a seller's catalog, never the seller's customer book.
func TestCustomerExport_RefusesTheCustomerPortal(t *testing.T) {
	t.Parallel()
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.Post(customerExportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}
