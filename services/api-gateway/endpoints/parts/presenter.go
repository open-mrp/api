package partep

import (
	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PartPresenter(p *pb.PartInfo) apiresource.Part {
	if p == nil || p.Item == nil {
		return apiresource.Part{}
	}

	item := itemep.ItemPresenter(p.Item)
	part := apiresource.Part{
		ID:        p.Id,
		Object:    constants.ObjectTypePart,
		Item:      &item,
		CreatedAt: grpcutil.TimestampToTime(p.Item.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.Item.UpdatedAt),
	}

	return part
}

func PartListPresenter(resp *pb.ListPartsResponse) *apiresource.List[apiresource.Part] {
	if resp == nil {
		return apiresource.NewList[apiresource.Part](nil, apiresource.PageInfo{})
	}

	parts := make([]apiresource.Part, len(resp.Parts))
	for i, part := range resp.Parts {
		parts[i] = PartPresenter(part)
	}

	return apiresource.NewList(parts, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
