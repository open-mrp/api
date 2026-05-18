package accountstatusep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountStatusPresenter(as *pb.AccountStatusInfo) apiresource.AccountStatus {
	if as == nil {
		return apiresource.AccountStatus{}
	}

	return apiresource.AccountStatus{
		ID:        as.Id,
		Object:    constants.ObjectTypeAccountStatus,
		Code:      constants.AccountStatusCode(as.Code),
		Name:      as.Name,
		Owner:     apiresource.SystemOwner(),
		CreatedAt: grpcutil.TimestampToTime(as.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(as.UpdatedAt),
	}
}

func AccountStatusListPresenter(ctx context.Context, resp *pb.ListAccountStatusesResponse) *apiresource.List[apiresource.AccountStatus] {
	if resp == nil {
		return apiresource.NewList[apiresource.AccountStatus](nil, apiresource.PageInfo{})
	}

	statuses := make([]apiresource.AccountStatus, len(resp.AccountStatuses))
	for i, as := range resp.AccountStatuses {
		statuses[i] = AccountStatusPresenter(as)
	}

	return apiresource.NewList(statuses, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
