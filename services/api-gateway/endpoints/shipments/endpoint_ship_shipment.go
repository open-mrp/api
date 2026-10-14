package shipmentep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to mark a shipment as shipped.
type ShipShipmentRequest struct {
	// ID of the shipment to ship.
	ShipmentID string `path:"id" validate:"required"`
	// Whether to email the customer a shipping notification.
	//
	// Whether to email the customer the invoice raised for this shipment.
	EmailCustomer bool `json:"email_customer"`
}

var sampleShipShipmentRequest = &ShipShipmentRequest{
	EmailCustomer: true,
}

func (*ShipShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleShipShipmentRequest)
}

// Dispatches a packed shipment asynchronously, marking it and its cases as shipped.
//
// Returns `202 Accepted` with a job. Poll the job at the returned `Location`; once it reports `completed`, its first result carries the shipment's `id`. The job sets the shipment status to `shipped`, records `shipped_at` and the acting user as `shipped_by`, marks all shipping cases as shipped, and assigns an SSCC to any case that does not already have one. Fails with a conflict error if the shipment has already been shipped, so shipping is a one-way move that can only be reversed with the void action.
//
// Shipping raises the shipment's invoice, numbered after the shipment, and queues it for EDI transmission when the customer trades over EDI. When the carrier is linked to the account's Shippo integration, a label is bought for every case first; each case must have a freight weight, and the carrier's charge is recorded as the cost of the order's freight line (zero when no label is bought). Labels already bought for the cases by an earlier attempt are reused, never bought twice. A shipment is shipped, voided or deleted by one request at a time: a concurrent ship of the same shipment fails its job with a conflict error and can be retried once the first finishes.
type ShipShipmentEndpoint struct{}

func (e *ShipShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*ShipShipmentRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ShipShipmentRequest, *apiresource.AsyncJob]{
		Title:             "Ship Shipment",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/{id}/actions/ship",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ShipShipmentRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ShipmentSvc).ShipShipment
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
