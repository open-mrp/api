// Package apiversion declares OpenMRP's API versions and registers them with apikit's version package at init, so
// every binary that parses or propagates a version (every service imports this through shared/contracts) knows them.
//
// 1.0.forge is the first stable version. 1.0.forge-preview.9 stays supported while its last clients migrate; the
// gateway's versiontransforms downgrade forge responses to it and upgrade its requests.
package apiversion

import "github.com/open-mrp/apikit/version"

// Header is the request header that names the API version.
const Header = "OpenMRP-Version"

var (
	// V1_0_Forge is the first stable version, and the latest.
	V1_0_Forge = version.MustNew("1.0.forge")
	// V1_0_Forge_Preview9 is the last preview, kept while its clients move to 1.0.forge.
	V1_0_Forge_Preview9 = version.MustNew("1.0.forge-preview.9")
)

func init() {
	version.SetHeader(Header)
	version.RegisterVersions(V1_0_Forge, V1_0_Forge_Preview9)
}
