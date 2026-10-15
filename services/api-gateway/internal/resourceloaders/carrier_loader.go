package resourceloaders

import (
	"context"

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

	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Carriers))
	for _, c := range resp.Carriers {
		out[c.Id] = CarrierFromProto(c)

		// FK metadata — read back by SubField closures via LoadMeta.
		var accountID string
		if c.AccountId != nil {
			accountID = *c.AccountId
		}
		meta.Set(constants.ObjectTypeCarrier, c.Id, "owner_account_id", accountID)
		meta.Set(constants.ObjectTypeCarrier, c.Id, "service_level_ids", append([]string(nil), c.ServiceLevelIdsPreview...))
		meta.Set(constants.ObjectTypeCarrier, c.Id, "service_levels_has_more", c.ServiceLevelsHasMore)
		var defaultServiceLevelID string
		if c.DefaultServiceLevelId != nil {
			defaultServiceLevelID = *c.DefaultServiceLevelId
		}
		meta.Set(constants.ObjectTypeCarrier, c.Id, "default_service_level_id", defaultServiceLevelID)
	}
	return out, nil
}

// CarrierFromProto maps a proto CarrierInfo to a clean apiresource.Carrier. Fields that depend on includes (DefaultServiceLevel, Owner, ServiceLevels) are left nil — they only become populated when the resolver fires their SubField.Populate.
func CarrierFromProto(c *pb.CarrierInfo) *apiresource.Carrier {
	return &apiresource.Carrier{
		ID:               c.Id,
		Object:           constants.ObjectTypeCarrier,
		Name:             c.Name,
		Type:             constants.CarrierType(c.Type),
		SCAC:             c.Scac,
		AccountNumber:    c.AccountNumber,
		PortalVisibility: carrierPortalVisibility(c.IsPortalEnabled),
		Status:           constants.ArchiveStatus(c.Status),
		CreatedAt:        grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(c.UpdatedAt),
	}
}

func carrierPortalVisibility(enabled bool) constants.CustomerPortalVisibility {
	if enabled {
		return constants.CustomerPortalVisibilityVisible
	}
	return constants.CustomerPortalVisibilityHidden
}

// CarrierServiceLevelsListPath is the list a carrier's embedded service levels continue in.
func CarrierServiceLevelsListPath(carrierID string) string {
	return "/v1/operations/carriers/" + carrierID + "/service-levels"
}
