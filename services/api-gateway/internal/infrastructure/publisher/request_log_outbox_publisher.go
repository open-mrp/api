package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var requestLogPublisherTracer = tracing.GetTracer("api-gateway.request_log_publisher")

type requestLogOutboxPublisher struct {
	outboxRepo   messaging.OutboxRepo
	frontendURL  string
	platformMode constants.PlatformMode
}

func NewRequestLogOutboxPublisher(outboxRepo messaging.OutboxRepo, frontendURL string, platformMode constants.PlatformMode) domain.RequestLogPublisher {
	return &requestLogOutboxPublisher{
		outboxRepo:   outboxRepo,
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
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
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
	go func() {
		err := messaging.WithOutboxDBLockRetry(context.Background(), messaging.OutboxDBRetryConfig(p.platformMode), "request_log_outbox.create", func() error {
			_, err := p.outboxRepo.Create(context.Background(), input)
			return err
		})
		if err != nil {
			slog.Error("Failed to save request log to outbox", "error", err, "request_id", rl.ID)
		}

		// Send an email alert for 5xx errors and 408 timeouts (skip in development mode)
		if (rl.StatusCode >= 500 || rl.StatusCode == http.StatusRequestTimeout) && p.platformMode != constants.PlatformModeDevelopment {
			p.publishErrorAlert(rl)
		}
	}()

	return nil
}

func (p *requestLogOutboxPublisher) publishErrorAlert(rl *appctx.RequestLog) {
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
	setOptionalParam(params, "ActorID", rl.ActorID)
	setOptionalParam(params, "AccountID", rl.AccountID)
	setOptionalParam(params, "TargetAccountID", rl.TargetAccountID)
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

func setOptionalParam(params map[string]any, key string, val *string) {
	if val != nil {
		params[key] = *val
	}
}
