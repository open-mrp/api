package subscriptionep

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
)

// Request to follow a record.
type CreateSubscriptionRequest struct {
	// The record to follow: a `sales_order`, `purchase_order`, `production_run`, `invoice`, `customer` or `supplier`.
	Record apiresource.RecordInput `json:"record" validate:"required"`
}

var sampleCreateSubscriptionRequest = &CreateSubscriptionRequest{Record: *apiresource.SampleRecordInput}

func (*CreateSubscriptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSubscriptionRequest)
}

// Subscribes you to a record, so you're notified about it.
//
// A new subscription has reason `manual`. Following a record you already follow returns that subscription unchanged; following one you unsubscribed from subscribes you again. Following a customer or supplier also covers their orders and invoices.
type CreateSubscriptionEndpoint struct{}

func (e *CreateSubscriptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSubscriptionRequest, *apiresource.Subscription] {
	return (&apiendpoint.APIEndpoint[*CreateSubscriptionRequest, *apiresource.Subscription]{
		Title:             "Create Subscription",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             subscriptionsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSubscription,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSubscriptionRequest) (*apiresource.Subscription, *apierror.APIError) {
			return svc.(Svc).CreateSubscription
		},
		LocationFunc: func(resp *apiresource.Subscription) string {
			return subscriptionsRoute + "/" + resp.ID
		},
	})
}
