package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// A parcel's weight and dimensions for shipping rate calculations.
type ParcelInput struct {
	// Parcel weight in pounds.
	Weight float64 `json:"weight" validate:"required"`
	// Parcel length in inches.
	Length float64 `json:"length" validate:"required"`
	// Parcel width in inches.
	Width float64 `json:"width" validate:"required"`
	// Parcel height in inches.
	Height float64 `json:"height" validate:"required"`
}

// Request to estimate a shipping rate.
type EstimateRateRequest struct {
	// ID of the carrier to rate.
	CarrierID string `json:"carrier_id" validate:"required"`
	// ID of the carrier service level to rate.
	ServiceLevelID string `json:"service_level_id" validate:"required"`
	// Product lines of the items being shipped, used to apply freight exemptions.
	//
	// If any listed product line is freight exempt, the estimated rate is `0`.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// ID of the customer the shipment is for, used to apply the customer's freight policy and default shipping term.
	//
	// A customer that is freight exempt through its own policy or through one of its groups, or whose shipping term is free freight, yields a rate of `0`; a flat-rate shipping term returns the flat rate. Omitting the customer skips all of these rules and quotes the plain carrier rate.
	CustomerID field.Optional[string] `json:"customer_id,omitzero"`
	// Origin address.
	//
	// A live carrier rate requires a postal code and country here; without them the request fails rather than returning a meaningless estimate.
	FromAddress apirequest.AddressInput `json:"from_address" validate:"required"`
	// Destination address.
	ToAddress apirequest.AddressInput `json:"to_address" validate:"required"`
	// Parcels to estimate rates for.
	Parcels []ParcelInput `json:"parcels" validate:"required,min=1"`
	// Total value of the order, used to evaluate the free-shipping minimum order value on the customer's shipping term.
	//
	// Free shipping applies only when the total is strictly above the threshold, and only for the service levels the shipping term allows.
	OrderTotal field.Optional[float64] `json:"order_total,omitzero"`
}

var (
	sampleEstimateFromStreetLine1 = apiresource.SampleAddressLine1
	sampleEstimateFromLocality    = apiresource.SampleAddressCity
	sampleEstimateFromState       = apiresource.SampleAddressState
	sampleEstimateFromPostalCode  = apiresource.SampleAddressPostalCode
	sampleEstimateToStreetLine1   = "456 Oak Avenue"
	sampleEstimateToLocality      = "Los Angeles"
	sampleEstimateToState         = "CA"
	sampleEstimateToPostalCode    = "90001"
)

var sampleEstimateRateRequest = &EstimateRateRequest{
	CarrierID:      apiresource.SampleCarrierID,
	ServiceLevelID: apiresource.SampleServiceLevelID,
	FromAddress: apirequest.AddressInput{
		Name:        "Origin Warehouse",
		StreetLine1: field.SomePtr(&sampleEstimateFromStreetLine1),
		Locality:    field.SomePtr(&sampleEstimateFromLocality),
		State:       field.SomePtr(&sampleEstimateFromState),
		PostalCode:  field.SomePtr(&sampleEstimateFromPostalCode),
		Country:     apiresource.SampleAddressCountry,
	},
	ToAddress: apirequest.AddressInput{
		Name:        "Destination",
		StreetLine1: field.SomePtr(&sampleEstimateToStreetLine1),
		Locality:    field.SomePtr(&sampleEstimateToLocality),
		State:       field.SomePtr(&sampleEstimateToState),
		PostalCode:  field.SomePtr(&sampleEstimateToPostalCode),
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

func (*EstimateRateRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEstimateRateRequest)
}

// Estimates the shipping rate for one specific carrier and service level.
//
// Freight rules are applied before live rating, in order: freight-exempt product lines, a freight-exempt customer or customer group, then the customer's default shipping term. A free-freight term and a met free-shipping minimum order value both return `0`, and a flat-rate term returns its flat rate without contacting the carrier. Live rates require the Shippo integration; without it, or for a carrier that is not linked to a live-rating account, the estimate is `0`.
//
// Use rate shop instead to compare every carrier and service level at once.
type EstimateRateEndpoint struct{}

func (e *EstimateRateEndpoint) Materialize() *apiendpoint.APIEndpoint[*EstimateRateRequest, *apiresource.EstimateRateResult] {
	return (&apiendpoint.APIEndpoint[*EstimateRateRequest, *apiresource.EstimateRateResult]{
		Title:               "Estimate Rate",
		Method:              http.MethodPost,
		Route:               "/v1/operations/shipments/actions/estimate-rate",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *EstimateRateRequest) (*apiresource.EstimateRateResult, *apierror.APIError) {
			return svc.(ShipmentSvc).EstimateRate
		},
	})
}
