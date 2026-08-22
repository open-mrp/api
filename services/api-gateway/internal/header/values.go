package header

const (
	IdempotencyKeyHeader     = "Idempotency-Key"
	IdempotentReplayedHeader = "Idempotent-Replayed"
	ContentTypeHeader        = "Content-Type"
	VersionHeader            = "OpenMRP-Version"
	TargetAccountIDHeader    = "OpenMRP-Account"
	ActorAccountIDHeader     = "OpenMRP-Actor-Account"
	AuthorizationHeader      = "Authorization"
	RequestIDHeader          = "Request-ID"
	WwwAuthenticateHeader    = "WWW-Authenticate"
	RetryAfterHeader         = "Retry-After"
	RateLimitLimitHeader     = "RateLimit-Limit"
	RateLimitRemainingHeader = "RateLimit-Remaining"
	RateLimitResetHeader     = "RateLimit-Reset"
	LocationHeader           = "Location"

	// InternalIdentityHeader carries a JSON-serialized agent identity on the internal listener. It is trusted ONLY when InternalServiceTokenHeader matches the configured secret. The edge/ingress must strip any X-OpenMRP-Internal-* headers from external traffic.
	InternalIdentityHeader = "X-OpenMRP-Internal-Identity"
	// InternalServiceTokenHeader carries the shared service token that gates the internal listener's identity trust.
	InternalServiceTokenHeader = "X-OpenMRP-Service-Token" // #nosec G101 -- header name, not a credential
)
