package shipmentep

import (
	"context"
	"testing"

	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeRateShopCore struct {
	pb.CoreShippingServiceClient
	resp *pb.RateShopResponse
}

func (f *fakeRateShopCore) RateShop(context.Context, *pb.RateShopRequest, ...grpc.CallOption) (*pb.RateShopResponse, error) {
	return f.resp, nil
}

// The carriers and service levels come back with the quote; nothing is loaded again under carriers:read, so a nil core client in the loaders would panic if it were.
func TestRateShop_OptionsCarryTheCarriersAndServiceLevelsReadWithTheQuote(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	token := "fedex_ground"
	core := &fakeRateShopCore{resp: &pb.RateShopResponse{
		Options:       []*pb.RateShopOptionInfo{{CarrierId: "cr_truck", CarrierName: "Truck", ServiceLevelId: "crop_ground", ServiceLevelName: "Ground", Rate: 12.5}},
		Carriers:      []*pb.CarrierInfo{{Id: "cr_truck", Name: "Truck", IsPortalEnabled: true, CreatedAt: now, UpdatedAt: now}},
		ServiceLevels: []*pb.ServiceLevelInfo{{Id: "crop_ground", Name: "Ground", ServiceLevelToken: &token, CarrierId: "cr_truck", CreatedAt: now, UpdatedAt: now}},
	}}
	svc := NewShipmentSvc(&ShipmentSvcConfig{CoreClient: core})

	result, apiErr := svc.RateShop(context.Background(), &RateShopRequest{
		ToAddress: apirequest.AddressInput{Name: "Destination", Country: "US"},
		Parcels:   []ParcelInput{{Weight: 5, Length: 12, Width: 8, Height: 6}},
	})
	if apiErr != nil {
		t.Fatalf("RateShop() error = %v", apiErr)
	}
	if len(result.Options.Data) != 1 {
		t.Fatalf("options = %+v", result.Options.Data)
	}
	option := result.Options.Data[0]
	if option.Carrier == nil || option.Carrier.ID != "cr_truck" || option.Carrier.CustomerPortalVisibility != constants.CustomerPortalVisibilityVisible || option.Carrier.CreatedAt.IsZero() {
		t.Errorf("carrier = %+v, want the one read with the quote", option.Carrier)
	}
	if option.ServiceLevel == nil || option.ServiceLevel.ServiceLevelToken != constants.ServiceLevelCode(token) || option.ServiceLevel.CreatedAt.IsZero() {
		t.Errorf("service level = %+v, want the one read with the quote", option.ServiceLevel)
	}
}
