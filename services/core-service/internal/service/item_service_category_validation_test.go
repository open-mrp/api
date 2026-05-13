package service

import (
	"testing"

	"github.com/augno/api/shared/constants"
)

func TestCategoryTypeMatchesItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		itemType       string
		categoryType   string
		wantCompatible bool
	}{
		{"material to material category", string(constants.ItemTypeCodeMaterial), string(constants.ItemCategoryTypeMaterial), true},
		{"material to product category", string(constants.ItemTypeCodeMaterial), string(constants.ItemCategoryTypeProduct), false},
		{"product to product category", string(constants.ItemTypeCodeProduct), string(constants.ItemCategoryTypeProduct), true},
		{"product to material category", string(constants.ItemTypeCodeProduct), string(constants.ItemCategoryTypeMaterial), false},
		{"part to product category", string(constants.ItemTypeCodePart), string(constants.ItemCategoryTypeProduct), true},
		{"part to material category", string(constants.ItemTypeCodePart), string(constants.ItemCategoryTypeMaterial), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := categoryTypeMatchesItem(tt.itemType, tt.categoryType)
			if got != tt.wantCompatible {
				t.Fatalf("categoryTypeMatchesItem(%q, %q) = %v, want %v", tt.itemType, tt.categoryType, got, tt.wantCompatible)
			}
		})
	}
}
