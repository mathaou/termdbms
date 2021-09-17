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
		"left",
		"right",
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

func handleEditInput(m *TuiModel, str, val string) (ret bool) {
	selectedInput := &m.textInput.Model
	input := selectedInput.Value()
	inputLen := len(input)

	if str == "backspace" {
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

		ret = true
	} else if str == "enter" { // writes your selection
		m.textInput.EnterBehavior(m, selectedInput, input)
		ret = true
	}

	return ret
}

func handleEditMovement(m *TuiModel, str, val string) (ret bool) {
	selectedInput := &m.textInput.Model
	if str == "home" {
		selectedInput.setCursor(0)

		ret = true
	} else if str == "end" {
		if len(val) > 0 {
			selectedInput.setCursor(len(val) - 1)
		}

		ret = true
	} else if str == "left" {
		cursorPosition := selectedInput.Cursor()

		if cursorPosition == selectedInput.offset && cursorPosition != 0 {
			selectedInput.offset--
			selectedInput.offsetRight--
		}

		if cursorPosition != 0 {
			selectedInput.SetCursor(cursorPosition - 1)
		}

		ret = true
	} else if str == "right" {
		cursorPosition := selectedInput.Cursor()

		if cursorPosition == selectedInput.offsetRight {
			selectedInput.offset++
			selectedInput.offsetRight++
		}

		selectedInput.setCursor(cursorPosition + 1)

		ret = true
	}

	return ret
}

func handleFormatMovement(m *TuiModel, str string) (ret bool) {
	switch str {
	case "pgdown":
		l := len(m.Format.Text) - 1
		for i := 0; i < m.viewport.Height && m.viewport.YOffset < l; i++ {
			scrollDown(m)
		}
		break
	case "pgup":
		for i := 0; i < m.viewport.Height && m.viewport.YOffset > 0; i++ {
			scrollUp(m)
		}
		break
	case "home":
		m.viewport.YOffset = 0
		break
	case "end":
		m.viewport.YOffset = len(m.Format.Text) - m.viewport.Height
		break
	case "right":
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

		break
	case "left":
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

		break
	case "up":
		ret = true
		if m.Format.CursorY > 0 {
			m.Format.CursorY--
		} else if m.viewport.YOffset > 0 {
			scrollUp(m)
		}

		break
	case "down":
		ret = true
		if m.Format.CursorY < m.viewport.Height-footerHeight && m.Format.CursorY < len(m.Format.Slices) {
			m.Format.CursorY++
		} else {
			scrollDown(m)
		}
	}

	return ret
}

//TODO: format mode delete/insert doesn't work

func handleFormatInput(m *TuiModel, str string) (ret bool) {
	switch str {
	case "enter":

		break
	case "backspace":
		cursor := m.Format.CursorX + formatModeOffset
		input := m.Format.Slices[m.Format.CursorY]
		inputLen := len(*input)
		runes := []rune(*input)
		if m.Format.CursorX > 0 {
			if cursor == inputLen && inputLen > 0 {
				*input = (*input)[0 : inputLen-1]
			} else if cursor > 0 {
				min := Max(cursor, 0)
				min = Min(min, inputLen-1)
				first := runes[:min-1]
				last := runes[min:]
				*input = string(first) + string(last)
			}

			break
		} else if m.Format.CursorY > 0 && m.Format.CursorX == 0 {
			cursor := m.Format.RunningOffsets[m.Format.CursorY+m.viewport.YOffset] + m.Format.CursorX
			runes := []rune(m.selectionText)
			newline := runes[cursor]
			if newline == '\n' {
				min := Max(cursor, 0)
				min = Min(min, len(m.selectionText)-1)
				first := runes[:min-1]
				last := runes[min:]
				m.selectionText = string(first) + string(last)
				if m.viewport.YOffset+m.viewport.Height == len(m.Format.Text) {
					m.viewport.YOffset--
				}
				m.Format.Text = getFormattedTextBuffer(m)
				m.SetViewSlices()
				m.Format.CursorY--
				m.Format.CursorX = m.Format.RunningOffsets[m.Format.CursorY] - 1
				ret = true
			}
		} else {
			ret = true
		}

		break
	}

	return ret
}

func handleFormatMode(m *TuiModel, str string) {
	var (
		val         string
		replacement string
	)
	if handleFormatMovement(m, str) || handleFormatInput(m, str) {
		return
	}

	for _, v := range inputBlacklist {
		if str == v {
			return
		}
	}

	lineNumberOffset := getOffsetForLineNumber(m.Format.CursorY)

	pString := m.Format.Slices[m.Format.CursorY]
	delta := 1
	if str != "backspace" {
		// update UI
		if *pString != "" {
			min := Max(m.Format.CursorX+lineNumberOffset+1, 0)
			min = Min(min, len(*pString))
			first := (*pString)[:min]
			last := (*pString)[min:]
			val = first + str + last
		} else {
			val = *pString + str
		}
	} else {
		delta = -1
		val = *pString
	}

	_, err := formatJson(m.selectionText)
	validJson := err == nil

	// if json special rules
	replacement = m.selectionText
	cursor := m.Format.RunningOffsets[m.viewport.YOffset+m.Format.CursorY]

	fIndex := Max(cursor, 0)
	lIndex := m.viewport.YOffset + m.Format.CursorY + 1

	first := replacement[:fIndex]
	middle := strings.TrimSpace(val[lineNumberOffset:])
	last := replacement[m.Format.RunningOffsets[lIndex]:]

	if !validJson {
		middle += "\n"
	}

	replacement = first + // replace the entire line the edit appears on
		middle + // insert the edit
		last // top the edit off with the rest of the string

	m.selectionText = replacement
	if len(*pString) == formatModeOffset && str != "backspace" { // insert on empty lines behaves funny
		*pString = *pString + str
	} else {
		*pString = val
	}

	m.Format.CursorX += delta

	for i := m.viewport.YOffset + m.Format.CursorY + 1; i < len(m.Format.RunningOffsets); i++ {
		m.Format.RunningOffsets[i] += delta
	}
}

// handleEditMode implementation is kind of jank, but we can clean it up later
func handleEditMode(m *TuiModel, str string) {
	var (
		input string
		val   string
	)
	selectedInput := &m.textInput.Model
	input = selectedInput.Value()
	if input != "" && selectedInput.Cursor() <= len(input)-1 {
		min := Max(selectedInput.Cursor(), 0)
		min = Min(min, len(input)-1)
		first := input[:min]
		last := input[min:]
		val = first + str + last
	} else {
		val = input + str
	}

	if str == "esc" {
		selectedInput.SetValue("")
		return
	}

	if handleEditMovement(m, str, val) || handleEditInput(m, str, val) {
		return
	}

	for _, v := range inputBlacklist {
		if str == v {
			return
		}
	}

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
