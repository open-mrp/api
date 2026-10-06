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
// The export takes no filters and is not paginated: it covers the same items the inventory list does, one row per item, ordered by SKU. Non-sale products — the service, shipping, tax, credit and return products that carry charges on orders — are left out. On hand is available stock net of what has been allocated, converted into the base unit of the item's category, and the Unit column names that unit by its abbreviation.
//
// A catalog of more than 50,000 items is refused with a validation error rather than exported partially.
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
