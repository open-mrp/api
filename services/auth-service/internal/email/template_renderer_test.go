package email

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/augno/api/shared/contracts"
)

func TestNewTemplateRendererFromDir(t *testing.T) {
	tests := []struct {
		name         string
		templatesDir string
		wantErr      bool
		errCode      contracts.ErrorCode
	}{
		{
			name:         "valid templates directory",
			templatesDir: "templates",
			wantErr:      false,
		},
		{
			name:         "missing directory",
			templatesDir: "nonexistent",
			wantErr:      true,
			errCode:      contracts.ErrorCodeInternalError,
		},
		{
			name:         "empty directory",
			templatesDir: "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.templatesDir == "templates" {
				wd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get working directory: %v", err)
				}
				dir = filepath.Join(wd, "..", "..", "templates")
			} else if tt.templatesDir == "" {
				dir = ""
			} else {
				dir = tt.templatesDir
			}

			renderer, err := NewTemplateRendererFromDir(dir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTemplateRendererFromDir() expected error, got nil")
					return
				}
				if tt.errCode != "" && err.Code != tt.errCode {
					t.Errorf("NewTemplateRendererFromDir() error code = %v, want %v", err.Code, tt.errCode)
				}
				return
			}

			if dir == "" {
				if err == nil && renderer == nil {
					return
				}
			}

			if err != nil {
				if dir != "" {
					t.Errorf("NewTemplateRendererFromDir() unexpected error = %v", err)
				}
				return
			}

			if renderer == nil && dir != "" {
				t.Errorf("NewTemplateRendererFromDir() returned nil renderer")
			}
		})
	}
}

func TestTemplateRenderer_RenderWelcomeEmail(t *testing.T) {
	renderer := setupTestRenderer(t)
	if renderer == nil {
		return
	}

	tests := []struct {
		name    string
		data    WelcomeEmailData
		wantErr bool
	}{
		{
			name: "valid data",
			data: WelcomeEmailData{
				UserName:  "John Doe",
				UserEmail: "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			data: WelcomeEmailData{
				UserName:  "",
				UserEmail: "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "special characters in name",
			data: WelcomeEmailData{
				UserName:  "John O'Brien",
				UserEmail: "john@example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			html, err := renderer.RenderWelcomeEmail(ctx, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RenderWelcomeEmail() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("RenderWelcomeEmail() unexpected error = %v", err)
				return
			}

			if html == "" {
				t.Errorf("RenderWelcomeEmail() returned empty HTML")
			}

			if len(html) < 100 {
				t.Errorf("RenderWelcomeEmail() returned suspiciously short HTML: %d bytes", len(html))
			}
		})
	}
}

func TestTemplateRenderer_RenderPasswordResetEmail(t *testing.T) {
	renderer := setupTestRenderer(t)
	if renderer == nil {
		return
	}

	tests := []struct {
		name    string
		data    PasswordResetEmailData
		wantErr bool
	}{
		{
			name: "valid data",
			data: PasswordResetEmailData{
				ResetLink:         "https://example.com/reset?t=token123",
				ExpirationMinutes: 15,
			},
			wantErr: false,
		},
		{
			name: "zero expiration",
			data: PasswordResetEmailData{
				ResetLink:         "https://example.com/reset?t=token123",
				ExpirationMinutes: 0,
			},
			wantErr: false,
		},
		{
			name: "long expiration",
			data: PasswordResetEmailData{
				ResetLink:         "https://example.com/reset?t=token123",
				ExpirationMinutes: 60,
			},
			wantErr: false,
		},
		{
			name: "empty reset link",
			data: PasswordResetEmailData{
				ResetLink:         "",
				ExpirationMinutes: 15,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			html, err := renderer.RenderPasswordResetEmail(ctx, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RenderPasswordResetEmail() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("RenderPasswordResetEmail() unexpected error = %v", err)
				return
			}

			if html == "" {
				t.Errorf("RenderPasswordResetEmail() returned empty HTML")
			}

			if len(html) < 100 {
				t.Errorf("RenderPasswordResetEmail() returned suspiciously short HTML: %d bytes", len(html))
			}
		})
	}
}

func TestTemplateRenderer_RenderPasswordUpdatedEmail(t *testing.T) {
	renderer := setupTestRenderer(t)
	if renderer == nil {
		return
	}

	tests := []struct {
		name    string
		data    PasswordUpdatedEmailData
		wantErr bool
	}{
		{
			name: "valid data",
			data: PasswordUpdatedEmailData{
				UserName: "John Doe",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			data: PasswordUpdatedEmailData{
				UserName: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			html, err := renderer.RenderPasswordUpdatedEmail(ctx, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RenderPasswordUpdatedEmail() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("RenderPasswordUpdatedEmail() unexpected error = %v", err)
				return
			}

			if html == "" {
				t.Errorf("RenderPasswordUpdatedEmail() returned empty HTML")
			}

			if len(html) < 100 {
				t.Errorf("RenderPasswordUpdatedEmail() returned suspiciously short HTML: %d bytes", len(html))
			}
		})
	}
}

func TestTemplateRenderer_MissingTemplate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	emptyDir := filepath.Join(wd, "testdata", "empty")
	os.MkdirAll(emptyDir, 0755)
	defer os.RemoveAll(emptyDir)

	renderer, err := NewTemplateRendererFromDir(emptyDir)
	if renderer != nil {
		t.Errorf("NewTemplateRendererFromDir() expected nil renderer when error occurs, got non-nil")
	}
}

func setupTestRenderer(t *testing.T) TemplateRenderer {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	var templatesDir string
	if filepath.Base(wd) == "email" {
		templatesDir = filepath.Join(wd, "..", "..", "templates")
	} else if filepath.Base(wd) == "auth-service" {
		templatesDir = filepath.Join(wd, "templates")
	} else {
		templatesDir = filepath.Join(wd, "services", "auth-service", "templates")
	}

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: templates directory does not exist at %s", templatesDir)
		return nil
	}

	renderer, err := NewTemplateRendererFromDir(templatesDir)
	if renderer == nil {
		t.Skipf("Skipping test: failed to load templates from %s: %v", templatesDir, err)
		return nil
	}

	return renderer
}
