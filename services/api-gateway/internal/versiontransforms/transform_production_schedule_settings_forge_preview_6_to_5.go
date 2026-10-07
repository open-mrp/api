package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&productionScheduleSettingsForgePreview6To5{})
}

// productionScheduleSettingsForgePreview6To5 downgrades production schedule settings from 1.0.forge-preview.6 to 1.0.forge-preview.5.
//
// preview.6 made `changeover_labor_rate` nullable: it is null unless the caller holds costs:read. preview.5 typed it as a number, so a preview.5 caller who cannot read costs receives 0 there instead; a caller who can receives the rate in both versions. Requests did not change.
type productionScheduleSettingsForgePreview6To5 struct{}

var productionScheduleSettingsCostKeys = map[string][]string{
	string(constants.ObjectTypeProductionScheduleSettings): {"changeover_labor_rate"},
}

func (t *productionScheduleSettingsForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *productionScheduleSettingsForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *productionScheduleSettingsForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeProductionScheduleSettings}
}

func (t *productionScheduleSettingsForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroHiddenCostsIn(data, productionScheduleSettingsCostKeys)
	return data
}

func (t *productionScheduleSettingsForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}
