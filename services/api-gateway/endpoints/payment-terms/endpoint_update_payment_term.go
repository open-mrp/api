package paymenttermep

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

// Request to partially update a payment term.
type UpdatePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
	// Display name, such as `Net 30`.
	//
	// Unique (case-insensitive) among your account's terms and the system defaults.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Whether the term can be assigned to customers and sales orders.
	//
	// `archived` terms stay attached to the records already using them.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleUpdatePaymentTermRequest = &UpdatePaymentTermRequest{
	Name: field.Some("Net 60"),
}

func (*UpdatePaymentTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePaymentTermRequest)
}

// Updates a payment term.
//
// Only fields you send are changed. System defaults can't be updated.
type UpdatePaymentTermEndpoint struct{}

func (e *UpdatePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm] {
	return (&apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:             "Update Payment Term",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/payment-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypePaymentTerm,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).UpdatePaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
