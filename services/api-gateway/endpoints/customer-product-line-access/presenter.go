package customerproductlineaccessep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func CustomerProductLineAccessPresenter(item *pb.CustomerProductLineAccessInfo) apiresource.CustomerProductLineAccess {
	if item == nil {
		return apiresource.CustomerProductLineAccess{}
	}

	productLines := make([]apiresource.ProductLine, len(item.ProductLines))
	for i, pl := range item.ProductLines {
		productLines[i] = apiresource.ProductLine{
			ID:     pl.Id,
			Object: constants.ObjectTypeProductLine,
			Name:   pl.Name,
		}
	}

	return apiresource.CustomerProductLineAccess{
		Customer: &apiresource.Customer{
			ID:     item.CustomerId,
			Object: constants.ObjectTypeAccount,
			Name:   item.CustomerName,
			Number: item.CustomerNumber,
		},
		Object:       constants.ObjectTypeCustomerProductLineAccess,
		ProductLines: apiresource.NewList(productLines, apiresource.PageInfo{}),
		CreatedAt:    grpcutil.TimestampToTime(item.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(item.UpdatedAt),
	}
}

func CustomerProductLineAccessListPresenter(ctx context.Context, resp *pb.ListCustomerProductLineAccessResponse) *apiresource.List[apiresource.CustomerProductLineAccess] {
	if resp == nil {
		return apiresource.NewList[apiresource.CustomerProductLineAccess](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.CustomerProductLineAccess, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = CustomerProductLineAccessPresenter(item)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
