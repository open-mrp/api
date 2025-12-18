package email

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/tracing"
)

var ErrTemplateNotFound = errors.New("template not found")

var templateRendererTracer = tracing.GetTracer("auth-service.email.template_renderer")

type TemplateRenderer interface {
	RenderWelcomeEmail(ctx context.Context, data WelcomeEmailData) (string, *contracts.APIError)
	RenderPasswordResetEmail(ctx context.Context, data PasswordResetEmailData) (string, *contracts.APIError)
	RenderPasswordUpdatedEmail(ctx context.Context, data PasswordUpdatedEmailData) (string, *contracts.APIError)
}

type templateRendererImpl struct {
	templates map[string]*template.Template
}

func NewTemplateRenderer(templatesFS embed.FS) (TemplateRenderer, *contracts.APIError) {
	ctx := context.Background()
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.new")
	defer span.End()

	templates := make(map[string]*template.Template)

	templateFiles := []string{
		"welcome.html",
		"password_reset.html",
		"password_updated.html",
	}

	for _, filename := range templateFiles {
		tmpl, err := template.ParseFS(templatesFS, filename)
		if err != nil {
			return nil, tracing.Trace(span, contracts.NewInternalError(err, fmt.Sprintf("Failed to parse template %s", filename)))
		}
		templates[filename] = tmpl
	}

	return &templateRendererImpl{
		templates: templates,
	}, nil
}

func NewTemplateRendererFromDir(templatesDir string) (TemplateRenderer, *contracts.APIError) {
	ctx := context.Background()
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.new_from_dir")
	defer span.End()

	templates := make(map[string]*template.Template)

	templateFiles := []string{
		"welcome.html",
		"password_reset.html",
		"password_updated.html",
	}

	for _, filename := range templateFiles {
		path := filepath.Join(templatesDir, filename)
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			return nil, tracing.Trace(span, contracts.NewInternalError(err, fmt.Sprintf("Failed to parse template %s", path)))
		}
		templates[filename] = tmpl
	}

	return &templateRendererImpl{
		templates: templates,
	}, nil
}

func (r *templateRendererImpl) RenderWelcomeEmail(ctx context.Context, data WelcomeEmailData) (string, *contracts.APIError) {
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.render_welcome")
	defer span.End()

	tmpl, ok := r.templates["welcome.html"]
	if !ok {
		return "", tracing.Trace(span, contracts.NewInternalError(ErrTemplateNotFound, "Welcome email template not found"))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", tracing.Trace(span, contracts.NewInternalError(err, "Failed to render welcome email template"))
	}

	return buf.String(), nil
}

func (r *templateRendererImpl) RenderPasswordResetEmail(ctx context.Context, data PasswordResetEmailData) (string, *contracts.APIError) {
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.render_password_reset")
	defer span.End()

	tmpl, ok := r.templates["password_reset.html"]
	if !ok {
		return "", tracing.Trace(span, contracts.NewInternalError(ErrTemplateNotFound, "Password reset email template not found"))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", tracing.Trace(span, contracts.NewInternalError(err, "Failed to render password reset email template"))
	}

	return buf.String(), nil
}

func (r *templateRendererImpl) RenderPasswordUpdatedEmail(ctx context.Context, data PasswordUpdatedEmailData) (string, *contracts.APIError) {
	_, span := templateRendererTracer.Start(ctx, "email.template_renderer.render_password_updated")
	defer span.End()

	tmpl, ok := r.templates["password_updated.html"]
	if !ok {
		return "", tracing.Trace(span, contracts.NewInternalError(ErrTemplateNotFound, "Password updated email template not found"))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", tracing.Trace(span, contracts.NewInternalError(err, "Failed to render password updated email template"))
	}

	return buf.String(), nil
}
