package stub

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// ShippoClientFactory is a no-op factory for use in test mode.
type ShippoClientFactory struct{}

func (f *ShippoClientFactory) Build(_ string) domain.ShippoClient {
	return &shippoClient{}
}

type shippoClient struct{}

// Rate scenarios are selected by the destination postal code, because that is the one input an end-to-end test can set freely: an order's ship-to address is created by the test, where the carrier, service levels and account integration all come from seed data.
//
// Every code outside this table gets transitStubDefaultRates, so the ordinary path needs no special address and existing tests are unaffected.
const (
	// ZipStubNoRates is a lane the carrier will not serve, so it returns no rates at all.
	ZipStubNoRates = "99910"
	// ZipStubNoTransit returns priced rates that carry no transit estimate, which is how a carrier answers when it will quote a price but not a delivery commitment.
	ZipStubNoTransit = "99911"
	// ZipStubError fails the rate call outright, standing in for a carrier outage or a rejected request.
	ZipStubError = "99912"
	// ZipStubSameDay returns zero-day transit, which is a real answer (a local or will-call lane) and not the same as having none.
	ZipStubSameDay = "99913"
	// ZipStubUnknownService returns a service level token no account carries, so the quote has nowhere to be filed.
	ZipStubUnknownService = "99914"
)

// Tokens the stub quotes against. They match the service levels seeded for the e2e transit carrier.
const (
	stubTokenGround    = "fedex_ground"
	stubTokenTwoDay    = "fedex_2_day"
	stubTokenOvernight = "fedex_priority_overnight"
)

func days(n int32) *int32 { return new(n) }

// transitStubDefaultRates is the ordinary three-service answer: a slow, a middle and a fast option, each with the transit a real carrier would quote.
func transitStubDefaultRates() []domain.ShippoRateOption {
	return []domain.ShippoRateOption{
		{ServiceLevelName: "Ground Shipping", ServiceLevelToken: stubTokenGround, Amount: 12.50, EstimatedDays: days(3)},
		{ServiceLevelName: "2 Day", ServiceLevelToken: stubTokenTwoDay, Amount: 28.00, EstimatedDays: days(2)},
		{ServiceLevelName: "Priority Overnight", ServiceLevelToken: stubTokenOvernight, Amount: 54.75, EstimatedDays: days(1)},
	}
}

// ratesForDestination returns the scenario the destination postal code selects. A nil slice with a nil error means the carrier answered with nothing, which is distinct from the error case.
func ratesForDestination(zip string) ([]domain.ShippoRateOption, *apierror.APIError) {
	switch normalizeStubZip(zip) {
	case ZipStubNoRates:
		return nil, nil

	case ZipStubNoTransit:
		rates := transitStubDefaultRates()
		for i := range rates {
			rates[i].EstimatedDays = nil
		}
		return rates, nil

	case ZipStubError:
		// Internal errors are classed transient, which is what a carrier outage looks like to the warm path.
		return nil, apierror.NewInternalError(nil, "Stub carrier is unavailable for this lane.")

	case ZipStubSameDay:
		return []domain.ShippoRateOption{
			{ServiceLevelName: "Ground Shipping", ServiceLevelToken: stubTokenGround, Amount: 9.00, EstimatedDays: days(0)},
		}, nil

	case ZipStubUnknownService:
		return []domain.ShippoRateOption{
			{ServiceLevelName: "Interstellar", ServiceLevelToken: "stub_service_no_account_carries", Amount: 99.00, EstimatedDays: days(4)},
		}, nil

	default:
		return transitStubDefaultRates(), nil
	}
}

// normalizeStubZip mirrors the lane normalization applied before a postal code is stored, so a test can pass a ZIP+4 and still select its scenario.
func normalizeStubZip(zip string) string {
	z := strings.ToUpper(strings.Join(strings.Fields(zip), ""))
	if base, _, found := strings.Cut(z, "-"); found {
		return base
	}
	return z
}

func (s *shippoClient) FindOrRegisterCarrierAccount(_ context.Context, _ string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	return &domain.ShippoCarrierAccount{}, nil
}

func (s *shippoClient) ConnectCarrierAccount(_ context.Context, _, _ string, _ map[string]string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	return &domain.ShippoCarrierAccount{}, nil
}

func (s *shippoClient) GetCarrierAccount(_ context.Context, _ string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	return &domain.ShippoCarrierAccount{}, nil
}

func (s *shippoClient) DeactivateCarrierAccount(_ context.Context, _ string) *apierror.APIError {
	return nil
}

func (s *shippoClient) GetCarrierServiceLevels(_ context.Context, _ string) ([]domain.ShippoServiceLevel, *apierror.APIError) {
	return nil, nil
}

func (s *shippoClient) InitiateOAuth(_ context.Context, _, _ string, _ *string) (string, *apierror.APIError) {
	return "https://stub.local/oauth", nil
}

// FetchShippingRate answers the pricing path from the same scenario table the transit path reads, so a lane that quotes three days also quotes the price that goes with it.
//
// ZipStubError is the deliberate exception: it fails only the all-rates call that warming uses. Order pricing calls this method synchronously and propagates its error, so failing here would abort order create and the order whose warm was meant to fail would never exist. Scoping the outage to the warm path is what makes it observable at all.
func (s *shippoClient) FetchShippingRate(_ context.Context, params domain.FetchShippingRateParams) (float64, *apierror.APIError) {
	zip := normalizeStubZip(params.ToAddress.Zip)
	if zip == ZipStubError {
		return transitStubDefaultRates()[0].Amount, nil
	}

	rates, apiErr := ratesForDestination(zip)
	if apiErr != nil {
		return 0, apiErr
	}
	if len(rates) == 0 {
		return 0, nil
	}

	if params.ServiceLevelToken != "" {
		for _, r := range rates {
			if r.ServiceLevelToken == params.ServiceLevelToken {
				return r.Amount, nil
			}
		}
		return 0, nil
	}

	cheapest := rates[0].Amount
	for _, r := range rates[1:] {
		if r.Amount < cheapest {
			cheapest = r.Amount
		}
	}
	return cheapest, nil
}

func (s *shippoClient) FetchAllShippingRates(_ context.Context, params domain.FetchAllShippingRatesParams) ([]domain.ShippoRateOption, *apierror.APIError) {
	return ratesForDestination(params.ToAddress.Zip)
}

func (s *shippoClient) CreateTransactionInstantLabel(_ context.Context, _ domain.CreateLabelParams) (*domain.LabelResult, *apierror.APIError) {
	return &domain.LabelResult{}, nil
}

func (s *shippoClient) RefundTransaction(_ context.Context, _ string) *apierror.APIError {
	return nil
}
