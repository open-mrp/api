package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var requestLogLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.request_log")

func LoadRequestLogs(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}

	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, requestLogLoaderTracer, "loader.request_logs.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRequestLogResponse, error) {
				// Base resource only: this loader populates a request_log embedded as
				// another resource's sub-resource (e.g. an audit event's "request").
				// Nested expansions like "request.account"/"request.actor" are not part
				// of any endpoint's allowed includes, so the embedded request_log must
				// not carry account/actor data the client never asked for.
				return loggingClient.GetRequestLog(ctx, &pb.GetRequestLogRequest{
					Id: id,
				}, opts...)
			})
		if apiErr != nil || resp == nil || resp.RequestLog == nil {
			continue
		}
		out[id] = requestLogFromProto(resp.RequestLog)
	}
	return out, nil
}

func requestLogFromProto(rl *pb.RequestLogInfo) *apiresource.RequestLog {
	if rl == nil {
		return nil
	}

	result := &apiresource.RequestLog{
		ID:              rl.Id,
		Object:          constants.ObjectTypeRequestLog,
		Method:          rl.Method,
		Host:            rl.Host,
		Path:            rl.Path,
		NormalizedRoute: rl.NormalizedRoute,
		StatusCode:      rl.StatusCode,
		LatencyUs:       rl.LatencyUs,
		APIVersion:      rl.ApiVersion,
		ClientIP:        rl.ClientIp,
		UserAgent:       rl.UserAgent,
		Referrer:        rl.Referrer,
		ErrorCode:       rl.ErrorCode,
		ErrorMessage:    rl.ErrorMessage,
		OccurredAt:      grpcutil.TimestampToTime(rl.OccurredAt),
		CreatedAt:       grpcutil.TimestampToTime(rl.CreatedAt),
		IdempotencyKey:  rl.IdempotencyKey,
	}

	if rl.AccountId != nil && rl.AccountName != nil {
		result.Account = &apiresource.Account{
			ID:     *rl.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   *rl.AccountName,
		}
		if rl.AccountCreatedAt != nil {
			result.Account.CreatedAt = rl.AccountCreatedAt.AsTime()
		}
		if rl.AccountUpdatedAt != nil {
			result.Account.UpdatedAt = rl.AccountUpdatedAt.AsTime()
		}
	}

	if rl.Actor != nil {
		actorType := constants.ActorType(rl.Actor.ActorType)
		var handle *string
		switch actorType {
		case constants.ActorTypeAPIKey:
			handle = rl.Actor.RedactedValue
		case constants.ActorTypeUser:
			handle = rl.Actor.Email
		}
		result.Actor = apiresource.NewActor(rl.Actor.Id, actorType, rl.Actor.Name, handle)
	}

	return result
}
