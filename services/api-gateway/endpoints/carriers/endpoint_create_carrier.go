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
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Carrier code.
	Code field.Optional[constants.CarrierCode] `json:"code,omitzero"`
	// Carrier account number. Required for UPS and USPS carriers.
	AccountNumber field.Optional[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
	// Carrier visibility in the customer portal.
	//
	// If `visible`, this carrier will be available for your customers to utilize when they go to checkout. If `hidden`, this carrier will not be an option on checkout.
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

// Creates a carrier. If a Shippo-supported carrier code is provided, the carrier will be registered with Shippo and service levels will be auto-synced as options.
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
