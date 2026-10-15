package carrierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create a carrier.
type CreateCarrierRequest struct {
	// Display name, such as `FedEx`.
	//
	// Unique (case-insensitive) among your account's carriers and the system carriers.
	Name string `json:"name" validate:"required,max=255"`
	// What kind of carrier this is. Fixed after creation.
	//
	// `fedex`, `ups`, `usps`: connected through Shippo for live rates and labels. `ltl`: less-than-truckload freight. `will_call`: the customer picks up. `local_delivery`: your own vehicles. `other`: any other carrier you manage yourself.
	Type constants.CarrierType `json:"type" validate:"required"`
	// Standard Carrier Alpha Code, such as `FDEG`.
	//
	// The 2–4 uppercase letters that identify the carrier on EDI documents and bills of lading.
	SCAC field.Optional[string] `json:"scac,omitzero" validate:"omitempty,min=2,max=4,alpha,uppercase"`
	// Your account number with the carrier. Fixed after creation.
	//
	// Required for `ups` and `usps`, which connect using it. FedEx connects through OAuth instead.
	AccountNumber field.Optional[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
	// Whether customers can see and pick this carrier at checkout in the customer portal. Defaults to `visible`.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero" default:"visible"`
	// `active` or `archived`. Defaults to `active`. Archived carriers can't be newly set on orders or customer defaults.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleCreateCarrierRequest = &CreateCarrierRequest{
	Name:             "FedEx",
	Type:             constants.CarrierTypeFedEx,
	SCAC:             field.Some("FDEG"),
	AccountNumber:    field.Some("1234567890"),
	PortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
}

func (*CreateCarrierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCarrierRequest)
}

// Creates a carrier.
//
// A `fedex`, `ups` or `usps` carrier is connected through Shippo for live rates and labels, and gets a service level for each service the carrier offers, hidden from the customer portal until you show it. This needs an active Shippo integration and is skipped for sandbox accounts.
type CreateCarrierEndpoint struct{}

func (e *CreateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*CreateCarrierRequest, *apiresource.Carrier]{
		Title:             "Create Carrier",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).CreateCarrier
		},
		LocationFunc: func(resp *apiresource.Carrier) string {
			return "/v1/operations/carriers/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"default_service_level", "owner", "owner.account", "service_levels"},
		}),
	})
}
