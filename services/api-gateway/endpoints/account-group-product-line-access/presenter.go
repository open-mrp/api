package accountgroupproductlineaccessep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountGroupProductLineAccessPresenter(item *pb.AccountGroupProductLineAccessInfo) apiresource.AccountGroupProductLineAccess {
	if item == nil {
		return apiresource.AccountGroupProductLineAccess{}
	}

	productLines := make([]apiresource.ProductLine, len(item.ProductLines))
	for i, pl := range item.ProductLines {
		productLines[i] = apiresource.ProductLine{
			ID:     pl.Id,
			Object: constants.ObjectTypeProductLine,
			Name:   pl.Name,
		}
	}

	return apiresource.AccountGroupProductLineAccess{
		AccountGroup: &apiresource.AccountGroup{
			ID:     item.AccountGroupId,
			Object: constants.ObjectTypeAccountGroup,
			Name:   item.AccountGroupName,
		},
		Object:       constants.ObjectTypeAccountGroupProductLineAccess,
		ProductLines: apiresource.NewList(productLines, apiresource.PageInfo{}),
		CreatedAt:    grpcutil.TimestampToTime(item.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(item.UpdatedAt),
	}
}

func AccountGroupProductLineAccessListPresenter(resp *pb.ListAccountGroupProductLineAccessResponse) *apiresource.List[apiresource.AccountGroupProductLineAccess] {
	if resp == nil {
		return apiresource.NewList[apiresource.AccountGroupProductLineAccess](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.AccountGroupProductLineAccess, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = AccountGroupProductLineAccessPresenter(item)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
