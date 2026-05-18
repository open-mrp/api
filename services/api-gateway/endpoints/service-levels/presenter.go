package servicelevelep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ServiceLevelPresenter(o *pb.ServiceLevelInfo, ownerAccount *apiresource.Account) apiresource.ServiceLevel {
	if o == nil {
		return apiresource.ServiceLevel{}
	}

	visibility := constants.CustomerPortalVisibilityHidden
	if o.IsPortalEnabled {
		visibility = constants.CustomerPortalVisibilityVisible
	}

	return apiresource.ServiceLevel{
		ID:                       o.Id,
		Object:                   constants.ObjectTypeServiceLevel,
		Name:                     o.Name,
		ServiceLevelToken:        constants.ServiceLevelCode(o.Code),
		CustomerPortalVisibility: visibility,
		IsDefault:                o.IsDefault,
		Owner:                    apiresource.NewOwnerWithAccount(o.AccountId, ownerAccount),
		CreatedAt:                grpcutil.TimestampToTime(o.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(o.UpdatedAt),
	}
}

func ServiceLevelListPresenter(ctx context.Context, resp *pb.ListServiceLevelsResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.ServiceLevel] {
	if resp == nil {
		return apiresource.NewList[apiresource.ServiceLevel](nil, apiresource.PageInfo{})
	}

	serviceLevels := make([]apiresource.ServiceLevel, len(resp.ServiceLevels))
	for i, o := range resp.ServiceLevels {
		serviceLevels[i] = ServiceLevelPresenter(o, ownerAccount)
	}

	return apiresource.NewList(serviceLevels, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
