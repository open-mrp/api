package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// stubBrandingStore implements s3.ObjectStore, recording what a branding lookup asked the bucket for.
type stubBrandingStore struct {
	object       []byte
	getBucket    string
	getKey       string
	presignKey   string
	presignedURL string
	getErr       *apierror.APIError
}

func (s *stubBrandingStore) Upload(context.Context, string, string, io.Reader, string) *apierror.APIError {
	return nil
}

func (s *stubBrandingStore) GetPresignedURL(_ context.Context, _, key string, _ time.Duration) (string, *apierror.APIError) {
	s.presignKey = key
	return s.presignedURL, nil
}

func (s *stubBrandingStore) GetPresignedPutURL(context.Context, string, string, string, time.Duration) (string, *apierror.APIError) {
	return "", nil
}

func (s *stubBrandingStore) FileExists(context.Context, string, string) (bool, *apierror.APIError) {
	return true, nil
}

func (s *stubBrandingStore) Get(_ context.Context, bucket, key string) ([]byte, *apierror.APIError) {
	s.getBucket, s.getKey = bucket, key
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.object, nil
}

func (s *stubBrandingStore) Delete(context.Context, string, string) *apierror.APIError { return nil }

func (s *stubBrandingStore) Copy(context.Context, string, string, string) *apierror.APIError {
	return nil
}

func redPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// account_branding.logo_url holds the S3 object key the upload wrote, not a URL. Handing that key to an HTTP client is a no-op that fails silently, so the price list and the acknowledgement printed a text-only letterhead in production while working in dev off a seeded absolute URL.
func TestBrandingAssets_ResolvesAStoredKeyAgainstTheBucket(t *testing.T) {
	store := &stubBrandingStore{object: redPNG(t), presignedURL: "https://s3.example.com/signed"}
	assets := NewBrandingAssets(store, "account-photos")

	imageType, data := assets.LogoImage(context.Background(), "ac_1/logo.png")
	if imageType != "PNG" || len(data) == 0 {
		t.Fatalf("LogoImage = (%q, %d bytes), want PNG bytes", imageType, len(data))
	}
	if store.getBucket != "account-photos" || store.getKey != "ac_1/logo.png" {
		t.Errorf("read %s/%s, want account-photos/ac_1/logo.png", store.getBucket, store.getKey)
	}

	if url := assets.LogoURL(context.Background(), "ac_1/logo.png"); url != "https://s3.example.com/signed" {
		t.Errorf("LogoURL = %q, want the signed URL", url)
	}
	if store.presignKey != "ac_1/logo.png" {
		t.Errorf("signed %q, want the stored key", store.presignKey)
	}
}

// Rows predating the key-based upload, and the dev seed, still hold an absolute URL. Signing one against the bucket would produce a link to an object that does not exist.
func TestBrandingAssets_PassesAnAbsoluteURLThrough(t *testing.T) {
	store := &stubBrandingStore{presignedURL: "https://s3.example.com/signed"}
	assets := NewBrandingAssets(store, "account-photos")

	const stored = "https://cdn.example.com/acme/logo.webp"
	if url := assets.LogoURL(context.Background(), stored); url != stored {
		t.Errorf("LogoURL = %q, want it unchanged", url)
	}
	if store.presignKey != "" {
		t.Errorf("signed %q, want no bucket lookup at all", store.presignKey)
	}
}

// Every caller treats a missing logo as a text-only letterhead, so no input may panic or fail the document — including a service that was never wired an object store.
func TestBrandingAssets_DegradesToNoLogo(t *testing.T) {
	cases := []struct {
		name   string
		assets BrandingAssets
		stored string
	}{
		{"no stored reference", NewBrandingAssets(&stubBrandingStore{}, "account-photos"), ""},
		{"blank stored reference", NewBrandingAssets(&stubBrandingStore{}, "account-photos"), "   "},
		{"object missing from the bucket", NewBrandingAssets(&stubBrandingStore{getErr: apierror.NewResourceNotFoundError("gone")}, "account-photos"), "ac_1/logo.png"},
		{"object is not a decodable image", NewBrandingAssets(&stubBrandingStore{object: []byte("not an image")}, "account-photos"), "ac_1/logo.png"},
		{"no object store wired", BrandingAssets{}, "ac_1/logo.png"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imageType, data := tc.assets.LogoImage(context.Background(), tc.stored)
			if imageType != "" || data != nil {
				t.Errorf("LogoImage = (%q, %d bytes), want nothing", imageType, len(data))
			}
			if url := tc.assets.LogoURL(context.Background(), tc.stored); url != "" {
				t.Errorf("LogoURL = %q, want empty", url)
			}
		})
	}
}
