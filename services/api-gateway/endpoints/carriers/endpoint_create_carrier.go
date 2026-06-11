package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a carrier.
type CreateCarrierRequest struct {
	// Human-readable name for the carrier.
	//
	// Must be unique among your account's carriers.
	Name string `json:"name" validate:"required,max=255"`
	// Well-known carrier code.
	//
	// Omit for a custom carrier. Providing a Shippo-supported code (`fedex`, `ups`, `usps`) connects the carrier through Shippo and auto-syncs its service levels.
	Code field.Optional[constants.CarrierCode] `json:"code,omitzero"`
	// Your account number with this carrier.
	//
	// Required when `code` is `ups` or `usps`, which connect to Shippo using this number; FedEx connects via OAuth instead.
	AccountNumber field.Optional[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
	// Carrier visibility in the customer portal.
	//
	// A `visible` carrier can be selected by your customers at checkout; a `hidden` carrier is not offered there. New carriers are visible unless set to `hidden`.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero" default:"visible"`
}

var sampleCreateCarrierCode = constants.CarrierCodeFedEx
var sampleCreateCarrierAccountNumber = "1234567890"
var sampleCreateCarrierVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateCarrierRequest = &CreateCarrierRequest{
	Name:                     "FedEx",
	Code:                     field.Some(sampleCreateCarrierCode),
	AccountNumber:            field.Some(sampleCreateCarrierAccountNumber),
	CustomerPortalVisibility: field.Some(sampleCreateCarrierVisibility),
}

func (*CreateCarrierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCarrierRequest)
}

// Creates a carrier.
//
// If a Shippo-supported code (`fedex`, `ups`, `usps`) is provided, the carrier is connected through Shippo and its service levels are auto-synced, initially hidden from the customer portal. Sandbox accounts skip the Shippo connection.
type CreateCarrierEndpoint struct{}

func (e *CreateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier]{
		Title:             "Create Carrier",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).CreateCarrier
		},
		LocationFunc: func(resp *apiresource.Carrier) string {
			return "/v1/operations/carriers/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "owner.account", "service_levels"},
		}),
	})
}
