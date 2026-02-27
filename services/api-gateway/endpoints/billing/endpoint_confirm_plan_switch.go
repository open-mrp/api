package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const confirmPlanSwitchDescription string = `Confirms a plan switch after a Stripe checkout redirect completes.
Called with the checkout session ID and target plan ID to finalize the upgrade.`

// The request to confirm a plan switch after a Stripe checkout redirect.
type ConfirmPlanSwitchRequest struct {
	// The Stripe checkout session ID from the redirect.
	SessionID string `json:"session_id" validate:"required"`
	// The target plan ID to switch to.
	PlanID string `json:"plan_id" validate:"required"`
}

var sampleConfirmPlanSwitchRequest = &ConfirmPlanSwitchRequest{
	SessionID: "cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG",
	PlanID:    "pt_pro_01gf7a8200eaj8fke1xvw4h50x",
}

func (*ConfirmPlanSwitchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConfirmPlanSwitchRequest)
}

type ConfirmPlanSwitchEndpoint struct{}

func (e *ConfirmPlanSwitchEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConfirmPlanSwitchRequest, *apiresource.ConfirmPlanSwitchResponse] {
	return &apiendpoint.APIEndpoint[*ConfirmPlanSwitchRequest, *apiresource.ConfirmPlanSwitchResponse]{
		Title:             "Confirm Plan Switch",
		Description:       confirmPlanSwitchDescription,
		Method:            http.MethodPost,
		Route:             "/v1/billing/plan-switches/confirm",
		ContentType:       "application/json",
		Request:           &ConfirmPlanSwitchRequest{},
		Response:          apiresource.SampleConfirmPlanSwitchResponse,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConfirmPlanSwitchRequest) (*apiresource.ConfirmPlanSwitchResponse, *apierror.APIError) {
			return svc.(BillingSvc).ConfirmPlanSwitch
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
