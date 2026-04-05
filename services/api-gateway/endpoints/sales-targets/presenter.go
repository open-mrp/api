package salestargetep

import (
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SalesTargetPresenter(st *pb.SalesTargetProto) apiresource.SalesTarget {
	if st == nil {
		return apiresource.SalesTarget{}
	}

	startAt, _ := time.Parse(time.RFC3339, st.StartDate)
	endAt, _ := time.Parse(time.RFC3339, st.EndDate)
	createdAt, _ := time.Parse(time.RFC3339, st.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, st.UpdatedAt)

	result := apiresource.SalesTarget{
		ID:      st.Id,
		Object:  constants.ObjectTypeSalesTarget,
		StartAt: startAt,
		EndAt:   endAt,
		SalesRep: &apiresource.User{
			ID:     st.SalesRepId,
			Object: constants.ObjectTypeUser,
		},
		Amount: &apiresource.Quantity{
			ID:     st.AmountId,
			Object: constants.ObjectTypeQuantity,
			Value:  st.AmountValue,
			Unit: &apiresource.Unit{
				ID:     st.AmountUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return result
}

func SalesTargetListPresenter(resp *pb.ListSalesTargetsResponse) *apiresource.List[apiresource.SalesTarget] {
	if resp == nil {
		return apiresource.NewList[apiresource.SalesTarget](nil, apiresource.PageInfo{})
	}

	targets := make([]apiresource.SalesTarget, len(resp.SalesTargets))
	for i, st := range resp.SalesTargets {
		targets[i] = SalesTargetPresenter(st)
	}

	return apiresource.NewList(targets, apiresource.PageInfo{})
}
