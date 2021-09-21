package viewer

const (
	HighlightKey                = "Highlight"
	HeaderBackgroundKey         = "HeaderBackground"
	HeaderBorderBackgroundKey   = "HeaderBorderBackground"
	HeaderForegroundKey         = "HeaderForeground"
	FooterForegroundColorKey    = "FooterForeground"
	HeaderBottomColorKey        = "HeaderBottom"
	HeaderTopForegroundColorKey = "HeaderTopForeground"
	BorderColorKey              = "BorderColor"
	TextColorKey                = "TextColor"
)

var (
	SelectedTheme = 0
	ValidThemes   = []string{
		"default",   // 0
		"nord",      // 1
		"solarized", // not accurate but whatever
	}
	ThemesMap = map[int]map[string]string{
		2: {
			HeaderBackgroundKey:         "#268bd2",
			HeaderBorderBackgroundKey:   "#268bd2",
			HeaderBottomColorKey:        "#586e75",
			BorderColorKey:              "#586e75",
			TextColorKey:                "#fdf6e3",
			HeaderForegroundKey:         "#fdf6e3",
			HighlightKey:                "#2aa198",
			FooterForegroundColorKey:    "#d33682",
			HeaderTopForegroundColorKey: "#d33682",
		},
		1: {
			HeaderBackgroundKey:         "#5e81ac",
			HeaderBorderBackgroundKey:   "#5e81ac",
			HeaderBottomColorKey:        "#5e81ac",
			BorderColorKey:              "#eceff4",
			TextColorKey:                "#eceff4",
			HeaderForegroundKey:         "#eceff4",
			HighlightKey:                "#88c0d0",
			FooterForegroundColorKey:    "#b48ead",
			HeaderTopForegroundColorKey: "#b48ead",
		},
		0: {
			HeaderBackgroundKey:         "#505050",
			HeaderBorderBackgroundKey:   "#505050",
			HeaderBottomColorKey:        "#FFFFFF",
			BorderColorKey:              "#FFFFFF",
			TextColorKey:                "#FFFFFF",
			HeaderForegroundKey:         "#FFFFFF",
			HighlightKey:                "#A0A0A0",
			FooterForegroundColorKey:    "#C2C2C2",
			HeaderTopForegroundColorKey: "#C2C2C2",
		},
	}
)
