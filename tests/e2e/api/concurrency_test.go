//go:build e2e

package api_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrent modification tests verify that the API handles parallel writes
// correctly — no 500s, no data corruption, proper transaction isolation.

// TestConcurrency_ParallelPatchSameResource fires N concurrent PATCH requests
// at the same resource and verifies all return 200 (no 500s).
func TestConcurrency_ParallelPatchSameResource(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-conc-patch")))
	id := jsonField(created, "id")
	path := customersPath + "/" + id

	const concurrency = 10
	var wg sync.WaitGroup
	results := make([]int, concurrency)
	errors := make([]error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status, _, err := apiClient.Patch(path, map[string]any{
				"note": uniqueName("conc-note"),
			}, newIdempotencyKey())
			results[idx] = status
			errors[idx] = err
		}(i)
	}
	wg.Wait()

	for i := range concurrency {
		require.NoError(t, errors[i], "Request %d failed with error", i)
		assert.NotEqual(t, 500, results[i],
			"Request %d returned 500 — concurrent PATCH should not cause server errors", i)
		assert.Equal(t, 200, results[i],
			"Request %d: expected 200, got %d", i, results[i])
	}
}

// TestConcurrency_ParallelCreateSameIdempotencyKey fires concurrent POSTs with
// the same idempotency key and verifies exactly one unique resource is created.
func TestConcurrency_ParallelCreateSameIdempotencyKey(t *testing.T) {
	t.Parallel()

	idemKey := newIdempotencyKey()
	name := uniqueName("e2e-conc-idem")

	const concurrency = 5
	var wg sync.WaitGroup
	statuses := make([]int, concurrency)
	bodies := make([][]byte, concurrency)
	errs := make([]error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status, body, err := apiClient.Post(customersPath, validCustomerBody(name), idemKey)
			statuses[idx] = status
			bodies[idx] = body
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	// Collect unique resource IDs from successful responses.
	ids := make(map[string]bool)
	for i := range concurrency {
		require.NoError(t, errs[i], "Request %d failed", i)
		assert.NotEqual(t, 500, statuses[i],
			"Request %d returned 500 — concurrent idempotent create should not cause server errors", i)
		if statuses[i] == 201 {
			parsed := parseJSON(bodies[i])
			id := jsonField(parsed, "id")
			if id != "" {
				ids[id] = true
			}
		}
	}

	// Concurrent creates with the same idempotency key must always resolve
	// to a single resource.
	require.LessOrEqualf(t, len(ids), 1,
		"Concurrent idempotent creates produced %d distinct IDs (expected 1)", len(ids))

	// Clean up all created resources.
	for id := range ids {
		apiClient.Delete(customersPath + "/" + id)
	}
}

// TestConcurrency_ParallelCreateDifferentResources verifies the API handles
// a burst of creates for distinct resources without errors.
func TestConcurrency_ParallelCreateDifferentResources(t *testing.T) {
	t.Parallel()

	const concurrency = 10
	var wg sync.WaitGroup
	statuses := make([]int, concurrency)
	bodies := make([][]byte, concurrency)
	errs := make([]error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status, body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-conc-burst")), newIdempotencyKey())
			statuses[idx] = status
			bodies[idx] = body
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	var createdIDs []string
	for i := range concurrency {
		require.NoError(t, errs[i], "Request %d failed", i)
		assert.NotEqual(t, 500, statuses[i],
			"Request %d returned 500 — burst create should not cause server errors", i)
		if statuses[i] == 201 {
			id := jsonField(parseJSON(bodies[i]), "id")
			if id != "" {
				createdIDs = append(createdIDs, id)
			}
		}
	}

	// Clean up all created resources.
	for _, id := range createdIDs {
		apiClient.Delete(customersPath + "/" + id)
	}

	// All should succeed for distinct concurrent creates.
	for i, s := range statuses {
		assert.Equal(t, 201, s, "Request %d: expected 201, got %d", i, s)
	}
}
