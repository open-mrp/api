//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview7APIVersion = "1.0.forge-preview.7"

// preview.8 matches an invoice search against the start of each field and leaves notes out. A client still
// pinned to preview.7 keeps matching anywhere in every field, notes included.
func TestVersionCompat_Preview7InvoiceSearchMatchesAnywhere(t *testing.T) {
	t.Parallel()
	pinned := apiClient.WithAPIVersion(preview7APIVersion)
	customer := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customer, nil)

	alias, notes := uniqueName("e2e-compat-alias"), uniqueName("e2e-compat-notes")
	_, err := authDB(t).Exec(`UPDATE account_relation SET alias = ?, notes = ? WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		alias, notes, SeedAccountID, customer)
	require.NoError(t, err)
	invoiceNote := uniqueName("e2e-compat-invoice-note")
	patchInvoice(t, inv.invoiceID, map[string]any{"note": invoiceNote})
	name := getCustomerName(t, customer)

	for what, q := range map[string]string{
		"invoice note":        invoiceNote,
		"customer notes":      notes,
		"middle of the alias": alias[4:],
		"middle of the name":  name[4:],
	} {
		params := url.Values{"q": {q}, "customer_ids": {customer}}
		assert.Contains(t, invoiceListIDs(t, pinned, invoicesPath, params), inv.invoiceID, "preview.7 %s %q", what, q)
		assert.NotContains(t, invoiceListIDs(t, apiClient, invoicesPath, params), inv.invoiceID, "latest %s %q", what, q)
	}

	// A preview.7 client that already asks for prefix matching gets it.
	params := url.Values{"q": {alias[4:]}, "q_match": {"prefix"}, "customer_ids": {customer}}
	assert.NotContains(t, invoiceListIDs(t, pinned, invoicesPath, params), inv.invoiceID)
}
