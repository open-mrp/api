package subscriptionep

import (
	"context"
	"fmt"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/notification"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

// Svc backs the subscription endpoints via the notification-service MessagingService gRPC client.
type Svc interface {
	ListSubscriptions(ctx context.Context, req *ListSubscriptionsRequest) (*apiresource.List[apiresource.Subscription], *apierror.APIError)
	GetSubscription(ctx context.Context, req *RetrieveSubscriptionRequest) (*apiresource.Subscription, *apierror.APIError)
	CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) (*apiresource.Subscription, *apierror.APIError)
	Subscribe(ctx context.Context, req *lifecycle.Request) (*apiresource.Subscription, *apierror.APIError)
	Unsubscribe(ctx context.Context, req *lifecycle.Request) (*apiresource.Subscription, *apierror.APIError)
}

type SvcConfig struct {
	// MessagingClient (required) is the notification-service MessagingService gRPC client.
	MessagingClient pb.MessagingServiceClient
}

func (c *SvcConfig) validate() error {
	if c.MessagingClient == nil {
		return fmt.Errorf("subscription endpoint service: messaging client is required")
	}
	return nil
}

type svcImpl struct {
	messagingClient pb.MessagingServiceClient
}

var tracer = tracing.GetTracer("api-gateway.endpoints.subscriptions.service")

func NewSvc(config *SvcConfig) Svc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &svcImpl{messagingClient: config.MessagingClient}
}

func (s *svcImpl) ListSubscriptions(ctx context.Context, req *ListSubscriptionsRequest) (*apiresource.List[apiresource.Subscription], *apierror.APIError) {
	pbReq := &pb.ListSubscriptionsRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		RecordIds:   req.RecordIDs,
		RecordTypes: enumStrings(req.RecordTypes),
		Statuses:    enumStrings(req.Statuses),
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, tracer, "service.subscriptions.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSubscriptionsResponse, error) {
			return s.messagingClient.ListSubscriptions(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	subs := present(ctx, resp.Subscriptions)
	items := make([]apiresource.Subscription, len(subs))
	for i, sub := range subs {
		items[i] = *sub
	}
	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

func (s *svcImpl) GetSubscription(ctx context.Context, req *RetrieveSubscriptionRequest) (*apiresource.Subscription, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, tracer, "service.subscriptions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SubscriptionInfo, error) {
			return s.messagingClient.GetSubscription(ctx, &pb.GetSubscriptionRequest{Id: req.ID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return present(ctx, []*pb.SubscriptionInfo{resp})[0], nil
}

func (s *svcImpl) CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) (*apiresource.Subscription, *apierror.APIError) {
	if !req.Record.Type.Followable() {
		return nil, apierror.NewInvalidFormatError("record.type", fmt.Sprintf("Records of type %q can't be followed.", req.Record.Type))
	}
	if apiErr := requireRecord(ctx, "record.id", req.Record.Type, req.Record.ID); apiErr != nil {
		return nil, apiErr
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, tracer, "service.subscriptions.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.FollowRecordResponse, error) {
			return s.messagingClient.FollowRecord(ctx, &pb.FollowRecordRequest{RecordType: string(req.Record.Type), RecordId: req.Record.ID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return present(ctx, []*pb.SubscriptionInfo{resp.Subscription})[0], nil
}

func (s *svcImpl) Subscribe(ctx context.Context, req *lifecycle.Request) (*apiresource.Subscription, *apierror.APIError) {
	return s.setStatus(ctx, req.ID, constants.RecordSubscriptionStatusSubscribed)
}

func (s *svcImpl) Unsubscribe(ctx context.Context, req *lifecycle.Request) (*apiresource.Subscription, *apierror.APIError) {
	return s.setStatus(ctx, req.ID, constants.RecordSubscriptionStatusUnsubscribed)
}

func (s *svcImpl) setStatus(ctx context.Context, id string, status constants.RecordSubscriptionStatus) (*apiresource.Subscription, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, tracer, "service.subscriptions.set_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SubscriptionInfo, error) {
			return s.messagingClient.SetSubscriptionStatus(ctx, &pb.SetSubscriptionStatusRequest{Id: id, Status: string(status)}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return present(ctx, []*pb.SubscriptionInfo{resp})[0], nil
}

// requireRecord refuses following a record that doesn't exist, for every record type with a stub loader.
func requireRecord(ctx context.Context, param string, t constants.RecordType, id string) *apierror.APIError {
	if !recordstub.Registered(t) {
		return nil
	}
	loaded, apiErr := recordstub.Load(include.WithIncludeReads(ctx), []string{recordstub.Key(t, id)})
	if apiErr != nil {
		return apiErr
	}
	if recordstub.One(loaded, t, id) == nil {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, fmt.Sprintf("No %s with this ID exists.", t))
	}
	return nil
}

// present is the response shape of subscriptions. Each record is its stub, filled from its type's loader where it has
// one; a record of another type, or one that no longer exists, keeps its type and ID.
func present(ctx context.Context, infos []*pb.SubscriptionInfo) []*apiresource.Subscription {
	out := make([]*apiresource.Subscription, len(infos))
	var keys []string
	for i, info := range infos {
		rt := constants.RecordType(info.RecordType)
		stub := apiresource.NewRecord(info.RecordId, rt)
		stub.Metadata = map[string]string{}
		out[i] = &apiresource.Subscription{
			ID:        info.Id,
			Object:    constants.ObjectTypeSubscription,
			Record:    stub,
			Reason:    constants.SubscriptionReason(info.Reason),
			Status:    constants.RecordSubscriptionStatus(info.Status),
			CreatedAt: info.CreatedAt.AsTime(),
			UpdatedAt: info.UpdatedAt.AsTime(),
		}
		if recordstub.Registered(rt) {
			keys = append(keys, recordstub.Key(rt, info.RecordId))
		}
	}
	if len(keys) == 0 {
		return out
	}
	loaded, apiErr := recordstub.Load(include.WithIncludeReads(ctx), keys)
	if apiErr != nil {
		return out
	}
	for _, sub := range out {
		if s := recordstub.One(loaded, sub.Record.Type, sub.Record.ID); s != nil {
			if s.Metadata == nil {
				s.Metadata = map[string]string{}
			}
			sub.Record = s
		}
	}
	return out
}

func enumStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
