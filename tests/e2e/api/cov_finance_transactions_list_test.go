//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The transaction list chooses its page from the transaction table under a per-filter index hint and
// joins the rest after, resolving a customer group to its customers first. These pin what each of
// those filters returns, since the plan tests only measure how much they read.

// customerInGroup creates a customer whose type group is groupID.
func customerInGroup(t *testing.T, groupID string) string {
	t.Helper()
	body := validCustomerBody(uniqueName("e2e-tx-list-cust"))
	body["customer_type_group_id"] = groupID
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	id := jsonField(parseJSON(resp), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + id) })
	return id
}

type transactionListFixture struct {
	groupID, inGroup, outside string
	// funds is when inGroupOld and outsideOld received their funds; inGroupNew received them a day later.
	funds                              time.Time
	inGroupOld, inGroupNew, outsideOld string
}

func newTransactionListFixture(t *testing.T) transactionListFixture {
	t.Helper()
	f := transactionListFixture{groupID: newAccountGroup(t)}
	f.inGroup = customerInGroup(t, f.groupID)
	f.outside = customerInGroup(t, SeedCustomerGroupID)

	// A second within 1999 no other run shares, so a funds window holds only this fixture's payments.
	f.funds = time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(time.Now().UnixNano()%(300*24*3600)) * time.Second)
	later := f.funds.Add(24 * time.Hour)
	now := time.Now().UTC().Truncate(time.Second)
	at := func(ago time.Duration) map[string]any {
		return map[string]any{"occurred_at": rfc3339(now.Add(-ago))}
	}
	f.inGroupOld = jsonField(createPayment(t, f.inGroup, "1.00", &f.funds, at(3*time.Hour)), "id")
	f.outsideOld = jsonField(createPayment(t, f.outside, "2.00", &f.funds, at(2*time.Hour)), "id")
	f.inGroupNew = jsonField(createPayment(t, f.inGroup, "3.00", &later, at(1*time.Hour)), "id")
	return f
}

func listTransactionIDs(t *testing.T, params url.Values) []string {
	t.Helper()
	rows, _ := listPayments(t, financeTransactionsPath, params)
	return rowIDs(rows)
}

func TestTransactionsList_CustomerGroup(t *testing.T) {
	t.Parallel()
	f := newTransactionListFixture(t)

	assert.Equal(t, []string{f.inGroupNew, f.inGroupOld},
		listTransactionIDs(t, url.Values{"customer_group_ids": {f.groupID}}),
		"the group's customers' transactions, newest first")
	assert.Equal(t, []string{f.inGroupNew, f.inGroupOld},
		listTransactionIDs(t, url.Values{"customer_group_ids": {f.groupID}, "status": {"unallocated"}}),
		"combined with status")
	assert.Empty(t, listTransactionIDs(t, url.Values{"customer_group_ids": {f.groupID}, "customer_ids": {f.outside}}),
		"a customer outside the group is excluded even when named")
	assert.Empty(t, listTransactionIDs(t, url.Values{"customer_group_ids": {newAccountGroup(t)}}),
		"a group with no customers lists nothing")
}

func TestTransactionsList_FundsReceivedRange(t *testing.T) {
	t.Parallel()
	f := newTransactionListFixture(t)
	window := url.Values{
		"starts_at": {rfc3339(f.funds.Add(-time.Second))},
		"ends_at":   {rfc3339(f.funds.Add(time.Second))},
	}

	assert.Equal(t, []string{f.outsideOld, f.inGroupOld}, listTransactionIDs(t, window),
		"every transaction whose funds arrived in the window, newest first")

	withCustomer := url.Values{"customer_ids": {f.inGroup}}
	for k, v := range window {
		withCustomer[k] = v
	}
	assert.Equal(t, []string{f.inGroupOld}, listTransactionIDs(t, withCustomer), "combined with a customer")

	withStatus := url.Values{"status": {"allocated"}}
	for k, v := range window {
		withStatus[k] = v
	}
	assert.Empty(t, listTransactionIDs(t, withStatus), "combined with a status none of them has")

	twoDays := url.Values{
		"customer_group_ids": {f.groupID},
		"starts_at":          {rfc3339(f.funds.Add(-time.Second))},
		"ends_at":            {rfc3339(f.funds.Add(25 * time.Hour))},
	}
	assert.Equal(t, []string{f.inGroupNew, f.inGroupOld}, listTransactionIDs(t, twoDays), "combined with a group")
}

func TestTransactionsList_PagesBackFromAFilteredPage(t *testing.T) {
	t.Parallel()
	f := newTransactionListFixture(t)

	first, pageInfo := listPayments(t, financeTransactionsPath, url.Values{"customer_ids": {f.inGroup}, "limit": {"1"}})
	require.Equal(t, []string{f.inGroupNew}, rowIDs(first))
	require.Equal(t, true, pageInfo["has_next_page"], "page_info: %v", pageInfo)

	next := jsonField(pageInfo, "next_page_url")
	status, body, err := apiClient.GetListRawFromPageURL(&next)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	second := parseJSON(body)
	var secondIDs []string
	for _, r := range jsonArray(second, "data") {
		secondIDs = append(secondIDs, jsonField(r.(map[string]any), "id"))
	}
	require.Equal(t, []string{f.inGroupOld}, secondIDs)

	secondInfo := jsonObject(second, "page_info")
	require.Equal(t, true, secondInfo["has_prev_page"], "page_info: %v", secondInfo)
	prev := jsonField(secondInfo, "previous_page_url")
	status, body, err = apiClient.GetListRawFromPageURL(&prev)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var prevIDs []string
	for _, r := range jsonArray(parseJSON(body), "data") {
		prevIDs = append(prevIDs, jsonField(r.(map[string]any), "id"))
	}
	assert.Equal(t, []string{f.inGroupNew}, prevIDs, "the previous page is the first page again")
}
