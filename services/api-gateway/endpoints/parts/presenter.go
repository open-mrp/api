package partep

import (
	itemep "github.com/open-mrp/api/services/api-gateway/endpoints/items"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
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
