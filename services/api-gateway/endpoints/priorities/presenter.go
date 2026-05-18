package priorityep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PriorityPresenter(p *pb.PriorityInfo) apiresource.Priority {
	if p == nil {
		return apiresource.Priority{}
	}

	return apiresource.Priority{
		ID:        p.Id,
		Object:    constants.ObjectTypePriority,
		Name:      p.Name,
		Code:      constants.PriorityCode(p.Code),
		Owner:     apiresource.SystemOwner(),
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func PriorityListPresenter(ctx context.Context, resp *pb.ListPrioritiesResponse) *apiresource.List[apiresource.Priority] {
	if resp == nil {
		return apiresource.NewList[apiresource.Priority](nil, apiresource.PageInfo{})
	}

	priorities := make([]apiresource.Priority, len(resp.Priorities))
	for i, p := range resp.Priorities {
		priorities[i] = PriorityPresenter(p)
	}

	return apiresource.NewList(priorities, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
