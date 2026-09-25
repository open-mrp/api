package shippo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/cache"
)

// countingShippo answers the carrier-account list and shipment rating, counting each.
type countingShippo struct {
	accountLists atomic.Int32
	shipments    atomic.Int32
	noRates      bool
}

func (s *countingShippo) handler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/carrier_accounts/":
		s.accountLists.Add(1)
		_, _ = w.Write([]byte(`{"results":[{"object_id":"byoa_ups","carrier":"ups","active":true},{"object_id":"shippo_ups","carrier":"ups","active":true,"is_shippo_account":true}]}`))
	case r.Method == http.MethodPost && r.URL.Path == "/shipments/":
		s.shipments.Add(1)
		w.WriteHeader(http.StatusCreated)
		if s.noRates {
			_, _ = w.Write([]byte(`{"rates":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"rates":[{"amount":"10.00","servicelevel":{"name":"Ground","token":"ups_ground"}},{"amount":"30.00","servicelevel":{"name":"Next Day","token":"ups_next_day_air"}}]}`))
	default:
		http.NotFound(w, r)
	}
}

// newCachingStubClient builds clients that share one in-memory cache, as a factory's clients do.
func newCachingStubClient(t *testing.T, shippo *countingShippo) func() *clientImpl {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(shippo.handler))
	t.Cleanup(server.Close)
	store, err := cache.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	caches, err := newRateCaches((&ClientFactoryConfig{Store: store}).WithDefaults())
	if err != nil {
		t.Fatalf("newRateCaches: %v", err)
	}
	return func() *clientImpl {
		return &clientImpl{apiKey: "test_key", httpClient: server.Client(), baseURL: server.URL, caches: caches}
	}
}

func rateParams() domain.FetchShippingRateParams {
	return domain.FetchShippingRateParams{
		CarrierAccountObjectID: "byoa_ups",
		ServiceLevelToken:      "ups_ground",
		FromAddress:            domain.ShippingAddress{Street1: "1 Mill Rd", City: "Hickory", State: "NC", Zip: "28601", Country: "US"},
		ToAddress:              domain.ShippingAddress{Name: "Receiving", Street1: "9 Main St", City: "Austin", State: "TX", Zip: "73301", Country: "US"},
		Parcels:                []domain.Parcel{{Weight: "4.5", Length: "23.5", Width: "13", Height: "9.5"}},
	}
}

// Each request builds its own client, so reuse has to come from the shared cache, not the client.
func TestFetchShippingRate_ReusesAQuoteAcrossClients(t *testing.T) {
	shippo := &countingShippo{}
	build := newCachingStubClient(t, shippo)

	for range 3 {
		rate, apiErr := build().FetchShippingRate(context.Background(), rateParams())
		if apiErr != nil {
			t.Fatalf("FetchShippingRate: %v", apiErr)
		}
		if rate != 11 {
			t.Fatalf("rate = %v, want the marked-up ground rate 11", rate)
		}
	}
	if got := shippo.shipments.Load(); got != 1 {
		t.Errorf("rated the shipment %d times, want 1", got)
	}
	if got := shippo.accountLists.Load(); got != 1 {
		t.Errorf("listed carrier accounts %d times, want 1", got)
	}
}

func TestFetchShippingRate_KeyIgnoresWhoItIsFor(t *testing.T) {
	shippo := &countingShippo{}
	build := newCachingStubClient(t, shippo)

	params := rateParams()
	if _, apiErr := build().FetchShippingRate(context.Background(), params); apiErr != nil {
		t.Fatalf("FetchShippingRate: %v", apiErr)
	}

	renamed := rateParams()
	renamed.ToAddress.Name = "Dock 4"
	renamed.ToAddress.City = "  AUSTIN "
	if _, apiErr := build().FetchShippingRate(context.Background(), renamed); apiErr != nil {
		t.Fatalf("FetchShippingRate: %v", apiErr)
	}
	if got := shippo.shipments.Load(); got != 1 {
		t.Errorf("a renamed recipient re-rated the shipment (%d calls)", got)
	}

	elsewhere := rateParams()
	elsewhere.ToAddress.Zip = "10001"
	if _, apiErr := build().FetchShippingRate(context.Background(), elsewhere); apiErr != nil {
		t.Fatalf("FetchShippingRate: %v", apiErr)
	}
	heavier := rateParams()
	heavier.Parcels[0].Weight = "9"
	if _, apiErr := build().FetchShippingRate(context.Background(), heavier); apiErr != nil {
		t.Fatalf("FetchShippingRate: %v", apiErr)
	}
	if got := shippo.shipments.Load(); got != 3 {
		t.Errorf("rated %d times, want a new quote each for a new destination and a new weight", got)
	}
}

func TestFetchShippingRate_CachedOnlyNeverCallsShippo(t *testing.T) {
	shippo := &countingShippo{}
	build := newCachingStubClient(t, shippo)

	cachedOnly := rateParams()
	cachedOnly.CachedOnly = true
	if _, apiErr := build().FetchShippingRate(context.Background(), cachedOnly); apiErr != domain.ErrShippingRateNotCached {
		t.Fatalf("cold cached-only lookup = %v, want ErrShippingRateNotCached", apiErr)
	}
	if shippo.shipments.Load()+shippo.accountLists.Load() != 0 {
		t.Fatal("a cached-only lookup called Shippo")
	}

	// A rate-shop for the same shipment warms what the order's freight estimate reads.
	options, apiErr := build().FetchAllShippingRates(context.Background(), domain.FetchAllShippingRatesParams{
		CarrierAccountObjectID: cachedOnly.CarrierAccountObjectID,
		FromAddress:            cachedOnly.FromAddress,
		ToAddress:              cachedOnly.ToAddress,
		Parcels:                cachedOnly.Parcels,
	})
	if apiErr != nil || len(options) != 2 {
		t.Fatalf("FetchAllShippingRates = (%v, %v)", options, apiErr)
	}

	rate, apiErr := build().FetchShippingRate(context.Background(), cachedOnly)
	if apiErr != nil || rate != 11 {
		t.Fatalf("warm cached-only lookup = (%v, %v), want 11", rate, apiErr)
	}
	if got := shippo.shipments.Load(); got != 1 {
		t.Errorf("rated %d times, want only the rate-shop's call", got)
	}
}

func TestFetchShippingRate_WithoutAStoreCallsShippoEveryTime(t *testing.T) {
	shippo := &countingShippo{}
	server := httptest.NewServer(http.HandlerFunc(shippo.handler))
	t.Cleanup(server.Close)

	for range 2 {
		client := &clientImpl{apiKey: "test_key", httpClient: server.Client(), baseURL: server.URL}
		if _, apiErr := client.FetchShippingRate(context.Background(), rateParams()); apiErr != nil {
			t.Fatalf("FetchShippingRate: %v", apiErr)
		}
	}
	if got := shippo.shipments.Load(); got != 2 {
		t.Errorf("rated %d times, want 2 with caching off", got)
	}
}
