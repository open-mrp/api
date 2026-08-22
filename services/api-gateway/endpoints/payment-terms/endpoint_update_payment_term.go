package paymenttermep

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

// Request to partially update a payment term.
type UpdatePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
	// New display name for the payment term.
	//
	// Must be unique among the payment terms visible to your account, including system defaults.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdatePaymentTermRequest = &UpdatePaymentTermRequest{
	Name: field.Some("Net 60"),
}

func (*UpdatePaymentTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePaymentTermRequest)
}

// Partially updates a payment term.
//
// Only payment terms created by your account can be updated; system-owned default terms cannot be.
type UpdatePaymentTermEndpoint struct{}

func (e *UpdatePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm] {
	return (&apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:               "Update Payment Term",
		Method:              http.MethodPatch,
		Route:               "/v1/finance/payment-terms/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypePaymentTerm,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).UpdatePaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
