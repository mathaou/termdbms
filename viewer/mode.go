package viewer

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

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

func moveCursorWithinBounds(m *TuiModel) {
	offset := getOffsetForLineNumber(m.formatCursorY)
	l := len(*m.FormatSlices[m.formatCursorY])
	end := l - 1 - offset
	if m.formatCursorX > end {
		m.formatCursorX = end
	}
}

func handleFormatModeMovement(m *TuiModel, str string) (ret bool) {
	if str == "right" {
		ret = true
		m.formatCursorX++

		offset := getOffsetForLineNumber(m.formatCursorY)
		x := m.formatCursorX + offset + 1 // for the space at the end
		l := len(*m.FormatSlices[m.formatCursorY])
		maxY := len(m.FormatSlices) - 1
		if l < x && m.formatCursorY < maxY {
			m.formatCursorX = 0
			m.formatCursorY++
		} else if l < x && m.formatCursorY < len(m.FormatText)-1 {
			go Program.Send(
				tea.KeyMsg{
					Type: tea.KeyDown,
					Alt:  false,
				},
			)
		} else if m.formatCursorY > maxY {
			m.formatCursorX = maxY
		}
	} else if str == "left" {
		ret = true
		m.formatCursorX--

		if m.formatCursorX < 0 && m.formatCursorY > 0 {
			m.formatCursorY--

			offset := getOffsetForLineNumber(m.formatCursorY)
			l := len(*m.FormatSlices[m.formatCursorY])
			m.formatCursorX = l - 1 - offset
		} else if m.formatCursorX < 0 &&
			m.formatCursorY == 0 &&
			m.viewport.YOffset > 0 {
			go Program.Send(
				tea.KeyMsg{
					Type: tea.KeyUp,
					Alt:  false,
				},
			)
		} else if m.formatCursorX < 0 {
			m.formatCursorX = 0
		}
	} else if str == "up" {
		ret = true
		if m.formatCursorY > 0 {
			m.formatCursorY--
		} else if m.viewport.YOffset > 0 {
			scrollUp(m)
		}
	} else if str == "down" {
		ret = true
		if m.formatCursorY < m.viewport.Height-footerHeight && m.formatCursorY < len(m.FormatSlices) {
			m.formatCursorY++
		} else {
			scrollDown(m)
		}
	}

	return ret
}

func handleFormatMode(m *TuiModel, str string) {
	var (
		val         string
		replacement string
	)
	if handleFormatModeMovement(m, str) {
		return
	}

	for _, v := range inputBlacklist {
		if str == v {
			return
		}
	}

	lineNumberOffset := getOffsetForLineNumber(m.formatCursorY)

	// update UI
	pString := m.FormatSlices[m.formatCursorY]
	if *pString != "" {
		min := Max(m.formatCursorX+lineNumberOffset+1, 0)
		min = Min(min, len(*pString)-1)
		first := (*pString)[:min]
		last := (*pString)[min:]
		val = first + str + last
	} else {
		val = *pString + str
	}


	// if json special rules
	if _, err := formatJson(replacement); err != nil {
		replacement = m.selectionText
		cursor := m.FormatRunningOffsets[m.viewport.YOffset+m.formatCursorY]

		first := replacement[:Max(cursor, 0)]
		middle := strings.TrimSpace(val[lineNumberOffset:])
		last := replacement[m.FormatRunningOffsets[m.viewport.YOffset+m.formatCursorY+1]:]

		replacement = first + // replace the entire line the edit appears on
			middle + // insert the edit
			last // top the edit off with the rest of the string
		// text input from here on out
		var i interface{}
		if replacement != "" {
			i = GetInterfaceFromString(replacement, m.formatInput.Original)
		} else {
			i = GetInterfaceFromString(replacement, m.formatInput.Original)
		}

		// valid json or other, commit
		*m.formatInput.Original = i
	}

	*pString = val
	m.formatCursorX++
}

// handleEditMode implementation is kind of jank, but we can clean it up later
func handleEditMode(m *TuiModel, str string) {
	var (
		input string
		val   string
	)
	line := m.textInput
	input = line.Model.Value()
	if input != "" && line.Model.Cursor() <= len(input)-1 {
		min := Max(line.Model.Cursor(), 0)
		min = Min(min, len(input)-1)
		first := input[:min]
		last := input[min:]
		val = first + str + last
	} else {
		val = input + str
	}

	inputLen := len(input)
	selectedInput := &m.textInput.Model
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
		m.textInput.EnterBehavior(m, selectedInput, input)
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
