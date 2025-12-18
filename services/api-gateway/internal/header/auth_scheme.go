package header

type AuthScheme string

const (
	AuthSchemeBasic  AuthScheme = "basic"
	AuthSchemeBearer AuthScheme = "bearer"
)
