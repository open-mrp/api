package resourceloaders

import (
	"context"
	"time"

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

var addressLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.address")

// LoadAddresses fetches addresses by ID via BatchGetAddressesByIDs.
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
		out[a.Id] = AddressFromProto(ctx, a)
	}
	return out, nil
}

// AddressFromProto builds an address and stashes its receive calendar's ID for the receive_calendar include.
func AddressFromProto(ctx context.Context, a *pb.AddressInfo) *apiresource.Address {
	addr := AddressFromInfo(a)
	if addr != nil {
		StashReceiveCalendar(ctx, addr.ID, a.ReceiveCalendarId)
	}
	return addr
}

// AddressFromInfo builds an address without making its receive calendar includable. nil stays nil.
func AddressFromInfo(a *pb.AddressInfo) *apiresource.Address {
	if a == nil {
		return nil
	}
	parts := AddressParts{
		ID:         a.Id,
		Name:       a.Name,
		IsDropShip: a.IsDropShip,
		Phone:      a.Phone,
		Email:      a.Email,
		CreatedAt:  grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(a.UpdatedAt),
	}
	if g := a.Geolocation; g != nil {
		parts.Line1, parts.Line2, parts.City, parts.State, parts.PostalCode, parts.Country = g.StreetLine_1, g.StreetLine_2, g.Locality, g.State, g.PostalCode, g.Country
	}
	return NewAddress(parts)
}

// AddressParts are the fields of an address, whichever RPC carried it.
type AddressParts struct {
	ID, Name                              string
	IsDropShip                            bool
	Line1, Line2, City, State, PostalCode *string
	Country                               string
	Phone, Email                          *string
	CreatedAt, UpdatedAt                  time.Time
}

// NewAddress is the address response for p. receive_calendar stays nil until included.
func NewAddress(p AddressParts) *apiresource.Address {
	addressType := constants.AddressTypeStandard
	if p.IsDropShip {
		addressType = constants.AddressTypeDropShip
	}
	return &apiresource.Address{
		ID:         p.ID,
		Object:     constants.ObjectTypeAddress,
		Name:       p.Name,
		Type:       addressType,
		Line1:      p.Line1,
		Line2:      p.Line2,
		City:       p.City,
		State:      p.State,
		PostalCode: p.PostalCode,
		Country:    p.Country,
		Phone:      p.Phone,
		Email:      p.Email,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// StashReceiveCalendar records the calendar an address names, for its receive_calendar include. nil records nothing.
func StashReceiveCalendar(ctx context.Context, addressID string, calendarID *string) {
	if calendarID != nil && *calendarID != "" {
		include.GetLoadMeta(ctx).Set(constants.ObjectTypeAddress, addressID, "receive_calendar_id", *calendarID)
	}
}
