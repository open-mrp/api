package apierror

const (
	docsBaseURL = "https://docs.augno.com"

	// Auth & authorization
	docURLExpiredAPIKey      = docsBaseURL + "/api/errors#api-key-expired"
	docURLRevokedAPIKey      = docsBaseURL + "/api/errors#api-key-revoked"
	docURLInvalidCredentials = docsBaseURL + "/api/errors#invalid-credentials"
	docURLInsufficientPerms  = docsBaseURL + "/api/errors#insufficient-permissions"
	docURLPaymentRequired    = docsBaseURL + "/api/errors#payment-required"

	// Validation
	docURLValidationFailed = docsBaseURL + "/api/errors#validation-failed"
	docURLMissingField     = docsBaseURL + "/api/errors#missing-field"
	docURLInvalidFormat    = docsBaseURL + "/api/errors#invalid-format"
	docURLMethodNotAllowed = docsBaseURL + "/api/errors#method-not-allowed"

	// Resources
	docURLResourceNotFound = docsBaseURL + "/api/errors#resource-not-found"
	docURLResourceConflict = docsBaseURL + "/api/errors#resource-conflict"
	docURLResourceGone     = docsBaseURL + "/api/errors#resource-gone"

	// Idempotency
	docURLIdempotencyInProgress = docsBaseURL + "/api/errors#idempotency-in-progress"

	// Limits
	docURLLimitExceeded      = docsBaseURL + "/api/errors#limit-exceeded"
	docURLRegistrationClosed = docsBaseURL + "/api/errors#registration-closed"

	// Rate limiting
	docURLRateLimit = docsBaseURL + "/api/errors#rate-limit-exceeded"

	// Parameters
	docURLParameterMissing    = docsBaseURL + "/api/errors#parameter-missing"
	docURLParameterInvalid    = docsBaseURL + "/api/errors#parameter-invalid"
	docURLParameterUnknown    = docsBaseURL + "/api/errors#parameter-unknown"
	docURLParametersExclusive = docsBaseURL + "/api/errors#parameters-exclusive"

	// Server errors
	docURLInternalError       = docsBaseURL + "/api/errors#internal-error"
	docURLRequestTimeout      = docsBaseURL + "/api/errors#request-timeout"
	docURLClientClosedRequest = docsBaseURL + "/api/errors#client-closed-request"

	// API version
	docURLAPIVersionRequired = docsBaseURL + "/api/errors#api-version-required"
	docURLAPIVersionInvalid  = docsBaseURL + "/api/errors#api-version-invalid"
	docURLAPIVersionTooOld   = docsBaseURL + "/api/errors#api-version-too-old"
)
