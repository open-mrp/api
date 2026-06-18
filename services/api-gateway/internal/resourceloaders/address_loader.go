package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var addressLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.address")

// LoadAddresses fetches addresses by ID via BatchGetAddressesByIDs. Geolocation is an inline sub-record (always present, not expandable) so it's populated directly from the proto response. Address exposes no expandable sub-resources so no LoadMeta is needed.
func LoadAddresses(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, addressLoaderTracer, "loader.addresses.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAddressesByIDsResponse, error) {
			return coreClient.BatchGetAddressesByIDs(ctx, &pb.BatchGetAddressesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Addresses))
	for _, a := range resp.Addresses {
		out[a.Id] = addressFromProto(a)
	}
	return out, nil
}

func addressFromProto(a *pb.AddressInfo) *apiresource.Address {
	addressType := constants.AddressTypeStandard
	if a.IsDropShip {
		addressType = constants.AddressTypeDropShip
	}
	return &apiresource.Address{
		ID:          a.Id,
		Object:      constants.ObjectTypeAddress,
		Name:        a.Name,
		Phone:       a.Phone,
		Email:       a.Email,
		Type:        addressType,
		Geolocation: geolocationFromProto(a.Geolocation),
		CreatedAt:   grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func geolocationFromProto(g *pb.GeolocationInfo) *apiresource.Geolocation {
	if g == nil {
		return nil
	}
	return &apiresource.Geolocation{
		ID:          g.Id,
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: g.StreetLine_1,
		StreetLine2: g.StreetLine_2,
		Locality:    g.Locality,
		State:       g.State,
		PostalCode:  g.PostalCode,
		Country:     g.Country,
	}
}
