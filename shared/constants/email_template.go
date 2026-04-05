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
	// EmailTemplateNewUserWelcome indicates that the email template is for a new user welcome email with a temporary password.
	EmailTemplateNewUserWelcome EmailTemplate = "new_user_welcome"
	// EmailTemplateOrderCheckout indicates that the email template is for an order checkout email.
	EmailTemplateOrderCheckout EmailTemplate = "order_checkout"
	// EmailTemplatePurchaseOrderSubmission indicates that the email template is for a purchase order submission email.
	EmailTemplatePurchaseOrderSubmission EmailTemplate = "purchase_order_submission"
	// EmailTemplateStatementOfAccount indicates that the email template is for a statement of account email.
	EmailTemplateStatementOfAccount EmailTemplate = "statement_of_account"
	// EmailTemplateInvoice indicates that the email template is for an invoice email.
	EmailTemplateInvoice EmailTemplate = "invoice"
	// EmailTemplateOrderAcknowledgement indicates that the email template is for an order acknowledgement email.
	EmailTemplateOrderAcknowledgement EmailTemplate = "order_acknowledgement"
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
		EmailTemplateRegistrationLimitAlert,
		EmailTemplateNewUserWelcome,
		EmailTemplateOrderCheckout,
		EmailTemplatePurchaseOrderSubmission,
		EmailTemplateStatementOfAccount,
		EmailTemplateInvoice,
		EmailTemplateOrderAcknowledgement:
		return true
	}
	return false
}

func (t EmailTemplate) EnumValues() []string {
	return []string{string(EmailTemplateWelcome), string(EmailTemplatePasswordReset), string(EmailTemplatePasswordUpdated), string(EmailTemplateRegistrationVerify), string(EmailTemplateRegistrationVerifyExisting), string(EmailTemplateEnterpriseRequest), string(EmailTemplateInternalErrorAlert), string(EmailTemplateNewRegistrationAlert), string(EmailTemplatePlanChangeAlert), string(EmailTemplateRegistrationLimitAlert), string(EmailTemplateNewUserWelcome), string(EmailTemplateOrderCheckout), string(EmailTemplatePurchaseOrderSubmission), string(EmailTemplateStatementOfAccount), string(EmailTemplateInvoice), string(EmailTemplateOrderAcknowledgement)}
}
