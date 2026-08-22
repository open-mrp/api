package itemep

import (
	"context"
	"net/http"

	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to export items.
type ExportItemsRequest struct{}

// Downloads every item in your account, with its category and on-hand inventory, as an Excel workbook named `items.xlsx`.
//
// The export takes no filters and is not paginated: it always covers the whole catalog, one row per item, ordered by SKU.
type ExportItemsEndpoint struct{}

func (e *ExportItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportItemsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportItemsRequest, *httptransport.FileDownload]{
		Title:               "Export Items",
		Method:              http.MethodGet,
		ContentType:         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:               "/v1/catalog/items/actions/export",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportItemsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(ItemSvc).ExportItems
		},
	})
}
