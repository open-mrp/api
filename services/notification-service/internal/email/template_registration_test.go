package email

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/constants"
)

// A template that is a valid EmailTemplate but missing from the renderer's map is not an error anyone sees: the send fails inside the notification-service after the publishing request has already returned 200, so the mail is silently dropped. purchase_order_submission and statement_of_account shipped that way. This walks the enum so a new template cannot repeat it.
func TestEveryEmailTemplateIsRegistered(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	if apiErr != nil {
		t.Fatalf("construct renderer: %v", apiErr)
	}

	for _, value := range constants.EmailTemplate("").EnumValues() {
		templateID := constants.EmailTemplate(value)
		if _, apiErr := renderer.RenderTemplate(context.Background(), templateID, map[string]any{}); apiErr != nil {
			t.Errorf("template %q does not render: %v", value, apiErr)
		}
	}
}
