package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExternalUserPhoto(t *testing.T) {
	for _, tc := range []struct {
		name      string
		imageURL  *string
		wantURL   string
		wantShown bool
	}{
		{"no photo", nil, "", false},
		{"identity provider avatar", new("https://lh3.googleusercontent.com/a/abc=s96-c"), "https://lh3.googleusercontent.com/a/abc=s96-c", true},
		{"uploaded here", new("/v1/core/users/us_1/photo"), "", false},
		{"uploaded through the dashboard", new("/v1/users/us_1/photo"), "", false},
		{"a signed link into the photos bucket", new("https://augno-user-photos.s3.us-east-1.amazonaws.com/a/b.jpg?X-Amz-Signature=x"), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, shown := externalUserPhoto(tc.imageURL)
			assert.Equal(t, tc.wantShown, shown)
			assert.Equal(t, tc.wantURL, url)
		})
	}
}
