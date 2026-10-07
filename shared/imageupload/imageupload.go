// Package imageupload decides whether an uploaded file is an image the platform will store. Logos and
// favicons are served from a public CDN under the content type they were stored with, so the type is
// read from the file's own bytes; the one a client names is never trusted.
package imageupload

import (
	"fmt"
	"net/http"
	"slices"

	apierror "github.com/open-mrp/api/shared/errors"
)

// MaxBytes is the largest image an upload accepts: the 10 MB the dashboard API's upload routes took.
const MaxBytes = 10 << 20

// bodyParam names the raw request body an image is sent as, which has no field name of its own.
const bodyParam = "body"

// photoTypes are the raster formats every browser renders in an <img>. SVG is left out: it can carry
// script.
var photoTypes = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

// faviconTypes add the ICO format browsers also take for a tab icon.
var faviconTypes = append(slices.Clone(photoTypes), "image/x-icon")

// Photo returns the media type of a logo or user photo, or a client error when it is not a PNG, JPEG,
// GIF, or WebP image.
func Photo(data []byte) (string, *apierror.APIError) {
	return detect(data, photoTypes, "PNG, JPEG, GIF, or WebP")
}

// Favicon returns the media type of a favicon, or a client error when it is not a PNG, JPEG, GIF, WebP,
// or ICO image.
func Favicon(data []byte) (string, *apierror.APIError) {
	return detect(data, faviconTypes, "PNG, JPEG, GIF, WebP, or ICO")
}

func detect(data []byte, allowed []string, formats string) (string, *apierror.APIError) {
	if len(data) == 0 {
		return "", apierror.NewMissingFieldError("The request body is empty. Send the image itself as the raw request body.", bodyParam)
	}
	if len(data) > MaxBytes {
		return "", apierror.NewRequestTooLargeError(fmt.Sprintf("The image is larger than the %d MB limit.", MaxBytes>>20))
	}
	contentType := http.DetectContentType(data)
	if !slices.Contains(allowed, contentType) {
		return "", apierror.NewInvalidFormatError(fmt.Sprintf("The request body is not a %s image.", formats), bodyParam)
	}
	return contentType, nil
}
