package orderdiscountep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func OrderDiscountPresenter(d *pb.OrderDiscountInfo) apiresource.OrderDiscount {
	if d == nil {
		return apiresource.OrderDiscount{}
	}

	return apiresource.OrderDiscount{
		ID:           d.Id,
		Object:       constants.ObjectTypeOrderDiscount,
		Name:         d.Name,
		Code:         d.Code,
		Percentage:   d.Percentage,
		Amount:       d.Amount,
		DiscountType: constants.OrderDiscountType(d.DiscountType),
		OrderCount:   d.OrderCount,
		CreatedAt:    grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func OrderDiscountListPresenter(resp *pb.ListOrderDiscountsResponse) *apiresource.List[apiresource.OrderDiscount] {
	if resp == nil {
		return apiresource.NewList[apiresource.OrderDiscount](nil, apiresource.PageInfo{})
	}

	discounts := make([]apiresource.OrderDiscount, len(resp.OrderDiscounts))
	for i, d := range resp.OrderDiscounts {
		discounts[i] = OrderDiscountPresenter(d)
	}

	return apiresource.NewList(discounts, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
