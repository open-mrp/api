package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArchiveStatus_RoundTripsIsActive(t *testing.T) {
	for _, s := range []ArchiveStatus{ArchiveStatusActive, ArchiveStatusArchived} {
		assert.Equal(t, s, ArchiveStatusFromIsActive(s.IsActive()))
	}
	assert.True(t, ArchiveStatusActive.IsActive())
	assert.False(t, ArchiveStatusArchived.IsActive())
}

func TestPauseStatus_RoundTripsIsActive(t *testing.T) {
	for _, s := range []PauseStatus{PauseStatusActive, PauseStatusDisabled} {
		assert.Equal(t, s, PauseStatusFromIsActive(s.IsActive()))
	}
	assert.True(t, PauseStatusActive.IsActive())
	assert.False(t, PauseStatusDisabled.IsActive())
}

func TestIsActiveFilter(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name     string
		statuses []ArchiveStatus
		want     *bool
	}{
		{"omitted selects all", nil, nil},
		{"active", []ArchiveStatus{ArchiveStatusActive}, &yes},
		{"archived", []ArchiveStatus{ArchiveStatusArchived}, &no},
		{"both select all", []ArchiveStatus{ArchiveStatusArchived, ArchiveStatusActive}, nil},
		{"repeats", []ArchiveStatus{ArchiveStatusArchived, ArchiveStatusArchived}, &no},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsActiveFilter(tc.statuses))
		})
	}

	assert.Equal(t, &no, IsActiveFilter([]PauseStatus{PauseStatusDisabled}))
	assert.Equal(t, &yes, IsActiveFilter([]PauseStatus{PauseStatusActive}))
	assert.Nil(t, IsActiveFilter([]PauseStatus{PauseStatusActive, PauseStatusDisabled}))
}
