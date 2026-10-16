//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const orderDiscountsPath = "/v1/sales/order-discounts"

func percentOff(pct string) map[string]any {
	return map[string]any{"type": "percent_off", "percent_off": pct}
}

func amountOff(amount string) map[string]any {
	return map[string]any{"type": "amount_off", "amount_off": map[string]any{"amount": amount, "currency": "usd"}}
}

// createOrderDiscount creates a percentage discount with a unique code and registers cleanup.
func createOrderDiscount(t *testing.T, overrides map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"name":     uniqueName("e2e-ords"),
		"code":     uniqueName("E2EORDS"),
		"discount": percentOff("10"),
	}
	for k, v := range overrides {
		body[k] = v
	}
	return createAndCleanup(t, orderDiscountsPath, body)
}

// ──────────────────────────────────────────────
// OrderDiscount — List
// ──────────────────────────────────────────────

func TestOrderDiscounts_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(orderDiscountsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assertListContainsID(t, orderDiscountsPath, nil, SeedOrderDiscountID)
}

// Order discounts have no expandable sub-objects, so every field is always populated.
func TestOrderDiscounts_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(orderDiscountsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "order_discount", jsonField(m, "object"))
		assertIDFormat(t, jsonField(m, "id"), id.OrderDiscountIDPrefix)
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "code"))
		assert.Contains(t, []string{"active", "archived"}, jsonField(m, "status"))
		discount := jsonObject(m, "discount")
		require.NotNil(t, discount, "discount is always present")
		switch jsonField(discount, "type") {
		case "percent_off":
			assert.NotEmpty(t, jsonField(discount, "percent_off"))
			assertNilField(t, discount, "amount_off")
		case "amount_off":
			assert.NotNil(t, jsonObject(discount, "amount_off"))
			assertNilField(t, discount, "percent_off")
		default:
			t.Errorf("discount.type %q is not percent_off or amount_off", jsonField(discount, "type"))
		}
		require.NotNil(t, m["times_redeemed"], "times_redeemed should always be present")
		assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")
	}
}

func TestOrderDiscounts_ListCursorPagination(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, orderDiscountsPath, nil)
}

func TestOrderDiscounts_ListSearchMatchesNameAndCode(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})
	discountID := jsonField(created, "id")

	t.Run("by name", func(t *testing.T) {
		list, _, err := apiClient.GetList(orderDiscountsPath, url.Values{"q": {jsonField(created, "name")}})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(list.Data), 1)
		assert.Equal(t, discountID, DataItemField(list.Data[0], "id"))
	})

	t.Run("by code", func(t *testing.T) {
		list, _, err := apiClient.GetList(orderDiscountsPath, url.Values{"q": {jsonField(created, "code")}})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(list.Data), 1, "the search term also matches the code")
		assert.Equal(t, discountID, DataItemField(list.Data[0], "id"))
	})
}

// ──────────────────────────────────────────────
// OrderDiscount — Create
// ──────────────────────────────────────────────

func TestOrderDiscounts_CreatePercentageResponseShape(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-ords-pct")
	code := uniqueName("E2EPCT")
	resp, err := apiClient.PostFull(orderDiscountsPath, map[string]any{
		"name":     name,
		"code":     code,
		"discount": percentOff("15"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	discountID := jsonField(got, "id")
	require.NotEmpty(t, discountID)
	t.Cleanup(func() { apiClient.Delete(orderDiscountsPath + "/" + discountID) })

	assertIDFormat(t, discountID, id.OrderDiscountIDPrefix)
	assertCreatedLocation(t, resp.Header, discountID)
	assertObjectField(t, got, "order_discount")
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, code, jsonField(got, "code"))
	assert.Equal(t, "active", jsonField(got, "status"), "status defaults to active")
	discount := jsonObject(got, "discount")
	assert.Equal(t, "percent_off", jsonField(discount, "type"))
	// percent_off is a whole percent.
	assert.Equal(t, "15", jsonField(discount, "percent_off"))
	assertNilField(t, discount, "amount_off")
	assert.EqualValues(t, 0, got["times_redeemed"], "a new discount has not been applied to any order")
}

func TestOrderDiscounts_CreateAmountType(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{"discount": amountOff("25.00"), "status": "archived"})

	discount := jsonObject(created, "discount")
	assert.Equal(t, "amount_off", jsonField(discount, "type"))
	assertDecimalEquals(t, 25.00, jsonField(jsonObject(discount, "amount_off"), "amount"), "amount_off.amount")
	assert.Equal(t, "usd", jsonField(jsonObject(discount, "amount_off"), "currency"))
	assertNilField(t, discount, "percent_off")
	assert.Equal(t, "archived", jsonField(created, "status"))
}

// discount.type is a closed enum, so the gateway rejects any value outside it.
func TestOrderDiscounts_CreateRejectsInvalidDiscountType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(orderDiscountsPath, map[string]any{
		"name":     uniqueName("e2e-ords-badtype"),
		"code":     uniqueName("E2EBAD"),
		"discount": map[string]any{"type": "banana", "percent_off": "10"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "discount.type")
}

func TestOrderDiscounts_CreateValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{"code": uniqueName("E2ENONAME"), "discount": percentOff("10")}},
		{"missing code", map[string]any{"name": uniqueName("e2e-ords-nocode"), "discount": percentOff("10")}},
		{"missing discount", map[string]any{"name": uniqueName("e2e-ords-notype"), "code": uniqueName("E2ENOTYPE")}},
		{"percent over 100", map[string]any{"name": uniqueName("e2e-ords-pct"), "code": uniqueName("E2EPCT"), "discount": percentOff("150")}},
		{"other currency", map[string]any{"name": uniqueName("e2e-ords-eur"), "code": uniqueName("E2EEUR"),
			"discount": map[string]any{"type": "amount_off", "amount_off": map[string]any{"amount": "5", "currency": "eur"}}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(orderDiscountsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"%s should be rejected, got %d: %s", tc.name, status, string(body))
		})
	}
}

// Codes are unique per account and compared case-insensitively, so SAVE10 collides with save10.
func TestOrderDiscounts_CreateDuplicateCodeConflict(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})
	code := jsonField(created, "code")

	for _, variant := range []string{code, strings.ToLower(code)} {
		status, body, err := apiClient.Post(orderDiscountsPath, map[string]any{
			"name":     uniqueName("e2e-ords-dupe"),
			"code":     variant,
			"discount": percentOff("10"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "code")
	}
}

func TestOrderDiscounts_CreateIdempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	body := map[string]any{
		"name":     uniqueName("e2e-ords-idem"),
		"code":     uniqueName("E2EIDEM"),
		"discount": percentOff("10"),
	}

	status1, body1, err := apiClient.Post(orderDiscountsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	firstID := jsonField(parseJSON(body1), "id")
	t.Cleanup(func() { apiClient.Delete(orderDiscountsPath + "/" + firstID) })

	status2, body2, err := apiClient.Post(orderDiscountsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, firstID, jsonField(parseJSON(body2), "id"),
		"replaying the key must not create a second discount, which would also trip the code conflict")
}

// ──────────────────────────────────────────────
// OrderDiscount — Find by code
// ──────────────────────────────────────────────

func TestOrderDiscounts_FindByCode(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Post(orderDiscountsPath+"/actions/find-by-code",
		map[string]any{"code": jsonField(created, "code")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	found := parseJSON(body)
	assert.Equal(t, jsonField(created, "id"), jsonField(found, "id"))
	assertObjectField(t, found, "order_discount")
}

func TestOrderDiscounts_FindByCodeIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Post(orderDiscountsPath+"/actions/find-by-code",
		map[string]any{"code": strings.ToLower(jsonField(created, "code"))}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, jsonField(created, "id"), jsonField(parseJSON(body), "id"))
}

func TestOrderDiscounts_FindByUnknownCodeReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(orderDiscountsPath+"/actions/find-by-code",
		map[string]any{"code": uniqueName("E2ENOSUCHCODE")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// OrderDiscount — Update
// ──────────────────────────────────────────────

func TestOrderDiscounts_UpdateFields(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})
	discountID := jsonField(created, "id")

	newName := uniqueName("e2e-ords-renamed")
	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+discountID, map[string]any{
		"name":     newName,
		"discount": percentOff("25"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Equal(t, discountID, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "25", jsonField(jsonObject(updated, "discount"), "percent_off"))
	assert.Equal(t, jsonField(created, "code"), jsonField(updated, "code"), "an omitted code is unchanged")
}

func TestOrderDiscounts_UpdateSwitchesDiscountType(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+jsonField(created, "id"), map[string]any{
		"discount": amountOff("40.00"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	discount := jsonObject(parseJSON(body), "discount")
	assert.Equal(t, "amount_off", jsonField(discount, "type"))
	assertDecimalEquals(t, 40.00, jsonField(jsonObject(discount, "amount_off"), "amount"), "amount_off.amount")
	assertNilField(t, discount, "percent_off")
}

func TestOrderDiscounts_UpdateRejectsInvalidDiscountType(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+jsonField(created, "id"),
		map[string]any{"discount": map[string]any{"type": "banana"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
}

// The legacy implementation only checked code uniqueness on create, so an update could
// collide two discounts onto one code.
func TestOrderDiscounts_UpdateDuplicateCodeConflict(t *testing.T) {
	t.Parallel()
	first := createOrderDiscount(t, map[string]any{})
	second := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+jsonField(second, "id"),
		map[string]any{"code": jsonField(first, "code")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "code")
}

func TestOrderDiscounts_UpdateNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+mustGenID(t, id.OrderDiscountIDPrefix),
		map[string]any{"name": uniqueName("e2e-ords-missing")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// OrderDiscount — Retrieve / Delete
// ──────────────────────────────────────────────

func TestOrderDiscounts_Retrieve(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})
	discountID := jsonField(created, "id")

	status, body, err := apiClient.GetListRaw(orderDiscountsPath+"/"+discountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, discountID, jsonField(got, "id"))
	assert.Equal(t, jsonField(created, "code"), jsonField(got, "code"))
}

func TestOrderDiscounts_DeleteRemovesDiscount(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})
	discountID := jsonField(created, "id")

	status, body, err := apiClient.Delete(orderDiscountsPath + "/" + discountID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertDeletedStub(t, body, discountID, "order_discount")

	getStatus, getBody, err := apiClient.GetListRaw(orderDiscountsPath+"/"+discountID, nil)
	require.NoError(t, err)
	requireStatus(t, 410, getStatus, getBody)
}

func TestOrderDiscounts_DeleteNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(orderDiscountsPath + "/" + mustGenID(t, id.OrderDiscountIDPrefix))
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestOrderDiscounts_RetrieveNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(orderDiscountsPath+"/"+mustGenID(t, id.OrderDiscountIDPrefix), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestOrderDiscounts_ListStatusFilter(t *testing.T) {
	t.Parallel()
	active := createOrderDiscount(t, map[string]any{})
	archived := createOrderDiscount(t, map[string]any{"status": "archived"})

	list, _, err := apiClient.GetList(orderDiscountsPath, url.Values{"statuses[]": {"archived"}, "q": {jsonField(archived, "code")}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, jsonField(archived, "id"), DataItemField(list.Data[0], "id"))

	list, _, err = apiClient.GetList(orderDiscountsPath, url.Values{"statuses[]": {"archived"}, "q": {jsonField(active, "code")}})
	require.NoError(t, err)
	assert.Empty(t, list.Data)
}

// An archived code can't be applied to a new order.
func TestOrderDiscounts_ArchivedCodeCannotBeApplied(t *testing.T) {
	t.Parallel()
	archived := createOrderDiscount(t, map[string]any{"status": "archived"})

	status, body, err := getCustomerPortalClient().Post(orderDiscountsPath+"/actions/find-by-code",
		map[string]any{"code": jsonField(archived, "code")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// preview.9 callers still read and write the fraction fields.
func TestOrderDiscounts_Preview9Shape(t *testing.T) {
	t.Parallel()
	p9 := p9Client()

	status, body, err := p9.Post(orderDiscountsPath, map[string]any{
		"name": uniqueName("e2e-ords-p9"), "code": uniqueName("E2EP9"), "discount_type": "percentage", "percentage": "0.05",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	discountID := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(orderDiscountsPath + "/" + discountID) })
	assert.Equal(t, "percentage", jsonField(got, "discount_type"))
	assertDecimalEquals(t, 0.05, jsonField(got, "percentage"), "percentage")
	require.NotNil(t, got["order_count"])
	assert.NotContains(t, got, "discount")

	status, body, err = apiClient.GetListRaw(orderDiscountsPath+"/"+discountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "5", jsonField(jsonObject(parseJSON(body), "discount"), "percent_off"), "forge reads the whole percent")

	status, body, err = p9.Patch(orderDiscountsPath+"/"+discountID, map[string]any{"discount_type": "amount", "amount": "12.50"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got = parseJSON(body)
	assert.Equal(t, "amount", jsonField(got, "discount_type"))
	assertDecimalEquals(t, 12.5, jsonField(got, "amount"), "amount")
}
