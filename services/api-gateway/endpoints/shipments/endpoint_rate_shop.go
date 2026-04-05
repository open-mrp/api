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

// RateShopRequest is the request to rate shop across carriers.
type RateShopRequest struct {
	// The product line IDs for the shipment.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// The customer ID for the shipment.
	CustomerID *string `json:"customer_id,omitempty"`
	// The origin address.
	FromAddress apirequest.ShippingAddressInput `json:"from_address" validate:"required"`
	// The destination address.
	ToAddress apirequest.ShippingAddressInput `json:"to_address" validate:"required"`
	// The parcels to rate shop for.
	Parcels []ParcelInput `json:"parcels" validate:"required,min=1"`
	// The total order value.
	OrderTotal *float64 `json:"order_total,omitempty"`
}

var sampleRateShopRequest = &RateShopRequest{
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

func (*RateShopRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRateShopRequest)
}

type RateShopEndpoint struct{}

func (e *RateShopEndpoint) Materialize() *apiendpoint.APIEndpoint[*RateShopRequest, *apiresource.RateShopResult] {
	return &apiendpoint.APIEndpoint[*RateShopRequest, *apiresource.RateShopResult]{
		Title:             "Rate Shop",
		Description:       "Compares shipping rates across all available carriers and options for the given addresses and parcels.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/actions/rate-shop",
		ContentType:       "application/json",
		Request:           &RateShopRequest{},
		Response:          &apiresource.RateShopResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RateShopRequest) (*apiresource.RateShopResult, *apierror.APIError) {
			return svc.(ShipmentSvc).RateShop
		},
	}
}
