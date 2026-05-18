package accountgroupep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountGroupPresenter(ag *pb.AccountGroupInfo) apiresource.AccountGroup {
	if ag == nil {
		return apiresource.AccountGroup{}
	}

	return apiresource.AccountGroup{
		ID:               ag.Id,
		Object:           constants.ObjectTypeAccountGroup,
		Name:             ag.Name,
		Description:      ag.Description,
		CommissionPolicy: constants.CommissionPolicy(ag.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(ag.FreightPolicy),
		Type:             constants.AccountGroupType(ag.Type),
		CreatedAt:        grpcutil.TimestampToTime(ag.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(ag.UpdatedAt),
	}
}

func AccountGroupListPresenter(ctx context.Context, resp *pb.ListAccountGroupsResponse) *apiresource.List[apiresource.AccountGroup] {
	if resp == nil {
		return apiresource.NewList[apiresource.AccountGroup](nil, apiresource.PageInfo{})
	}

	groups := make([]apiresource.AccountGroup, len(resp.AccountGroups))
	for i, ag := range resp.AccountGroups {
		groups[i] = AccountGroupPresenter(ag)
	}

	return apiresource.NewList(groups, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
