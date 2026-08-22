package notificationpreferenceep

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationPreferenceSvc backs the notification-preference endpoints via the notification-service
// MessagingService gRPC client.
type NotificationPreferenceSvc interface {
	ListNotificationPreferences(ctx context.Context, req *ListNotificationPreferencesRequest) (*apiresource.List[apiresource.NotificationPreference], *apierror.APIError)
	UpsertNotificationPreference(ctx context.Context, req *UpsertNotificationPreferenceRequest) (*apiresource.NotificationPreference, *apierror.APIError)
}

type NotificationPreferenceSvcConfig struct {
	// MessagingClient (required) is the notification-service MessagingService gRPC client.
	MessagingClient pb.MessagingServiceClient
}

type notificationPreferenceSvcImpl struct {
	messagingClient pb.MessagingServiceClient
}

var notificationPreferenceSvcTracer = tracing.GetTracer("api-gateway.endpoints.notification-preferences.service")

func (c *NotificationPreferenceSvcConfig) validate() error {
	if c.MessagingClient == nil {
		return fmt.Errorf("notification preference endpoint service: messaging client is required")
	}
	return nil
}

func NewNotificationPreferenceSvc(config *NotificationPreferenceSvcConfig) NotificationPreferenceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &notificationPreferenceSvcImpl{messagingClient: config.MessagingClient}
}

func (s *notificationPreferenceSvcImpl) ListNotificationPreferences(ctx context.Context, _ *ListNotificationPreferencesRequest) (*apiresource.List[apiresource.NotificationPreference], *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationPreferenceSvcTracer, "service.notification_preferences.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListNotificationPreferencesResponse, error) {
			return s.messagingClient.ListNotificationPreferences(ctx, &pb.ListNotificationPreferencesRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	items := make([]apiresource.NotificationPreference, 0, len(resp.Preferences))
	for _, p := range resp.Preferences {
		items = append(items, notificationPreferenceFromProto(p))
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (s *notificationPreferenceSvcImpl) UpsertNotificationPreference(ctx context.Context, req *UpsertNotificationPreferenceRequest) (*apiresource.NotificationPreference, *apierror.APIError) {
	category := ""
	if c, ok := req.Category.Value(); ok {
		category = c
	}
	digest := string(constants.NotificationDigestInstant)
	if d, ok := req.Digest.Value(); ok {
		digest = string(d)
	}
	pbReq := &pb.UpsertNotificationPreferenceRequest{
		Category:     category,
		InAppEnabled: req.InAppEnabled,
		EmailEnabled: req.EmailEnabled,
		PushEnabled:  req.PushEnabled,
		Digest:       digest,
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationPreferenceSvcTracer, "service.notification_preferences.upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationPreferenceInfo, error) {
			return s.messagingClient.UpsertNotificationPreference(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := notificationPreferenceFromProto(resp)
	return &result, nil
}

func notificationPreferenceFromProto(p *pb.NotificationPreferenceInfo) apiresource.NotificationPreference {
	if p == nil {
		return apiresource.NotificationPreference{}
	}
	result := apiresource.NotificationPreference{
		ID:           p.Id,
		Object:       constants.ObjectTypeNotificationPreference,
		InAppEnabled: p.InAppEnabled,
		EmailEnabled: p.EmailEnabled,
		PushEnabled:  p.PushEnabled,
		Digest:       constants.NotificationDigest(p.Digest),
		CreatedAt:    tsToTime(p.CreatedAt),
		UpdatedAt:    tsToTime(p.UpdatedAt),
	}
	// An empty category is the global default, surfaced to clients as null.
	if p.Category != "" {
		category := p.Category
		result.Category = &category
	}
	return result
}

func tsToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
