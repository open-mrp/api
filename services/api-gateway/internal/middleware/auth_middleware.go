package middleware

import (
	"context"
	"fmt"
	"net/http"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel"
)

type AuthMiddlewareConfig struct {
	AuthClient *grpcclient.AuthServiceClient
}

func AuthMiddleware(config AuthMiddlewareConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health check endpoint to allow ALB health checks to pass.
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			tracer := otel.Tracer("api-gateway/internal/middleware")
			spanName := fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(r.Context(), spanName)

			authHeader := r.Header.Get("Authorization")
			augnoAccountIDHeader := r.Header.Get("Augno-Account-ID")

			var authToken string

			if authHeader != "" {
				authResult, apiErr := header.ValidateAuthHeader(authHeader)
				if apiErr != nil {
					httptransport.RespondWithAPIError(r.Context(), w, apiErr)
					tracing.RecordControllerError(span, apiErr)
					span.End()
					return
				}
				authToken = authResult.TokenString
			} else {
				tokenValue, apiErr := cookie.GetAccessTokenFromRequest(r)
				if apiErr == nil && tokenValue != "" {
					authResult, apiErr := header.ValidateAuthHeader(fmt.Sprintf("Bearer %s", tokenValue))
					if apiErr != nil {
						httptransport.RespondWithAPIError(r.Context(), w, apiErr)
						tracing.RecordControllerError(span, apiErr)
						span.End()
						return
					}
					authToken = authResult.TokenString
				}
			}

			identity, err := config.AuthClient.Client.ValidateCredential(ctx, &pb.Credential{
				Token:           authToken,
				TargetAccountId: augnoAccountIDHeader,
			})

			apiErr := contracts.ConvertGRPCError(ctx, err, "auth-service")
			if apiErr != nil {
				if rl, ok := apicontext.GetRequestLogFromContext(r.Context()); ok && rl != nil && (rl.ErrorMessage == nil || *rl.ErrorMessage == "") {
					rl.ErrorMessage = &apiErr.PublicMessage
				}
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				tracing.RecordControllerError(span, apiErr)
				span.End()
				return
			}
			span.End()

			if rl, ok := apicontext.GetRequestLogFromContext(r.Context()); ok && rl != nil {
				if identity.Actor != nil {
					actorType := normalizeActorType(identity.Actor.Type)
					rl.ActorType = &actorType
					rl.ActorID = &identity.Actor.Id
				}
				if identity.TargetAccountId != nil {
					rl.AccountID = identity.TargetAccountId
				}
				identityType := normalizeIdentityType(identity.Type)
				rl.IdentityType = &identityType
			}

			identityType := types.IdentityFromProto(identity)
			ctx = context.WithValue(r.Context(), apicontext.AuthIdentityKey, identityType)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		}
	}
}

func normalizeIdentityType(t pb.IdentityType) string {
	switch t {
	case pb.IdentityType_IDENTITY_TYPE_USER:
		return "user"
	case pb.IdentityType_IDENTITY_TYPE_API_KEY:
		return "api_key"
	case pb.IdentityType_IDENTITY_TYPE_UNAUTHENTICATED:
		return "unauthenticated"
	default:
		return ""
	}
}

func normalizeActorType(t pb.IdentityActorType) string {
	switch t {
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_INTERNAL:
		return "internal"
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_CUSTOMER:
		return "customer"
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_SUPPLIER:
		return "supplier"
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNASSIGNED:
		return "unassigned"
	default:
		return ""
	}
}
