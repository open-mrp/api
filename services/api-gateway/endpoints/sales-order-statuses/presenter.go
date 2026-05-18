package salesorderstatusep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SalesOrderStatusPresenter(s *pb.SalesOrderStatusInfo) apiresource.SalesOrderStatus {
	if s == nil {
		return apiresource.SalesOrderStatus{}
	}

	return apiresource.SalesOrderStatus{
		ID:        s.Id,
		Object:    constants.ObjectTypeSalesOrderStatus,
		Code:      constants.SalesOrderStatusCode(s.Code),
		Name:      s.Name,
		Owner:     apiresource.SystemOwner(),
		CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func SalesOrderStatusListPresenter(ctx context.Context, resp *pb.ListSalesOrderStatusesResponse) *apiresource.List[apiresource.SalesOrderStatus] {
	if resp == nil {
		return apiresource.NewList[apiresource.SalesOrderStatus](nil, apiresource.PageInfo{})
	}

	statuses := make([]apiresource.SalesOrderStatus, len(resp.SalesOrderStatuses))
	for i, s := range resp.SalesOrderStatuses {
		statuses[i] = SalesOrderStatusPresenter(s)
	}

	return apiresource.NewList(statuses, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
