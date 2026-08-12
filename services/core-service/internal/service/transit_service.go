package service

import (
	"context"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var transitSvcTracer = tracing.GetTracer("core-service.transit_service")

// transitEstimateTTL is how long a harvested lane estimate is treated as current before a warm re-quotes it. Carrier transit for a lane changes when a carrier reworks its network, which is a matter of years, so this is a hedge against drift rather than a real expiry — it exists mostly so a lane quoted once during a trial does not stay authoritative forever.
const transitEstimateTTL = 30 * 24 * time.Hour

// normalizePostal reduces a postal code to the form the lane key is stored in, so a warm and a read of the same lane always agree.
//
// US ZIP+4 collapses to the five-digit base: the last four digits address a building, not a route, and keeping them would shard one lane into a row per customer site while quoting identical transit for each.
func normalizePostal(country, postal string) string {
	p := strings.ToUpper(strings.Join(strings.Fields(postal), ""))
	if strings.EqualFold(country, "US") {
		if base, _, found := strings.Cut(p, "-"); found {
			return base
		}
	}
	return p
}

// buildTransitLane projects an order's service level and its two endpoints into a lane key. The zero lane is returned when any part is missing, which IsComplete reports and every caller treats as "transit is unknown" rather than as a failure.
func buildTransitLane(serviceLevelID string, origin, dest domain.ShippingAddress) domain.TransitLane {
	return domain.TransitLane{
		CarrierOptionID: serviceLevelID,
		OriginCountry:   strings.ToUpper(strings.TrimSpace(origin.Country)),
		OriginPostal:    normalizePostal(origin.Country, origin.Zip),
		DestCountry:     strings.ToUpper(strings.TrimSpace(dest.Country)),
		DestPostal:      normalizePostal(dest.Country, dest.Zip),
	}
}

// selectTransit picks which of the two candidates an order commits against.
//
// The cached lane wins whenever it exists, stale or not. Staleness governs whether a warm should re-quote, not whether a reader may trust the answer: a months-old estimate for this exact lane is still a measurement of this journey, where the service-level default is a single number standing in for every lane the account ships. Preferring the default on age would trade a specific answer for a general one in the name of freshness.
func selectTransit(candidates *domain.CarrierTransitCandidates) *scheduling.Transit {
	if candidates == nil {
		return nil
	}
	if candidates.LaneDays != nil {
		return &scheduling.Transit{Days: *candidates.LaneDays, Source: string(constants.TransitSourceCarrierLane)}
	}
	if candidates.ServiceLevelDefaultDays != nil {
		return &scheduling.Transit{Days: *candidates.ServiceLevelDefaultDays, Source: string(constants.TransitSourceServiceLevel)}
	}
	return nil
}

// resolveOrderTransit works out how long the carrier needs for an order's lane, or nil when that cannot be known.
//
// Every miss along the way is a nil rather than an error. An order with no service level, an account with no ship-from, a lane nobody has quoted and a service level with no default are all ordinary states, and the commitment they produce — ship-by equal to the promised date — is exactly the behaviour that predates transit.
func (s *salesOrderSvcImpl) resolveOrderTransit(ctx context.Context, accountID string, order *domain.SalesOrder) (*scheduling.Transit, *apierror.APIError) {
	ctx, span := transitSvcTracer.Start(ctx, "service.transit.resolve_order_transit")
	defer span.End()

	if order.ServiceLevelID == nil || *order.ServiceLevelID == "" {
		return nil, nil
	}

	origin, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if origin == nil {
		return nil, nil
	}

	dest, apiErr := s.resolveOrderAddress(ctx, s.repos.NewAddressRepo(), accountID, order.BuyerAccountID, order.ShippingAddressID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lane := buildTransitLane(*order.ServiceLevelID, *origin, dest)
	if !lane.IsComplete() {
		return nil, nil
	}

	candidates, apiErr := s.repos.NewCarrierTransitEstimateRepo().Resolve(ctx, accountID, lane)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return selectTransit(candidates), nil
}

// WarmForOrder quotes an order's lane and records the transit the carrier reports for every service level it returns.
//
// This runs off the back of an order event rather than inside the create or issue path on purpose. Rating a shipment is a call to Shippo with a fifteen-second timeout, and order create is the request users feel most; the whole point of caching transit is that the slow part happens once, out of band, and the issue that depends on it does a single indexed read.
//
// One quote fills every service level on the lane, not just the one the order chose. The carrier returns them all anyway, and an account that later switches an order from ground to two-day finds that lane already warm.
func (s *salesOrderSvcImpl) WarmForOrder(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := transitSvcTracer.Start(ctx, "service.transit.warm_for_order")
	defer span.End()

	order, apiErr := s.repos.NewSalesOrderRepo().Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// A lane needs a carrier to quote and a service level to key on. Orders routinely have neither at create; the shipping-updated event brings them back here once one is chosen.
	if order.CarrierID == nil || *order.CarrierID == "" || order.ServiceLevelID == nil || *order.ServiceLevelID == "" {
		return nil
	}

	origin, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if origin == nil {
		return nil
	}

	dest, apiErr := s.resolveOrderAddress(ctx, s.repos.NewAddressRepo(), accountID, order.BuyerAccountID, order.ShippingAddressID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	lane := buildTransitLane(*order.ServiceLevelID, *origin, dest)
	if !lane.IsComplete() {
		return nil
	}

	// Skip the carrier call when this lane was quoted recently. Warming is driven by order events, so a busy customer would otherwise re-rate the same lane on every order.
	candidates, apiErr := s.repos.NewCarrierTransitEstimateRepo().Resolve(ctx, accountID, lane)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if candidates != nil && candidates.LaneRefreshedAt != nil && time.Since(*candidates.LaneRefreshedAt) < transitEstimateTTL {
		return nil
	}

	rates, apiErr := s.fetchLaneRates(ctx, accountID, order, *origin, dest)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return tracing.Trace(span, s.recordLaneRates(ctx, accountID, lane, rates))
}

// fetchLaneRates rates the order's parcel with its carrier and returns every option the carrier offers, each carrying its own transit estimate. Returns nil when the account cannot be rated at all, which is a no-op rather than a failure.
func (s *salesOrderSvcImpl) fetchLaneRates(ctx context.Context, accountID string, order *domain.SalesOrder, origin, dest domain.ShippingAddress) ([]domain.ShippoRateOption, *apierror.APIError) {
	if s.shippoFactory == nil {
		return nil, nil
	}

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: accountID, CarrierID: *order.CarrierID})
	if apiErr != nil {
		return nil, apiErr
	}
	if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
		return nil, nil
	}

	integrationRepo := s.repos.NewAccountIntegrationRepo()
	hasIntegration, apiErr := integrationRepo.HasIntegration(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	if !hasIntegration {
		return nil, nil
	}

	encryptedCreds, _, apiErr := integrationRepo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	apiKey, apiErr := decryptShippoAPIKey(encryptedCreds, s.encryptionKey, accountID)
	if apiErr != nil {
		return nil, apiErr
	}

	return s.shippoFactory.Build(apiKey).FetchAllShippingRates(ctx, domain.FetchAllShippingRatesParams{
		CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
		FromAddress:            origin,
		ToAddress:              dest,
		Parcels:                transitProbeParcels(),
	})
}

// transitProbeParcels is the parcel sent purely to make the carrier quote a lane.
//
// The weight is nominal rather than the order's real weight, because transit is a function of distance and service and not of what is in the box: ground from Ohio to Texas is three days whether the parcel is one pound or forty. Shippo will not return rates without a parcel, so this is the smallest input that gets an answer. Dimensions match the ones the pricing path already sends, so a lane is described to the carrier the same way in both places.
func transitProbeParcels() []domain.Parcel {
	return []domain.Parcel{{
		Weight: "1",
		Length: "23.5",
		Width:  "13",
		Height: "9.5",
	}}
}

// recordLaneRates writes one estimate per service level the carrier priced, matching its token back to the account's own service levels. Options the account does not carry are dropped: the lane key is a service level ID, so there is nowhere to file them.
func (s *salesOrderSvcImpl) recordLaneRates(ctx context.Context, accountID string, lane domain.TransitLane, rates []domain.ShippoRateOption) *apierror.APIError {
	if len(rates) == 0 {
		return nil
	}

	carrierOptionIDByToken, apiErr := s.serviceLevelIDsByToken(ctx, accountID, lane.CarrierOptionID)
	if apiErr != nil {
		return apiErr
	}

	estimateRepo := s.repos.NewCarrierTransitEstimateRepo()
	for _, rate := range rates {
		if rate.EstimatedDays == nil || *rate.EstimatedDays < 0 {
			continue
		}
		carrierOptionID, ok := carrierOptionIDByToken[rate.ServiceLevelToken]
		if !ok {
			continue
		}

		estimateID, apiErr := id.GenID(id.CarrierTransitEstimateIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		laneForOption := lane
		laneForOption.CarrierOptionID = carrierOptionID
		if apiErr := estimateRepo.Upsert(ctx, domain.UpsertTransitEstimateParams{
			ID:          estimateID,
			AccountID:   accountID,
			Lane:        laneForOption,
			TransitDays: int(*rate.EstimatedDays),
			SourceCode:  string(constants.TransitEstimateSourceShippo),
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// serviceLevelIDsByToken indexes the carrier's service levels by the token the carrier rates against, so a quote can be filed under the account's own IDs.
func (s *salesOrderSvcImpl) serviceLevelIDsByToken(ctx context.Context, accountID, serviceLevelID string) (map[string]string, *apierror.APIError) {
	serviceLevel, apiErr := s.repos.NewServiceLevelRepo().Get(ctx, accountID, serviceLevelID)
	if apiErr != nil {
		return nil, apiErr
	}

	siblings, apiErr := s.repos.NewCarrierRepo().ListOptionsByCarrierID(ctx, accountID, serviceLevel.CarrierID)
	if apiErr != nil {
		return nil, apiErr
	}

	byToken := make(map[string]string, len(siblings))
	for _, sl := range siblings {
		if sl.ServiceLevelToken != nil && *sl.ServiceLevelToken != "" {
			byToken[*sl.ServiceLevelToken] = sl.ID
		}
	}
	return byToken, nil
}
