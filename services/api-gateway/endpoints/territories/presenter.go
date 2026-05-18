package territoryep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SalesRepPresenter(sr *pb.TerritoryAccountUserInfo) *apiresource.AccountUser {
	if sr == nil {
		return nil
	}
	au := &apiresource.AccountUser{
		ID:     sr.Id,
		Object: constants.ObjectTypeAccountUser,
		Name:   sr.Name,
		Email:  sr.Email,
		Status: constants.AccountUserStatus(sr.GetStatus()),
	}
	if sr.CreatedAt != nil {
		au.CreatedAt = sr.CreatedAt.AsTime()
	}
	if sr.UpdatedAt != nil {
		au.UpdatedAt = sr.UpdatedAt.AsTime()
	}
	return au
}

func ProductLinePresenter(pl *pb.TerritoryProductLineInfo) *apiresource.ProductLine {
	if pl == nil {
		return nil
	}
	resource := &apiresource.ProductLine{
		ID:               pl.Id,
		Object:           constants.ObjectTypeProductLine,
		Name:             pl.Name,
		CommissionPolicy: constants.CommissionPolicyFromBool(pl.GetIsCommissionExempt()),
		FreightPolicy:    constants.FreightPolicyFromBool(pl.GetIsFreightExempt()),
	}
	if pl.CreatedAt != nil {
		resource.CreatedAt = pl.CreatedAt.AsTime()
	}
	if pl.UpdatedAt != nil {
		resource.UpdatedAt = pl.UpdatedAt.AsTime()
	}
	return resource
}

func TerritoryPresenter(t *pb.TerritoryInfo) apiresource.Territory {
	if t == nil {
		return apiresource.Territory{}
	}

	return apiresource.Territory{
		ID:           t.Id,
		Object:       constants.ObjectTypeTerritory,
		State:        t.State,
		StartZipcode: t.StartZipcode,
		EndZipcode:   t.EndZipcode,
		SalesRep:     SalesRepPresenter(t.SalesRep),
		ProductLine:  ProductLinePresenter(t.ProductLine),
		CreatedAt:    grpcutil.TimestampToTime(t.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(t.UpdatedAt),
	}
}

func TerritoryListPresenter(ctx context.Context, resp *pb.ListTerritoriesResponse) *apiresource.List[apiresource.Territory] {
	if resp == nil {
		return apiresource.NewList[apiresource.Territory](nil, apiresource.PageInfo{})
	}

	territories := make([]apiresource.Territory, len(resp.Territories))
	for i, t := range resp.Territories {
		territories[i] = TerritoryPresenter(t)
	}

	return apiresource.NewList(territories, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
