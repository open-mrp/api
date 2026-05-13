package materialep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apierror "github.com/augno/api/shared/errors"
)

// ExportMaterialsRequest is the request to export materials as an Excel file.
type ExportMaterialsRequest struct {
	// Optional search query.
	Query *string `query:"q"`
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Start of creation date range.
	StartDate *time.Time `query:"start_date"`
	// End of creation date range.
	EndDate *time.Time `query:"end_date"`
}

type ExportMaterialsEndpoint struct{}

func (e *ExportMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportMaterialsRequest, *httptransport.FileDownload] {
	return &apiendpoint.APIEndpoint[*ExportMaterialsRequest, *httptransport.FileDownload]{
		Title:             "Export Materials",
		Description:       "Exports all matching materials as an Excel file.",
		Method:            http.MethodGet,
		ContentType:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:             "/v1/catalog/materials/actions/export",
		Request:           &ExportMaterialsRequest{},
		Response:          &httptransport.FileDownload{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportMaterialsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(MaterialSvc).ExportMaterials
		},
	}
}
