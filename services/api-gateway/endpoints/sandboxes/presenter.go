package sandboxep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SandboxPresenter(s *pb.SandboxInfo) apiresource.Sandbox {
	if s == nil {
		return apiresource.Sandbox{}
	}

	result := apiresource.Sandbox{
		ID:        s.Id,
		Object:    constants.ObjectTypeSandbox,
		Name:      s.Name,
		CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
	}

	if s.OwnerAccountId != nil && s.OwnerAccountName != nil {
		result.OwnerAccount = &apiresource.LightAccount{
			ID:     *s.OwnerAccountId,
			Object: constants.ObjectTypeAccount,
			Name:   *s.OwnerAccountName,
		}
	}

	return result
}

func SandboxListPresenter(resp *pb.ListSandboxAccountsResponse) *apiresource.List[apiresource.Sandbox] {
	if resp == nil {
		return apiresource.NewList[apiresource.Sandbox](nil, apiresource.PageInfo{})
	}

	sandboxes := make([]apiresource.Sandbox, len(resp.Sandboxes))
	for i, s := range resp.Sandboxes {
		sandboxes[i] = SandboxPresenter(s)
	}

	return apiresource.NewList(sandboxes, mapProtoPageInfo(resp.PageInfo))
}

func mapProtoPageInfo(pi *pb.PageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}
