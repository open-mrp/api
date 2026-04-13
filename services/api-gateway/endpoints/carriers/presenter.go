package carrierep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func CarrierPresenter(c *pb.CarrierInfo, ownerAccount *apiresource.Account) apiresource.Carrier {
	if c == nil {
		return apiresource.Carrier{}
	}

	var serviceLevels *apiresource.List[apiresource.ServiceLevel]
	if c.ServiceLevels != nil {
		items := make([]apiresource.ServiceLevel, len(c.ServiceLevels))
		for i, o := range c.ServiceLevels {
			items[i] = ServiceLevelPresenter(o)
		}
		serviceLevels = apiresource.NewList(items, apiresource.PageInfo{})
	}

	carrierVisibility := constants.CustomerPortalVisibilityHidden
	if c.IsPortalEnabled {
		carrierVisibility = constants.CustomerPortalVisibilityVisible
	}

	carrier := apiresource.Carrier{
		ID:                       c.Id,
		Object:                   constants.ObjectTypeCarrier,
		Name:                     c.Name,
		Code:                     carrierCodePtr(c.Code),
		AccountNumber:            c.AccountNumber,
		CustomerPortalVisibility: carrierVisibility,
		Owner:                    apiresource.NewOwnerWithAccount(c.AccountId, ownerAccount),
		ServiceLevels:            serviceLevels,
		CreatedAt:                grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(c.UpdatedAt),
	}
	if c.DeletedAt != nil {
		t := grpcutil.TimestampToTime(c.DeletedAt)
		carrier.DeletedAt = &t
	}
	return carrier
}

func ServiceLevelPresenter(o *pb.ServiceLevelInfo) apiresource.ServiceLevel {
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
		CreatedAt:                grpcutil.TimestampToTime(o.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(o.UpdatedAt),
	}
}

func carrierCodePtr(s *string) *constants.CarrierCode {
	if s == nil {
		return nil
	}
	c := constants.CarrierCode(*s)
	return &c
}

func CarrierListPresenter(resp *pb.ListCarriersResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.Carrier] {
	if resp == nil {
		return apiresource.NewList[apiresource.Carrier](nil, apiresource.PageInfo{})
	}

	carriers := make([]apiresource.Carrier, len(resp.Carriers))
	for i, c := range resp.Carriers {
		carriers[i] = CarrierPresenter(c, ownerAccount)
	}

	return apiresource.NewList(carriers, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
