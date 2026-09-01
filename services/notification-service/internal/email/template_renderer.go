package email

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

//go:embed templates/*.html templates/partials/*.html
var templatesFS embed.FS

var templateRendererTracer = tracing.GetTracer("notification-service.email.template_renderer")

type TemplateRenderer interface {
	RenderTemplate(ctx context.Context, templateID constants.EmailTemplate, params map[string]any) (string, *apierror.APIError)
}

type templateRendererImpl struct {
	templates map[constants.EmailTemplate]*template.Template
}

func NewTemplateRenderer() (TemplateRenderer, *apierror.APIError) {
	ctx := context.Background()
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.new")
	defer span.End()

	templates := make(map[constants.EmailTemplate]*template.Template)

	templateFiles := map[constants.EmailTemplate]string{
		constants.EmailTemplateWelcome:                    "templates/welcome.html",
		constants.EmailTemplatePasswordReset:              "templates/password_reset.html",
		constants.EmailTemplatePasswordUpdated:            "templates/password_updated.html",
		constants.EmailTemplateRegistrationVerify:         "templates/registration_verify.html",
		constants.EmailTemplateRegistrationVerifyExisting: "templates/registration_verify_existing.html",
		constants.EmailTemplateEnterpriseRequest:          "templates/enterprise_request.html",
		constants.EmailTemplateInternalErrorAlert:         "templates/internal_error_alert.html",
		constants.EmailTemplateNewRegistrationAlert:       "templates/new_registration_alert.html",
		constants.EmailTemplatePlanChangeAlert:            "templates/plan_change_alert.html",
		constants.EmailTemplateRegistrationLimitAlert:     "templates/registration_limit_alert.html",
		constants.EmailTemplateNewUserWelcome:             "templates/new_user_welcome.html",
		constants.EmailTemplateInvoice:                    "templates/invoice_email.html",
		constants.EmailTemplateOrderAcknowledgement:       "templates/order_acknowledgement_email.html",
		constants.EmailTemplateOrderCheckout:              "templates/order_checkout.html",
		constants.EmailTemplatePurchaseOrderSubmission:    "templates/purchase_order_submission.html",
		constants.EmailTemplateStatementOfAccount:         "templates/statement_of_account.html",
		constants.EmailTemplateAlreadyRegistered:          "templates/already_registered.html",
		constants.EmailTemplateChatMessage:                "templates/chat_message.html",
		constants.EmailTemplateMessageFailureAlert:        "templates/message_failure_alert.html",
		constants.EmailTemplateDemoRequest:                "templates/demo_request.html",
		constants.EmailTemplateDashboardFeedback:          "templates/dashboard_feedback.html",
	}

	// The partials carry the shared merchant letterhead and footer, so every merchant-facing email renders the same branding. They hold only {{define}} blocks, and ParseFS names the result after the first file, so the per-template Execute still resolves to the template itself.
	for templateID, filename := range templateFiles {
		tmpl, err := template.ParseFS(templatesFS, filename, "templates/partials/*.html")
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, fmt.Sprintf("Failed to parse template %s", filename)))
		}
		templates[templateID] = tmpl
	}

	return &templateRendererImpl{
		templates: templates,
	}, nil
}

func (r *templateRendererImpl) RenderTemplate(ctx context.Context, templateID constants.EmailTemplate, params map[string]any) (string, *apierror.APIError) {
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.render")
	defer span.End()

	if !templateID.IsValid() {
		return "", tracing.Trace(span, apierror.NewInternalError(nil, fmt.Sprintf("Invalid template ID: %s", templateID)))
	}

	tmpl, ok := r.templates[templateID]
	if !ok {
		return "", tracing.Trace(span, apierror.NewInternalError(nil, fmt.Sprintf("Template not found: %s", templateID)))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, fmt.Sprintf("Failed to render template %s", templateID)))
	}

	return buf.String(), nil
}
