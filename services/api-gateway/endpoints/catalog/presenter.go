package catalogep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func CatalogProductLineListPresenter(ctx context.Context, resp *pb.ListCatalogProductLinesResponse) *apiresource.List[apiresource.CatalogProductLine] {
	if resp == nil {
		return apiresource.NewList[apiresource.CatalogProductLine](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.CatalogProductLine, len(resp.ProductLines))
	for i, pl := range resp.ProductLines {
		items[i] = apiresource.CatalogProductLine{
			ID:     pl.Id,
			Object: constants.ObjectTypeCatalogProductLine,
			Name:   pl.Name,
		}
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
