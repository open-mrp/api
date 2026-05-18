package addressep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func GeolocationPresenter(g *pb.GeolocationInfo) *apiresource.Geolocation {
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

func AddressPresenter(a *pb.AddressInfo) apiresource.Address {
	if a == nil {
		return apiresource.Address{}
	}

	return apiresource.Address{
		ID:          a.Id,
		Object:      constants.ObjectTypeAddress,
		Name:        a.Name,
		Phone:       a.Phone,
		Email:       a.Email,
		Type:        addressTypeFromDropShip(a.IsDropShip),
		Geolocation: GeolocationPresenter(a.Geolocation),
		CreatedAt:   grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func addressTypeFromDropShip(isDropShip bool) constants.AddressType {
	if isDropShip {
		return constants.AddressTypeDropShip
	}
	return constants.AddressTypeStandard
}

func AddressListPresenter(ctx context.Context, resp *pb.ListAddressesResponse) *apiresource.List[apiresource.Address] {
	if resp == nil {
		return apiresource.NewList[apiresource.Address](nil, apiresource.PageInfo{})
	}

	addresses := make([]apiresource.Address, len(resp.Addresses))
	for i, a := range resp.Addresses {
		addresses[i] = AddressPresenter(a)
	}

	return apiresource.NewList(addresses, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
