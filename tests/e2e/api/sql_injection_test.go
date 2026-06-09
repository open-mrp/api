//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SQL injection tests verify that user-controllable inputs cannot escape
// parameterized queries. We exercise three independent surfaces:
//
//  1. Path params  — IDs like /v1/sales/customers/{id}
//  2. Query params — search (q=) and filter values
//  3. Body fields  — JSON strings stored via POST
//
// The contract for a properly parameterized stack:
//
//   - Path / query injection never returns 5xx and never returns rows the
//     attacker isn't entitled to (no row-count blowup, no cross-tenant leak).
//   - Body strings round-trip verbatim — they're stored as literals, never
//     interpreted by the SQL engine.

// sqlInjectionPayloads are classic SQLi attempts: string terminators, comment
// sequences, tautologies, stacked statements, and UNION attempts.
var sqlInjectionPayloads = []string{
	`'`,
	`"`,
	`';`,
	`';--`,
	`' OR '1'='1`,
	`' OR 1=1 --`,
	`' OR 1=1 #`,
	`" OR ""="`,
	`' UNION SELECT NULL --`,
	`' UNION SELECT id,name FROM account --`,
	`'; DROP TABLE customer; --`,
	`'; DELETE FROM customer WHERE '1'='1`,
	`' AND SLEEP(5) --`,
	`1) OR ('1'='1`,
	`admin'--`,
	"\\'",
}

// TestSQLInjection_BodyField_RoundTripsLiterally posts a customer with a SQL
// injection payload as its name and verifies the value is stored verbatim. If
// the SQL engine were interpreting the input, the round-trip would fail or
// extra rows would appear.
func TestSQLInjection_BodyField_RoundTripsLiterally(t *testing.T) {
	t.Parallel()

	for i, payload := range sqlInjectionPayloads {
		payload := payload
		t.Run(fmt.Sprintf("payload_%d", i), func(t *testing.T) {
			t.Parallel()

			// Embed the payload inside a unique marker so the resource is
			// addressable via search even when the payload itself is junk
			// (e.g. just `'`). The marker stays ASCII-clean.
			marker := uniqueName("e2e-sqli-body")
			name := marker + " " + payload

			body := validCustomerBody(name)
			status, raw, err := apiClient.Post(customersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 201, status, raw)

			created := parseJSON(raw)
			id := jsonField(created, "id")
			require.NotEmpty(t, id, "created customer should have an id")
			t.Cleanup(func() { apiClient.Delete(customersPath + "/" + id) })

			// The name must round-trip exactly — if SQL injection had taken
			// effect, the stored value would diverge from the input.
			assert.Equal(t, name, jsonField(created, "name"),
				"create response: name should round-trip verbatim")

			status, raw, err = apiClient.GetListRaw(customersPath+"/"+id, nil)
			require.NoError(t, err)
			requireStatus(t, 200, status, raw)

			fetched := parseJSON(raw)
			assert.Equal(t, name, jsonField(fetched, "name"),
				"get response: name should round-trip verbatim")
			assert.Equal(t, id, jsonField(fetched, "id"),
				"fetched customer id should match created id")
		})
	}
}

// TestSQLInjection_SearchQuery_DoesNotBypassWhereClause verifies that classic
// tautology payloads (e.g. `' OR 1=1 --`) in the `q=` search parameter cannot
// bypass the WHERE clause and return all rows. A correctly parameterized query
// treats the payload as a literal search term, so it should match at most the
// few rows that literally contain the payload string — never the full list.
func TestSQLInjection_SearchQuery_DoesNotBypassWhereClause(t *testing.T) {
	t.Parallel()

	// Establish a baseline of how many rows exist with limit=100. If injection
	// worked, an `' OR 1=1 --` search would return this many rows. A safe
	// implementation returns at most the rows that literally contain the
	// payload string — typically zero.
	baselineList, _, err := apiClient.GetList(customersPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	baselineCount := len(baselineList.Data)
	if baselineCount < 2 {
		t.Fatalf("need at least 2 customers in the tenant for this test; got %d", baselineCount)
	}

	// A benign search for a random unique string establishes the expected
	// "no match" upper bound. Any safe injection attempt should return at
	// most this many rows.
	benign := uniqueName("e2e-sqli-zzz-no-match")
	benignList, _, err := apiClient.GetList(customersPath, url.Values{"q": {benign}})
	require.NoError(t, err)
	benignCount := len(benignList.Data)

	for i, payload := range sqlInjectionPayloads {
		payload := payload
		t.Run(fmt.Sprintf("payload_%d", i), func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(customersPath, url.Values{
				"q":     {payload},
				"limit": {"100"},
			})
			require.NoError(t, err)
			// 400 (rejected by validation) is acceptable. 500 is never
			// acceptable — that would mean the SQL layer choked on
			// attacker-controlled input.
			assert.NotEqual(t, 500, status,
				"SQLi payload %q should not 500: %s", payload, string(body))
			if status != 200 {
				return
			}

			list, _, err := apiClient.GetList(customersPath, url.Values{
				"q":     {payload},
				"limit": {"100"},
			})
			require.NoError(t, err)

			// The decisive check: an injection that bypassed the WHERE
			// clause would return the full row set. A safe query returns
			// at most the (small) number of rows whose name literally
			// contains the payload — bounded above by the benign count + a
			// small constant for any accidental literal matches.
			assert.Less(t, len(list.Data), baselineCount,
				"SQLi payload %q returned %d rows (baseline=%d) — looks like the WHERE clause was bypassed",
				payload, len(list.Data), baselineCount)

			// Tighter check when the payload contains no characters that
			// could match a real customer name. Most payloads here are
			// SQL-only punctuation, so we expect very few matches.
			// We allow up to len(sqlInjectionPayloads) extra matches
			// because the parallel BodyField test creates customers
			// whose names contain the payload strings, and short
			// payloads like `'` are substrings of those names.
			if !containsAlphaNumRun(payload, 3) {
				maxAllowed := benignCount + len(sqlInjectionPayloads)
				assert.LessOrEqual(t, len(list.Data), maxAllowed,
					"SQLi payload %q returned more matches (%d) than expected (benign=%d + body-test headroom=%d)",
					payload, len(list.Data), benignCount, len(sqlInjectionPayloads))
			}
		})
	}
}

// TestSQLInjection_PathParam_RejectedSafely verifies that SQL injection
// payloads in path parameters never produce a 500 or accidentally return a
// real resource. The expected outcomes are 400 (validation rejects the ID
// format), 404 (no row matches), or 405 (method not allowed for the parsed
// path) — anything else suggests the payload reached the SQL layer.
func TestSQLInjection_PathParam_RejectedSafely(t *testing.T) {
	t.Parallel()

	for i, payload := range sqlInjectionPayloads {
		payload := payload
		t.Run(fmt.Sprintf("payload_%d", i), func(t *testing.T) {
			t.Parallel()

			path := customersPath + "/" + url.PathEscape(payload)
			status, body, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err)
			assert.NotEqual(t, 500, status,
				"SQLi path param %q should not 500: %s", payload, string(body))
			assert.NotEqual(t, 200, status,
				"SQLi path param %q should not match a real resource: %s", payload, string(body))
		})
	}
}

// TestSQLInjection_FilterParam_RejectedSafely verifies that ID-shaped filter
// parameters reject SQL injection payloads without 500ing. A parameterized
// query receiving an invalid ID returns 400 (validation) or 200 with an
// empty result set — never the full table and never a 500.
func TestSQLInjection_FilterParam_RejectedSafely(t *testing.T) {
	t.Parallel()

	for i, payload := range sqlInjectionPayloads {
		payload := payload
		t.Run(fmt.Sprintf("payload_%d", i), func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(customersPath, url.Values{
				"customer_group_ids": {payload},
				"limit":              {"100"},
			})
			require.NoError(t, err)
			assert.NotEqual(t, 500, status,
				"SQLi filter param %q should not 500: %s", payload, string(body))

			if status != 200 {
				return
			}

			// Any 200 response with an invalid group id should return an
			// empty list — never the full customer table.
			list, _, err := apiClient.GetList(customersPath, url.Values{
				"customer_group_ids": {payload},
				"limit":              {"100"},
			})
			require.NoError(t, err)
			assertEmptyListData(t, list.Data,
				fmt.Sprintf("SQLi filter %q should match no rows", payload))
		})
	}
}

// TestSQLInjection_NoCrossTenantLeak verifies that a tenant cannot use SQL
// injection payloads in search or filter parameters to read resources owned
// by another tenant. We create a uniquely-named customer in tenant A, then
// from tenant B run SQLi payloads against the search and filter parameters
// and assert that tenant A's customer never appears.
func TestSQLInjection_NoCrossTenantLeak(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	uniqueA := uniqueName("e2e-sqli-tenanta-secret")
	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueA))
	tenantACustomerID := jsonField(created, "id")
	require.NotEmpty(t, tenantACustomerID)

	probes := append([]string{uniqueA, tenantACustomerID}, sqlInjectionPayloads...)

	for i, probe := range probes {
		probe := probe
		t.Run(fmt.Sprintf("probe_%d", i), func(t *testing.T) {
			t.Parallel()

			// Search from tenant B with payload — must never expose
			// tenant A's customer.
			status, body, err := clientB.GetListRaw(customersPath, url.Values{
				"q":     {probe},
				"limit": {"100"},
			})
			require.NoError(t, err)
			assert.NotEqual(t, 500, status,
				"SQLi probe %q should not 500 (tenant B): %s", probe, string(body))
			if status == 200 {
				list, _, err := clientB.GetList(customersPath, url.Values{
					"q":     {probe},
					"limit": {"100"},
				})
				require.NoError(t, err)
				for _, item := range list.Data {
					id := DataItemField(item, "id")
					assert.NotEqual(t, tenantACustomerID, id,
						"tenant B search for %q leaked tenant A customer %s", probe, tenantACustomerID)
				}
			}

			// Direct GET-by-id from tenant B with the real tenant A id is
			// already covered by TestTenantIsolation_GetByID. Here we
			// also try the injection payloads as the path id — they must
			// never resolve to tenant A's customer.
			status, body, err = clientB.GetListRaw(customersPath+"/"+url.PathEscape(probe), nil)
			require.NoError(t, err)
			assert.NotEqual(t, 500, status,
				"SQLi path probe %q should not 500 (tenant B): %s", probe, string(body))
			if status == 200 {
				fetched := parseJSON(body)
				assert.NotEqual(t, tenantACustomerID, jsonField(fetched, "id"),
					"tenant B path GET for %q leaked tenant A customer %s", probe, tenantACustomerID)
			}
		})
	}
}

// containsAlphaNumRun reports whether s contains a run of at least n
// consecutive ASCII alphanumeric characters. Payloads with no such run
// cannot match any real customer name as a literal substring, which lets
// the search test assert "zero matches" rather than just "fewer than the
// baseline".
func containsAlphaNumRun(s string, n int) bool {
	run := 0
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			run++
			if run >= n {
				return true
			}
		default:
			run = 0
		}
	}
	return false
}
