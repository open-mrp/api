package salesorderep

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/metadata"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// captureSalesClient records the request each sales RPC receives and answers with a bare order or
// line carrying replyMetadata, so a test sees both what the gateway sent and how it presents the reply.
type captureSalesClient struct {
	pb.CoreSalesServiceClient
	replyMetadata map[string]string

	createOrder *pb.CreateSalesOrderRequest
	updateOrder *pb.UpdateSalesOrderRequest
	createLine  *pb.CreateSalesOrderLineRequest
	updateLine  *pb.UpdateSalesOrderLineRequest
}

func (c *captureSalesClient) order() *pb.SalesOrderInfo {
	now := timestamppb.Now()
	return &pb.SalesOrderInfo{Id: "or_meta", Number: "SO-1", Metadata: c.replyMetadata, CreatedAt: now, UpdatedAt: now}
}

func (c *captureSalesClient) line() *pb.SalesOrderLineInfo {
	now := timestamppb.Now()
	return &pb.SalesOrderLineInfo{Id: "orln_meta", LineItemNumber: 1, ProductSku: "SKU", Metadata: c.replyMetadata, CreatedAt: now, UpdatedAt: now}
}

func (c *captureSalesClient) CreateSalesOrder(_ context.Context, in *pb.CreateSalesOrderRequest, _ ...grpc.CallOption) (*pb.CreateSalesOrderResponse, error) {
	c.createOrder = in
	return &pb.CreateSalesOrderResponse{SalesOrder: c.order()}, nil
}

func (c *captureSalesClient) UpdateSalesOrder(_ context.Context, in *pb.UpdateSalesOrderRequest, _ ...grpc.CallOption) (*pb.UpdateSalesOrderResponse, error) {
	c.updateOrder = in
	return &pb.UpdateSalesOrderResponse{SalesOrder: c.order()}, nil
}

func (c *captureSalesClient) CreateSalesOrderLine(_ context.Context, in *pb.CreateSalesOrderLineRequest, _ ...grpc.CallOption) (*pb.CreateSalesOrderLineResponse, error) {
	c.createLine = in
	return &pb.CreateSalesOrderLineResponse{SalesOrderLine: c.line()}, nil
}

func (c *captureSalesClient) UpdateSalesOrderLine(_ context.Context, in *pb.UpdateSalesOrderLineRequest, _ ...grpc.CallOption) (*pb.UpdateSalesOrderLineResponse, error) {
	c.updateLine = in
	return &pb.UpdateSalesOrderLineResponse{SalesOrderLine: c.line()}, nil
}

// serveMetadata runs the real endpoint — decode, validation, service, presenter — against client. The
// permission gate is dropped: these tests are about the body, not who may send it.
func serveMetadata[TReq, TResp any, T interface {
	Materialize() *apiendpoint.APIEndpoint[TReq, TResp]
}](t *testing.T, source T, client *captureSalesClient, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	ep := apiendpoint.From(source)
	ep.RequiredPermissions = nil
	ep.WithService(nil, NewSalesOrderSvc(&SalesOrderSvcConfig{CoreClient: client}))

	r := httptest.NewRequest(ep.GetMethod(), path, strings.NewReader(body))
	r = r.WithContext(appctx.WithPathParams(r.Context(), pathParams(ep.GetRoute(), path)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ep.GetHandler()(w, r)
	return w
}

// pathParams reads route's {name} segments out of path, as the gateway router does.
func pathParams(route, path string) map[string]string {
	params := map[string]string{}
	routeParts, pathParts := strings.Split(route, "/"), strings.Split(path, "/")
	for i, part := range routeParts {
		if strings.HasPrefix(part, "{") && i < len(pathParts) {
			params[strings.Trim(part, "{}")] = pathParts[i]
		}
	}
	return params
}

func responseMetadata(t *testing.T, w *httptest.ResponseRecorder) any {
	t.Helper()
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got), w.Body.String())
	return got["metadata"]
}

func errorParam(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error struct {
			Param string `json:"param"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	return env.Error.Param
}

const createOrderBody = `{
	"buyer_account_id": "ac_buyer",
	"priority_code": "normal",
	"bill_to_address_id": "ad_bill",
	"ship_to_address_id": "ad_ship",
	"lines": [{"product_id": "pr_1", "quantity": {"value": "1", "unit_id": "un_ea"}%s}]%s
}`

func createOrderWith(orderMetadata, lineMetadata string) string {
	order, line := "", ""
	if orderMetadata != "" {
		order = `, "metadata": ` + orderMetadata
	}
	if lineMetadata != "" {
		line = `, "metadata": ` + lineMetadata
	}
	return fmt.Sprintf(createOrderBody, line, order)
}

func TestCreateSalesOrder_MetadataReachesCore(t *testing.T) {
	t.Parallel()

	t.Run("order and line maps are sent as given; a null value is dropped and an empty string kept", func(t *testing.T) {
		t.Parallel()
		client := &captureSalesClient{}
		w := serveMetadata(t, &CreateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders",
			createOrderWith(`{"edi_po": "81078093", "blank": "", "skip": null}`, `{"edi_line_item_id": "00010", "skip": null}`))
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		require.NotNil(t, client.createOrder)
		assert.Equal(t, map[string]string{"edi_po": "81078093", "blank": ""}, client.createOrder.Metadata)
		require.Len(t, client.createOrder.Lines, 1)
		assert.Equal(t, map[string]string{"edi_line_item_id": "00010"}, client.createOrder.Lines[0].Metadata)
	})

	t.Run("omitted and null metadata send nothing", func(t *testing.T) {
		t.Parallel()
		for _, body := range []string{createOrderWith("", ""), createOrderWith("null", "null")} {
			client := &captureSalesClient{}
			w := serveMetadata(t, &CreateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders", body)
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			assert.Empty(t, client.createOrder.Metadata)
			assert.Empty(t, client.createOrder.Lines[0].Metadata)
		}
	})

	t.Run("an order with no metadata is shown as an empty object, never null", func(t *testing.T) {
		t.Parallel()
		w := serveMetadata(t, &CreateSalesOrderEndpoint{}, &captureSalesClient{}, "/v1/sales/sales-orders", createOrderWith("", ""))
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.Equal(t, map[string]any{}, responseMetadata(t, w))
	})

	t.Run("an invalid line map is rejected before reaching core", func(t *testing.T) {
		t.Parallel()
		client := &captureSalesClient{}
		w := serveMetadata(t, &CreateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders",
			createOrderWith("", fmt.Sprintf(`{%q: "v"}`, strings.Repeat("k", metadata.MaxKeyLength+1))))
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		assert.Nil(t, client.createOrder)
	})
}

func TestUpdateSalesOrder_MetadataPatch(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body string
		want *pb.MetadataPatch
	}{
		{"omitted leaves metadata alone", `{"note": "x"}`, nil},
		{"null clears every key", `{"metadata": null}`, &pb.MetadataPatch{Clear: true}},
		{"empty object changes nothing", `{"metadata": {}}`, &pb.MetadataPatch{Set: map[string]string{}}},
		{"a string sets, null removes, an empty string is a value",
			`{"metadata": {"keep": "v", "drop": null, "blank": ""}}`,
			&pb.MetadataPatch{Set: map[string]string{"keep": "v", "blank": ""}, Remove: []string{"drop"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &captureSalesClient{}
			w := serveMetadata(t, &UpdateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders/or_meta", tc.body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, client.updateOrder)
			assertPatch(t, tc.want, client.updateOrder.Metadata)
		})
	}

	t.Run("the reply's metadata is shown as stored", func(t *testing.T) {
		t.Parallel()
		client := &captureSalesClient{replyMetadata: map[string]string{"edi_po": "81078093", "blank": ""}}
		w := serveMetadata(t, &UpdateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders/or_meta", `{"metadata": {"edi_po": "81078093"}}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, map[string]any{"edi_po": "81078093", "blank": ""}, responseMetadata(t, w))
	})
}

func TestCreateAndUpdateSalesOrderLine_Metadata(t *testing.T) {
	t.Parallel()

	t.Run("create sends the map, dropping null values", func(t *testing.T) {
		t.Parallel()
		client := &captureSalesClient{}
		w := serveMetadata(t, &CreateSalesOrderLineEndpoint{}, client, "/v1/sales/sales-orders/or_meta/lines",
			`{"product_id": "pr_1", "product_sku": "SKU", "quantity": {"value": "1", "unit_id": "un_ea"}, "metadata": {"edi_line_item_id": "00020", "skip": null, "blank": ""}}`)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.Equal(t, map[string]string{"edi_line_item_id": "00020", "blank": ""}, client.createLine.Metadata)
	})

	for _, tc := range []struct {
		name string
		body string
		want *pb.MetadataPatch
	}{
		{"update omitted", `{"product_sku": "SKU-2"}`, nil},
		{"update null clears", `{"metadata": null}`, &pb.MetadataPatch{Clear: true}},
		{"update merges", `{"metadata": {"a": "1", "b": null}}`, &pb.MetadataPatch{Set: map[string]string{"a": "1"}, Remove: []string{"b"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &captureSalesClient{}
			w := serveMetadata(t, &UpdateSalesOrderLineEndpoint{}, client, "/v1/sales/sales-orders/or_meta/lines/orln_meta", tc.body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assertPatch(t, tc.want, client.updateLine.Metadata)
		})
	}
}

// Every shape the edge rejects, on the update endpoint where the type is most permissive.
func TestUpdateSalesOrder_InvalidMetadataIsRejected(t *testing.T) {
	t.Parallel()

	tooMany := map[string]string{}
	for i := range metadata.MaxKeys + 1 {
		tooMany[fmt.Sprintf("k%d", i)] = "v"
	}
	tooManyJSON, _ := json.Marshal(map[string]any{"metadata": tooMany})

	for _, tc := range []struct {
		name string
		body string
	}{
		{"key over 40 characters", fmt.Sprintf(`{"metadata": {%q: "v"}}`, strings.Repeat("k", metadata.MaxKeyLength+1))},
		{"empty key", `{"metadata": {"": "v"}}`},
		{"key with an opening bracket", `{"metadata": {"a[b": "v"}}`},
		{"key with a closing bracket", `{"metadata": {"a]": "v"}}`},
		{"value over 500 characters", fmt.Sprintf(`{"metadata": {"k": %q}}`, strings.Repeat("v", metadata.MaxValueLength+1))},
		{"more than 50 keys in one request", string(tooManyJSON)},
		{"a number value", `{"metadata": {"k": 5}}`},
		{"a boolean value", `{"metadata": {"k": true}}`},
		{"a nested object value", `{"metadata": {"k": {"x": "y"}}}`},
		{"an array instead of an object", `{"metadata": ["k"]}`},
		{"a string instead of an object", `{"metadata": "k=v"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &captureSalesClient{}
			w := serveMetadata(t, &UpdateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders/or_meta", tc.body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Nil(t, client.updateOrder, "an invalid body must not reach core")
			assert.Contains(t, errorParam(t, w), "metadata", w.Body.String())
		})
	}

	t.Run("keys and values exactly at the limits are accepted", func(t *testing.T) {
		t.Parallel()
		atLimit := map[string]string{}
		for i := range metadata.MaxKeys - 1 {
			atLimit[fmt.Sprintf("k%d", i)] = "v"
		}
		atLimit[strings.Repeat("k", metadata.MaxKeyLength)] = strings.Repeat("v", metadata.MaxValueLength)
		body, _ := json.Marshal(map[string]any{"metadata": atLimit})

		client := &captureSalesClient{}
		w := serveMetadata(t, &UpdateSalesOrderEndpoint{}, client, "/v1/sales/sales-orders/or_meta", string(body))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Len(t, client.updateOrder.Metadata.Set, metadata.MaxKeys)
	})
}

func assertPatch(t *testing.T, want, got *pb.MetadataPatch) {
	t.Helper()
	if want == nil {
		assert.Nil(t, got)
		return
	}
	require.NotNil(t, got)
	assert.Equal(t, want.Clear, got.Clear, "clear")
	assert.Equal(t, len(want.Set), len(got.Set), "set: %v", got.Set)
	for k, v := range want.Set {
		assert.Equal(t, v, got.Set[k], "set[%s]", k)
	}
	assert.ElementsMatch(t, want.Remove, got.Remove, "remove")
}
