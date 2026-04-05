package constants

// Color represents a color code.
type Color string

const (
	ColorBlue    Color = "blue"
	ColorBrown   Color = "brown"
	ColorDefault Color = "default"
	ColorGray    Color = "gray"
	ColorGreen   Color = "green"
	ColorOrange  Color = "orange"
	ColorPink    Color = "pink"
	ColorPurple  Color = "purple"
	ColorRed     Color = "red"
	ColorYellow  Color = "yellow"
)

func (m Color) IsValid() bool {
	switch m {
	case ColorBlue, ColorBrown, ColorDefault, ColorGray, ColorGreen,
		ColorOrange, ColorPink, ColorPurple, ColorRed, ColorYellow:
		return true
	default:
		return false
	}
}

func (m Color) EnumValues() []string {
	return []string{
		string(ColorBlue),
		string(ColorBrown),
		string(ColorDefault),
		string(ColorGray),
		string(ColorGreen),
		string(ColorOrange),
		string(ColorPink),
		string(ColorPurple),
		string(ColorRed),
		string(ColorYellow),
	}
}
