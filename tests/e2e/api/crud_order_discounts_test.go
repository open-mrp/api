//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/augno/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const orderDiscountsPath = "/v1/sales/order-discounts"

// createOrderDiscount creates a percentage discount with a unique code and registers cleanup.
func createOrderDiscount(t *testing.T, overrides map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"name":          uniqueName("e2e-ords"),
		"code":          uniqueName("E2EORDS"),
		"discount_type": "percentage",
		"percentage":    "0.1",
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
		assert.Contains(t, []string{"percentage", "amount"}, jsonField(m, "discount_type"),
			"discount_type is a closed enum")
		assert.NotEmpty(t, jsonField(m, "percentage"))
		assert.NotEmpty(t, jsonField(m, "amount"))
		require.NotNil(t, m["order_count"], "order_count should always be present")
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
		"name":          name,
		"code":          code,
		"discount_type": "percentage",
		"percentage":    "0.15",
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
	assert.Equal(t, "percentage", jsonField(got, "discount_type"))
	// percentage is a multiplier, not a whole percent.
	assertDecimalEquals(t, 0.15, jsonField(got, "percentage"), "percentage")
	assert.EqualValues(t, 0, got["order_count"], "a new discount has not been applied to any order")
}

func TestOrderDiscounts_CreateAmountType(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{
		"discount_type": "amount",
		"amount":        "25.00",
		"percentage":    "0",
	})

	assert.Equal(t, "amount", jsonField(created, "discount_type"))
	assertDecimalEquals(t, 25.00, jsonField(created, "amount"), "amount")
}

// discount_type is declared as constants.OrderDiscountType, so the gateway rejects any
// value outside the enum before the request reaches core-service.
func TestOrderDiscounts_CreateRejectsInvalidDiscountType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(orderDiscountsPath, map[string]any{
		"name":          uniqueName("e2e-ords-badtype"),
		"code":          uniqueName("E2EBAD"),
		"discount_type": "banana",
		"percentage":    "0.1",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"an unknown discount_type should be rejected, got %d: %s", status, string(body))
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "discount_type")
}

func TestOrderDiscounts_CreateValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{"code": uniqueName("E2ENONAME"), "discount_type": "percentage"}},
		{"missing code", map[string]any{"name": uniqueName("e2e-ords-nocode"), "discount_type": "percentage"}},
		{"missing discount_type", map[string]any{"name": uniqueName("e2e-ords-notype"), "code": uniqueName("E2ENOTYPE")}},
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
			"name":          uniqueName("e2e-ords-dupe"),
			"code":          variant,
			"discount_type": "percentage",
			"percentage":    "0.1",
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
		"name":          uniqueName("e2e-ords-idem"),
		"code":          uniqueName("E2EIDEM"),
		"discount_type": "percentage",
		"percentage":    "0.1",
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
		"name":       newName,
		"percentage": "0.25",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Equal(t, discountID, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, newName, jsonField(updated, "name"))
	assertDecimalEquals(t, 0.25, jsonField(updated, "percentage"), "percentage")
	assert.Equal(t, jsonField(created, "code"), jsonField(updated, "code"), "an omitted code is unchanged")
}

func TestOrderDiscounts_UpdateSwitchesDiscountType(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+jsonField(created, "id"), map[string]any{
		"discount_type": "amount",
		"amount":        "40.00",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Equal(t, "amount", jsonField(updated, "discount_type"))
	assertDecimalEquals(t, 40.00, jsonField(updated, "amount"), "amount")
}

func TestOrderDiscounts_UpdateRejectsInvalidDiscountType(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})

	status, body, err := apiClient.Patch(orderDiscountsPath+"/"+jsonField(created, "id"),
		map[string]any{"discount_type": "banana"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"an unknown discount_type should be rejected on update too, got %d: %s", status, string(body))
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

	getStatus, getBody, err := apiClient.GetListRaw(orderDiscountsPath+"/"+discountID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, getStatus, getBody)
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
