package productep

import (
	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	productlineep "github.com/augno/api/services/api-gateway/endpoints/product-lines"
	producttypeep "github.com/augno/api/services/api-gateway/endpoints/product-types"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ProductPresenter(proto *pb.ProductFullInfo) apiresource.Product {
	if proto == nil {
		return apiresource.Product{}
	}

	result := apiresource.Product{
		ID:            proto.Id,
		Object:        constants.ObjectTypeProduct,
		IsPortalReady: proto.IsPortalReady,
		CreatedAt:     grpcutil.TimestampToTime(proto.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTime(proto.UpdatedAt),
	}

	if proto.Item != nil {
		i := itemep.ItemPresenter(proto.Item)
		result.Item = &i
	}

	if proto.ProductLine != nil {
		pl := productlineep.ProductLinePresenter(proto.ProductLine, nil)
		result.ProductLine = &pl
	}

	if proto.ProductType != nil {
		pt := producttypeep.ProductTypePresenter(proto.ProductType)
		result.ProductType = &pt
	}

	return result
}

func ProductListPresenter(resp *pb.ListProductsFullResponse) *apiresource.List[apiresource.Product] {
	if resp == nil {
		return apiresource.NewList[apiresource.Product](nil, apiresource.PageInfo{})
	}

	products := make([]apiresource.Product, len(resp.Products))
	for i, p := range resp.Products {
		products[i] = ProductPresenter(p)
	}

	return apiresource.NewList(products, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func ValidateProductsPresenter(resp *pb.ValidateProductsResponse) *apiresource.ValidateProductsResponse {
	if resp == nil {
		return &apiresource.ValidateProductsResponse{
			Object:   constants.ObjectTypeMap,
			Products: map[string]*apiresource.Product{},
		}
	}

	products := make(map[string]*apiresource.Product, len(resp.Products))
	for key, proto := range resp.Products {
		p := ProductPresenter(proto)
		products[key] = &p
	}

	return &apiresource.ValidateProductsResponse{
		Object:   constants.ObjectTypeMap,
		Products: products,
	}
}
