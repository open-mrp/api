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

var carrierLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.carrier")

// CarrierServiceLevelsLimit caps the size of the service_levels list returned inline on a Carrier. Clients that need more must page through GET /v1/operations/carriers/{id}/service-levels.
const CarrierServiceLevelsLimit = 10

// LoadCarriers fetches carriers by ID via BatchGetCarriersByIDs. Each loaded carrier is mapped to a clean *apiresource.Carrier with no FK fields. FK info (owner account_id, service_level_ids preview, has_more flag) is stashed in the request-scoped LoadMeta side-table for SubField closures to read back.
func LoadCarriers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, carrierLoaderTracer, "loader.carriers.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetCarriersByIDsResponse, error) {
			return coreClient.BatchGetCarriersByIDs(ctx, &pb.BatchGetCarriersByIDsRequest{
				Ids:                ids,
				ServiceLevelsLimit: CarrierServiceLevelsLimit,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Carriers))
	for _, c := range resp.Carriers {
		out[c.Id] = carrierFromProto(c)

		// FK metadata — read back by SubField closures via LoadMeta.
		var accountID string
		if c.AccountId != nil {
			accountID = *c.AccountId
		}
		meta.Set(constants.ObjectTypeCarrier, c.Id, "owner_account_id", accountID)
		meta.Set(constants.ObjectTypeCarrier, c.Id, "service_level_ids", append([]string(nil), c.ServiceLevelIdsPreview...))
		meta.Set(constants.ObjectTypeCarrier, c.Id, "service_levels_has_more", c.ServiceLevelsHasMore)
	}
	return out, nil
}

// carrierFromProto maps a proto CarrierInfo to a clean apiresource.Carrier. Fields that depend on includes (Owner, ServiceLevels) are left nil — they only become populated when the resolver fires their SubField.Populate.
func carrierFromProto(c *pb.CarrierInfo) *apiresource.Carrier {
	var code *constants.CarrierCode
	if c.Code != nil {
		v := constants.CarrierCode(*c.Code)
		code = &v
	}
	visibility := constants.CustomerPortalVisibilityHidden
	if c.IsPortalEnabled {
		visibility = constants.CustomerPortalVisibilityVisible
	}
	carrier := &apiresource.Carrier{
		ID:                       c.Id,
		Object:                   constants.ObjectTypeCarrier,
		Name:                     c.Name,
		Code:                     code,
		AccountNumber:            c.AccountNumber,
		CustomerPortalVisibility: visibility,
		CreatedAt:                grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(c.UpdatedAt),
	}
	if c.DeletedAt != nil {
		t := grpcutil.TimestampToTime(c.DeletedAt)
		carrier.DeletedAt = &t
	}
	return carrier
}
