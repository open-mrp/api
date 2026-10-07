package invoiceep

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

// captureInvoiceClient records the UpdateInvoice request and answers with a bare invoice carrying
// replyMetadata.
type captureInvoiceClient struct {
	pb.CoreServiceClient
	replyMetadata map[string]string
	update        *pb.UpdateInvoiceRequest
}

func (c *captureInvoiceClient) UpdateInvoice(_ context.Context, in *pb.UpdateInvoiceRequest, _ ...grpc.CallOption) (*pb.UpdateInvoiceResponse, error) {
	c.update = in
	now := timestamppb.Now()
	return &pb.UpdateInvoiceResponse{Invoice: &pb.InvoiceInfo{
		Id: "iv_meta", Number: "INV-1", TotalInvoiced: "0", Metadata: c.replyMetadata, CreatedAt: now, UpdatedAt: now,
	}}, nil
}

// patchInvoice runs the real Update Invoice endpoint against client, without its permission gate.
func patchInvoice(t *testing.T, client *captureInvoiceClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	ep := apiendpoint.From(&UpdateInvoiceEndpoint{})
	ep.RequiredPermissions = nil
	ep.WithService(nil, NewInvoiceSvc(&InvoiceSvcConfig{CoreClient: client}))

	r := httptest.NewRequest(http.MethodPatch, "/v1/finance/invoices/iv_meta", strings.NewReader(body))
	r = r.WithContext(appctx.WithPathParams(r.Context(), map[string]string{"id": "iv_meta"}))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ep.GetHandler()(w, r)
	return w
}

func TestUpdateInvoice_MetadataPatch(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		body       string
		wantNil    bool
		wantClear  bool
		wantSet    map[string]string
		wantRemove []string
	}{
		{name: "omitted leaves metadata alone", body: `{"is_edi_sent": true}`, wantNil: true},
		{name: "null clears every key", body: `{"metadata": null}`, wantClear: true},
		{name: "empty object changes nothing", body: `{"metadata": {}}`, wantSet: map[string]string{}},
		{
			name:       "a string sets, null removes, an empty string is a value",
			body:       `{"metadata": {"edi_filename": "a.csv", "edi_sent_at": null, "blank": ""}}`,
			wantSet:    map[string]string{"edi_filename": "a.csv", "blank": ""},
			wantRemove: []string{"edi_sent_at"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &captureInvoiceClient{}
			w := patchInvoice(t, client, tc.body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, client.update)

			got := client.update.Metadata
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantClear, got.Clear)
			assert.Equal(t, len(tc.wantSet), len(got.Set), "set: %v", got.Set)
			for k, v := range tc.wantSet {
				assert.Equal(t, v, got.Set[k], "set[%s]", k)
			}
			assert.ElementsMatch(t, tc.wantRemove, got.Remove)
		})
	}
}

func TestUpdateInvoice_MetadataIsShownAsStored(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		stored map[string]string
		want   map[string]any
	}{
		"empty is an object, never null": {nil, map[string]any{}},
		"values come back unchanged":     {map[string]string{"edi_filename": "a.csv", "blank": ""}, map[string]any{"edi_filename": "a.csv", "blank": ""}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := patchInvoice(t, &captureInvoiceClient{replyMetadata: tc.stored}, `{"is_edi_sent": true}`)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var got map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			assert.Equal(t, tc.want, got["metadata"])
		})
	}
}

func TestUpdateInvoice_InvalidMetadataIsRejected(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"key over 40 characters":    fmt.Sprintf(`{"metadata": {%q: "v"}}`, strings.Repeat("k", metadata.MaxKeyLength+1)),
		"key with a bracket":        `{"metadata": {"a[0]": "v"}}`,
		"value over 500 characters": fmt.Sprintf(`{"metadata": {"k": %q}}`, strings.Repeat("v", metadata.MaxValueLength+1)),
		"a number value":            `{"metadata": {"k": 1}}`,
		"an array":                  `{"metadata": []}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := &captureInvoiceClient{}
			w := patchInvoice(t, client, body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Nil(t, client.update, "an invalid body must not reach core")
		})
	}
}
