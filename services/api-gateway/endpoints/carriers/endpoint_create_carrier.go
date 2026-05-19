package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a carrier.
type CreateCarrierRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Carrier code.
	Code *constants.CarrierCode `json:"code"`
	// Carrier account number. Required for UPS and USPS carriers.
	AccountNumber *string `json:"account_number" validate:"omitempty,max=255"`
	// Whether this carrier will be available for customers to select in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible"`
}

var sampleCreateCarrierCode = constants.CarrierCodeFedEx
var sampleCreateCarrierAccountNumber = "1234567890"
var sampleCreateCarrierVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateCarrierRequest = &CreateCarrierRequest{
	Name:                     "FedEx",
	Code:                     &sampleCreateCarrierCode,
	AccountNumber:            &sampleCreateCarrierAccountNumber,
	CustomerPortalVisibility: &sampleCreateCarrierVisibility,
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
