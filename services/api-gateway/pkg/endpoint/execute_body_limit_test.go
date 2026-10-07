package apiendpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	apierror "github.com/open-mrp/api/shared/errors"
)

// nameBodyOfSize is a JSON body of exactly n bytes carrying one name.
func nameBodyOfSize(t *testing.T, n int) string {
	t.Helper()
	empty := `{"name":""}`
	body := `{"name":"` + strings.Repeat("n", n-len(empty)) + `"}`
	if len(body) != n {
		t.Fatalf("built a %d byte body, want %d", len(body), n)
	}
	return body
}

func executeNamePost(t *testing.T, extras APIEndpointExtras, body string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	ran := false
	ep := &APIEndpoint[*stubRequest, *stubResponse]{
		Method:            http.MethodPost,
		Route:             "/v1/things",
		SuccessStatusCode: http.StatusOK,
		Extras:            extras,
		ServiceHandler: func(svc any) func(context.Context, *stubRequest) (*stubResponse, *apierror.APIError) {
			return func(_ context.Context, req *stubRequest) (*stubResponse, *apierror.APIError) {
				ran = true
				return &stubResponse{ID: "th_1", Name: req.Name}, nil
			}
		},
	}
	bindHandler(ep)

	r := httptest.NewRequest(http.MethodPost, "/v1/things", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.ContentLength = int64(len(body))
	w := httptest.NewRecorder()
	ep.Execute(w, r)
	return w, ran
}

// A body over the cap used to be cut at the cap and handed to the decoder, which reported the
// truncation as malformed JSON. It is a size problem, and the client is told so.
func TestExecute_JSONBodyOverDefaultCap_is413(t *testing.T) {
	t.Parallel()

	w, ran := executeNamePost(t, APIEndpointExtras{}, nameBodyOfSize(t, httptransport.DefaultMaxJSONBodyBytes+1))

	if ran {
		t.Fatal("an oversized body must not reach the handler")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413: %s", w.Code, w.Body.String())
	}
	env := decodeErrEnvelope(t, w)
	if env.Error.Code != apierror.ErrorCodeRequestTooLarge {
		t.Fatalf("code=%q, want request_too_large", env.Error.Code)
	}
	if !strings.Contains(env.Error.Message, "1 MB") {
		t.Errorf("the message names the limit: %q", env.Error.Message)
	}
}

func TestExecute_JSONBodyExactlyAtDefaultCap_isAccepted(t *testing.T) {
	t.Parallel()

	w, ran := executeNamePost(t, APIEndpointExtras{}, nameBodyOfSize(t, httptransport.DefaultMaxJSONBodyBytes))

	if w.Code != http.StatusOK || !ran {
		t.Fatalf("a body exactly at the cap is whole, got %d: %.200s", w.Code, w.Body.String())
	}
}

func TestExecute_JSONBodyUnderARaisedCap_isAcceptedWhole(t *testing.T) {
	t.Parallel()

	size := 3 << 20
	w, ran := executeNamePost(t, APIEndpointExtras{MaxJSONBodyBytes: httptransport.MaxJSONBodyBytes}, nameBodyOfSize(t, size))

	if w.Code != http.StatusOK || !ran {
		t.Fatalf("got %d: %.200s", w.Code, w.Body.String())
	}
	var resp stubResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Name) != size-len(`{"name":""}`) {
		t.Errorf("the handler saw a %d byte name; the body was cut", len(resp.Name))
	}
}

func TestExecute_JSONBodyOverARaisedCap_is413(t *testing.T) {
	t.Parallel()

	w, ran := executeNamePost(t, APIEndpointExtras{MaxJSONBodyBytes: 2 << 20}, nameBodyOfSize(t, 2<<20+1))

	if ran || w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d (handler ran: %v): %.200s", w.Code, ran, w.Body.String())
	}
	if env := decodeErrEnvelope(t, w); !strings.Contains(env.Error.Message, "2 MB") {
		t.Errorf("the message names the endpoint's own limit: %q", env.Error.Message)
	}
}

type overCapEndpoint struct{}

func (overCapEndpoint) Materialize() *APIEndpoint[*stubRequest, *stubResponse] {
	return &APIEndpoint[*stubRequest, *stubResponse]{
		Method: http.MethodPost,
		Route:  "/v1/things",
		Extras: APIEndpointExtras{MaxJSONBodyBytes: httptransport.MaxJSONBodyBytes + 1},
	}
}

// The idempotency middleware buffers up to the gateway ceiling before any endpoint cap applies, so an
// endpoint declaring more would be refused upstream of its own limit. That is caught at startup.
func TestFrom_RefusesAJSONBodyCapOverTheCeiling(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for a cap over the ceiling")
		}
	}()
	From[*stubRequest, *stubResponse](overCapEndpoint{})
}
