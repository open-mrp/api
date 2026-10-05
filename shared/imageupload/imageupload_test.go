package imageupload

import (
	"bytes"
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
)

var (
	png  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpeg = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	gif  = []byte("GIF89a\x01\x00\x01\x00")
	webp = []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
	ico  = []byte("\x00\x00\x01\x00\x01\x00\x10\x10")
)

func TestPhoto_TakesTheTypeFromTheBytes(t *testing.T) {
	t.Parallel()
	for want, data := range map[string][]byte{
		"image/png":  png,
		"image/jpeg": jpeg,
		"image/gif":  gif,
		"image/webp": webp,
	} {
		got, apiErr := Photo(data)
		if apiErr != nil {
			t.Errorf("%s: unexpected error %v", want, apiErr)
			continue
		}
		if got != want {
			t.Errorf("Photo() = %q, want %q", got, want)
		}
	}
}

func TestPhoto_RefusesWhatIsNotARasterImage(t *testing.T) {
	t.Parallel()
	for name, data := range map[string][]byte{
		"html": []byte("<html><script>alert(1)</script></html>"),
		"svg":  []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"text": []byte("not an image"),
		"pdf":  []byte("%PDF-1.4\n"),
		"ico":  ico,
	} {
		_, apiErr := Photo(data)
		if apiErr == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		if apiErr.Code != apierror.ErrorCodeInvalidFormat || apiErr.Param != "body" {
			t.Errorf("%s: got %s on %q, want invalid_format on body", name, apiErr.Code, apiErr.Param)
		}
	}
}

func TestFavicon_AlsoTakesICO(t *testing.T) {
	t.Parallel()
	got, apiErr := Favicon(ico)
	if apiErr != nil || got != "image/x-icon" {
		t.Fatalf("Favicon(ico) = %q, %v", got, apiErr)
	}
	if _, apiErr := Favicon([]byte("<html></html>")); apiErr == nil {
		t.Error("Favicon accepted HTML")
	}
}

func TestPhoto_RefusesAnEmptyBody(t *testing.T) {
	t.Parallel()
	_, apiErr := Photo(nil)
	if apiErr == nil || apiErr.Code != apierror.ErrorCodeMissingField || apiErr.Param != "body" {
		t.Fatalf("Photo(nil) = %v, want missing_field on body", apiErr)
	}
}

func TestPhoto_RefusesAnImageOverTheLimit(t *testing.T) {
	t.Parallel()
	if _, apiErr := Photo(append(bytes.Clone(png), make([]byte, MaxBytes-len(png))...)); apiErr != nil {
		t.Fatalf("an image of exactly MaxBytes was refused: %v", apiErr)
	}
	_, apiErr := Photo(append(bytes.Clone(png), make([]byte, MaxBytes)...))
	if apiErr == nil || apierror.GetHTTPStatusCode(apiErr.Code) != 413 {
		t.Fatalf("an image over MaxBytes = %v, want 413", apiErr)
	}
}
