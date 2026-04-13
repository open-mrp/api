package productlineep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ProductLinePresenter(pl *pb.ProductLineInfo, ownerAccount *apiresource.Account) apiresource.ProductLine {
	if pl == nil {
		return apiresource.ProductLine{}
	}

	result := apiresource.ProductLine{
		ID:               pl.Id,
		Object:           constants.ObjectTypeProductLine,
		Name:             pl.Name,
		Description:      pl.Description,
		Notes:            pl.Notes,
		CommissionPolicy: constants.CommissionPolicy(pl.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(pl.FreightPolicy),
		Owner:            apiresource.NewOwnerWithAccount(pl.AccountId, ownerAccount),
		CreatedAt:        grpcutil.TimestampToTime(pl.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(pl.UpdatedAt),
	}

	if pl.UnitGroup != nil {
		result.UnitGroup = &apiresource.UnitGroup{
			ID:     pl.UnitGroup.Id,
			Object: constants.ObjectTypeUnitGroup,
			Name:   pl.UnitGroup.Name,
			Type:   constants.UnitType(pl.UnitGroup.Type),
		}
	}

	return result
}

func ProductLineListPresenter(resp *pb.ListProductLinesResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.ProductLine] {
	if resp == nil {
		return apiresource.NewList[apiresource.ProductLine](nil, apiresource.PageInfo{})
	}

	productLines := make([]apiresource.ProductLine, len(resp.ProductLines))
	for i, pl := range resp.ProductLines {
		productLines[i] = ProductLinePresenter(pl, ownerAccount)
	}

	return apiresource.NewList(productLines, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
