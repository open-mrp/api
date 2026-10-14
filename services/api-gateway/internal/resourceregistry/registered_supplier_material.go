package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSupplierMaterial,
		Load:       resourceloaders.LoadSupplierMaterials,
		Subs: []include.SubField{
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
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSupplierMaterial, sm.ID, "material")
	if !ok {
		return
	}
	sm.Material = v.(*apiresource.Material)
}
