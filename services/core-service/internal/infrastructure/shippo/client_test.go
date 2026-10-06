package shippo

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

func TestNormalizeShippoDecimal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"float accumulation garbage", "1.0499999999999998", "1.05"},
		{"clean integer", "5", "5"},
		{"clean integer with point", "13", "13"},
		{"clean decimal", "23.5", "23.5"},
		{"trims trailing zeros", "9.5000", "9.5"},
		{"rounds to four places", "0.30000000000000004", "0.3"},
		{"rounds long fraction", "12.123456789", "12.1235"},
		{"zero", "0", "0"},
		{"unparseable passes through", "abc", "abc"},
		{"empty passes through", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeShippoDecimal(tc.in); got != tc.want {
				t.Errorf("normalizeShippoDecimal(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestPickShippingRate_MissingRateIsUnavailable(t *testing.T) {
	ground := ShipmentRate{Amount: "10.00", ServiceLevel: &RateService{Token: "ups_ground"}}
	tests := []struct {
		name  string
		rates []ShipmentRate
		token string
	}{
		{"no rates at all", nil, "ups_ground"},
		{"no rate for the requested service", []ShipmentRate{ground}, "ups_next_day_air"},
		{"no priced rate when any service will do", []ShipmentRate{{Amount: "n/a"}}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rate, apiErr := pickShippingRate(tc.rates, tc.token)
			if apiErr == nil {
				t.Fatalf("a missing rate must not quote %v as free freight", rate)
			}
			if apiErr.Code != apierror.ErrorCodeSvcUnavailable || !apiErr.IsTransient {
				t.Errorf("want a transient 503, got %s (transient %v)", apiErr.Code, apiErr.IsTransient)
			}
		})
	}

	rate, apiErr := pickShippingRate([]ShipmentRate{ground}, "ups_ground")
	if apiErr != nil || rate != applyShippingMarkup(10) {
		t.Errorf("the requested service's rate: got (%v, %v)", rate, apiErr)
	}
}

func TestClientFactory_RefusesLiveKeysOutsideProduction(t *testing.T) {
	tests := []struct {
		mode    constants.PlatformMode
		key     string
		wantErr bool
	}{
		{constants.PlatformModeProduction, "shippo_live_x", false},
		{constants.PlatformModeProduction, "shippo_test_x", false},
		{constants.PlatformModeDevelopment, "shippo_live_x", true},
		{constants.PlatformModeDevelopment, "shippo_test_x", false},
		{constants.PlatformModeTest, "shippo_live_x", true},
	}
	for _, tc := range tests {
		t.Run(string(tc.mode)+"/"+tc.key, func(t *testing.T) {
			factory, err := NewClientFactory(&ClientFactoryConfig{PlatformMode: tc.mode})
			if err != nil {
				t.Fatalf("building the factory: %v", err)
			}
			client, apiErr := factory.Build(tc.key)
			if tc.wantErr && (apiErr == nil || client != nil) {
				t.Errorf("a live key must not build a client in %s", tc.mode)
			}
			if !tc.wantErr && (apiErr != nil || client == nil) {
				t.Errorf("unexpected refusal: %v", apiErr)
			}
		})
	}
}

func TestClientFactory_DefaultsToProduction(t *testing.T) {
	factory, err := NewClientFactory(nil)
	if err != nil {
		t.Fatalf("building the factory: %v", err)
	}
	if _, apiErr := factory.Build("shippo_live_x"); apiErr != nil {
		t.Errorf("an unset platform mode is production, as PLATFORM is: %v", apiErr)
	}
}
