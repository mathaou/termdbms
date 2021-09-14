package viewer

var (
	inputBlacklist = []string{
		"alt+[",
		"up",
		"down",
		"tab",
		"pgdown",
		"pgup",
	}
)

func handleFormatMode(m *TuiModel, str, input, val string) {
	if str == "right" {
		m.formatCursorX++
		offset := getOffsetForLineNumber(m.formatCursorY)
		x := m.formatCursorX + offset + 1 // for the space at the end
		l := len(m.FormatSlices[m.formatCursorY])
		if l <= x {
			m.formatCursorX = 0
			m.formatCursorY++
		}
	}
}

// handleEditMode implementation is kind of jank, but we can clean it up later
func handleEditMode(m *TuiModel, str, input, val string) {
	inputLen := len(input)
	lineEdit := m.GetSelectedLineEdit()
	selectedInput := &lineEdit.Model
	if str == "esc" {
		selectedInput.SetValue("")
		return
	}

	for _, v := range inputBlacklist {
		if str == v {
			return
		}
	}

	if str == "home" {
		selectedInput.setCursor(0)
	} else if str == "end" {
		if len(val) > 0 {
			selectedInput.setCursor(len(val) - 1)
		}
	} else if str == "left" {
		cursorPosition := selectedInput.Cursor()

		if cursorPosition == selectedInput.offset && cursorPosition != 0 {
			selectedInput.offset--
			selectedInput.offsetRight--
		}

		if cursorPosition != 0 {
			selectedInput.SetCursor(cursorPosition - 1)
		}
	} else if str == "right" {
		cursorPosition := selectedInput.Cursor()

		if cursorPosition == selectedInput.offsetRight {
			selectedInput.offset++
			selectedInput.offsetRight++
		}

		selectedInput.setCursor(cursorPosition + 1)
	} else if str == "backspace" {
		cursor := selectedInput.Cursor()
		runes := []rune(input)
		if cursor == inputLen && inputLen > 0 {
			selectedInput.SetValue(input[0 : inputLen-1])
		} else if cursor > 0 {
			min := Max(selectedInput.Cursor(), 0)
			min = Min(min, inputLen-1)
			first := runes[:min-1]
			last := runes[min:]
			selectedInput.SetValue(string(first) + string(last))
			selectedInput.SetCursor(selectedInput.Cursor() - 1)
		}
	} else if str == "enter" { // writes your selection
		lineEdit.EnterBehavior(m, selectedInput, input)
	} else {
		prePos := selectedInput.Cursor()
		if val != "" {
			selectedInput.SetValue(val)
		} else {
			selectedInput.SetValue(str)
		}

		if prePos != 0 {
			prePos = selectedInput.Cursor()
		}
		selectedInput.setCursor(prePos + 1)
	}
}
