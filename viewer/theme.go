package viewer

const (
	highlightKey = "highlight"
	headerBackgroundKey = "headerBackground"
	headerBorderBackgroundKey = "headerBorderBackground"
	headerForegroundKey = "headerForeground"
	footerForegroundColorKey = "footerForegroundColor"
	headerBottomColorKey = "headerBottomColor"
	headerTopForegroundColorKey = "headerTopForegroundColor"
	borderColorKey = "borderColor"
	textColorKey = "textColor"
)

var (
	SelectedTheme = 0
	ValidThemes   = []string{
		"default", // 0
		"nord",   // 1
		"solarized", // not accurate but whatever
	}
	ThemesMap = map[int]map[string]string{
		2: {
			headerBackgroundKey:         "#268bd2",
			headerBorderBackgroundKey:   "#268bd2",
			headerBottomColorKey:        "#586e75",
			borderColorKey:              "#586e75",
			textColorKey:                "#fdf6e3",
			headerForegroundKey:         "#fdf6e3",
			highlightKey:                "#2aa198", // change to whatever
			footerForegroundColorKey:    "#d33682",
			headerTopForegroundColorKey: "#d33682",
		},
		1: {
			headerBackgroundKey:         "#5e81ac",
			headerBorderBackgroundKey:   "#5e81ac",
			headerBottomColorKey:        "#5e81ac",
			borderColorKey:              "#eceff4",
			textColorKey:                "#eceff4",
			headerForegroundKey:         "#eceff4",
			highlightKey:                "#88c0d0", // change to whatever
			footerForegroundColorKey:    "#b48ead",
			headerTopForegroundColorKey: "#b48ead",
		},
		0: {
			headerBackgroundKey:         "#505050",
			headerBorderBackgroundKey:   "#505050",
			headerBottomColorKey:        "#FFFFFF",
			borderColorKey:              "#FFFFFF",
			textColorKey:                "#FFFFFF",
			headerForegroundKey:         "#FFFFFF",
			highlightKey:                "#A0A0A0", // change to whatever
			footerForegroundColorKey:    "#C2C2C2",
			headerTopForegroundColorKey: "#C2C2C2",
		},
	}
)
