//go:build e2e

package api_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Numbers handed out from a per-account counter: production runs and customers. Runs created at once
// must each get their own number without the creates deadlocking, a number someone typed in or renamed
// to is never handed out again, and a typed-in customer number goes to one customer when two people
// save it at once.
//
// The tests that renumber ahead of the counter work in tenant B, where nothing else creates runs or
// customers, so the counter lands on the number they took.

const numberingRaceWidth = 8

func TestNumbering_RunsCreatedAtOnceGetTheirOwnNumbers(t *testing.T) {
	t.Parallel()
	once := dashSuppliersOnce(apiClient)

	results := race(numberingRaceWidth, func(int) (int, []byte, error) {
		return once.Post(productionRunsPath, map[string]any{"responsible_user_id": SeedUserID}, newIdempotencyKey())
	})

	numbers := map[string]bool{}
	for _, r := range results {
		require.Equal(t, http.StatusCreated, r.status, "every run created at once is created, none deadlocks: %s", r)
		run := parseJSON(r.body)
		runID := jsonField(run, "id")
		t.Cleanup(func() { apiClient.Delete(productionRunPath(runID)) })
		number := jsonField(run, "number")
		assert.False(t, numbers[number], "run number %s handed out twice", number)
		numbers[number] = true
	}
}

func TestNumbering_ARunNumberRenamedAheadIsSkipped(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()
	createRunB := func() map[string]any {
		t.Helper()
		status, raw, err := clientB.Post(productionRunsPath, map[string]any{"responsible_user_id": SeedTenantBAccountUserID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, http.StatusCreated, status, raw)
		run := parseJSON(raw)
		runID := jsonField(run, "id")
		t.Cleanup(func() { clientB.Delete(productionRunPath(runID)) })
		return run
	}

	first := createRunB()
	n, err := strconv.Atoi(jsonField(first, "number"))
	require.NoError(t, err, "tenant B's runs are numbered by the counter")
	taken := strconv.Itoa(n + 1)
	status, raw, err := clientB.Patch(productionRunPath(jsonField(first, "id")), map[string]any{"number": taken}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)

	next := createRunB()
	assert.NotEqual(t, taken, jsonField(next, "number"), "the counter skips the number the renamed run holds")
}

func TestNumbering_ACustomerNumberSavedTwiceAtOnceGoesToOneCustomer(t *testing.T) {
	t.Parallel()
	once := dashSuppliersOnce(apiClient)
	number := uniqueName("CUST-RACE")

	results := race(6, func(i int) (int, []byte, error) {
		body := validCustomerBody(uniqueName("e2e-number-race"))
		body["number"] = number
		return once.Post(customersPath, body, newIdempotencyKey())
	})

	created := 0
	for _, r := range results {
		if r.status == http.StatusCreated {
			created++
			id := jsonField(parseJSON(r.body), "id")
			t.Cleanup(func() { apiClient.Delete(customersPath + "/" + id) })
			continue
		}
		require.Equal(t, http.StatusConflict, r.status, "a create that loses the number conflicts: %s", r)
		assert.Equal(t, "number", jsonField(jsonObject(parseJSON(r.body), "error"), "param"))
	}
	assert.Equal(t, 1, created, "exactly one customer takes the number: %v", results)
}

func TestNumbering_AnAllocatedCustomerNumberSkipsOneTypedIn(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()
	status, raw, err := clientB.Post(accountGroupsPath, map[string]any{"name": uniqueName("e2e-number-skip"), "type": "type_group"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, raw)
	typeGroupID := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { clientB.Delete(accountGroupsPath + "/" + typeGroupID) })
	createCustomerB := func(number string) map[string]any {
		t.Helper()
		name := uniqueName("e2e-number-skip")
		// Tenant B has no carrier or terms of its own; the platform's are open to every account.
		body := map[string]any{
			"name":                     name,
			"status":                   "normal",
			"default_carrier_id":       SeedSystemCarrierID,
			"default_payment_term_id":  SeedDefaultPaymentTermID,
			"default_shipping_term_id": SeedShippingTermID,
			"customer_type_group_id":   typeGroupID,
			"bill_to_address":          map[string]any{"name": name + " Billing", "country": "US"},
			"ship_to_address":          map[string]any{"name": name + " Shipping", "country": "US"},
		}
		if number != "" {
			body["number"] = number
		}
		status, raw, err := clientB.Post(customersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, http.StatusCreated, status, raw)
		customer := parseJSON(raw)
		id := jsonField(customer, "id")
		t.Cleanup(func() { clientB.Delete(customersPath + "/" + id) })
		return customer
	}

	first := createCustomerB("")
	n, err := strconv.Atoi(jsonField(first, "number"))
	require.NoError(t, err, "an allocated customer number is numeric")
	typedIn := strconv.Itoa(n + 1)
	createCustomerB(typedIn)

	next := createCustomerB("")
	assert.NotEqual(t, typedIn, jsonField(next, "number"), "the counter skips the number someone typed in")
}

// A scan starts its run. Runs created at the same moment must not stop that: allocating a run number used
// to share-lock every run of the account, and the scan starting its run lost the deadlock that followed.
func TestNumbering_ScansStartTheirRunsWhileOtherRunsAreCreated(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	const scans = 6
	runIDs, planned := make([]string, scans), make([]string, scans)
	for i := range planned {
		run := createRunWithBatches(t, map[string]any{"item_id": c.a.id, "quantity_value": "1", "quantity_unit_id": unitEach})
		runIDs[i] = jsonField(run, "id")
		batches := runBatches(t, runIDs[i], "run")
		require.Len(t, batches, 1)
		planned[i] = jsonField(batches[0], "id")
	}
	once := dashSuppliersOnce(apiClient)

	results := race(2*scans, func(i int) (int, []byte, error) {
		if i < scans {
			return once.Post(batchesPath+"/actions/initialize", map[string]any{"batch_id": planned[i], "scanning_station_id": c.initStation}, newIdempotencyKey())
		}
		return once.Post(productionRunsPath, map[string]any{"responsible_user_id": SeedUserID}, newIdempotencyKey())
	})

	for i, r := range results {
		require.Equal(t, http.StatusCreated, r.status, "%s", r)
		id := jsonField(parseJSON(r.body), "id")
		if i < scans {
			settleScanAtCleanup(t, "initialize", id)
		} else {
			t.Cleanup(func() { apiClient.Delete(productionRunPath(id)) })
		}
	}
	for _, runID := range runIDs {
		status, raw, err := apiClient.GetListRaw(productionRunPath(runID), nil)
		require.NoError(t, err)
		requireStatus(t, http.StatusOK, status, raw)
		assert.NotEmpty(t, jsonField(parseJSON(raw), "started_at"), "a scanned run %s is started", runID)
	}
}
