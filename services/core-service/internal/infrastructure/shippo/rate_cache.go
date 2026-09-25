package shippo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/cache"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	defaultRateTTL            = time.Hour
	defaultCarrierAccountsTTL = time.Hour
	// A shipment Shippo returned no rates for is often a carrier hiccup rather than an unserved lane,
	// so it is retried soon instead of quoting nothing for an hour.
	noRatesTTL = time.Minute
)

// ClientFactoryConfig configures the clients a ClientFactory builds.
type ClientFactoryConfig struct {
	// Store (optional; default: nil) shares carrier account lists and shipment rates across requests
	// and replicas. Nil disables caching: every quote calls Shippo.
	Store cache.Store

	// RateTTL (optional; default: 1h) is how long a shipment's rates are reused. Carrier rates move
	// with published tariffs and surcharges, not per request, and nothing stores a Shippo rate id
	// (labels are bought by service-level token), so a cached quote never goes stale into a purchase.
	RateTTL time.Duration

	// CarrierAccountsTTL (optional; default: 1h) is how long a Shippo token's carrier account list is reused.
	CarrierAccountsTTL time.Duration
}

func (c *ClientFactoryConfig) WithDefaults() *ClientFactoryConfig {
	if c == nil {
		c = &ClientFactoryConfig{}
	}
	out := *c
	if out.RateTTL == 0 {
		out.RateTTL = defaultRateTTL
	}
	if out.CarrierAccountsTTL == 0 {
		out.CarrierAccountsTTL = defaultCarrierAccountsTTL
	}
	return &out
}

func (c *ClientFactoryConfig) validate() error {
	if c.RateTTL < 0 || c.CarrierAccountsTTL < 0 {
		return fmt.Errorf("shippo: cache TTLs must not be negative")
	}
	return nil
}

// rateCaches are shared by every client the factory builds.
type rateCaches struct {
	rates    *cache.Cache[[]ShipmentRate]
	accounts *cache.Cache[[]CarrierAccount]
}

func newRateCaches(cfg *ClientFactoryConfig) (*rateCaches, error) {
	rates, err := cache.New[[]ShipmentRate](&cache.Config{Name: "shippo.shipment_rates", Store: cfg.Store, TTL: cfg.RateTTL},
		cache.WithTTLFunc(func(rates []ShipmentRate) time.Duration {
			if len(rates) == 0 {
				return noRatesTTL
			}
			return 0
		}))
	if err != nil {
		return nil, err
	}
	accounts, err := cache.New[[]CarrierAccount](&cache.Config{Name: "shippo.carrier_accounts", Store: cfg.Store, TTL: cfg.CarrierAccountsTTL})
	if err != nil {
		return nil, err
	}
	return &rateCaches{rates: rates, accounts: accounts}, nil
}

// tokenKey scopes entries to one Shippo token without putting the secret in the key.
func tokenKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:8])
}

// rateKey identifies a shipment by what Shippo rates it on. The recipient's name, phone and email
// are left out so the same lane quoted from an address typed into the order form and from the
// stored address it becomes share an entry.
func rateKey(apiKey, carrierAccountObjectID string, from, to domain.ShippingAddress, parcels []domain.Parcel, billing *domain.ShippingBilling) string {
	type parcelKey struct{ W, L, Wd, H string }
	ps := make([]parcelKey, len(parcels))
	for i, p := range parcels {
		ps[i] = parcelKey{normalizeShippoDecimal(p.Weight), normalizeShippoDecimal(p.Length), normalizeShippoDecimal(p.Width), normalizeShippoDecimal(p.Height)}
	}
	raw, _ := json.Marshal(struct {
		Account  string
		From, To []string
		Parcels  []parcelKey
		Billing  *domain.ShippingBilling
	}{carrierAccountObjectID, addressKey(from), addressKey(to), ps, billing})
	sum := sha256.Sum256(raw)
	return tokenKey(apiKey) + ":" + hex.EncodeToString(sum[:16])
}

func addressKey(a domain.ShippingAddress) []string {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	return []string{norm(a.Street1), norm(a.City), norm(a.State), norm(a.Zip), norm(a.Country)}
}

// shipmentRates returns the rates Shippo quotes for a shipment, from the cache when it holds them.
// With cachedOnly a miss returns domain.ErrShippingRateNotCached instead of calling Shippo.
func (c *clientImpl) shipmentRates(ctx context.Context, carrierAccountObjectID string, from, to domain.ShippingAddress, parcels []domain.Parcel, billing *domain.ShippingBilling, cachedOnly bool) ([]ShipmentRate, *apierror.APIError) {
	load := func(ctx context.Context) ([]ShipmentRate, *apierror.APIError) {
		shipment, apiErr := c.createShipmentForRates(ctx, carrierAccountObjectID, from, to, parcels, billing)
		if apiErr != nil {
			return nil, apiErr
		}
		return shipment.Rates, nil
	}
	if c.caches == nil {
		if cachedOnly {
			return nil, domain.ErrShippingRateNotCached
		}
		return load(ctx)
	}

	key := cache.Key{ID: rateKey(c.apiKey, carrierAccountObjectID, from, to, parcels, billing)}
	if cachedOnly {
		rates, ok := c.caches.rates.Peek(ctx, key)
		if !ok {
			return nil, domain.ErrShippingRateNotCached
		}
		return rates, nil
	}
	return c.caches.rates.GetOrLoad(ctx, key, load)
}
