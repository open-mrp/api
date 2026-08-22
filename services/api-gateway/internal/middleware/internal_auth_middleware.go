package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/internal/header"
	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// InternalAuthMiddlewareConfig configures the trusted internal listener's auth.
type InternalAuthMiddlewareConfig struct {
	// ServiceToken (required) is the shared secret that gates identity trust on the internal listener. Requests must present a matching InternalServiceTokenHeader.
	ServiceToken string
}

func (c *InternalAuthMiddlewareConfig) validate() error {
	if c.ServiceToken == "" {
		return fmt.Errorf("internal auth middleware: service token is required")
	}
	return nil
}

// InternalAuthMiddleware authenticates requests on the gateway's internal listener. Instead of validating a user credential against auth-service, it trusts an agent identity supplied directly in InternalIdentityHeader — but ONLY when InternalServiceTokenHeader matches the configured secret (constant-time) and the identity is a well-formed agent identity. This listener must never be exposed behind the public ALB, and the edge must strip X-OpenMRP-Internal-* from external traffic.
func InternalAuthMiddleware(config *InternalAuthMiddlewareConfig) func(http.HandlerFunc) http.HandlerFunc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	expectedToken := []byte(config.ServiceToken)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			tracer := otel.Tracer("api-gateway/internal/middleware")
			ctx, span := tracer.Start(r.Context(), fmt.Sprintf("internal-auth %s %s", r.Method, r.URL.Path))

			// Gate: constant-time service-token check. Fail closed.
			presented := []byte(r.Header.Get(header.InternalServiceTokenHeader))
			if subtle.ConstantTimeCompare(presented, expectedToken) != 1 {
				respondInternalAuthError(r, w, span, apierror.NewAuthenticationError("Invalid internal service token."))
				return
			}

			rawIdentity := r.Header.Get(header.InternalIdentityHeader)
			if rawIdentity == "" {
				respondInternalAuthError(r, w, span, apierror.NewAuthenticationError("Missing internal identity."))
				return
			}

			var identity types.Identity
			if err := json.Unmarshal([]byte(rawIdentity), &identity); err != nil {
				respondInternalAuthError(r, w, span, apierror.NewAuthenticationError("Malformed internal identity."))
				return
			}

			// Shape allowlist: a leaked token must not be usable to mint anything other than an agent identity scoped to a target account.
			if identity.Type != types.IdentityActorTypeAgent || !identity.IsInternalActor() || !identity.IsTargetAccountSet() {
				respondInternalAuthError(r, w, span, apierror.NewAuthenticationError("Internal identity must be an agent identity scoped to a target account."))
				return
			}

			span.End()

			if rl, ok := appctx.GetRequestLog(r.Context()); ok && rl != nil {
				if identity.Actor != nil {
					actorType := string(identity.Actor.RelationType)
					rl.ActorType = &actorType
					rl.ActorID = &identity.Actor.ID
					rl.AccountID = identity.Actor.AccountID
				}
				if identity.IsTargetAccountSet() {
					accountID := identity.Target.AccountID
					rl.TargetAccountID = &accountID
				}
				identityType := string(identity.Type)
				rl.IdentityType = &identityType
			}

			ctx = appctx.WithIdentity(ctx, &identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

func respondInternalAuthError(r *http.Request, w http.ResponseWriter, span trace.Span, apiErr *apierror.APIError) {
	httptransport.RespondWithAPIError(r.Context(), w, apiErr)
	tracing.RecordControllerError(span, apiErr)
	span.End()
}
