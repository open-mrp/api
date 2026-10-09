//go:build e2e

package api_test

import (
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readyPortalRegistration struct {
	buyer       *Client
	sessionPath string
	userID      string
	sellerID    string
}

func readyNewCustomerRegistration(t *testing.T) readyPortalRegistration {
	t.Helper()
	buyer := newPortalBuyerClient(t)
	session := startPortalRegistration(t, buyer)
	sessionPath := portalRegSessionsPath + "/" + jsonField(session, "id")

	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"session_data":         fullNewCustomerData("E2E Concurrent Buyer " + uuid.New().String()[:8]),
		"is_existing_customer": false,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	return readyPortalRegistration{
		buyer:       buyer,
		sessionPath: sessionPath,
		userID:      jsonField(session, "user_id"),
		sellerID:    jsonField(session, "seller_account_id"),
	}
}

func registeredCustomerNumber(t *testing.T, sellerID, userID string) string {
	t.Helper()
	rows, err := authDB(t).Query(`SELECT ar.external_number FROM account_relation ar
		JOIN account_user au ON au.account_id = ar.counterparty_account_id
		WHERE ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer' AND au.user_id = ?`,
		sellerID, userID)
	require.NoError(t, err)
	defer rows.Close()
	var numbers []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		numbers = append(numbers, n)
	}
	require.NoError(t, rows.Err())
	require.Len(t, numbers, 1, "buyer %s is linked to exactly one customer of the seller", userID)
	return numbers[0]
}

func TestPortalRegistration_ConcurrentCompletionsGetDistinctCustomerNumbers(t *testing.T) {
	t.Parallel()
	const buyers = 8

	regs := make([]readyPortalRegistration, buyers)
	for i := range regs {
		regs[i] = readyNewCustomerRegistration(t)
	}

	type result struct {
		status int
		body   []byte
		err    error
	}
	results := make([]result, buyers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, reg := range regs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, body, err := reg.buyer.Post(reg.sessionPath+"/actions/complete", nil, newIdempotencyKey())
			results[i] = result{status: status, body: body, err: err}
		}()
	}
	close(start)
	wg.Wait()

	for _, r := range results {
		require.NoError(t, r.err)
		requireStatus(t, 200, r.status, r.body)
		assert.NotEmpty(t, jsonField(parseJSON(r.body), "completed_at"))
	}

	seen := map[string]bool{}
	for _, reg := range regs {
		number := registeredCustomerNumber(t, reg.sellerID, reg.userID)
		assert.False(t, seen[number], "customer number %s was handed to two registrants", number)
		seen[number] = true
		assert.Equal(t, 1, countInAccount(t, `SELECT COUNT(*) FROM account_relation
			WHERE owner_account_id = ? AND account_relation_role_code = 'customer' AND external_number = ?`,
			reg.sellerID, number), "customer number %s belongs to one customer", number)
	}
	assert.Len(t, seen, buyers)
}

func TestCustomerNumbers_AllocationSkipsANumberWrittenWithLeadingZeros(t *testing.T) {
	prop := sysPropertyByCode(t, apiClient, "customer_number")
	id := jsonField(prop, "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)

	base := 1_500_000_000 + rand.IntN(500_000_000)
	taken := strconv.Itoa(base + 1)
	require.False(t, numberInUse(t, SeedAccountID, "customer_number", base+1), "precondition: customer number %s is free", taken)

	padded := validCustomerBody(uniqueName("e2e-padded-number"))
	padded["number"] = "00" + taken
	created := createAndCleanup(t, customersPath, padded)
	require.Equal(t, "00"+taken, jsonField(created, "number"))

	setSysPropertyValue(t, apiClient, id, base)

	allocated := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-allocated-number")))
	number := jsonField(allocated, "number")
	assert.NotEqual(t, taken, number, "the counter must not hand out a number already written as 00%s", taken)
	assert.Equal(t, strconv.Itoa(base+2), number)
	assert.False(t, numberInUse(t, SeedAccountID, "customer_number", base+1), "no customer carries %s bare", taken)
}
