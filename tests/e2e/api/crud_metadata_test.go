//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Metadata on sales orders, their lines and invoices: a client-owned string map that OpenMRP stores and
// returns unchanged. On update, keys left out are kept, a key sent as null is removed, `metadata: null`
// removes every key, and an empty string is a value like any other.

// --- Helpers ---

// metadataOf reads a resource's metadata, failing when it is missing or not an object: it is always
// present, and `{}` when nothing is set.
func metadataOf(t *testing.T, resource map[string]any) map[string]any {
	t.Helper()
	raw, ok := resource["metadata"]
	require.True(t, ok, "metadata is always present: %v", resource)
	m, ok := raw.(map[string]any)
	require.True(t, ok, "metadata is an object, never null: %#v", raw)
	return m
}

func patchOK(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	status, b, err := apiClient.Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, b)
	return parseJSON(b)
}

func patchStatus(t *testing.T, path string, body map[string]any) (int, map[string]any) {
	t.Helper()
	status, b, err := apiClient.Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, parseJSON(b)
}

func errParam(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	p, _ := e["param"].(string)
	return p
}

func orderMetadata(t *testing.T, orderID string) map[string]any {
	t.Helper()
	return metadataOf(t, getSalesOrder(t, orderID, url.Values{}))
}

// lineMetadata reads a line's metadata from the order, the only place a line is read.
func lineMetadata(t *testing.T, orderID, lineID string) map[string]any {
	t.Helper()
	for _, line := range orderLines(t, orderID) {
		if line["id"] == lineID {
			return metadataOf(t, line)
		}
	}
	t.Fatalf("line %s not on order %s", lineID, orderID)
	return nil
}

func orderLines(t *testing.T, orderID string) []map[string]any {
	t.Helper()
	got := getSalesOrder(t, orderID, url.Values{"include": {"lines"}})
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")
	var out []map[string]any
	for _, l := range lines["data"].([]any) {
		out = append(out, l.(map[string]any))
	}
	return out
}

func createOrderWithMetadata(t *testing.T, orderMetadata, lineMetadata map[string]any) map[string]any {
	t.Helper()
	body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	if orderMetadata != nil {
		body["metadata"] = orderMetadata
	}
	if lineMetadata != nil {
		body["lines"].([]map[string]any)[0]["metadata"] = lineMetadata
	}
	return createAndCleanup(t, salesOrdersPath, body)
}

// --- Create ---

func TestSalesOrderMetadata_CreateStoresOrderAndLineMaps(t *testing.T) {
	t.Parallel()

	created := createOrderWithMetadata(t,
		map[string]any{"edi_po": "81078093", "edi_partner": "acme", "blank": "", "skip": nil},
		map[string]any{"edi_line_item_id": "00010", "skip": nil},
	)
	orderID := jsonField(created, "id")

	want := map[string]any{"edi_po": "81078093", "edi_partner": "acme", "blank": ""}
	assert.Equal(t, want, metadataOf(t, created), "create response: a null value is not stored, an empty string is")
	assert.Equal(t, want, orderMetadata(t, orderID), "retrieve")

	saleLineID := orderSaleLineID(t, orderID)
	for _, line := range orderLines(t, orderID) {
		if line["id"] == saleLineID {
			assert.Equal(t, map[string]any{"edi_line_item_id": "00010"}, metadataOf(t, line))
		} else {
			assert.Equal(t, map[string]any{}, metadataOf(t, line), "a line the order generates for itself has none")
		}
	}
}

func TestSalesOrderMetadata_OmittedOrNullOnCreateIsEmpty(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]any{"omitted": nil, "null": "null", "empty object": map[string]any{}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
			switch value {
			case nil:
			case "null":
				body["metadata"] = nil
				body["lines"].([]map[string]any)[0]["metadata"] = nil
			default:
				body["metadata"] = value
			}
			created := createAndCleanup(t, salesOrdersPath, body)
			orderID := jsonField(created, "id")

			assert.Equal(t, map[string]any{}, metadataOf(t, created))
			assert.Equal(t, map[string]any{}, orderMetadata(t, orderID))
			for _, line := range orderLines(t, orderID) {
				assert.Equal(t, map[string]any{}, metadataOf(t, line))
			}
		})
	}
}

func TestSalesOrderMetadata_ListShowsTheSameMetadata(t *testing.T) {
	t.Parallel()

	po := uniqueName("PO-META")
	body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	body["customer_purchase_order_number"] = po
	body["metadata"] = map[string]any{"edi_po": po}
	created := createAndCleanup(t, salesOrdersPath, body)

	// The list reads through its own projection, so it is checked separately from retrieve.
	raw := listFindByField(t, salesOrdersPath, url.Values{}, "id", jsonField(created, "id"))
	require.NotNil(t, raw, "order on the list")
	var row map[string]any
	require.NoError(t, json.Unmarshal(raw, &row))
	assert.Equal(t, map[string]any{"edi_po": po}, metadataOf(t, row))
}

// --- Update ---

// Each step is checked on the PATCH response and again on a fresh read, so a value that only appears
// in the response, or a write that never reached the row, both fail.
func TestSalesOrderMetadata_UpdateSemantics(t *testing.T) {
	t.Parallel()

	created := createOrderWithMetadata(t, map[string]any{"a": "1", "b": "2", "c": "3"}, nil)
	orderID := jsonField(created, "id")
	path := salesOrdersPath + "/" + orderID

	step := func(name string, patch map[string]any, want map[string]any) {
		t.Helper()
		got := patchOK(t, path, patch)
		assert.Equal(t, want, metadataOf(t, got), "%s: response", name)
		assert.Equal(t, want, orderMetadata(t, orderID), "%s: retrieve", name)
	}

	step("another field alone leaves metadata alone",
		map[string]any{"note": "unrelated"}, map[string]any{"a": "1", "b": "2", "c": "3"})
	step("an empty object changes nothing",
		map[string]any{"metadata": map[string]any{}}, map[string]any{"a": "1", "b": "2", "c": "3"})
	step("a new key is added and the rest kept",
		map[string]any{"metadata": map[string]any{"d": "4"}}, map[string]any{"a": "1", "b": "2", "c": "3", "d": "4"})
	step("an existing key is overwritten",
		map[string]any{"metadata": map[string]any{"a": "one"}}, map[string]any{"a": "one", "b": "2", "c": "3", "d": "4"})
	step("null removes a key",
		map[string]any{"metadata": map[string]any{"b": nil}}, map[string]any{"a": "one", "c": "3", "d": "4"})
	step("an empty string is stored, not a removal",
		map[string]any{"metadata": map[string]any{"c": ""}}, map[string]any{"a": "one", "c": "", "d": "4"})
	step("null on a key that is not there is a no-op",
		map[string]any{"metadata": map[string]any{"never_set": nil}}, map[string]any{"a": "one", "c": "", "d": "4"})
	step("set, remove and blank in one request",
		map[string]any{"metadata": map[string]any{"a": nil, "d": "", "e": "5"}}, map[string]any{"c": "", "d": "", "e": "5"})
	step("metadata: null removes every key",
		map[string]any{"metadata": nil}, map[string]any{})
	step("clearing what is already empty is a no-op",
		map[string]any{"metadata": nil}, map[string]any{})
	step("keys can be set again after a clear",
		map[string]any{"metadata": map[string]any{"x": "1"}}, map[string]any{"x": "1"})
	step("metadata and another field change together",
		map[string]any{"note": "both", "metadata": map[string]any{"y": "2"}}, map[string]any{"x": "1", "y": "2"})

	assert.Equal(t, "both", getSalesOrder(t, orderID, url.Values{})["note"])
}

func TestSalesOrderMetadata_ValuesRoundTripExactly(t *testing.T) {
	t.Parallel()

	created := createOrderWithMetadata(t, nil, nil)
	orderID := jsonField(created, "id")

	values := map[string]any{
		"json_looking": `{"not":"parsed"}`,
		"quotes":       `say "hi"`,
		"backslash":    `C:\edi\in`,
		"newline":      "line1\nline2",
		"unicode":      "Grüße 東京 🚚",
		"number_text":  "00010",
		"null_text":    "null",
		"spaces":       "  padded  ",
		"a.b":          "dotted key",
		"$path":        "dollar key",
		"with space":   "spaced key",
		"emoji_🚚":      "emoji key",
		"max_runes":    strings.Repeat("é", 500),
	}
	got := patchOK(t, salesOrdersPath+"/"+orderID, map[string]any{"metadata": values})
	assert.Equal(t, values, metadataOf(t, got))
	assert.Equal(t, values, orderMetadata(t, orderID), "every value is stored as the exact string sent")

	// A key with a dot or a $ is a plain key, not a JSON path: removing it removes only that key.
	got = patchOK(t, salesOrdersPath+"/"+orderID, map[string]any{"metadata": map[string]any{"a.b": nil, "$path": nil}})
	m := metadataOf(t, got)
	assert.NotContains(t, m, "a.b")
	assert.NotContains(t, m, "$path")
	assert.Len(t, m, len(values)-2)
}

// --- Limits and validation ---

func TestSalesOrderMetadata_FiftyKeyLimitAppliesToTheMergedResult(t *testing.T) {
	t.Parallel()

	created := createOrderWithMetadata(t, nil, nil)
	orderID := jsonField(created, "id")
	path := salesOrdersPath + "/" + orderID

	fifty := map[string]any{}
	for i := range 50 {
		fifty[fmt.Sprintf("k%02d", i)] = "v"
	}
	assert.Len(t, metadataOf(t, patchOK(t, path, map[string]any{"metadata": fifty})), 50, "exactly 50 is allowed")

	// One more key in a request that is itself valid pushes the object over the limit; nothing is written.
	status, body := patchStatus(t, path, map[string]any{"note": "must not stick", "metadata": map[string]any{"k50": "v"}})
	assert.Equal(t, 400, status, "%v", body)
	assert.Equal(t, "metadata", errParam(body))
	assert.Len(t, orderMetadata(t, orderID), 50, "the rejected update is rolled back")
	assert.NotEqual(t, "must not stick", getSalesOrder(t, orderID, url.Values{})["note"], "the whole update is rolled back")

	// Removing a key in the same request makes room for a new one.
	got := patchOK(t, path, map[string]any{"metadata": map[string]any{"k00": nil, "k50": "v"}})
	m := metadataOf(t, got)
	assert.Len(t, m, 50)
	assert.Contains(t, m, "k50")
	assert.NotContains(t, m, "k00")
}

func TestSalesOrderMetadata_InvalidMapsAreRejected(t *testing.T) {
	t.Parallel()

	created := createOrderWithMetadata(t, map[string]any{"keep": "me"}, nil)
	orderID := jsonField(created, "id")
	path := salesOrdersPath + "/" + orderID

	fiftyOne := map[string]any{}
	for i := range 51 {
		fiftyOne[fmt.Sprintf("k%02d", i)] = "v"
	}
	longKey := strings.Repeat("k", 41)

	for name, tc := range map[string]struct {
		metadata any
		param    string
	}{
		"key over 40 characters":    {map[string]any{longKey: "v"}, "metadata[" + longKey + "]"},
		"empty key":                 {map[string]any{"": "v"}, "metadata[]"},
		"key with [":                {map[string]any{"a[0": "v"}, "metadata[a[0]"},
		"key with ]":                {map[string]any{"a]": "v"}, "metadata[a]]"},
		"value over 500 characters": {map[string]any{"k": strings.Repeat("v", 501)}, "metadata[k]"},
		"51 keys in one request":    {fiftyOne, "metadata"},
		"number value":              {map[string]any{"k": 5}, ""},
		"boolean value":             {map[string]any{"k": true}, ""},
		"object value":              {map[string]any{"k": map[string]any{"x": "y"}}, ""},
		"array value":               {map[string]any{"k": []any{"x"}}, ""},
		"array instead of a map":    {[]any{"k"}, ""},
		"string instead of a map":   {"k=v", ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body := patchStatus(t, path, map[string]any{"metadata": tc.metadata})
			require.Equal(t, 400, status, "%v", body)
			if tc.param != "" {
				assert.Equal(t, tc.param, errParam(body))
			}
		})
	}

	assert.Equal(t, map[string]any{"keep": "me"}, orderMetadata(t, orderID), "no rejected update wrote anything")
}

func TestSalesOrderMetadata_InvalidMapOnCreateIsRejected(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(map[string]any){
		"order key too long": func(b map[string]any) { b["metadata"] = map[string]any{strings.Repeat("k", 41): "v"} },
		"line value too long": func(b map[string]any) {
			b["lines"].([]map[string]any)[0]["metadata"] = map[string]any{"k": strings.Repeat("v", 501)}
		},
		"order value not a string": func(b map[string]any) { b["metadata"] = map[string]any{"k": 1} },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
			mutate(body)
			status, b, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				_, _, _ = apiClient.Delete(salesOrdersPath + "/" + jsonField(parseJSON(b), "id"))
			}
			assert.Equal(t, 400, status, string(b))
		})
	}
}

func TestSalesOrderMetadata_IdempotentReplayReturnsTheSameResult(t *testing.T) {
	t.Parallel()

	created := createOrderWithMetadata(t, map[string]any{"a": "1", "b": "2"}, nil)
	path := salesOrdersPath + "/" + jsonField(created, "id")

	key := newIdempotencyKey()
	patch := map[string]any{"metadata": map[string]any{"b": nil, "c": "3"}}
	s1, b1, err := apiClient.Patch(path, patch, key)
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)
	s2, b2, err := apiClient.Patch(path, patch, key)
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)

	want := map[string]any{"a": "1", "c": "3"}
	assert.Equal(t, want, metadataOf(t, parseJSON(b1)))
	assert.Equal(t, want, metadataOf(t, parseJSON(b2)))
}

// --- Lines ---

func TestSalesOrderLineMetadata_CreateAndUpdateSemantics(t *testing.T) {
	t.Parallel()

	orderID := createLifecycleOrder(t)
	linesPath := salesOrdersPath + "/" + orderID + "/lines"

	created := createAndCleanupLine(t, linesPath, map[string]any{
		"product_id":  SeedProductID,
		"product_sku": uniqueName("SKU-META"),
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
		"metadata":    map[string]any{"edi_line_item_id": "00020", "blank": "", "skip": nil},
	})
	lineID := jsonField(created, "id")
	linePath := linesPath + "/" + lineID

	want := map[string]any{"edi_line_item_id": "00020", "blank": ""}
	assert.Equal(t, want, metadataOf(t, created))
	assert.Equal(t, want, lineMetadata(t, orderID, lineID))

	step := func(name string, patch map[string]any, want map[string]any) {
		t.Helper()
		got := patchOK(t, linePath, patch)
		assert.Equal(t, want, metadataOf(t, got), "%s: response", name)
		assert.Equal(t, want, lineMetadata(t, orderID, lineID), "%s: retrieve", name)
	}
	step("another field alone leaves metadata alone",
		map[string]any{"product_description": "changed"}, want)
	step("merge", map[string]any{"metadata": map[string]any{"blank": nil, "po_line": "1"}},
		map[string]any{"edi_line_item_id": "00020", "po_line": "1"})
	step("empty string", map[string]any{"metadata": map[string]any{"po_line": ""}},
		map[string]any{"edi_line_item_id": "00020", "po_line": ""})
	step("null clears", map[string]any{"metadata": nil}, map[string]any{})

	// Other lines on the order are untouched by this line's writes.
	for _, line := range orderLines(t, orderID) {
		if line["id"] != lineID {
			assert.Equal(t, map[string]any{}, metadataOf(t, line))
		}
	}

	status, body := patchStatus(t, linePath, map[string]any{"metadata": map[string]any{"a]": "v"}})
	assert.Equal(t, 400, status, "%v", body)
	assert.Equal(t, "metadata[a]]", errParam(body))
}

func createAndCleanupLine(t *testing.T, linesPath string, body map[string]any) map[string]any {
	t.Helper()
	status, b, err := apiClient.Post(linesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, b)
	created := parseJSON(b)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(linesPath + "/" + jsonField(created, "id")) })
	return created
}

// --- Invoices, and the order line as other records show it ---

// shipOrderWithLineMetadata ships a one-line order whose line carries lineMetadata and returns the
// shipment, the invoice it raised and the order.
func shipOrderWithLineMetadata(t *testing.T, lineMetadata map[string]any) (shipmentID, invoiceID, orderID string) {
	t.Helper()
	sale := shipOrder(t, setupOrderCustomer(t), []map[string]any{{
		"product_id": SeedProductID,
		"quantity":   map[string]any{"value": "1", "unit_id": SeedUnitID},
		"metadata":   lineMetadata,
	}})

	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+sale.shipmentID, url.Values{"include": {"related.invoice"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	invoiceID = jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "invoice"), "id")
	require.NotEmpty(t, invoiceID)

	status, body, err = apiClient.GetListRaw(financeInvoicesPath+"/"+invoiceID, url.Values{"include": {"order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	orderID = jsonField(jsonObject(parseJSON(body), "order"), "id")
	require.NotEmpty(t, orderID)
	return sale.shipmentID, invoiceID, orderID
}

func TestInvoiceMetadata_UpdateSemantics(t *testing.T) {
	t.Parallel()

	_, invoiceID, _ := shipOrderWithLineMetadata(t, nil)
	path := financeInvoicesPath + "/" + invoiceID

	assert.Equal(t, map[string]any{}, metadataOf(t, getInvoice(t, invoiceID)), "a new invoice has none")

	step := func(name string, patch map[string]any, want map[string]any) {
		t.Helper()
		got := patchOK(t, path, patch)
		assert.Equal(t, want, metadataOf(t, got), "%s: response", name)
		assert.Equal(t, want, metadataOf(t, getInvoice(t, invoiceID)), "%s: retrieve", name)
	}
	step("set", map[string]any{"metadata": map[string]any{"edi_filename": "INV.csv", "edi_sent_at": "2026-10-06T12:00:00Z"}},
		map[string]any{"edi_filename": "INV.csv", "edi_sent_at": "2026-10-06T12:00:00Z"})
	step("another field alone leaves metadata alone", map[string]any{"note": "x"},
		map[string]any{"edi_filename": "INV.csv", "edi_sent_at": "2026-10-06T12:00:00Z"})
	step("empty object changes nothing", map[string]any{"metadata": map[string]any{}},
		map[string]any{"edi_filename": "INV.csv", "edi_sent_at": "2026-10-06T12:00:00Z"})
	step("null removes a key, an empty string is kept", map[string]any{"metadata": map[string]any{"edi_sent_at": nil, "edi_filename": ""}},
		map[string]any{"edi_filename": ""})
	step("null clears", map[string]any{"metadata": nil}, map[string]any{})

	status, body := patchStatus(t, path, map[string]any{"metadata": map[string]any{"k": strings.Repeat("v", 501)}})
	assert.Equal(t, 400, status, "%v", body)
	assert.Equal(t, "metadata[k]", errParam(body))
}

// The order line's metadata is read live wherever the line is shown, so the EDI line number set on the
// order is on the invoice's lines without another call — and a later change to it shows there too.
func TestOrderLineMetadata_ShowsOnInvoiceShipmentAndPickLines(t *testing.T) {
	t.Parallel()

	shipmentID, invoiceID, orderID := shipOrderWithLineMetadata(t, map[string]any{"edi_line_item_id": "00010"})

	saleLineID := orderSaleLineID(t, orderID)

	// orderLineMetadataOn maps each order line shown on the record's lines to its metadata.
	orderLineMetadataOn := func(path, include, lineKey string) map[string]map[string]any {
		t.Helper()
		status, body, err := apiClient.GetListRaw(path, url.Values{"include": {include}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		lines := jsonObject(parseJSON(body), "lines")
		require.NotNil(t, lines, "%s: lines included", path)
		out := map[string]map[string]any{}
		for _, l := range lines["data"].([]any) {
			orderLine, ok := l.(map[string]any)[lineKey].(map[string]any)
			require.True(t, ok, "%s: %s included", path, lineKey)
			out[orderLine["id"].(string)] = metadataOf(t, orderLine)
		}
		require.Contains(t, out, saleLineID, "%s shows the sale line", path)
		return out
	}

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"related.pick"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	pickID := jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	require.NotEmpty(t, pickID)

	// The invoice also bills the order's own freight line, which carries no metadata.
	check := func(want map[string]any) {
		t.Helper()
		for name, byLine := range map[string]map[string]map[string]any{
			"invoice line":  orderLineMetadataOn(financeInvoicesPath+"/"+invoiceID, "lines.order_line", "order_line"),
			"shipment line": orderLineMetadataOn(shipmentsPath+"/"+shipmentID, "lines.sales_order_line", "sales_order_line"),
			"pick line":     orderLineMetadataOn(picksPath+"/"+pickID, "lines.sales_order_line", "sales_order_line"),
		} {
			for lineID, m := range byLine {
				if lineID == saleLineID {
					assert.Equal(t, want, m, "%s for the sale line", name)
				} else {
					assert.Equal(t, map[string]any{}, m, "%s for line %s", name, lineID)
				}
			}
		}
	}
	check(map[string]any{"edi_line_item_id": "00010"})

	patchOK(t, salesOrdersPath+"/"+orderID+"/lines/"+saleLineID, map[string]any{"metadata": map[string]any{"po_line": "1"}})
	check(map[string]any{"edi_line_item_id": "00010", "po_line": "1"})
}
