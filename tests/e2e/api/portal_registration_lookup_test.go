//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A buyer joining an existing customer finds it by its customer number, or by one of its sales order or
// invoice numbers, and confirms it with the ZIP on its default billing or shipping address. A number alone
// links nobody, and a miss on either part reads the same.

const (
	seedCustomerBillingPostalCode  = "94103"
	seedCustomerShippingPostalCode = "44113"
	seedCustomerSalesOrderNumber   = "ORD-001"
	seedCustomerInvoiceNumber      = "INV-001"
	existingCustomerNotFound       = "We couldn't find an account with that number and ZIP code."
)

// completeExistingLookup saves an existing-customer lookup on a fresh buyer's session and completes it.
func completeExistingLookup(t *testing.T, sessionData map[string]any) (buyer *Client, status int, body []byte) {
	t.Helper()
	buyer = newPortalBuyerClient(t)
	sessionPath := portalRegSessionsPath + "/" + jsonField(startPortalRegistration(t, buyer), "id")

	patchStatus, patchBody, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"is_existing_customer": true,
		"session_data":         sessionData,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	status, body, err = buyer.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	return buyer, status, body
}

func TestPortalRegistrationLookup_LinksByEachNumberType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data map[string]any
	}{
		{"customer number with billing ZIP", map[string]any{
			"lookup_type": "customer_number", "lookup_number": SeedCustomerNumber, "lookup_postal_code": seedCustomerBillingPostalCode,
		}},
		{"sales order number with ZIP+4 of the shipping address", map[string]any{
			"lookup_type": "sales_order_number", "lookup_number": seedCustomerSalesOrderNumber, "lookup_postal_code": seedCustomerShippingPostalCode + "-1234",
		}},
		{"invoice number with padded ZIP", map[string]any{
			"lookup_type": "invoice_number", "lookup_number": " " + seedCustomerInvoiceNumber + " ", "lookup_postal_code": " " + seedCustomerBillingPostalCode + " ",
		}},
		{"lookup type omitted defaults to customer number", map[string]any{
			"lookup_number": SeedCustomerNumber, "lookup_postal_code": seedCustomerBillingPostalCode,
		}},
		{"legacy customer_number field", map[string]any{
			"customer_number": SeedCustomerNumber, "lookup_postal_code": seedCustomerShippingPostalCode,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			buyer, status, body := completeExistingLookup(t, tc.data)
			requireStatus(t, 200, status, body)
			assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"))
			assert.Contains(t, dashCustomersVendorAccounts(t, buyer), SeedCustomerAccountID)
		})
	}
}

func TestPortalRegistrationLookup_SessionEchoesLookupFields(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionPath := portalRegSessionsPath + "/" + jsonField(startPortalRegistration(t, buyer), "id")

	data := jsonObject(startPortalRegistration(t, buyer), "session_data")
	if data != nil {
		assertNilField(t, data, "lookup_type")
	}

	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"is_existing_customer": true,
		"session_data": map[string]any{
			"lookup_type": "invoice_number", "lookup_number": seedCustomerInvoiceNumber, "lookup_postal_code": seedCustomerBillingPostalCode,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	saved := jsonObject(parseJSON(body), "session_data")
	assert.Equal(t, "invoice_number", jsonField(saved, "lookup_type"))
	assert.Equal(t, seedCustomerInvoiceNumber, jsonField(saved, "lookup_number"))
	assert.Equal(t, seedCustomerBillingPostalCode, jsonField(saved, "lookup_postal_code"))
}

func TestPortalRegistrationLookup_MissesReadTheSame(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data map[string]any
	}{
		{"right number, wrong ZIP", map[string]any{
			"lookup_type": "customer_number", "lookup_number": SeedCustomerNumber, "lookup_postal_code": "10001",
		}},
		{"unknown customer number", map[string]any{
			"lookup_type": "customer_number", "lookup_number": "no-such-customer", "lookup_postal_code": seedCustomerBillingPostalCode,
		}},
		{"unknown sales order number", map[string]any{
			"lookup_type": "sales_order_number", "lookup_number": "no-such-order", "lookup_postal_code": seedCustomerBillingPostalCode,
		}},
		{"invoice number given as an order number", map[string]any{
			"lookup_type": "sales_order_number", "lookup_number": seedCustomerInvoiceNumber, "lookup_postal_code": seedCustomerBillingPostalCode,
		}},
		{"right invoice, ZIP of an address that is not a default", map[string]any{
			"lookup_type": "invoice_number", "lookup_number": seedCustomerInvoiceNumber, "lookup_postal_code": "43004",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			buyer, status, body := completeExistingLookup(t, tc.data)
			requireStatus(t, 404, status, body)
			assert.Equal(t, existingCustomerNotFound, jsonField(jsonObject(parseJSON(body), "error"), "message"))
			assert.NotContains(t, dashCustomersVendorAccounts(t, buyer), SeedCustomerAccountID)
		})
	}
}

func TestPortalRegistrationLookup_RequiresNumberAndZIP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		data  map[string]any
		param string
	}{
		{"no ZIP", map[string]any{"lookup_number": SeedCustomerNumber}, "lookup_postal_code"},
		{"no number", map[string]any{"lookup_type": "invoice_number", "lookup_postal_code": seedCustomerBillingPostalCode}, "lookup_number"},
		{"legacy customer number with no ZIP", map[string]any{"customer_number": SeedCustomerNumber}, "lookup_postal_code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, status, body := completeExistingLookup(t, tc.data)
			requireStatus(t, 422, status, body)
			assert.Equal(t, tc.param, jsonField(jsonObject(parseJSON(body), "error"), "param"))
		})
	}
}

func TestPortalRegistrationLookup_RejectsAnUnknownLookupType(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionPath := portalRegSessionsPath + "/" + jsonField(startPortalRegistration(t, buyer), "id")

	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"is_existing_customer": true,
		"session_data": map[string]any{
			"lookup_type": "purchase_order_number", "lookup_number": "PO-1", "lookup_postal_code": seedCustomerBillingPostalCode,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 200 {
		status, body, err = buyer.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
		require.NoError(t, err)
	}
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "session_data.lookup_type")
}

func TestPortalRegistrationLookup_OneShotRegisterNeedsTheZIPToo(t *testing.T) {
	t.Parallel()

	buyer := newPortalBuyerClient(t)
	status, body, err := buyer.Post("/v1/sales/customers/registration", map[string]any{
		"account_slug":         SeedAccountSlug,
		"is_existing_customer": true,
		"customer_number":      SeedCustomerNumber,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)

	status, body, err = buyer.Post("/v1/sales/customers/registration", map[string]any{
		"account_slug":         SeedAccountSlug,
		"is_existing_customer": true,
		"lookup_type":          "sales_order_number",
		"lookup_number":        seedCustomerSalesOrderNumber,
		"lookup_postal_code":   seedCustomerShippingPostalCode,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	assert.Contains(t, dashCustomersVendorAccounts(t, buyer), SeedCustomerAccountID)
}
