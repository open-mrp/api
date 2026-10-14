package apierror

const (
	docsBaseURL = "https://docs.openmrp.ai"

	// Auth & authorization
	docURLExpiredToken       = docsBaseURL + "/api/errors#expired_token"
	docURLExpiredAPIKey      = docsBaseURL + "/api/errors#api_key_expired"
	docURLRevokedAPIKey      = docsBaseURL + "/api/errors#api_key_revoked"
	docURLInvalidCredentials = docsBaseURL + "/api/errors#invalid_credentials"
	docURLInsufficientPerms  = docsBaseURL + "/api/errors#insufficient_permissions"
	docURLPaymentRequired    = docsBaseURL + "/api/errors#payment_required"
	docURLAgentSpendingCap   = docsBaseURL + "/api/errors#agent_spending_cap_reached"

	// Signatures
	docURLSigningPINInvalid           = docsBaseURL + "/api/errors#signing_pin_invalid"
	docURLSigningPINLocked            = docsBaseURL + "/api/errors#signing_pin_locked"
	docURLSigningPINNotSet            = docsBaseURL + "/api/errors#signing_pin_not_set"
	docURLSigningPINChangeRequired    = docsBaseURL + "/api/errors#signing_pin_change_required"
	docURLBatchReleaseRequired        = docsBaseURL + "/api/errors#batch_release_required"
	docURLBatchOnHold                 = docsBaseURL + "/api/errors#batch_on_hold"
	docURLCorrectionSignatureRequired = docsBaseURL + "/api/errors#correction_signature_required"
	docURLOperatorCheckInRequired     = docsBaseURL + "/api/errors#operator_check_in_required"

	// Validation
	docURLValidationFailed = docsBaseURL + "/api/errors#validation_failed"
	docURLMissingField     = docsBaseURL + "/api/errors#missing_field"
	docURLInvalidFormat    = docsBaseURL + "/api/errors#invalid_format"
	docURLMethodNotAllowed = docsBaseURL + "/api/errors#method_not_allowed"
	docURLRequestTooLarge  = docsBaseURL + "/api/errors#request_too_large"

	// Resources
	docURLResourceNotFound = docsBaseURL + "/api/errors#resource_not_found"
	docURLResourceConflict = docsBaseURL + "/api/errors#resource_conflict"
	docURLResourceGone     = docsBaseURL + "/api/errors#resource_gone"

	// Idempotency
	docURLIdempotencyInProgress = docsBaseURL + "/api/errors#idempotency_in_progress"

	// Limits
	docURLLimitExceeded      = docsBaseURL + "/api/errors#limit_exceeded"
	docURLRegistrationClosed = docsBaseURL + "/api/errors#registration_closed"

	// Rate limiting
	docURLRateLimit = docsBaseURL + "/api/errors#rate_limit_exceeded"

	// Parameters
	docURLParameterMissing    = docsBaseURL + "/api/errors#parameter_missing"
	docURLParameterInvalid    = docsBaseURL + "/api/errors#parameter_invalid"
	docURLParameterUnknown    = docsBaseURL + "/api/errors#parameter_unknown"
	docURLParametersExclusive = docsBaseURL + "/api/errors#parameters_exclusive"

	// Server errors
	docURLInternalError       = docsBaseURL + "/api/errors#internal_error"
	docURLRequestTimeout      = docsBaseURL + "/api/errors#request_timeout"
	docURLClientClosedRequest = docsBaseURL + "/api/errors#client_closed_request"

	// API version
	docURLAPIVersionRequired = docsBaseURL + "/api/errors#api_version_required"
	docURLAPIVersionInvalid  = docsBaseURL + "/api/errors#api_version_invalid"
	docURLAPIVersionTooOld   = docsBaseURL + "/api/errors#api_version_too_old"
)
