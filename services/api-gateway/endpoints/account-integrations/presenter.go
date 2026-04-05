package accountintegrationep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountIntegrationPresenter(ai *pb.AccountIntegrationInfo) apiresource.AccountIntegration {
	if ai == nil {
		return apiresource.AccountIntegration{}
	}

	return apiresource.AccountIntegration{
		ID:              ai.Id,
		Object:          constants.ObjectTypeAccountIntegration,
		Name:            ai.Name,
		IntegrationCode: constants.IntegrationCode(ai.IntegrationCode),
		IsActive:        ai.IsActive,
		CreatedAt:       grpcutil.TimestampToTime(ai.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(ai.UpdatedAt),
	}
}

func AccountIntegrationListPresenter(resp *pb.ListAccountIntegrationsResponse) *apiresource.List[apiresource.AccountIntegration] {
	if resp == nil {
		return apiresource.NewList[apiresource.AccountIntegration](nil, apiresource.PageInfo{})
	}

	integrations := make([]apiresource.AccountIntegration, len(resp.AccountIntegrations))
	for i, ai := range resp.AccountIntegrations {
		integrations[i] = AccountIntegrationPresenter(ai)
	}

	return apiresource.NewList(integrations, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
