package productep

import (
	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	productlineep "github.com/augno/api/services/api-gateway/endpoints/product-lines"
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
		ID:        proto.Id,
		Object:    constants.ObjectTypeProduct,
		Type:      constants.ProductTypeCode(proto.GetProductTypeCode()),
		CreatedAt: grpcutil.TimestampToTime(proto.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(proto.UpdatedAt),
	}
	if proto.GetIsPortalReady() {
		result.PortalVisibility = constants.CustomerPortalVisibilityVisible
	} else {
		result.PortalVisibility = constants.CustomerPortalVisibilityHidden
	}

	if proto.Item != nil {
		i := itemep.ItemPresenter(proto.Item)
		result.Item = &i
	}

	if proto.ProductLine != nil {
		pl := productlineep.ProductLinePresenter(proto.ProductLine, nil)
		result.ProductLine = &pl
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
