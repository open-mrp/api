package invoiceep

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	pb "github.com/open-mrp/api/shared/proto/core"
)

type captureListClient struct {
	pb.CoreServiceClient
	list *pb.ListInvoicesRequest
}

func (c *captureListClient) ListInvoices(_ context.Context, in *pb.ListInvoicesRequest, _ ...grpc.CallOption) (*pb.ListInvoicesResponse, error) {
	c.list = in
	return &pb.ListInvoicesResponse{PageInfo: &pb.PageInfo{}}, nil
}

// listInvoices runs the real List Invoices endpoint against client, without its permission gate.
func listInvoices(t *testing.T, client *captureListClient, query url.Values) *httptest.ResponseRecorder {
	t.Helper()
	ep := apiendpoint.From(&ListInvoicesEndpoint{})
	ep.RequiredPermissions = nil
	ep.WithService(nil, NewInvoiceSvc(&InvoiceSvcConfig{CoreClient: client}))

	r := httptest.NewRequest(http.MethodGet, "/v1/finance/invoices?"+query.Encode(), nil)
	w := httptest.NewRecorder()
	ep.GetHandler()(w, r)
	return w
}

func TestListInvoices_NumbersReachCoreAsGiven(t *testing.T) {
	t.Parallel()

	client := &captureListClient{}
	w := listInvoices(t, client, url.Values{"numbers": {"0012345", "0012346"}, "customer_ids": {"ac_1", "ac_2"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, client.list)
	assert.Equal(t, []string{"0012345", "0012346"}, client.list.Numbers, "numbers are passed exactly, leading zeros kept")
	assert.Equal(t, []string{"ac_1", "ac_2"}, client.list.CustomerIds)
	assert.Nil(t, client.list.Query, "numbers are not turned into a search")
}

func TestListInvoices_OmittedNumbersSendNone(t *testing.T) {
	t.Parallel()

	client := &captureListClient{}
	w := listInvoices(t, client, url.Values{})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, client.list.Numbers)
}

func TestListInvoices_InvalidNumbersAreRejected(t *testing.T) {
	t.Parallel()

	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("%07d", i)
	}
	for name, numbers := range map[string][]string{
		"more than 100":     tooMany,
		"an empty number":   {"0012345", ""},
		"a number too long": {strings.Repeat("9", 256)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := &captureListClient{}
			w := listInvoices(t, client, url.Values{"numbers": numbers})
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Nil(t, client.list, "an invalid request must not reach core")
		})
	}
}
