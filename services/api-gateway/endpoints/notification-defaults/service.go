package notificationdefaultsep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/notifyprefs"
	pb "github.com/open-mrp/api/shared/proto/notification"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

// Svc backs the account notification defaults endpoints.
type Svc interface {
	Get(ctx context.Context, req *RetrieveNotificationDefaultsRequest) (*apiresource.NotificationDefaults, *apierror.APIError)
	Update(ctx context.Context, req *UpdateNotificationDefaultsRequest) (*apiresource.NotificationDefaults, *apierror.APIError)
}

type SvcConfig struct {
	// MessagingClient (required) is the notification-service MessagingService gRPC client.
	MessagingClient pb.MessagingServiceClient
}

func (c *SvcConfig) validate() error {
	if c.MessagingClient == nil {
		return fmt.Errorf("notification defaults endpoint service: messaging client is required")
	}
	return nil
}

type svcImpl struct {
	messagingClient pb.MessagingServiceClient
}

var tracer = tracing.GetTracer("api-gateway.endpoints.notification-defaults.service")

func NewSvc(config *SvcConfig) Svc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &svcImpl{messagingClient: config.MessagingClient}
}

func (s *svcImpl) Get(ctx context.Context, _ *RetrieveNotificationDefaultsRequest) (*apiresource.NotificationDefaults, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, tracer, "service.notification_defaults.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationDefaultsInfo, error) {
			return s.messagingClient.GetNotificationDefaults(ctx, &pb.GetNotificationDefaultsRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return apiresource.NotificationDefaultsOf(notifyprefs.Parse(resp.SettingsJson)), nil
}

func (s *svcImpl) Update(ctx context.Context, req *UpdateNotificationDefaultsRequest) (*apiresource.NotificationDefaults, *apierror.APIError) {
	patch, apiErr := apiresource.NotificationSettingsPatch{Default: req.Default, Types: req.Types, AutoSubscribe: req.AutoSubscribe}.ToPatch("")
	if apiErr != nil {
		return nil, apiErr
	}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to encode the notification defaults.")
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, tracer, "service.notification_defaults.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationDefaultsInfo, error) {
			return s.messagingClient.UpdateNotificationDefaults(ctx, &pb.UpdateNotificationDefaultsRequest{PatchJson: patchJSON}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return apiresource.NotificationDefaultsOf(notifyprefs.Parse(resp.SettingsJson)), nil
}
