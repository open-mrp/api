package email

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

//go:embed templates/*.html
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
	}

	for templateID, filename := range templateFiles {
		tmpl, err := template.ParseFS(templatesFS, filename)
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
