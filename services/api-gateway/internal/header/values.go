package header

const (
	IdempotencyKeyHeader     = "Idempotency-Key"
	IdempotentReplayedHeader = "Idempotent-Replayed"
	ContentTypeHeader        = "Content-Type"
	VersionHeader            = "Augno-Version"
	TargetAccountIDHeader    = "Augno-Account"
	ActorAccountIDHeader     = "Augno-Actor-Account"
	AuthorizationHeader      = "Authorization"
	RequestIDHeader          = "Request-ID"
	WwwAuthenticateHeader    = "WWW-Authenticate"
	RetryAfterHeader         = "Retry-After"
	RateLimitLimitHeader     = "RateLimit-Limit"
	RateLimitRemainingHeader = "RateLimit-Remaining"
	RateLimitResetHeader     = "RateLimit-Reset"
	LocationHeader           = "Location"

	// InternalIdentityHeader carries a JSON-serialized agent identity on the internal listener. It is trusted ONLY when InternalServiceTokenHeader matches the configured secret. The edge/ingress must strip any X-Augno-Internal-* headers from external traffic.
	InternalIdentityHeader = "X-Augno-Internal-Identity"
	// InternalServiceTokenHeader carries the shared service token that gates the internal listener's identity trust.
	InternalServiceTokenHeader = "X-Augno-Service-Token" // #nosec G101 -- header name, not a credential
)
