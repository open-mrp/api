package notificationpreferenceep

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/notifyprefs"
	pb "github.com/open-mrp/api/shared/proto/notification"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationPreferenceSvc backs the notification-preference endpoints via the notification-service
// MessagingService gRPC client.
type NotificationPreferenceSvc interface {
	ListNotificationPreferences(ctx context.Context, req *ListNotificationPreferencesRequest) (*apiresource.List[apiresource.NotificationPreference], *apierror.APIError)
	UpsertNotificationPreference(ctx context.Context, req *UpsertNotificationPreferenceRequest) (*apiresource.NotificationPreference, *apierror.APIError)
	GetNotificationPreferences(ctx context.Context, req *RetrieveNotificationPreferencesRequest) (*apiresource.NotificationPreferences, *apierror.APIError)
	UpdateNotificationPreferences(ctx context.Context, req *UpdateNotificationPreferencesRequest) (*apiresource.NotificationPreferences, *apierror.APIError)
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

func (s *notificationPreferenceSvcImpl) GetNotificationPreferences(ctx context.Context, _ *RetrieveNotificationPreferencesRequest) (*apiresource.NotificationPreferences, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationPreferenceSvcTracer, "service.notification_preferences.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationPreferencesInfo, error) {
			return s.messagingClient.GetNotificationPreferences(ctx, &pb.GetNotificationPreferencesRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return preferencesFromProto(resp), nil
}

func (s *notificationPreferenceSvcImpl) UpdateNotificationPreferences(ctx context.Context, req *UpdateNotificationPreferencesRequest) (*apiresource.NotificationPreferences, *apierror.APIError) {
	patch, apiErr := req.SettingsPatch().ToPatch("")
	if apiErr != nil {
		return nil, apiErr
	}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to encode the preferences.")
	}
	pbReq := &pb.UpdateNotificationPreferencesRequest{PatchJson: patchJSON, PausedUntilSet: req.PausedUntil.WasProvided()}
	if t, ok := req.PausedUntil.Value(); ok {
		pbReq.PausedUntil = timestamppb.New(t)
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, notificationPreferenceSvcTracer, "service.notification_preferences.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.NotificationPreferencesInfo, error) {
			return s.messagingClient.UpdateNotificationPreferences(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return preferencesFromProto(resp), nil
}

func preferencesFromProto(p *pb.NotificationPreferencesInfo) *apiresource.NotificationPreferences {
	var paused *time.Time
	if p.PausedUntil != nil {
		t := p.PausedUntil.AsTime()
		paused = &t
	}
	return apiresource.NotificationPreferencesOf(notifyprefs.Parse(p.SettingsJson), paused)
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
		category := constants.NotificationCategory(p.Category)
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
