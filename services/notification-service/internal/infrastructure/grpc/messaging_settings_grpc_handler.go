package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/notifyprefs"
	pb "github.com/open-mrp/api/shared/proto/notification"
	apierror "github.com/open-mrp/apikit/apierror"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *messagingGRPCHandler) GetNotificationPreferences(ctx context.Context, _ *pb.GetNotificationPreferencesRequest) (*pb.NotificationPreferencesInfo, error) {
	view, apiErr := h.messagingSvc.GetNotificationPreferences(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoPreferences(view), nil
}

func (h *messagingGRPCHandler) UpdateNotificationPreferences(ctx context.Context, req *pb.UpdateNotificationPreferencesRequest) (*pb.NotificationPreferencesInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	patch, apiErr := patchFromJSON(req.PatchJson)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	input := domain.UpdateNotificationPreferencesInput{Patch: patch, PausedUntilSet: req.PausedUntilSet}
	if req.PausedUntil != nil {
		t := req.PausedUntil.AsTime()
		input.PausedUntil = &t
	}
	view, apiErr := h.messagingSvc.UpdateNotificationPreferences(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoPreferences(view), nil
}

func (h *messagingGRPCHandler) GetNotificationDefaults(ctx context.Context, _ *pb.GetNotificationDefaultsRequest) (*pb.NotificationDefaultsInfo, error) {
	d, apiErr := h.messagingSvc.GetNotificationDefaults(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.NotificationDefaultsInfo{SettingsJson: d.Marshal()}, nil
}

func (h *messagingGRPCHandler) UpdateNotificationDefaults(ctx context.Context, req *pb.UpdateNotificationDefaultsRequest) (*pb.NotificationDefaultsInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	patch, apiErr := patchFromJSON(req.PatchJson)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	d, apiErr := h.messagingSvc.UpdateNotificationDefaults(ctx, patch)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.NotificationDefaultsInfo{SettingsJson: d.Marshal()}, nil
}

func (h *messagingGRPCHandler) ListSubscriptions(ctx context.Context, req *pb.ListSubscriptionsRequest) (*pb.ListSubscriptionsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.messagingSvc.ListSubscriptions(ctx, domain.ListSubscriptionsInput{
		RecordIDs:   req.RecordIds,
		RecordTypes: req.RecordTypes,
		Statuses:    req.Statuses,
		Limit:       req.Limit,
		Cursor:      req.Cursor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	subs := make([]*pb.SubscriptionInfo, 0, len(page.Subscriptions))
	for _, sub := range page.Subscriptions {
		subs = append(subs, toProtoSubscription(sub))
	}
	return &pb.ListSubscriptionsResponse{
		Subscriptions: subs,
		PageInfo:      &pb.PageInfo{NextCursor: page.NextCursor, HasNextPage: page.HasNextPage},
	}, nil
}

func (h *messagingGRPCHandler) GetSubscription(ctx context.Context, req *pb.GetSubscriptionRequest) (*pb.SubscriptionInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	sub, apiErr := h.messagingSvc.GetSubscription(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoSubscription(sub), nil
}

func (h *messagingGRPCHandler) FollowRecord(ctx context.Context, req *pb.FollowRecordRequest) (*pb.FollowRecordResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	sub, created, apiErr := h.messagingSvc.FollowRecord(ctx, req.RecordType, req.RecordId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.FollowRecordResponse{Subscription: toProtoSubscription(sub), Created: created}, nil
}

func (h *messagingGRPCHandler) SetSubscriptionStatus(ctx context.Context, req *pb.SetSubscriptionStatusRequest) (*pb.SubscriptionInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	sub, apiErr := h.messagingSvc.SetSubscriptionStatus(ctx, req.Id, req.Status)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return toProtoSubscription(sub), nil
}

func patchFromJSON(raw []byte) (notifyprefs.Patch, *apierror.APIError) {
	var p notifyprefs.Patch
	if len(raw) == 0 {
		return p, nil
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, apierror.NewInternalError(err, "Failed to read the notification settings patch.")
	}
	return p, nil
}

func toProtoPreferences(v *domain.NotificationPreferences) *pb.NotificationPreferencesInfo {
	d := v.Default
	out := &pb.NotificationPreferencesInfo{
		SettingsJson: notifyprefs.Defaults{Default: &d, Types: v.Types, AutoSubscribe: v.AutoSubscribe}.Marshal(),
	}
	if v.PausedUntil != nil && v.PausedUntil.After(time.Now()) {
		out.PausedUntil = timestamppb.New(*v.PausedUntil)
	}
	return out
}

func toProtoSubscription(s *domain.Subscription) *pb.SubscriptionInfo {
	return &pb.SubscriptionInfo{
		Id:            s.ID,
		AccountId:     s.AccountID,
		AccountUserId: s.AccountUserID,
		RecordType:    s.RecordType,
		RecordId:      s.RecordID,
		Reason:        s.Reason,
		Status:        s.Status,
		CreatedAt:     timestamppb.New(s.CreatedAt),
		UpdatedAt:     timestamppb.New(s.UpdatedAt),
	}
}
