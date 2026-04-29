package producttypeep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ProductTypePresenter(pt *pb.ProductTypeInfo) apiresource.ProductType {
	if pt == nil {
		return apiresource.ProductType{}
	}

	return apiresource.ProductType{
		ID:        pt.Id,
		Object:    constants.ObjectTypeProductType,
		Name:      pt.Name,
		Code:      constants.ProductTypeCode(pt.Code),
		CreatedAt: grpcutil.TimestampToTime(pt.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(pt.UpdatedAt),
	}
}

func ProductTypeListPresenter(resp *pb.ListProductTypesResponse) *apiresource.List[apiresource.ProductType] {
	if resp == nil {
		return apiresource.NewList[apiresource.ProductType](nil, apiresource.PageInfo{})
	}

	productTypes := make([]apiresource.ProductType, len(resp.ProductTypes))
	for i, pt := range resp.ProductTypes {
		productTypes[i] = ProductTypePresenter(pt)
	}

	return apiresource.NewList(productTypes, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
