package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	s3client "github.com/augno/api/shared/cloud/s3"
	apierror "github.com/augno/api/shared/errors"

	_ "golang.org/x/image/webp" // register WEBP decoder (account logos are often .webp)
)

// account_branding.logo_url does not hold a URL. UploadAccountLogo writes the object key it uploaded to ("<account_id>/logo.png") into that column, and every read path is expected to sign it. Rows predating that, and the dev seed, still hold an absolute URL — so both shapes have to resolve.
//
// Anything that hands the raw column to an HTTP client or an <img src> gets a broken image in production and works in dev, which is how this survived in the price list and the order acknowledgement.

// brandingLogoURLExpiry outlives the acknowledgement email in the recipient's inbox for a working day; the link is regenerated on every send.
const brandingLogoURLExpiry = 12 * time.Hour

// brandingAssets resolves stored branding references against the bucket they were uploaded to. Bound to one bucket so callers never repeat it.
type brandingAssets struct {
	store  s3client.ObjectStore
	bucket string
}

// NewBrandingAssets binds the object store to the account-photos bucket.
func NewBrandingAssets(store s3client.ObjectStore, bucket string) BrandingAssets {
	return BrandingAssets{brandingAssets{store: store, bucket: bucket}}
}

// BrandingAssets reads an account's branding images. The zero value resolves absolute URLs and nothing else, which is what a service that was never wired an object store can honestly do.
type BrandingAssets struct{ inner brandingAssets }

// LogoURL turns a stored branding logo reference into a URL a mail client can load. Best-effort: an unresolvable reference yields "" and the caller renders without a logo rather than with a broken one.
func (b BrandingAssets) LogoURL(ctx context.Context, stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	if isAbsoluteHTTPURL(stored) {
		return stored
	}
	if b.inner.store == nil {
		return ""
	}
	url, apiErr := b.inner.store.GetPresignedURL(ctx, b.inner.bucket, stored, brandingLogoURLExpiry)
	if apiErr != nil {
		return ""
	}
	return url
}

// LogoImage reads a stored branding logo reference and normalizes it to PNG bytes for embedding in a PDF. It decodes any supported source format — PNG, JPEG, GIF, or WEBP (account logos are commonly .webp, which fpdf cannot embed directly) — and re-encodes to PNG. Best-effort: any failure (missing object, network, timeout, undecodable) returns ("", nil) and the document falls back to a text-only letterhead.
func (b BrandingAssets) LogoImage(ctx context.Context, stored string) (imageType string, data []byte) {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return "", nil
	}

	var raw []byte
	if isAbsoluteHTTPURL(stored) {
		raw = fetchLogoBytesOverHTTP(ctx, stored)
	} else if b.inner.store != nil {
		var apiErr *apierror.APIError
		// Read the object directly rather than signing a URL and fetching it back over the internet: the service already holds credentials to the bucket.
		raw, apiErr = b.inner.store.Get(ctx, b.inner.bucket, stored)
		if apiErr != nil {
			raw = nil
		}
	}
	if len(raw) == 0 {
		return "", nil
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", nil
	}
	// Flatten any transparency onto white before encoding: fpdf renders alpha PNGs unreliably (logos come out faint/washed-out), and the PDF background is white.
	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)
	var out bytes.Buffer
	if err := png.Encode(&out, flat); err != nil {
		return "", nil
	}
	return "PNG", out.Bytes()
}

func isAbsoluteHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func fetchLogoBytesOverHTTP(ctx context.Context, url string) []byte {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL from account branding logo stored server-side
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil
	}
	return body
}
