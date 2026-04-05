package territoryep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SalesRepPresenter(sr *pb.TerritoryAccountUserInfo) *apiresource.AccountUser {
	if sr == nil {
		return nil
	}
	return &apiresource.AccountUser{
		ID:     sr.Id,
		Object: constants.ObjectTypeAccountUser,
		Name:   sr.Name,
		Email:  sr.Email,
	}
}

func ProductLinePresenter(pl *pb.TerritoryProductLineInfo) *apiresource.ProductLine {
	if pl == nil {
		return nil
	}
	return &apiresource.ProductLine{
		ID:     pl.Id,
		Object: constants.ObjectTypeProductLine,
		Name:   pl.Name,
	}
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

func TerritoryListPresenter(resp *pb.ListTerritoriesResponse) *apiresource.List[apiresource.Territory] {
	if resp == nil {
		return apiresource.NewList[apiresource.Territory](nil, apiresource.PageInfo{})
	}

	territories := make([]apiresource.Territory, len(resp.Territories))
	for i, t := range resp.Territories {
		territories[i] = TerritoryPresenter(t)
	}

	return apiresource.NewList(territories, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
