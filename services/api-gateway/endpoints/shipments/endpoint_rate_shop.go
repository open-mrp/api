package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
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
	// A customer that is freight exempt through its own policy or through one of its groups, or whose shipping term is free freight, returns no options with `exemption_type` set to `freight_exempt`; a flat-rate shipping term replaces carrier rates with the flat rate. Omitting the customer skips all of these rules and returns plain carrier rates.
	CustomerID field.Optional[string] `json:"customer_id,omitzero"`
	// Origin address.
	//
	// When omitted, the account's configured ship-from origin is used, which is how customer portal callers rate shop without knowing the seller's address.
	FromAddress field.Optional[apirequest.AddressInput] `json:"from_address,omitzero"`
	// Destination address.
	ToAddress apirequest.AddressInput `json:"to_address" validate:"required"`
	// Parcels to rate shop.
	Parcels []ParcelInput `json:"parcels" validate:"required,min=1"`
	// Total value of the order, used to evaluate the free-shipping minimum order value on the customer's shipping term.
	//
	// Free shipping applies only when the total is strictly above the threshold, and only for the service levels the shipping term allows.
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
	FromAddress: field.Some(apirequest.AddressInput{
		Name:        "Origin Warehouse",
		StreetLine1: field.SomePtr(&sampleRateShopFromStreetLine1),
		Locality:    field.SomePtr(&sampleRateShopFromLocality),
		State:       field.SomePtr(&sampleRateShopFromState),
		PostalCode:  field.SomePtr(&sampleRateShopFromPostalCode),
		Country:     apiresource.SampleAddressCountry,
	}),
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
// Returns options sorted by rate ascending, after applying the account's freight rules: freight-exempt product lines or customers and free-freight shipping terms return no options, a flat-rate shipping term replaces carrier rates with the flat rate, and a met free-shipping minimum order value zeroes the rate on eligible options.
//
// Live carrier rates require the Shippo integration. Carriers that are not linked to a live-rating account are returned at a rate of `0`, while carriers that are linked but whose rates cannot be fetched are left out of the results entirely. Customer portal callers only see carriers and service levels that have been enabled for the portal.
type RateShopEndpoint struct{}

func (e *RateShopEndpoint) Materialize() *apiendpoint.APIEndpoint[*RateShopRequest, *apiresource.RateShopResult] {
	return (&apiendpoint.APIEndpoint[*RateShopRequest, *apiresource.RateShopResult]{
		Title:               "Rate Shop",
		Method:              http.MethodPost,
		Route:               "/v1/operations/shipments/actions/rate-shop",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RateShopRequest) (*apiresource.RateShopResult, *apierror.APIError) {
			return svc.(ShipmentSvc).RateShop
		},
	})
}
