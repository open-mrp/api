package constants

import "testing"

func TestUploadedAttachmentKindForContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		contentType string
		want        MessageAttachmentKind
	}{
		{"image/png", MessageAttachmentKindImage},
		{"IMAGE/JPEG", MessageAttachmentKindImage},
		{" application/pdf ", MessageAttachmentKindFile},
		{"", MessageAttachmentKindFile},
	}
	for _, tt := range tests {
		if got := UploadedAttachmentKindForContentType(tt.contentType); got != tt.want {
			t.Errorf("UploadedAttachmentKindForContentType(%q) = %q, want %q", tt.contentType, got, tt.want)
		}
	}
}
