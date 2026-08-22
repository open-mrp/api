package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a carrier.
type CreateCarrierRequest struct {
	// Human-readable name for the carrier.
	//
	// Must not match another carrier already visible to your account, including the system-provided ones.
	Name string `json:"name" validate:"required,max=255"`
	// Well-known carrier code.
	//
	// Providing a Shippo-supported code (`fedex`, `ups`, `usps`) connects the carrier through Shippo and syncs its service levels; the other codes, such as `will_call` and `delivery`, simply describe a self-managed shipping method. Omit the code entirely when none of them fit. The code cannot be changed after the carrier is created.
	Code field.Optional[constants.CarrierCode] `json:"code,omitzero"`
	// Your account number with this carrier.
	//
	// Required when `code` is `ups` or `usps`, whose carrier accounts are connected to Shippo using this number; FedEx authorizes through OAuth instead, so no account number is needed.
	AccountNumber field.Optional[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
	// Whether customers can see and select this carrier at checkout in the customer portal.
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

// Creates a shipping carrier your account can ship orders with.
//
// Supplying a Shippo-supported code (`fedex`, `ups`, `usps`) connects a Shippo carrier account and creates a service level for every service that carrier offers, each hidden from the customer portal until you make it visible. This requires an active Shippo integration on the account and is skipped entirely for sandbox accounts, which get a carrier record with no service levels and no live rating.
type CreateCarrierEndpoint struct{}

func (e *CreateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier]{
		Title:               "Create Carrier",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/carriers",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCarrier,
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
