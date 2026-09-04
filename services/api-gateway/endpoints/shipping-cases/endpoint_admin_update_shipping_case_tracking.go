package shippingcaseep

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

// ! separate endpoint due to SDK type resttrictions
// Request to correct a shipped shipping case's tracking number.
type AdminUpdateShippingCaseTrackingRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
	// Carrier tracking number the case actually traveled under, replacing any number already recorded.
	TrackingNumber field.Optional[string] `json:"tracking_number,omitzero" validate:"omitempty,max=255"`
}

var sampleAdminUpdateShippingCaseTrackingRequest = &AdminUpdateShippingCaseTrackingRequest{
	TrackingNumber: field.Some(sampleTrackingNumber),
}

func (*AdminUpdateShippingCaseTrackingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAdminUpdateShippingCaseTrackingRequest)
}

// Rewrites the tracking number of a shipping case that has already shipped.
// Administrators only: it is the deliberate override for a case dispatched under the wrong number, and it rejects a case that has not shipped yet.
type AdminUpdateShippingCaseTrackingEndpoint struct{}

func (e *AdminUpdateShippingCaseTrackingEndpoint) Materialize() *apiendpoint.APIEndpoint[*AdminUpdateShippingCaseTrackingRequest, *apiresource.ShippingCase] {
	return (&apiendpoint.APIEndpoint[*AdminUpdateShippingCaseTrackingRequest, *apiresource.ShippingCase]{
		Title:             "Admin Update Shipping Case Tracking",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}/actions/admin-update-tracking",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// The admin gate and shipments:update are enforced in the service; declared here to match.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeShippingCase,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AdminUpdateShippingCaseTrackingRequest) (*apiresource.ShippingCase, *apierror.APIError) {
			return svc.(ShippingCaseSvc).AdminUpdateShippingCaseTracking
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingCase,
			Fields:     []string{"carrier", "shipment", "freight_amount", "freight_amount.unit", "freight_weight", "freight_weight.unit"},
		}),
	})
}
