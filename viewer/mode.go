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
	offset := getOffsetForLineNumber(m.Format.CursorY)
	l := len(*m.Format.Slices[m.Format.CursorY])
	end := l - 1 - offset
	if m.Format.CursorX > end {
		m.Format.CursorX = end
	}
}

func handleFormatModeMovement(m *TuiModel, str string) (ret bool) {
	if str == "right" {
		ret = true
		m.Format.CursorX++

		offset := getOffsetForLineNumber(m.Format.CursorY)
		x := m.Format.CursorX + offset + 1 // for the space at the end
		l := len(*m.Format.Slices[m.Format.CursorY])
		maxY := len(m.Format.Slices) - 1
		if l < x && m.Format.CursorY < maxY {
			m.Format.CursorX = 0
			m.Format.CursorY++
		} else if l < x && m.Format.CursorY < len(m.Format.Text)-1 {
			go Program.Send(
				tea.KeyMsg{
					Type: tea.KeyDown,
					Alt:  false,
				},
			)
		} else if m.Format.CursorY > maxY {
			m.Format.CursorX = maxY
		}
	} else if str == "left" {
		ret = true
		m.Format.CursorX--

		if m.Format.CursorX < 0 && m.Format.CursorY > 0 {
			m.Format.CursorY--

			offset := getOffsetForLineNumber(m.Format.CursorY)
			l := len(*m.Format.Slices[m.Format.CursorY])
			m.Format.CursorX = l - 1 - offset
		} else if m.Format.CursorX < 0 &&
			m.Format.CursorY == 0 &&
			m.viewport.YOffset > 0 {
			go Program.Send(
				tea.KeyMsg{
					Type: tea.KeyUp,
					Alt:  false,
				},
			)
		} else if m.Format.CursorX < 0 {
			m.Format.CursorX = 0
		}
	} else if str == "up" {
		ret = true
		if m.Format.CursorY > 0 {
			m.Format.CursorY--
		} else if m.viewport.YOffset > 0 {
			scrollUp(m)
		}
	} else if str == "down" {
		ret = true
		if m.Format.CursorY < m.viewport.Height-footerHeight && m.Format.CursorY < len(m.Format.Slices) {
			m.Format.CursorY++
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

	lineNumberOffset := getOffsetForLineNumber(m.Format.CursorY)

	// update UI
	pString := m.Format.Slices[m.Format.CursorY]
	if *pString != "" {
		min := Max(m.Format.CursorX+lineNumberOffset+1, 0)
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
		cursor := m.Format.RunningOffsets[m.viewport.YOffset+m.Format.CursorY]

		first := replacement[:Max(cursor, 0)]
		middle := strings.TrimSpace(val[lineNumberOffset:])
		last := replacement[m.Format.RunningOffsets[m.viewport.YOffset+m.Format.CursorY+1]:]

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
	m.Format.CursorX++
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
