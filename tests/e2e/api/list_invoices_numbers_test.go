//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listInvoiceNumbers(t *testing.T, params url.Values) (int, []string) {
	t.Helper()
	status, body, err := apiClient.GetListRaw(financeInvoicesPath, params)
	require.NoError(t, err)
	if status != 200 {
		return status, nil
	}
	var numbers []string
	for _, raw := range jsonArray(parseJSON(body), "data") {
		numbers = append(numbers, jsonField(raw.(map[string]any), "number"))
	}
	return status, numbers
}

// `numbers` finds exactly the invoices named — not ones whose number, note, customer or PO merely
// contains the text, as `q` does — so an integration can look an invoice up by its number.
func TestListInvoices_NumbersMatchExactly(t *testing.T) {
	t.Parallel()

	a := invoiceNewCustomer(t)
	b := invoiceNewCustomer(t)
	numA := jsonField(getInvoice(t, a.invoiceID), "number")
	numB := jsonField(getInvoice(t, b.invoiceID), "number")
	require.NotEqual(t, numA, numB)

	t.Run("one number", func(t *testing.T) {
		status, got := listInvoiceNumbers(t, url.Values{"numbers": {numA}})
		require.Equal(t, 200, status)
		assert.Equal(t, []string{numA}, got)
	})

	t.Run("several numbers", func(t *testing.T) {
		status, got := listInvoiceNumbers(t, url.Values{"numbers": {numA, numB}})
		require.Equal(t, 200, status)
		assert.ElementsMatch(t, []string{numA, numB}, got)
	})

	t.Run("part of a number matches nothing", func(t *testing.T) {
		status, got := listInvoiceNumbers(t, url.Values{"numbers": {numA[:len(numA)-1]}})
		require.Equal(t, 200, status)
		assert.Empty(t, got)
	})

	t.Run("an unknown number matches nothing", func(t *testing.T) {
		status, got := listInvoiceNumbers(t, url.Values{"numbers": {"ZZ-NO-SUCH-INVOICE"}})
		require.Equal(t, 200, status)
		assert.Empty(t, got)
	})

	t.Run("combines with other filters", func(t *testing.T) {
		status, got := listInvoiceNumbers(t, url.Values{"numbers": {numA, numB}, "customer_ids": {a.customerID}})
		require.Equal(t, 200, status)
		assert.Equal(t, []string{numA}, got, "only the named invoice of that customer")

		status, got = listInvoiceNumbers(t, url.Values{"numbers": {numA}, "customer_ids": {b.customerID}})
		require.Equal(t, 200, status)
		assert.Empty(t, got, "the number belongs to another customer")

		// A number can appear inside another customer's generated name, so search by B's whole name.
		status, body, err := apiClient.GetListRaw(customersPath+"/"+b.customerID, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		nameB := jsonField(parseJSON(body), "name")
		status, got = listInvoiceNumbers(t, url.Values{"numbers": {numA, numB}, "q": {nameB}})
		require.Equal(t, 200, status)
		assert.Equal(t, []string{numB}, got, "a search narrows the named invoices further")
	})

	t.Run("more than 100 numbers is rejected", func(t *testing.T) {
		numbers := make([]string, 101)
		for i := range numbers {
			numbers[i] = numA
		}
		status, _ := listInvoiceNumbers(t, url.Values{"numbers": numbers})
		assert.Equal(t, 400, status)
	})
}
