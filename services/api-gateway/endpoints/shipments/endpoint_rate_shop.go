package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to rate shop across carriers.
type RateShopRequest struct {
	// Product lines of the items being shipped, used to apply freight exemptions.
	//
	// If any listed product line is freight exempt, no options are returned and `exemption_type` is `freight_exempt`.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// ID of the customer the shipment is for, used to apply the customer's freight policy and default shipping term.
	//
	// A freight-exempt customer or a free-freight shipping term returns no options with `exemption_type` set to `freight_exempt`; a flat-rate shipping term replaces carrier rates with the flat rate.
	CustomerID field.Optional[string] `json:"customer_id,omitzero"`
	// Origin address.
	FromAddress apirequest.AddressInput `json:"from_address" validate:"required"`
	// Destination address.
	ToAddress apirequest.AddressInput `json:"to_address" validate:"required"`
	// Parcels to rate shop.
	Parcels []ParcelInput `json:"parcels" validate:"required,min=1"`
	// Total value of the order, used to evaluate the free-shipping minimum order value on the customer's shipping term.
	OrderTotal field.Optional[float64] `json:"order_total,omitzero"`
}

var (
	sampleRateShopFromStreetLine1 = apiresource.SampleAddressLine1
	sampleRateShopFromLocality    = apiresource.SampleAddressCity
	sampleRateShopFromState       = apiresource.SampleAddressState
	sampleRateShopFromPostalCode  = apiresource.SampleAddressPostalCode
	sampleRateShopToStreetLine1   = "456 Oak Avenue"
	sampleRateShopToLocality      = "Los Angeles"
	sampleRateShopToState         = "CA"
	sampleRateShopToPostalCode    = "90001"
)

var sampleRateShopRequest = &RateShopRequest{
	FromAddress: apirequest.AddressInput{
		Name:        "Origin Warehouse",
		StreetLine1: field.SomePtr(&sampleRateShopFromStreetLine1),
		Locality:    field.SomePtr(&sampleRateShopFromLocality),
		State:       field.SomePtr(&sampleRateShopFromState),
		PostalCode:  field.SomePtr(&sampleRateShopFromPostalCode),
		Country:     apiresource.SampleAddressCountry,
	},
	ToAddress: apirequest.AddressInput{
		Name:        "Destination",
		StreetLine1: field.SomePtr(&sampleRateShopToStreetLine1),
		Locality:    field.SomePtr(&sampleRateShopToLocality),
		State:       field.SomePtr(&sampleRateShopToState),
		PostalCode:  field.SomePtr(&sampleRateShopToPostalCode),
		Country:     "US",
	},
	Parcels: []ParcelInput{
		{
			Weight: 5.0,
			Length: 12.0,
			Width:  8.0,
			Height: 6.0,
		},
	},
}

func (*RateShopRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRateShopRequest)
}

// Compares shipping rates across all of the account's carriers and service levels for the given addresses and parcels.
//
// Returns options sorted by rate ascending, after applying the account's freight rules: freight-exempt product lines or customers and free-freight shipping terms return no options, a flat-rate shipping term replaces carrier rates with the flat rate, and a met free-shipping minimum order value zeroes the rate on eligible options. Live carrier rates require the Shippo integration; carriers without live rating configured are returned with a rate of `0`.
type RateShopEndpoint struct{}

func (e *RateShopEndpoint) Materialize() *apiendpoint.APIEndpoint[*RateShopRequest, *apiresource.RateShopResult] {
	return (&apiendpoint.APIEndpoint[*RateShopRequest, *apiresource.RateShopResult]{
		Title:             "Rate Shop",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/actions/rate-shop",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RateShopRequest) (*apiresource.RateShopResult, *apierror.APIError) {
			return svc.(ShipmentSvc).RateShop
		},
	})
}
