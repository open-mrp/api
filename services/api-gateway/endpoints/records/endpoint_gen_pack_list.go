package recordsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to generate a pack list for a shipment.
type GenPackListRequest struct {
	// ID of the shipment to generate the pack list for.
	ShipmentID string `json:"shipment_id" validate:"required"`
}

var sampleGenPackListRequest = &GenPackListRequest{
	ShipmentID: apiresource.SampleShipmentID,
}

func (*GenPackListRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGenPackListRequest)
}

// Assembles a printable pack-list document for a shipment.
//
// Gathers the shipment's packed line items and shipping cases (with carrier tracking numbers and case weights) together with its parent order's header, bill-to and ship-to parties, terms, and any order lines still back-ordered, and returns them as a single document ready to render. The document is a point-in-time snapshot and is not persisted.
type GenPackListEndpoint struct{}

func (e *GenPackListEndpoint) Materialize() *apiendpoint.APIEndpoint[*GenPackListRequest, *apiresource.PackList] {
	return (&apiendpoint.APIEndpoint[*GenPackListRequest, *apiresource.PackList]{
		Title:             "Generate Pack List",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/records/actions/generate-pack-list",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ObjectType:        constants.ObjectTypePackList,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainShipments, Action: types.ActionRead},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GenPackListRequest) (*apiresource.PackList, *apierror.APIError) {
			return svc.(RecordsSvc).GenPackList
		},
	})
}
