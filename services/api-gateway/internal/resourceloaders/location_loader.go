package resourceloaders

import (
	"context"
	"net/url"

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

var locationLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.location")

func LoadLocations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, locationLoaderTracer, "loader.locations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetLocationsByIDsResponse, error) {
			return coreClient.BatchGetLocationsByIDs(ctx, &pb.BatchGetLocationsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Locations))
	for _, loc := range resp.Locations {
		out[loc.Id] = locationFromProto(loc)

		meta.Set(constants.ObjectTypeLocation, loc.Id, "parent_id", loc.GetParentId())
		meta.Set(constants.ObjectTypeLocation, loc.Id, "address_id", loc.GetAddressId())

		childIDs := make([]string, len(loc.Children))
		for i, c := range loc.Children {
			childIDs[i] = c.Id
		}
		meta.Set(constants.ObjectTypeLocation, loc.Id, "child_ids", childIDs)
	}
	return out, nil
}

// LocationChildrenListPath is the location list narrowed to one location's children, where its embedded children
// continue.
func LocationChildrenListPath(locationID string) string {
	return "/v1/operations/locations?" + url.Values{"parent_ids[]": {locationID}}.Encode()
}

func locationFromProto(loc *pb.LocationInfo) *apiresource.Location {
	return &apiresource.Location{
		ID:        loc.Id,
		Object:    constants.ObjectTypeLocation,
		Name:      loc.Name,
		TypeCode:  constants.LocationTypeCode(loc.TypeCode),
		Status:    constants.ArchiveStatus(loc.Status),
		CreatedAt: grpcutil.TimestampToTime(loc.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(loc.UpdatedAt),
	}
}

// StashSiteID records a document's site for the `site` include that SiteSub registers on it. Call it from the
// document's loader for every row; an empty siteID (no site) is skipped.
func StashSiteID(ctx context.Context, parentType constants.ObjectType, parentID, siteID string) {
	if siteID != "" {
		include.GetLoadMeta(ctx).Set(parentType, parentID, SiteMetaKey, siteID)
	}
}

// SiteMetaKey is the load-meta key a document's site ID is stashed under.
const SiteMetaKey = "site_id"
