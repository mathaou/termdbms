package viewer

func GetHelpText() (help string) {
	help = "Help:\n" +
		"\t-p\tdatabase path (absolute)\n" +
		"\t-h\tprints this message\n" +
		"Controls:\n" +
		"MOUSE\n" +
		"\tScroll up + down to navigate table\n" +
		"\tMove cursor to select cells for full screen viewing\n" +
		"KEYBOARD\n" +
		"\t[WASD] to move around cells, and also move columns if close to edge.\n" +
		"\t[ENTER] to select selected cell for full screen view\n" +
		"\t[BACKSPACE] to delete text before cursor in edit mode.\n" +
		"\t[UP/K and DOWN/J] to navigate schemas\n" +
		"\t[LEFT/H and RIGHT/L] to move around columns if greater than available width\n" +
		"\t\tAlso to control the cursor of the text editor in edit mode.\n" +
		"\t[M(scroll up) and N(scroll down)] to scroll manually\n" +
		"\t[Q or CTRL+C] to quit program\n" +
		"\t[B] to toggle borders!\n" +
		"\t[C] to expand column!\n" +
		"\t[P] in selection mode to write cell to file\n" +
		"\t[ESC] to exit full screen view\n"

	return help
}
