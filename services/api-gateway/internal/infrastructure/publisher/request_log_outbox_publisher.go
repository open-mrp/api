package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	corepb "github.com/open-mrp/api/shared/proto/core"
	pb "github.com/open-mrp/api/shared/proto/platform"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var requestLogPublisherTracer = tracing.GetTracer("api-gateway.request_log_publisher")

// accountNameResolver is the slice of the core client the error-alert path uses to
// name the acting and target accounts. Satisfied by corepb.CoreServiceClient.
type accountNameResolver interface {
	GetAccountNames(ctx context.Context, in *corepb.GetAccountNamesRequest, opts ...grpc.CallOption) (*corepb.GetAccountNamesResponse, error)
}

type requestLogOutboxPublisher struct {
	outboxRepo   messaging.OutboxRepo
	coreClient   accountNameResolver
	frontendURL  string
	platformMode constants.PlatformMode
}

func NewRequestLogOutboxPublisher(outboxRepo messaging.OutboxRepo, coreClient accountNameResolver, frontendURL string, platformMode constants.PlatformMode) domain.RequestLogPublisher {
	return &requestLogOutboxPublisher{
		outboxRepo:   outboxRepo,
		coreClient:   coreClient,
		frontendURL:  frontendURL,
		platformMode: platformMode,
	}
}

func (p *requestLogOutboxPublisher) Create(ctx context.Context, rl *appctx.RequestLog) error {
	ctx, span := requestLogPublisherTracer.Start(ctx, "publisher.request_log.create")
	defer span.End()

	pbLog := &pb.RequestLog{
		Id:                   rl.ID,
		Method:               rl.Method,
		Host:                 rl.Host,
		Path:                 rl.Path,
		NormalizedRoute:      rl.NormalizedRoute,
		QueryJson:            rl.QueryJSON,
		StatusCode:           int32(rl.StatusCode), // #nosec G115
		LatencyUs:            rl.LatencyUs,
		AccountId:            rl.AccountID,
		ClientIp:             rl.ClientIP,
		ClientIpString:       rl.ClientIPString,
		UserAgent:            rl.UserAgent,
		Referrer:             rl.Referrer,
		ErrorCode:            rl.ErrorCode,
		ErrorMessage:         rl.ErrorMessage,
		OccurredAt:           timestamppb.New(rl.OccurredAt),
		IdempotencyKeyId:     rl.IdempotencyKeyID,
		TargetAccountId:      rl.TargetAccountID,
		ActorId:              rl.ActorID,
		ActorType:            rl.ActorType,
		InternalErrorMessage: rl.InternalErrorMessage,
		StackTrace:           rl.StackTrace,
		IdentityType:         rl.IdentityType,
		CreatedAt:            timestamppb.Now(),
		ApiVersion:           rl.APIVersion,
		TraceId:              rl.TraceID,
		PublicEndpoint:       rl.PublicEndpoint,
		Hidden:               rl.Hidden,
		BodyJson:             rl.BodyJSON,
		ResponseJson:         rl.ResponseJSON,
	}

	_, marshalSpan := requestLogPublisherTracer.Start(ctx, "publisher.request_log.marshal")
	data, err := protojson.Marshal(pbLog)
	marshalSpan.End()
	if err != nil {
		slog.Error("Failed to marshal request log", "error", err, "request_id", rl.ID)
		return err
	}

	msg := contracts.AmqpMessage{
		RequestID: rl.ID,
		Data:      data,
	}
	// Capture the actor's display name from the identity while the request context is still available; the error-alert goroutine below runs on a detached context that no longer carries it.
	var actorName *string
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
		if identity != nil && identity.Actor != nil {
			actorName = identity.Actor.Name
		}
	}

	input := messaging.OutboxMessageInput{
		ServiceName: "api-gateway",
		MessageType: string(contracts.LoggingEventRequestLogged),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.LoggingEventRequestLogged),
		Payload:     msg,
	}

	// Save to outbox asynchronously - don't block the HTTP response
	// No tracing here since this runs after the request completes
	go func() { // #nosec G118 - runs after the response; a request-scoped context would cancel it
		err := messaging.WithOutboxDBLockRetry(context.Background(), messaging.OutboxDBRetryConfig(p.platformMode), "request_log_outbox.create", func() error {
			_, err := p.outboxRepo.Create(context.Background(), input)
			return err
		})
		if err != nil {
			slog.Error("Failed to save request log to outbox", "error", err, "request_id", rl.ID)
		}

		// Send an email alert for 5xx errors (skip in development mode)
		if rl.StatusCode >= 500 && p.platformMode != constants.PlatformModeDevelopment {
			p.publishErrorAlert(rl, actorName)
		}
	}()

	return nil
}

func (p *requestLogOutboxPublisher) publishErrorAlert(rl *appctx.RequestLog, actorName *string) {
	params := map[string]any{
		"RequestID":       rl.ID,
		"Method":          rl.Method,
		"Path":            rl.Path,
		"NormalizedRoute": rl.NormalizedRoute,
		"StatusCode":      rl.StatusCode,
		"LatencyUs":       rl.LatencyUs,
		"OccurredAt":      rl.OccurredAt.Format("2006-01-02 15:04:05 UTC"),
	}

	setOptionalParam(params, "ErrorCode", rl.ErrorCode)
	setOptionalParam(params, "ErrorMessage", rl.ErrorMessage)
	setOptionalParam(params, "InternalErrorMessage", rl.InternalErrorMessage)
	setOptionalParam(params, "StackTrace", rl.StackTrace)
	setOptionalParam(params, "RequestBody", rl.BodyJSON)
	setOptionalParam(params, "ResponseBody", rl.ResponseJSON)
	setOptionalParam(params, "UserName", actorName)
	setOptionalParam(params, "ActorID", rl.ActorID)
	setOptionalParam(params, "AccountID", rl.AccountID)
	setOptionalParam(params, "TargetAccountID", rl.TargetAccountID)

	// Resolve the acting and target account names so the alert reads as "who, on what account" rather than opaque IDs.
	accountNames := p.resolveAccountNames(rl.AccountID, rl.TargetAccountID)
	if rl.AccountID != nil {
		if name := accountNames[*rl.AccountID]; name != "" {
			params["AccountName"] = name
		}
	}
	if rl.TargetAccountID != nil {
		if name := accountNames[*rl.TargetAccountID]; name != "" {
			params["TargetAccountName"] = name
		}
	}
	setOptionalParam(params, "ClientIP", rl.ClientIPString)
	setOptionalParam(params, "UserAgent", rl.UserAgent)
	setOptionalParam(params, "TraceID", rl.TraceID)
	setOptionalParam(params, "APIVersion", rl.APIVersion)

	if p.frontendURL != "" {
		params["RequestLogURL"] = p.frontendURL + "/dashboard/request-logs/" + rl.ID
	}

	emailData := messaging.EmailSendData{
		To:         []string{"dev@augno.com"},
		Subject:    fmt.Sprintf("[%d Alert] %s %s", rl.StatusCode, rl.Method, rl.Path),
		TemplateID: constants.EmailTemplateInternalErrorAlert,
		Params:     params,
	}

	emailJSON, err := json.Marshal(emailData)
	if err != nil {
		slog.Error("Failed to marshal error alert email data", "error", err, "request_id", rl.ID)
		return
	}

	emailMsg := contracts.AmqpMessage{
		RequestID: rl.ID,
		Data:      emailJSON,
	}

	emailInput := messaging.OutboxMessageInput{
		ServiceName: "api-gateway",
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     emailMsg,
	}

	err = messaging.WithOutboxDBLockRetry(context.Background(), messaging.OutboxDBRetryConfig(p.platformMode), "request_log_outbox.error_alert.create", func() error {
		_, err := p.outboxRepo.Create(context.Background(), emailInput)
		return err
	})
	if err != nil {
		slog.Error("Failed to save error alert email to outbox", "error", err, "request_id", rl.ID)
	}
}

// resolveAccountNames looks up the display names for the given account IDs (nils and
// duplicates ignored). Best-effort: a lookup failure just yields an empty map so the
// alert still sends with IDs alone. Runs on context.Background() since the request has
// already completed.
func (p *requestLogOutboxPublisher) resolveAccountNames(ids ...*string) map[string]string {
	if p.coreClient == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	var lookup []string
	for _, id := range ids {
		if id == nil || *id == "" {
			continue
		}
		if _, ok := seen[*id]; ok {
			continue
		}
		seen[*id] = struct{}{}
		lookup = append(lookup, *id)
	}
	if len(lookup) == 0 {
		return nil
	}

	resp, apiErr := grpcutil.CallRPC(context.Background(), requestLogPublisherTracer, "request_log_outbox.get_account_names", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.GetAccountNamesResponse, error) {
			return p.coreClient.GetAccountNames(ctx, &corepb.GetAccountNamesRequest{AccountIds: lookup}, opts...)
		})
	if apiErr != nil {
		slog.Error("Failed to resolve account names for error alert", "error", apiErr)
		return nil
	}
	return resp.Names
}

func setOptionalParam(params map[string]any, key string, val *string) {
	if val != nil {
		params[key] = *val
	}
}
