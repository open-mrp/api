package shippo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
)

// Builds a client pointed at a local stub; no test ever reaches the real Shippo API.
func newStubClient(t *testing.T, handler http.HandlerFunc) *clientImpl {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &clientImpl{apiKey: "test_key", httpClient: server.Client(), baseURL: server.URL}
}

func TestCreateTransactionInstantLabel_MapsParcelTransactions(t *testing.T) {
	client := newStubClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/transactions":
			var req CreateTransactionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding transaction request: %v", err)
			}
			if req.CarrierAccount != "ca_1" || req.ServiceLevelToken != "ups_ground" {
				t.Errorf("unexpected carrier/service level: %+v", req)
			}
			if len(req.Shipment.Parcels) != 2 {
				t.Errorf("expected 2 parcels, got %d", len(req.Shipment.Parcels))
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"object_id":"txn_master","status":"SUCCESS","tracking_number":"1Z-MASTER","rate":{"object_id":"rate_1","amount":"24.75"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/transactions":
			if got := r.URL.Query().Get("rate"); got != "rate_1" {
				t.Errorf("expected the list to be filtered by the purchased rate, got %q", got)
			}
			// Shippo lists a rate's transactions newest-first.
			_, _ = w.Write([]byte(`{"results":[
				{"object_id":"txn_b","tracking_number":"1Z-B","label_url":"https://labels/b.png"},
				{"object_id":"txn_a","tracking_number":"1Z-A","label_url":"https://labels/a.png"}
			]}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	result, apiErr := client.CreateTransactionInstantLabel(context.Background(), domain.CreateLabelParams{
		CarrierAccountObjectID: "ca_1",
		ServiceLevelToken:      "ups_ground",
		Parcels: []domain.Parcel{
			{Weight: "10", Length: "23.5", Width: "13", Height: "9.5"},
			{Weight: "12", Length: "23.5", Width: "13", Height: "9.5"},
		},
	})
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if result.MasterTrackingNumber != "1Z-MASTER" {
		t.Errorf("master tracking = %q", result.MasterTrackingNumber)
	}
	if result.NegotiatedRate != 24.75 {
		t.Errorf("negotiated rate = %v", result.NegotiatedRate)
	}
	if len(result.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(result.Packages))
	}
	// Reversing the list aligns each package with the parcel it was bought for.
	if result.Packages[0].TrackingNumber != "1Z-A" || result.Packages[1].TrackingNumber != "1Z-B" {
		t.Errorf("packages are misaligned with their parcels: %+v", result.Packages)
	}
	if result.Packages[0].ShippoTransactionID != "txn_a" || result.Packages[0].LabelURL != "https://labels/a.png" {
		t.Errorf("package 0 = %+v", result.Packages[0])
	}
}

func TestCreateTransactionInstantLabel_FailedTransaction(t *testing.T) {
	client := newStubClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object_id":"txn","status":"ERROR","messages":[{"text":"Invalid postal code"}]}`))
	})

	_, apiErr := client.CreateTransactionInstantLabel(context.Background(), domain.CreateLabelParams{
		Parcels: []domain.Parcel{{Weight: "10"}},
	})
	if apiErr == nil {
		t.Fatal("a non-SUCCESS transaction must fail the purchase")
	}
}

func TestCreateTransactionInstantLabel_NoParcels(t *testing.T) {
	client := newStubClient(t, func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("no carrier call should be made: %s %s", r.Method, r.URL.Path)
	})

	if _, apiErr := client.CreateTransactionInstantLabel(context.Background(), domain.CreateLabelParams{}); apiErr == nil {
		t.Fatal("a purchase with no parcels must be rejected before the carrier call")
	}
}

func TestRefundTransaction(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"success", http.StatusCreated, `{"status":"QUEUED"}`, false},
		{"already refunded is idempotent", http.StatusBadRequest, `{"detail":"This transaction has already been refunded."}`, false},
		{"refund error", http.StatusCreated, `{"status":"ERROR"}`, true},
		{"other bad request", http.StatusBadRequest, `{"detail":"Unknown transaction."}`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newStubClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/refunds" {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})

			apiErr := client.RefundTransaction(context.Background(), "txn_1")
			if tc.wantErr && apiErr == nil {
				t.Error("expected an error")
			}
			if !tc.wantErr && apiErr != nil {
				t.Errorf("unexpected error: %v", apiErr)
			}
		})
	}
}

func TestParseTransactionRate(t *testing.T) {
	if id, amount := parseTransactionRate(json.RawMessage(`{"object_id":"rate_1","amount":"9.99"}`)); id != "rate_1" || amount != "9.99" {
		t.Errorf("embedded rate object: got (%q, %q)", id, amount)
	}
	if id, amount := parseTransactionRate(json.RawMessage(`"rate_2"`)); id != "rate_2" || amount != "" {
		t.Errorf("rate id string: got (%q, %q)", id, amount)
	}
	if id, _ := parseTransactionRate(nil); id != "" {
		t.Errorf("missing rate: got %q", id)
	}
}
