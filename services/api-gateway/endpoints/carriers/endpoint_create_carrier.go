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

// CreateCarrierRequest is the request to create a new carrier.
type CreateCarrierRequest struct {
	// The display name of the carrier.
	Name string `json:"name" validate:"required"`
	// The carrier code.
	Code *constants.CarrierCode `json:"code"`
	// The carrier account number, required for UPS and USPS carriers.
	AccountNumber *string `json:"account_number"`
	// Whether this carrier is visible in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible" nullable:"false"`
}

var sampleCreateCarrierCode = constants.CarrierCodeFedEx
var sampleCreateCarrierRequest = &CreateCarrierRequest{
	Name: "FedEx",
	Code: &sampleCreateCarrierCode,
}

func (*CreateCarrierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCarrierRequest)
}

type CreateCarrierEndpoint struct{}

func (e *CreateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier] {
	return &apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier]{
		Title:             "Create Carrier",
		Description:       "Creates a new carrier. If a Shippo-supported carrier code is provided, the carrier will be registered with Shippo and service levels will be auto-synced as options.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/carriers",
		Request:           &CreateCarrierRequest{},
		Response:          &apiresource.Carrier{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).CreateCarrier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "service_levels"},
		}),
	}
}
