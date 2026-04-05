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

// ParcelInput represents a parcel for rate estimation.
type ParcelInput struct {
	// The weight of the parcel.
	Weight float64 `json:"weight" validate:"required"`
	// The length of the parcel.
	Length float64 `json:"length" validate:"required"`
	// The width of the parcel.
	Width float64 `json:"width" validate:"required"`
	// The height of the parcel.
	Height float64 `json:"height" validate:"required"`
}

// EstimateRateRequest is the request to estimate a shipping rate.
type EstimateRateRequest struct {
	// The ID of the carrier to estimate rates for.
	CarrierID string `json:"carrier_id" validate:"required"`
	// The ID of the service level to estimate rates for.
	ServiceLevelID string `json:"service_level_id" validate:"required"`
	// The product line IDs for the shipment.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// The customer ID for the shipment.
	CustomerID *string `json:"customer_id,omitempty"`
	// The origin address.
	FromAddress apirequest.ShippingAddressInput `json:"from_address" validate:"required"`
	// The destination address.
	ToAddress apirequest.ShippingAddressInput `json:"to_address" validate:"required"`
	// The parcels to estimate rates for.
	Parcels []ParcelInput `json:"parcels" validate:"required,min=1"`
	// The total order value.
	OrderTotal *float64 `json:"order_total,omitempty"`
}

var sampleEstimateRateRequest = &EstimateRateRequest{
	CarrierID:      apiresource.SampleCarrierID,
	ServiceLevelID: apiresource.SampleServiceLevelID,
	FromAddress: apirequest.ShippingAddressInput{
		StreetLine1: apiresource.SampleAddressLine1,
		Locality:    apiresource.SampleAddressCity,
		State:       apiresource.SampleAddressState,
		PostalCode:  apiresource.SampleAddressPostalCode,
		Country:     apiresource.SampleAddressCountry,
	},
	ToAddress: apirequest.ShippingAddressInput{
		StreetLine1: "456 Oak Avenue",
		Locality:    "Los Angeles",
		State:       "CA",
		PostalCode:  "90001",
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

type EstimateRateEndpoint struct{}

func (e *EstimateRateEndpoint) Materialize() *apiendpoint.APIEndpoint[*EstimateRateRequest, *apiresource.EstimateRateResult] {
	return &apiendpoint.APIEndpoint[*EstimateRateRequest, *apiresource.EstimateRateResult]{
		Title:             "Estimate Rate",
		Description:       "Estimates a shipping rate for a given carrier, carrier option, addresses, and parcels.",
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
	}
}
