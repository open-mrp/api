package constants

// EmailTemplate represents the type of email template to render.
type EmailTemplate string

const (
	// EmailTemplateWelcome indicates that the email template is for a welcome email.
	EmailTemplateWelcome EmailTemplate = "welcome"
	// EmailTemplatePasswordReset indicates that the email template is for a password reset email.
	EmailTemplatePasswordReset EmailTemplate = "password_reset"
	// EmailTemplatePasswordUpdated indicates that the email template is for a password updated email.
	EmailTemplatePasswordUpdated EmailTemplate = "password_updated"
	// EmailTemplateRegistrationVerify indicates that the email template is for a registration verify email.
	EmailTemplateRegistrationVerify EmailTemplate = "registration_verify"
	// EmailTemplateRegistrationVerifyExisting indicates that the email template is for a registration verify existing email.
	EmailTemplateRegistrationVerifyExisting EmailTemplate = "registration_verify_existing"
	// EmailTemplateEnterpriseRequest indicates that the email template is for a enterprise request email.
	EmailTemplateEnterpriseRequest EmailTemplate = "enterprise_request"
	// EmailTemplateInternalErrorAlert indicates that the email template is for a 5xx internal error alert.
	EmailTemplateInternalErrorAlert EmailTemplate = "internal_error_alert"
	// EmailTemplateNewRegistrationAlert indicates that the email template is for a new account registration alert.
	EmailTemplateNewRegistrationAlert EmailTemplate = "new_registration_alert"
	// EmailTemplatePlanChangeAlert indicates that the email template is for a plan change alert.
	EmailTemplatePlanChangeAlert EmailTemplate = "plan_change_alert"
	// EmailTemplateRegistrationLimitAlert indicates that the email template is for a registration limit reached alert.
	EmailTemplateRegistrationLimitAlert EmailTemplate = "registration_limit_alert"
)

func (t EmailTemplate) IsValid() bool {
	switch t {
	case EmailTemplateWelcome,
		EmailTemplatePasswordReset,
		EmailTemplatePasswordUpdated,
		EmailTemplateRegistrationVerify,
		EmailTemplateRegistrationVerifyExisting,
		EmailTemplateEnterpriseRequest,
		EmailTemplateInternalErrorAlert,
		EmailTemplateNewRegistrationAlert,
		EmailTemplatePlanChangeAlert,
		EmailTemplateRegistrationLimitAlert:
		return true
	}
	return false
}

func (t EmailTemplate) EnumValues() []string {
	return []string{string(EmailTemplateWelcome), string(EmailTemplatePasswordReset), string(EmailTemplatePasswordUpdated), string(EmailTemplateRegistrationVerify), string(EmailTemplateRegistrationVerifyExisting), string(EmailTemplateEnterpriseRequest), string(EmailTemplateInternalErrorAlert), string(EmailTemplateNewRegistrationAlert), string(EmailTemplatePlanChangeAlert), string(EmailTemplateRegistrationLimitAlert)}
}
