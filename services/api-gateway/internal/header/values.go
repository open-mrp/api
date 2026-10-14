// Package header holds the HTTP header names that are OpenMRP's own; the generic ones are apikit's transport package.
package header

import (
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
)

const (
	// TargetAccountIDHeader names the account a request acts in, when the caller can act in more than one.
	TargetAccountIDHeader = "OpenMRP-Account"
	// ActorAccountIDHeader names the account the caller acts as.
	ActorAccountIDHeader = "OpenMRP-Actor-Account"
	// ContentDispositionHeader names a downloaded file.
	ContentDispositionHeader = "Content-Disposition"

	// InternalIdentityHeader carries a JSON-serialized agent identity on the internal listener. It is trusted ONLY when InternalServiceTokenHeader matches the configured secret. The edge/ingress must strip any X-OpenMRP-Internal-* headers from external traffic.
	InternalIdentityHeader = "X-OpenMRP-Internal-Identity"
	// InternalServiceTokenHeader carries the shared service token that gates the internal listener's identity trust.
	InternalServiceTokenHeader = "X-OpenMRP-Service-Token" // #nosec G101 -- header name, not a credential
)

// IsAPIKey reports whether a bearer token is an OpenMRP secret API key rather than a session token.
func IsAPIKey(token string) bool {
	return strings.HasPrefix(token, string(types.APIKeyPrefixSecretKey))
}
