package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Parcel for rate estimation.
type ParcelInput struct {
	// Weight.
	Weight float64 `json:"weight" validate:"required"`
	// Length.
	Length float64 `json:"length" validate:"required"`
	// Width.
	Width float64 `json:"width" validate:"required"`
	// Height.
	Height float64 `json:"height" validate:"required"`
}

// Request to estimate a shipping rate.
type EstimateRateRequest struct {
	// Carrier ID.
	CarrierID string `json:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `json:"service_level_id" validate:"required"`
	// Product line IDs.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Customer ID.
	CustomerID *string `json:"customer_id,omitempty"`
	// Origin address.
	FromAddress apirequest.AddressInput `json:"from_address" validate:"required"`
	// Destination address.
	ToAddress apirequest.AddressInput `json:"to_address" validate:"required"`
	// Parcels to estimate rates for.
	Parcels []ParcelInput `json:"parcels" validate:"required,min=1"`
	// Total order value.
	OrderTotal *float64 `json:"order_total,omitempty"`
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
		StreetLine1: &sampleEstimateFromStreetLine1,
		Locality:    &sampleEstimateFromLocality,
		State:       &sampleEstimateFromState,
		PostalCode:  &sampleEstimateFromPostalCode,
		Country:     apiresource.SampleAddressCountry,
	},
	ToAddress: apirequest.AddressInput{
		Name:        "Destination",
		StreetLine1: &sampleEstimateToStreetLine1,
		Locality:    &sampleEstimateToLocality,
		State:       &sampleEstimateToState,
		PostalCode:  &sampleEstimateToPostalCode,
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

// Estimates a shipping rate for a given carrier, carrier option, addresses, and parcels.
type EstimateRateEndpoint struct{}

func (e *EstimateRateEndpoint) Materialize() *apiendpoint.APIEndpoint[*EstimateRateRequest, *apiresource.EstimateRateResult] {
	return (&apiendpoint.APIEndpoint[*EstimateRateRequest, *apiresource.EstimateRateResult]{
		Title:             "Estimate Rate",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/actions/estimate-rate",
		ContentType:       "application/json",
		Request:           &EstimateRateRequest{},
		Response:          &apiresource.EstimateRateResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *EstimateRateRequest) (*apiresource.EstimateRateResult, *apierror.APIError) {
			return svc.(ShipmentSvc).EstimateRate
		},
	}).WithDocSource(e)
}
