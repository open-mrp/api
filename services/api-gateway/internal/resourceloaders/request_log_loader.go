package resourceloaders

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/platform"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var requestLogLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.request_log")

func LoadRequestLogs(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}

	out := make(map[string]any, len(ids))
	actors := make([]*apiresource.ActorRef, 0, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, requestLogLoaderTracer, "loader.request_logs.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRequestLogResponse, error) {
				// The embedded log (e.g. an audit event's "request") carries what every log does, but not its expandable account: no endpoint offers "request.account".
				return loggingClient.GetRequestLog(ctx, &pb.GetRequestLogRequest{
					Id:       id,
					Includes: []string{"actor", "query_params"},
				}, opts...)
			})
		if apiErr != nil || resp == nil || resp.RequestLog == nil {
			continue
		}
		rl := RequestLogFromProto(resp.RequestLog)
		actors = append(actors, rl.Actor)
		out[id] = &rl
	}
	ResolveActorRefs(ctx, actors)
	return out, nil
}

// RequestLogFromProto is the request log an RPC carried, without its expandable fields.
func RequestLogFromProto(rl *pb.RequestLogInfo) apiresource.RequestLog {
	if rl == nil {
		return apiresource.RequestLog{}
	}

	result := apiresource.RequestLog{
		ID:              rl.Id,
		Object:          constants.ObjectTypeRequestLog,
		Method:          constants.HTTPMethod(rl.Method),
		Host:            rl.Host,
		Path:            rl.Path,
		NormalizedRoute: rl.NormalizedRoute,
		QueryParams:     queryParamsFromJSON(rl.QueryJson),
		StatusCode:      rl.StatusCode,
		LatencyUs:       rl.LatencyUs,
		APIVersion:      rl.ApiVersion,
		ClientIP:        rl.ClientIp,
		UserAgent:       rl.UserAgent,
		Referrer:        rl.Referrer,
		ErrorCode:       constants.EnumPtr[apierror.Code](rl.ErrorCode),
		ErrorMessage:    rl.ErrorMessage,
		OccurredAt:      grpcutil.TimestampToTime(rl.OccurredAt),
		CreatedAt:       grpcutil.TimestampToTime(rl.CreatedAt),
		IdempotencyKey:  rl.IdempotencyKey,
	}
	// platform-service joins user and API key names; ResolveActorRefs fills the rest.
	if rl.Actor != nil {
		result.Actor = ActorRefFromRecorded(constants.ActorType(rl.Actor.ActorType), rl.Actor.Id, rl.Actor.Name)
	}

	// Never expose internal infrastructure (internal listener host, pod IP) for agent requests, embedded or not.
	result.ScrubInternalInfra(rl.IdentityType)
	return result
}

// queryParamsFromJSON reads stored query parameters, where a parameter sent once is a string and one sent more than
// once is a list of strings. Anything else reads as no parameters.
func queryParamsFromJSON(s *string) *map[string][]string {
	if s == nil || *s == "" {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(*s), &raw); err != nil || len(raw) == 0 {
		return nil
	}
	out := make(map[string][]string, len(raw))
	for k, v := range raw {
		switch vv := v.(type) {
		case string:
			out[k] = []string{vv}
		case []any:
			values := make([]string, 0, len(vv))
			for _, item := range vv {
				if str, ok := item.(string); ok {
					values = append(values, str)
				}
			}
			out[k] = values
		}
	}
	return &out
}
