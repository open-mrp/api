package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var serviceLevelLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.service_level")

// LoadServiceLevels fetches service levels by ID via BatchGetServiceLevelsByIDs. It builds clean *apiresource.ServiceLevel values and stashes each service level's owner_account_id in LoadMeta for the owner/owner.account sub-fields.
func LoadServiceLevels(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelLoaderTracer, "loader.service_levels.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetServiceLevelsByIDsResponse, error) {
			return coreClient.BatchGetServiceLevelsByIDs(ctx, &pb.BatchGetServiceLevelsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ServiceLevels))
	for _, sl := range resp.ServiceLevels {
		out[sl.Id] = serviceLevelFromProto(sl)
		var accountID string
		if sl.AccountId != nil {
			accountID = *sl.AccountId
		}
		meta.Set(constants.ObjectTypeServiceLevel, sl.Id, "owner_account_id", accountID)
	}
	return out, nil
}

func serviceLevelFromProto(sl *pb.ServiceLevelInfo) *apiresource.ServiceLevel {
	visibility := constants.CustomerPortalVisibilityHidden
	if sl.IsPortalEnabled {
		visibility = constants.CustomerPortalVisibilityVisible
	}
	var token constants.ServiceLevelCode
	if sl.ServiceLevelToken != nil {
		token = constants.ServiceLevelCode(*sl.ServiceLevelToken)
	} else {
		token = constants.ServiceLevelCode(sl.Code)
	}
	return &apiresource.ServiceLevel{
		ID:                       sl.Id,
		Object:                   constants.ObjectTypeServiceLevel,
		Name:                     sl.Name,
		ServiceLevelToken:        token,
		CustomerPortalVisibility: visibility,
		IsDefault:                sl.IsDefault,
		CreatedAt:                grpcutil.TimestampToTime(sl.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(sl.UpdatedAt),
	}
}
