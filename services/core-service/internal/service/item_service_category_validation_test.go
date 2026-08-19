package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"go.uber.org/mock/gomock"
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

// attributeValidationRepos wires just the two repos the attribute/category checks read.
func attributeValidationRepos(t *testing.T, attributes []*domain.Attribute, properties []*domain.ItemCategoryProperty) domain.RepoFactory {
	t.Helper()
	ctrl := gomock.NewController(t)

	attributeRepo := repositorymock.NewMockAttributeRepo(ctrl)
	attributeRepo.EXPECT().GetByIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(attributes, nil).AnyTimes()

	categoryRepo := repositorymock.NewMockItemCategoryRepo(ctrl)
	categoryRepo.EXPECT().GetProperties(gomock.Any(), gomock.Any()).Return(properties, nil).AnyTimes()

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAttributeRepo().Return(attributeRepo).AnyTimes()
	repos.EXPECT().NewItemCategoryRepo().Return(categoryRepo).AnyTimes()
	return repos
}

func TestValidateAttributesForCategory(t *testing.T) {
	t.Parallel()

	red := &domain.Attribute{ID: "at_red", Value: "Red", PropertyID: "pp_color"}
	color := &domain.ItemCategoryProperty{ID: "pp_color"}
	size := &domain.ItemCategoryProperty{ID: "pp_size"}

	tests := []struct {
		name       string
		attributes []*domain.Attribute
		properties []*domain.ItemCategoryProperty
		ids        []string
		wantCode   apierror.ErrorCode
	}{
		{"no attributes requested", nil, nil, nil, ""},
		{"property carried by the category", []*domain.Attribute{red}, []*domain.ItemCategoryProperty{color, size}, []string{"at_red"}, ""},
		{"property not carried by the category", []*domain.Attribute{red}, []*domain.ItemCategoryProperty{size}, []string{"at_red"}, apierror.ErrorCodeValidationFailed},
		{"category carries nothing", []*domain.Attribute{red}, nil, []string{"at_red"}, apierror.ErrorCodeValidationFailed},
		{"attribute outside the account", nil, []*domain.ItemCategoryProperty{color}, []string{"at_red"}, apierror.ErrorCodeResourceNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repos := attributeValidationRepos(t, tt.attributes, tt.properties)

			apiErr := validateAttributesForCategory(context.Background(), repos, "ac_test", "itcg_test", tt.ids, "attribute_ids")

			if tt.wantCode == "" {
				if apiErr != nil {
					t.Fatalf("validateAttributesForCategory() = %v, want nil", apiErr)
				}
				return
			}
			if apiErr == nil {
				t.Fatalf("validateAttributesForCategory() = nil, want %s", tt.wantCode)
			}
			if apiErr.Code != tt.wantCode {
				t.Fatalf("validateAttributesForCategory() code = %s, want %s", apiErr.Code, tt.wantCode)
			}
		})
	}
}

func TestValidateCategoryCarriesItemAttributes(t *testing.T) {
	t.Parallel()

	worn := &domain.Item{ItemCategoryID: "itcg_current", Attributes: []*domain.ItemAttribute{{ID: "at_red", Value: "Red", PropertyID: "pp_color"}}}

	tests := []struct {
		name       string
		item       *domain.Item
		properties []*domain.ItemCategoryProperty
		wantErr    bool
	}{
		{"item carries no attributes", &domain.Item{}, nil, false},
		{"target category carries the property", worn, []*domain.ItemCategoryProperty{{ID: "pp_color"}}, false},
		{"target category strands the attribute", worn, []*domain.ItemCategoryProperty{{ID: "pp_size"}}, true},
		{"target category carries nothing", worn, nil, true},
	}

	// Re-assigning the item's own category must stay a no-op even when the item predates the rule.
	t.Run("target is the item's current category", func(t *testing.T) {
		t.Parallel()
		repos := attributeValidationRepos(t, nil, nil)
		if apiErr := validateCategoryCarriesItemAttributes(context.Background(), repos, worn, "itcg_current", "category_id"); apiErr != nil {
			t.Fatalf("validateCategoryCarriesItemAttributes() = %v, want nil", apiErr)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repos := attributeValidationRepos(t, nil, tt.properties)

			apiErr := validateCategoryCarriesItemAttributes(context.Background(), repos, tt.item, "itcg_target", "category_id")

			if tt.wantErr && apiErr == nil {
				t.Fatal("validateCategoryCarriesItemAttributes() = nil, want a validation error")
			}
			if !tt.wantErr && apiErr != nil {
				t.Fatalf("validateCategoryCarriesItemAttributes() = %v, want nil", apiErr)
			}
			if tt.wantErr && apiErr.Code != apierror.ErrorCodeValidationFailed {
				t.Fatalf("validateCategoryCarriesItemAttributes() code = %s, want %s", apiErr.Code, apierror.ErrorCodeValidationFailed)
			}
		})
	}
}
