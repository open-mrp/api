package main

import "testing"

func TestValidateRequires(t *testing.T) {
	for tag, want := range map[string]bool{
		"":                                    false,
		"required":                            true,
		"required,min=1,dive,required":        true,
		"omitempty,max=100,dive,required":     false,
		"omitempty,dive,required,max=255":     false,
		"dive,required":                       false,
		"omitempty,required_with=other_field": true,
	} {
		if got := validateRequires(tag); got != want {
			t.Errorf("validateRequires(%q) = %v, want %v", tag, got, want)
		}
	}
}
