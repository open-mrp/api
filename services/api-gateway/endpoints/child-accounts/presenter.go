package childaccountep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ChildAccountPresenter(ca *pb.ChildAccountProto) apiresource.ChildAccount {
	if ca == nil {
		return apiresource.ChildAccount{}
	}

	return apiresource.ChildAccount{
		ID:     ca.RelationId,
		Object: constants.ObjectTypeChildAccount,
		Account: &apiresource.Account{
			ID:     ca.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   ca.AccountName,
		},
		ExternalNumber: nonEmptyStringPtr(ca.ExternalNumber),
		Email:          ca.Email,
		CreatedAt:      grpcutil.TimestampToTime(ca.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(ca.UpdatedAt),
	}
}

func ChildAccountListPresenter(ctx context.Context, resp *pb.ListChildAccountsResponse) *apiresource.List[apiresource.ChildAccount] {
	if resp == nil {
		return apiresource.NewList[apiresource.ChildAccount](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.ChildAccount, len(resp.Items))
	for i, ca := range resp.Items {
		items[i] = ChildAccountPresenter(ca)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func nonEmptyStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
