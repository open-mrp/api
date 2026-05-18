package itemep

import (
	"context"
	"net/http"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apierror "github.com/augno/api/shared/errors"
)

// ExportItemsRequest is the request to export items with inventory.
type ExportItemsRequest struct{}

// Exports all items with on-hand inventory as an Excel file.
type ExportItemsEndpoint struct{}

func (e *ExportItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportItemsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportItemsRequest, *httptransport.FileDownload]{
		Title:             "Export Items",
		Method:            http.MethodGet,
		ContentType:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:             "/v1/catalog/items/actions/export",
		Request:           &ExportItemsRequest{},
		Response:          &httptransport.FileDownload{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportItemsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(ItemSvc).ExportItems
		},
	}).WithDocSource(e)
}
