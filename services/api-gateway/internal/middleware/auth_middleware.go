package middleware

import (
	"fmt"
	"net/http"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel"
)

type AuthMiddlewareConfig struct {
	AuthClient *grpcclient.AuthServiceClient
}

func (c *AuthMiddlewareConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("auth middleware: auth client is required")
	}
	return nil
}

func AuthMiddleware(config *AuthMiddlewareConfig) func(http.HandlerFunc) http.HandlerFunc {
	if err := config.validate(); err != nil {
		panic(err)
	}

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

			authHeader := r.Header.Get(header.AuthorizationHeader)
			augnoAccountIDHeader := r.Header.Get(header.TargetAccountIDHeader)
			actorAccountIDHeader := r.Header.Get(header.ActorAccountIDHeader)

			platform, _ := appctx.GetPlatformFromContext(r.Context())
			isProduction := platform.IsProduction()

			cookieToken, _ := cookie.GetAccessTokenFromRequest(r)

			if authHeader != "" && cookieToken != "" {
				apiErr := apierror.NewAuthenticationError("Ambiguous authentication: both Authorization header and cookies are present. Please provide only one.")
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				tracing.RecordControllerError(span, apiErr)
				span.End()
				return
			}

			var authToken string

			if authHeader != "" {
				authResult, apiErr := header.ValidateAndExtractAuthHeader(authHeader)
				if apiErr != nil {
					httptransport.RespondWithAPIError(r.Context(), w, apiErr)
					tracing.RecordControllerError(span, apiErr)
					span.End()
					return
				}

				if isProduction && !header.IsAPIKey(authResult.TokenString) {
					apiErr := apierror.NewAuthenticationError("Access tokens are not allowed in the Authorization header. Please use cookies instead.")
					httptransport.RespondWithAPIError(r.Context(), w, apiErr)
					tracing.RecordControllerError(span, apiErr)
					span.End()
					return
				}

				authToken = authResult.TokenString
			} else if cookieToken != "" {
				if isProduction && header.IsAPIKey(cookieToken) {
					apiErr := apierror.NewAuthenticationError("API keys are not allowed in cookies. Please use the Authorization header instead.")
					httptransport.RespondWithAPIError(r.Context(), w, apiErr)
					tracing.RecordControllerError(span, apiErr)
					span.End()
					return
				}
				authToken = cookieToken
			}

			if authToken != "" && header.IsAPIKey(authToken) && actorAccountIDHeader != "" {
				apiErr := apierror.NewValidationErrorWithParam(
					"Augno-Actor-Account header is not allowed when authenticating with an API key. API keys always act on behalf of the account they were created by.",
					header.ActorAccountIDHeader,
				)
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				tracing.RecordControllerError(span, apiErr)
				span.End()
				return
			}

			var targetAccountID *string
			if augnoAccountIDHeader != "" {
				targetAccountID = &augnoAccountIDHeader
			}

			var actorAccountID *string
			if actorAccountIDHeader != "" {
				actorAccountID = &actorAccountIDHeader
			}

			identity, err := config.AuthClient.Client.ValidateCredential(ctx, &pb.Credential{
				Token:           authToken,
				TargetAccountId: targetAccountID,
				ActorAccountId:  actorAccountID,
			})

			apiErr := contracts.ConvertGRPCError(ctx, err, "auth-service")
			if apiErr != nil {
				if rl, ok := appctx.GetRequestLog(r.Context()); ok && rl != nil && (rl.ErrorMessage == nil || *rl.ErrorMessage == "") {
					rl.ErrorMessage = &apiErr.PublicMessage
				}
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				tracing.RecordControllerError(span, apiErr)
				span.End()
				return
			}
			span.End()

			if rl, ok := appctx.GetRequestLog(r.Context()); ok && rl != nil {
				if identity.Actor != nil {
					actorType := normalizeActorType(identity.Actor.RelationType)
					rl.ActorType = &actorType
					rl.ActorID = &identity.Actor.Id
					rl.AccountID = identity.Actor.AccountId
				}
				if identity.Target != nil && identity.Target.AccountId != "" {
					accountId := identity.Target.AccountId
					rl.TargetAccountID = &accountId
				}
				identityType := normalizeIdentityType(identity.Type)
				rl.IdentityType = &identityType
			}

			identityType := types.IdentityFromProto(identity)
			ctx = appctx.WithIdentity(r.Context(), identityType)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		}
	}
}

func normalizeIdentityType(t pb.IdentityActorType) string {
	switch t {
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_USER:
		return "user"
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_API_KEY:
		return "api_key"
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNAUTHENTICATED:
		return "unauthenticated"
	default:
		return ""
	}
}

func normalizeActorType(t pb.IdentityRelationType) string {
	switch t {
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_INTERNAL:
		return "internal"
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_CUSTOMER:
		return "customer"
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_SUPPLIER:
		return "supplier"
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_UNASSIGNED:
		return "unassigned"
	default:
		return ""
	}
}
