package constants

// Color represents a color code.
type Color string

const (
	// ColorBlue indicates the color blue.
	ColorBlue Color = "blue"
	// ColorBrown indicates the color brown.
	ColorBrown Color = "brown"
	// ColorDefault indicates the default color.
	ColorDefault Color = "default"
	// ColorGray indicates the color gray.
	ColorGray Color = "gray"
	// ColorGreen indicates the color green.
	ColorGreen Color = "green"
	// ColorOrange indicates the color orange.
	ColorOrange Color = "orange"
	// ColorPink indicates the color pink.
	ColorPink Color = "pink"
	// ColorPurple indicates the color purple.
	ColorPurple Color = "purple"
	// ColorRed indicates the color red.
	ColorRed Color = "red"
	// ColorYellow indicates the color yellow.
	ColorYellow Color = "yellow"
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
