// Package apperr registers OpenMRP's business error codes with apikit's apierror registry and holds their
// constructors. Registration happens at init, so any binary that can build or decode one of these errors
// (every service imports this package through shared/contracts) knows its status and docs page.
package apperr

import (
	"net/http"
	"time"

	"github.com/open-mrp/apikit/apierror"
)

const (
	// CodeLimitExceeded is a plan-imposed resource limit, such as the number of sandboxes. The error carries `quota`.
	CodeLimitExceeded apierror.Code = "limit_exceeded"
	// CodeRegistrationClosed is public sign-up for a plan that has reached its capacity.
	CodeRegistrationClosed apierror.Code = "registration_closed"
	// CodePaymentRequired is an account whose subscription is past due, canceled or unpaid.
	CodePaymentRequired apierror.Code = "payment_required"
	// CodeAgentSpendingCapReached is an agent run refused because the account reached its monthly agent spending cap.
	CodeAgentSpendingCapReached apierror.Code = "agent_spending_cap_reached"
	// CodeAPIKeyExpired is an API key past its expiration date.
	CodeAPIKeyExpired apierror.Code = "api_key_expired" // #nosec G101 - an error code, not a credential
	// CodeAPIKeyRevoked is an API key its owner or OpenMRP revoked.
	CodeAPIKeyRevoked apierror.Code = "api_key_revoked" // #nosec G101 - an error code, not a credential
	// CodeVerificationRequired is an action an unverified sandbox account cannot take until a person verifies it.
	CodeVerificationRequired apierror.Code = "verification_required"
	// CodeCurrencyNotSupported is a currency code the API does not accept yet.
	CodeCurrencyNotSupported apierror.Code = "currency_not_supported"
)

// QuotaInfo is the `quota` member of a limit_exceeded error, so a client can show usage or an upgrade prompt.
type QuotaInfo struct {
	// The most the current plan allows.
	Limit int32 `json:"limit"`
	// How many are in use.
	Used int32 `json:"used"`
	// When the quota resets, or null for a limit that does not reset.
	ResetAt *time.Time `json:"reset_at"`
}

// DocsBaseURL is where every error code's docs page lives, as an anchor named for the code.
const DocsBaseURL = "https://docs.openmrp.ai/api/errors#"

func init() {
	apierror.SetDocURL(func(c apierror.Code) string { return DocsBaseURL + string(c) })
	for _, s := range []apierror.Spec{
		{Code: CodeLimitExceeded, Type: apierror.TypeInvalidRequest, Status: http.StatusForbidden,
			Description:  "The account reached a limit of its plan. quota says how much is allowed and used. Upgrade the plan to raise it.",
			DetailsField: "quota", DetailsExample: QuotaInfo{Limit: 3, Used: 3}},
		{Code: CodeRegistrationClosed, Type: apierror.TypeInvalidRequest, Status: http.StatusForbidden,
			Description: "Sign-up for this plan is closed because it reached its capacity."},
		{Code: CodePaymentRequired, Type: apierror.TypeInvalidRequest, Status: http.StatusPaymentRequired,
			Description: "The account's subscription is past due, canceled or unpaid. Resolve billing to continue."},
		{Code: CodeAgentSpendingCapReached, Type: apierror.TypeInvalidRequest, Status: http.StatusPaymentRequired,
			Description: "The account reached its monthly agent spending cap. Raise or remove the cap to run agents."},
		{Code: CodeAPIKeyExpired, Type: apierror.TypeInvalidRequest, Status: http.StatusUnauthorized,
			Description: "The API key has expired. Create a new key."},
		{Code: CodeAPIKeyRevoked, Type: apierror.TypeInvalidRequest, Status: http.StatusUnauthorized,
			Description: "The API key was revoked. Create a new key."},
		{Code: CodeVerificationRequired, Type: apierror.TypeInvalidRequest, Status: http.StatusForbidden,
			Description: "An unverified account can't do this yet. A person must verify the account first."},
		{Code: CodeCurrencyNotSupported, Type: apierror.TypeInvalidRequest, Status: http.StatusUnprocessableEntity,
			Description: "The currency is not supported. Only usd is accepted."},
	} {
		apierror.Register(s)
	}
}

// NewLimitExceededError is a 403 for a plan-imposed limit. Pass apierror.WithDetails(QuotaInfo{...}) to report usage.
func NewLimitExceededError(message string, opts ...apierror.Option) *apierror.APIError {
	return apierror.New(CodeLimitExceeded, message, opts...)
}

// NewRegistrationClosedError is a 403 for sign-up to a plan at capacity.
func NewRegistrationClosedError(message string) *apierror.APIError {
	return apierror.New(CodeRegistrationClosed, message)
}

// NewPaymentRequiredError is a 402 for an account whose subscription must be resolved first.
func NewPaymentRequiredError(message string) *apierror.APIError {
	return apierror.New(CodePaymentRequired, message)
}

// NewAgentSpendingCapReachedError is a 402 for an agent run refused by the account's spending cap.
func NewAgentSpendingCapReachedError(message string) *apierror.APIError {
	return apierror.New(CodeAgentSpendingCapReached, message)
}

// NewExpiredAPIKeyError is a 401 for an expired API key.
func NewExpiredAPIKeyError(message string) *apierror.APIError {
	return apierror.New(CodeAPIKeyExpired, message)
}

// NewRevokedAPIKeyError is a 401 for a revoked API key.
func NewRevokedAPIKeyError(message string) *apierror.APIError {
	return apierror.New(CodeAPIKeyRevoked, message)
}

// NewVerificationRequiredError is a 403 for an action an unverified account cannot take.
func NewVerificationRequiredError(message string) *apierror.APIError {
	return apierror.New(CodeVerificationRequired, message)
}

// NewCurrencyNotSupportedError is a 422 on param for a currency the API does not accept.
func NewCurrencyNotSupportedError(param, message string) *apierror.APIError {
	return apierror.New(CodeCurrencyNotSupported, message,
		apierror.WithFieldErrors(apierror.Field(param, CodeCurrencyNotSupported, message)))
}
