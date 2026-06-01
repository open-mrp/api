package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSupplierMaterial,
		Load:       resourceloaders.LoadSupplierMaterials,
		Subs: []resourcekit.SubField{
			{
				Key:         "material",
				Target:      constants.ObjectTypeMaterial,
				ExtractRefs: extractMaterialRefsFromSupplierMaterial,
				Populate:    populateMaterialOnSupplierMaterial,
			},
		},
	})
}

func extractMaterialRefsFromSupplierMaterial(_ context.Context, parent any) []any {
	sm := parent.(*apiresource.SupplierMaterial)
	if sm.Material == nil {
		return nil
	}
	return []any{sm.Material}
}

func populateMaterialOnSupplierMaterial(ctx context.Context, parent any, _ map[string]any) {
	sm := parent.(*apiresource.SupplierMaterial)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSupplierMaterial, sm.ID, "material")
	if !ok {
		return
	}
	sm.Material = v.(*apiresource.Material)
}
