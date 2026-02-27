package apierror

const (
	docsBaseURL = "https://docs.augno.com"

	// Auth & authorization
	docURLExpiredAPIKey      = docsBaseURL + "/errors#api_key_expired"
	docURLRevokedAPIKey      = docsBaseURL + "/errors#api_key_revoked"
	docURLInvalidCredentials = docsBaseURL + "/errors#invalid_credentials"
	docURLInsufficientPerms  = docsBaseURL + "/errors#insufficient_permissions"
	docURLPaymentRequired    = docsBaseURL + "/errors#payment_required"

	// Validation
	docURLValidationFailed = docsBaseURL + "/errors#validation_failed"
	docURLMissingField     = docsBaseURL + "/errors#missing_field"
	docURLInvalidFormat    = docsBaseURL + "/errors#invalid_format"
	docURLMethodNotAllowed = docsBaseURL + "/errors#method_not_allowed"

	// Resources
	docURLResourceNotFound = docsBaseURL + "/errors#resource_not_found"
	docURLResourceConflict = docsBaseURL + "/errors#resource_conflict"
	docURLResourceGone     = docsBaseURL + "/errors#resource_gone"

	// Idempotency
	docURLIdempotencyInProgress = docsBaseURL + "/errors#idempotency_in_progress"

	// Limits
	docURLLimitExceeded      = docsBaseURL + "/errors#limit_exceeded"
	docURLRegistrationClosed = docsBaseURL + "/errors#registration_closed"

	// Rate limiting
	docURLRateLimit = docsBaseURL + "/errors#rate_limit_exceeded"

	// Parameters
	docURLParameterMissing    = docsBaseURL + "/errors#parameter_missing"
	docURLParameterInvalid    = docsBaseURL + "/errors#parameter_invalid"
	docURLParameterUnknown    = docsBaseURL + "/errors#parameter_unknown"
	docURLParametersExclusive = docsBaseURL + "/errors#parameters_exclusive"

	// Server errors
	docURLInternalError       = docsBaseURL + "/errors#internal_error"
	docURLRequestTimeout      = docsBaseURL + "/errors#request_timeout"
	docURLClientClosedRequest = docsBaseURL + "/errors#client_closed_request"

	// API version
	docURLAPIVersionRequired = docsBaseURL + "/errors#api_version_required"
	docURLAPIVersionInvalid  = docsBaseURL + "/errors#api_version_invalid"
	docURLAPIVersionTooOld   = docsBaseURL + "/errors#api_version_too_old"
)
