//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/augno/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountPricesPath = "/v1/sales/account-prices"

// assertDecimalEquals compares an API decimal string numerically. Decimals come back at
// full scale ("33.330000000000000000000000000000"), so a string compare against the value
// that was sent would fail on padding alone.
func assertDecimalEquals(t *testing.T, expected float64, actual, field string) {
	t.Helper()
	got, err := strconv.ParseFloat(actual, 64)
	require.NoError(t, err, "%s is not a decimal: %q", field, actual)
	assert.InDelta(t, expected, got, 0.0001, "%s mismatch", field)
}

// ──────────────────────────────────────────────
// AccountPrice — Include Tests
// ──────────────────────────────────────────────

func TestAccountPrices_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["recipient_account"], "recipient_account should be null without include")
	assert.Nil(t, got["product_line"], "product_line should be null without include")
	assert.Nil(t, got["categories"], "categories should be null without include")
	assert.Nil(t, got["attributes"], "attributes should be null without include")
}

func TestAccountPrices_IncludeRecipientAccount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"recipient_account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["recipient_account"]
	assert.True(t, ok, "recipient_account key should be present with ?include=recipient_account")
	if acc := jsonObject(got, "recipient_account"); acc != nil {
		// account or customer object (recipient is typically an account)
		obj := jsonField(acc, "object")
		assert.NotEmpty(t, obj)
		assert.NotEmpty(t, jsonField(acc, "id"))
	}
}

func TestAccountPrices_IncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["product_line"]
	assert.True(t, ok, "product_line key should be present with ?include=product_line")
	if pl := jsonObject(got, "product_line"); pl != nil {
		assert.Equal(t, "product_line", jsonField(pl, "object"))
	}
}

func TestAccountPrices_IncludeCategories(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"categories"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cats := jsonObject(got, "categories")
	require.NotNil(t, cats, "categories should be present with ?include=categories")
	assert.Equal(t, "list", jsonField(cats, "object"))
}

func TestAccountPrices_IncludeAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+SeedAccountPriceID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present with ?include=attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
}

// ──────────────────────────────────────────────
// AccountPrice — List
// ──────────────────────────────────────────────

func TestAccountPrices_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountPricesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assertListContainsID(t, accountPricesPath, nil, SeedAccountPriceID)
}

func TestAccountPrices_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountPricesPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "account_price", jsonField(m, "object"))
		assertIDFormat(t, jsonField(m, "id"), id.AccountPriceIDPrefix)
		assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")
		// The rate is not expandable, so it is always populated.
		rate := jsonObject(m, "rate")
		require.NotNil(t, rate, "rate should always be present")
		assert.NotEmpty(t, jsonField(rate, "value"))
	}
}

func TestAccountPrices_ListCursorPagination(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, accountPricesPath, nil)
}

// The list is the only account-price endpoint that takes includes as a whole page,
// so this pins that the gated sub-objects hydrate for every row rather than only on retrieve.
func TestAccountPrices_ListWithIncludesHydratesEveryRow(t *testing.T) {
	t.Parallel()
	params := url.Values{"include": {"recipient_account", "product_line", "categories", "attributes"}}
	list, _, err := apiClient.GetList(accountPricesPath, params)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "expected at least the seeded account price")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.NotNil(t, jsonObject(m, "recipient_account"), "recipient_account should hydrate on list")
		assert.NotNil(t, jsonObject(m, "product_line"), "product_line should hydrate on list")
		assert.NotNil(t, jsonObject(m, "categories"), "categories should hydrate on list")
		assert.NotNil(t, jsonObject(m, "attributes"), "attributes should hydrate on list")
	}
}

func TestAccountPrices_ListExpandableNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountPricesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assertNilField(t, m, "recipient_account")
		assertNilField(t, m, "product_line")
		assertNilField(t, m, "categories")
		assertNilField(t, m, "attributes")
	}
}

// The filter resolves to the customer plus its parent, and nothing else. SeedCustomerAccountID
// is seeded as a child of the house account (SeedAccountID), so both are admissible here and
// any third recipient would be a leak.
func TestAccountPrices_ListFilterByRecipientAccount(t *testing.T) {
	t.Parallel()
	created := createAccountPrice(t, SeedCustomerAccountID, "11.11")

	list, _, err := apiClient.GetList(accountPricesPath, url.Values{
		"recipient_account_id": {SeedCustomerAccountID},
		"include":              {"recipient_account"},
		"limit":                {"100"},
	})
	require.NoError(t, err)

	var found bool
	for _, item := range list.Data {
		m := parseJSON(item)
		recipient := jsonObject(m, "recipient_account")
		require.NotNil(t, recipient, "filter should not return rows without a recipient")
		assert.Contains(t, []string{SeedCustomerAccountID, SeedAccountID}, jsonField(recipient, "id"),
			"recipient_account_id filter must return only the customer and its parent")
		if jsonField(m, "id") == jsonField(created, "id") {
			found = true
		}
	}
	assert.True(t, found, "the price just created for this customer should match its own filter")
}

// A price recorded against a parent account also prices its children's orders, so filtering
// on a child must surface the parent's prices too. SeedChildAccountID1's parent relation is
// the house account (SeedAccountID), so a price on the house account must appear here.
func TestAccountPrices_ListFilterByChildIncludesParentPrices(t *testing.T) {
	t.Parallel()
	parentPrice := createAccountPrice(t, SeedAccountID, "22.22")

	list, _, err := apiClient.GetList(accountPricesPath, url.Values{
		"recipient_account_id": {SeedChildAccountID1},
		"limit":                {"100"},
	})
	require.NoError(t, err)

	var found bool
	for _, item := range list.Data {
		if DataItemField(item, "id") == jsonField(parentPrice, "id") {
			found = true
			break
		}
	}
	assert.True(t, found,
		"a price on the parent account should be listed when filtering by the child account")
}

// ──────────────────────────────────────────────
// AccountPrice — Create
// ──────────────────────────────────────────────

// createAccountPrice creates an account price for the given recipient and registers cleanup.
func createAccountPrice(t *testing.T, recipientAccountID, rateValue string) map[string]any {
	t.Helper()
	return createAndCleanup(t, accountPricesPath, map[string]any{
		"recipient_account_id": recipientAccountID,
		"product_line_id":      SeedProductLineID,
		"rate": map[string]any{
			"value":               rateValue,
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	})
}

func TestAccountPrices_CreateAllFieldsAndResponseShape(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(accountPricesPath+"?include=recipient_account&include=product_line&include=categories&include=attributes", map[string]any{
		"recipient_account_id": SeedCustomerAccountID,
		"product_line_id":      SeedProductLineID,
		"rate": map[string]any{
			"value":               "33.33",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
		"category_ids":  []string{SeedItemCategoryID},
		"attribute_ids": []string{SeedAttributeID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	priceID := jsonField(got, "id")
	require.NotEmpty(t, priceID)
	t.Cleanup(func() { apiClient.Delete(accountPricesPath + "/" + priceID) })

	assertIDFormat(t, priceID, id.AccountPriceIDPrefix)
	assertCreatedLocation(t, resp.Header, priceID)
	assertObjectField(t, got, "account_price")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	rate := jsonObject(got, "rate")
	require.NotNil(t, rate)
	assert.Equal(t, "rate", jsonField(rate, "object"))
	assertDecimalEquals(t, 33.33, jsonField(rate, "value"), "rate.value")
	assert.Equal(t, e2eCurrencyUnitID, jsonField(jsonObject(rate, "numerator_unit"), "id"))
	assert.Equal(t, SeedUnitID, jsonField(jsonObject(rate, "denominator_unit"), "id"))

	recipient := jsonObject(got, "recipient_account")
	require.NotNil(t, recipient)
	assert.Equal(t, SeedCustomerAccountID, jsonField(recipient, "id"))

	productLine := jsonObject(got, "product_line")
	require.NotNil(t, productLine)
	assert.Equal(t, SeedProductLineID, jsonField(productLine, "id"))

	categories := jsonListData(got, "categories")
	require.Len(t, categories, 1, "the one category sent should come back")
	assert.Equal(t, SeedItemCategoryID, jsonField(categories[0].(map[string]any), "id"))

	attributes := jsonListData(got, "attributes")
	require.Len(t, attributes, 1, "the one attribute sent should come back")
	assert.Equal(t, SeedAttributeID, jsonField(attributes[0].(map[string]any), "id"))
}

func TestAccountPrices_CreateValidation(t *testing.T) {
	t.Parallel()

	validRate := map[string]any{
		"value":               "1.00",
		"numerator_unit_id":   e2eCurrencyUnitID,
		"denominator_unit_id": SeedUnitID,
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing recipient_account_id", map[string]any{"product_line_id": SeedProductLineID, "rate": validRate}},
		{"missing product_line_id", map[string]any{"recipient_account_id": SeedCustomerAccountID, "rate": validRate}},
		{"missing rate", map[string]any{"recipient_account_id": SeedCustomerAccountID, "product_line_id": SeedProductLineID}},
		{"rate missing value", map[string]any{
			"recipient_account_id": SeedCustomerAccountID,
			"product_line_id":      SeedProductLineID,
			"rate":                 map[string]any{"numerator_unit_id": e2eCurrencyUnitID, "denominator_unit_id": SeedUnitID},
		}},
		{"rate missing denominator unit", map[string]any{
			"recipient_account_id": SeedCustomerAccountID,
			"product_line_id":      SeedProductLineID,
			"rate":                 map[string]any{"value": "1.00", "numerator_unit_id": e2eCurrencyUnitID},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(accountPricesPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"%s should be rejected, got %d: %s", tc.name, status, string(body))
		})
	}
}

func TestAccountPrices_CreateRejectsUnknownField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountPricesPath, map[string]any{
		"recipient_account_id": SeedCustomerAccountID,
		"product_line_id":      SeedProductLineID,
		"rate": map[string]any{
			"value":               "1.00",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
		// The pre-RateInput spelling: it must not be silently accepted now that the
		// rate is taken as one object.
		"rate_value": "1.00",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "rate_value")
}

func TestAccountPrices_CreateIdempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	body := map[string]any{
		"recipient_account_id": SeedCustomerAccountID,
		"product_line_id":      SeedProductLineID,
		"rate": map[string]any{
			"value":               "44.44",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}

	status1, body1, err := apiClient.Post(accountPricesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	firstID := jsonField(parseJSON(body1), "id")
	t.Cleanup(func() { apiClient.Delete(accountPricesPath + "/" + firstID) })

	status2, body2, err := apiClient.Post(accountPricesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, firstID, jsonField(parseJSON(body2), "id"),
		"replaying the idempotency key must not create a second price")
}

// ──────────────────────────────────────────────
// AccountPrice — Update
// ──────────────────────────────────────────────

func TestAccountPrices_UpdateReplacesRateWhole(t *testing.T) {
	t.Parallel()
	created := createAccountPrice(t, SeedCustomerAccountID, "55.55")
	priceID := jsonField(created, "id")

	status, body, err := apiClient.Patch(accountPricesPath+"/"+priceID, map[string]any{
		"rate": map[string]any{
			"value":               "66.66",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Equal(t, priceID, jsonField(updated, "id"), "ID must not change")
	assertDecimalEquals(t, 66.66, jsonField(jsonObject(updated, "rate"), "value"), "rate.value")

	// Re-read so the assertion covers what was persisted, not just what was echoed.
	status, body, err = apiClient.GetListRaw(accountPricesPath+"/"+priceID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDecimalEquals(t, 66.66, jsonField(jsonObject(parseJSON(body), "rate"), "value"), "persisted rate.value")
}

func TestAccountPrices_UpdateOmittedFieldsUnchanged(t *testing.T) {
	t.Parallel()
	created := createAccountPrice(t, SeedCustomerAccountID, "77.77")
	priceID := jsonField(created, "id")

	// Change only the product line; the rate and recipient must survive untouched.
	status, body, err := apiClient.Patch(accountPricesPath+"/"+priceID+"?include=recipient_account&include=product_line", map[string]any{
		"product_line_id": SeedProductLineID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assertDecimalEquals(t, 77.77, jsonField(jsonObject(updated, "rate"), "value"),
		"omitting rate must leave it unchanged")
	assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(updated, "recipient_account"), "id"),
		"omitting recipient_account_id must leave it unchanged")
}

// category_ids/attribute_ids replace the whole set rather than merging, and an empty
// list clears it — the behaviour the details form depends on.
func TestAccountPrices_UpdateReplacesCategoriesAndAttributes(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountPricesPath, map[string]any{
		"recipient_account_id": SeedCustomerAccountID,
		"product_line_id":      SeedProductLineID,
		"rate": map[string]any{
			"value":               "88.88",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
		"category_ids":  []string{SeedItemCategoryID},
		"attribute_ids": []string{SeedAttributeID},
	})
	priceID := jsonField(created, "id")

	status, body, err := apiClient.Patch(accountPricesPath+"/"+priceID+"?include=categories&include=attributes", map[string]any{
		"category_ids":  []string{},
		"attribute_ids": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Empty(t, jsonListData(updated, "categories"), "an empty category_ids clears the set")
	assert.Empty(t, jsonListData(updated, "attributes"), "an empty attribute_ids clears the set")
}

func TestAccountPrices_UpdateNonexistentReturns404(t *testing.T) {
	t.Parallel()
	missingID := mustGenID(t, id.AccountPriceIDPrefix)
	status, body, err := apiClient.Patch(accountPricesPath+"/"+missingID, map[string]any{
		"rate": map[string]any{
			"value":               "1.00",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// AccountPrice — Delete
// ──────────────────────────────────────────────

func TestAccountPrices_DeleteRemovesPrice(t *testing.T) {
	t.Parallel()
	created := createAccountPrice(t, SeedCustomerAccountID, "99.99")
	priceID := jsonField(created, "id")

	status, body, err := apiClient.Delete(accountPricesPath + "/" + priceID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	getStatus, getBody, err := apiClient.GetListRaw(accountPricesPath+"/"+priceID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, getStatus, getBody)
	requireErrorResponse(t, getBody, "resource_not_found", "invalid_request_error")
}

func TestAccountPrices_DeleteNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(accountPricesPath + "/" + mustGenID(t, id.AccountPriceIDPrefix))
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestAccountPrices_RetrieveNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountPricesPath+"/"+mustGenID(t, id.AccountPriceIDPrefix), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}
