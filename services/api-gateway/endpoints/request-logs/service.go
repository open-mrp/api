package requestlogep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/platform"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RequestLogSvc interface {
	ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError)
	GetRequestLog(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError)
}

type RequestLogSvcConfig struct {
	// LoggingClient (required) is the platform-service logging gRPC client.
	LoggingClient pb.LoggingServiceClient
}

type requestLogSvcImpl struct {
	loggingClient pb.LoggingServiceClient
}

var requestLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.request_logs.service")

func (c *RequestLogSvcConfig) validate() error {
	if c.LoggingClient == nil {
		return fmt.Errorf("request log endpoint service: logging client is required")
	}
	return nil
}

func NewRequestLogSvc(config *RequestLogSvcConfig) RequestLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &requestLogSvcImpl{
		loggingClient: config.LoggingClient,
	}
}

func (m *requestLogSvcImpl) ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
	methods := make([]string, len(req.Methods))
	for i, m := range req.Methods {
		methods[i] = string(m)
	}

	errorCodes := make([]string, len(req.ErrorCodes))
	for i, ec := range req.ErrorCodes {
		errorCodes[i] = string(ec)
	}

	excludeErrorCodes := make([]string, len(req.ExcludeErrorCodes))
	for i, ec := range req.ExcludeErrorCodes {
		excludeErrorCodes[i] = string(ec)
	}

	actorTypes := make([]string, len(req.ActorTypes))
	for i, at := range req.ActorTypes {
		actorTypes[i] = string(at)
	}

	pbReq := &pb.ListRequestLogsRequest{
		Query:             req.Query,
		Methods:           methods,
		StatusCodes:       req.StatusCodes,
		StatusCodeClasses: req.StatusCodeClasses,
		ErrorCodes:        errorCodes,
		ExcludeErrorCodes: excludeErrorCodes,
		ActorAccountIds:   req.ActorAccountIDs,
		TargetAccountIds:  req.TargetAccountIDs,
		ActorIds:          req.ActorIDs,
		ActorTypes:        actorTypes,
		NormalizedRoutes:  req.NormalizedRoutes,
		Hosts:             req.Hosts,
		MinLatencyUs:      req.MinLatencyUs,
		IdempotencyKey:    req.IdempotencyKey,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Includes:          resourcekit.FilterIncludes(ctx, "account", "actor", "actor.role"),
	}

	if req.StartDate != nil && !req.StartDate.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRequestLogsResponse, error) {
			return m.loggingClient.ListRequestLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.RequestLog](nil, apiresource.PageInfo{}), nil
	}

	meta := resourcekit.GetLoadMeta(ctx)
	logs := make([]apiresource.RequestLog, len(resp.RequestLogs))
	var agentActors []*apiresource.Actor
	for i, rl := range resp.RequestLogs {
		logs[i] = requestLogFromProto(rl)
		if actor := stashRequestLogMeta(ctx, meta, rl); actor != nil && actor.Type == constants.ActorTypeAgent {
			agentActors = append(agentActors, actor)
		}
	}
	hydrateAgentActors(ctx, agentActors)

	return apiresource.NewList(logs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *requestLogSvcImpl) GetRequestLog(ctx context.Context, req *RetrieveRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
	pbReq := &pb.GetRequestLogRequest{
		Id:       req.ID,
		Includes: resourcekit.FilterIncludes(ctx, "account", "actor", "actor.role", "query_params", "request_body", "response_body"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRequestLogResponse, error) {
			return m.loggingClient.GetRequestLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := requestLogFromProto(resp.RequestLog)
	if actor := stashRequestLogMeta(ctx, meta, resp.RequestLog); actor != nil && actor.Type == constants.ActorTypeAgent {
		hydrateAgentActors(ctx, []*apiresource.Actor{actor})
	}
	return &result, nil
}

// hydrateAgentActors fills in agent actors' display name + handle from agent-service.
// Unlike user/api_key actors — whose names are joined in platform-service — agent
// definitions live in a separate datastore, so their names must be resolved here.
// Best-effort and a no-op unless the caller expanded the actor (the only case where
// the name is rendered), avoiding a needless agent-service round-trip otherwise.
func hydrateAgentActors(ctx context.Context, actors []*apiresource.Actor) {
	if len(actors) == 0 || !resourcekit.RequestedIncludeSet(ctx)["actor"] {
		return
	}
	resourceloaders.HydrateActorNames(ctx, actors)
}

func requestLogFromProto(rl *pb.RequestLogInfo) apiresource.RequestLog {
	if rl == nil {
		return apiresource.RequestLog{}
	}

	result := apiresource.RequestLog{
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
	// Never expose internal infrastructure (internal listener host, pod IP) for agent requests.
	result.ScrubInternalInfra(rl.IdentityType)
	return result
}

// stashRequestLogMeta preheats the request log's account/actor and sub-resource
// payloads into the request-scoped resolver cache. It returns the actor it built
// (nil when the log has none) so the caller can post-process it — e.g. resolve agent
// display names that platform-service cannot join.
func stashRequestLogMeta(ctx context.Context, meta *resourcekit.LoadMeta, rl *pb.RequestLogInfo) *apiresource.Actor {
	if rl == nil {
		return nil
	}

	if rl.AccountId != nil && rl.AccountName != nil {
		account := &apiresource.Account{
			ID:     *rl.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   *rl.AccountName,
		}
		if rl.AccountCreatedAt != nil {
			account.CreatedAt = rl.AccountCreatedAt.AsTime()
		}
		if rl.AccountUpdatedAt != nil {
			account.UpdatedAt = rl.AccountUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "account", account)
	}

	var actor *apiresource.Actor
	if rl.Actor != nil {
		actorType := constants.ActorType(rl.Actor.ActorType)
		var handle *string
		switch actorType {
		case constants.ActorTypeAPIKey:
			handle = rl.Actor.RedactedValue
		case constants.ActorTypeUser:
			handle = rl.Actor.Email
		}
		actor = apiresource.NewActor(rl.Actor.Id, actorType, rl.Actor.Name, handle)
		resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "actor_id", actor.ID)
		if rl.Actor.RoleId != nil {
			meta.Set(constants.ObjectTypeActor, actor.ID, "role_id", *rl.Actor.RoleId)
		}
	}

	if rl.QueryJson != nil {
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "query_params", rawMessageFromString(*rl.QueryJson))
	}
	if rl.BodyJson != nil {
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "request_body", rawMessageFromString(*rl.BodyJson))
	}
	if rl.ResponseJson != nil {
		meta.Set(constants.ObjectTypeRequestLog, rl.Id, "response_body", rawMessageFromString(*rl.ResponseJson))
	}

	return actor
}

func rawMessageFromString(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	if json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return json.RawMessage(encoded)
}
