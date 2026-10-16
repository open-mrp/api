package resourceloaders

import (
	"context"
	"strconv"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var accountGroupLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_group")

// LoadAccountGroups fetches account groups by ID via BatchGetAccountGroupsByIDs. AccountGroup exposes no expandable sub-resources (not even an Owner) so no LoadMeta needs to be stashed.
// AccountGroupMetaPriceListIDs is the load meta key holding a group's price list IDs.
const AccountGroupMetaPriceListIDs = "price_list_ids"

func LoadAccountGroups(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupLoaderTracer, "loader.account_groups.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountGroupsByIDsResponse, error) {
			return coreClient.BatchGetAccountGroupsByIDs(ctx, &pb.BatchGetAccountGroupsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.AccountGroups))
	for _, ag := range resp.AccountGroups {
		out[ag.Id] = accountGroupFromProto(ag)
		meta.Set(constants.ObjectTypeAccountGroup, ag.Id, AccountGroupMetaPriceListIDs, ag.PriceListIds)
	}
	return out, nil
}

func accountGroupFromProto(ag *pb.AccountGroupInfo) *apiresource.AccountGroup {
	return &apiresource.AccountGroup{
		ID:               ag.Id,
		Object:           constants.ObjectTypeAccountGroup,
		Name:             ag.Name,
		Description:      ag.Description,
		CommissionPolicy: new(constants.CommissionPolicy(ag.CommissionPolicy)),
		FreightPolicy:    constants.FreightPolicy(ag.FreightPolicy),
		Type:             constants.AccountGroupType(ag.Type),
		Defaults:         &apiresource.AccountGroupDefaults{LeadTime: DaysQuantity(ag.DefaultLeadTimeDays)},
		CreatedAt:        grpcutil.TimestampToTime(ag.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(ag.UpdatedAt),
	}
}

// DaysQuantity is a lead time stored in whole days as a Quantity in the day unit, or nil when none is set.
func DaysQuantity(days *int32) *apiresource.Quantity {
	if days == nil {
		return nil
	}
	return &apiresource.Quantity{Value: strconv.FormatInt(int64(*days), 10), Unit: apiresource.UnitStub(DayUnitID)}
}

// DayUnitID is the system unit every lead time is counted in.
const DayUnitID = "day"
