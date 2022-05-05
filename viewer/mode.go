package viewer

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	InputBlacklist = []string{
		"alt+",
		"ctrl+",
		"up",
		"down",
		"tab",
		"left",
		"enter",
		"right",
		"pgdown",
		"pgup",
	}
)

func PrepareFormatMode(m *TuiModel) {
	m.UI.FormatModeEnabled = true
	m.UI.EditModeEnabled = false
	m.TextInput.Model.SetValue("")
	m.FormatInput.Model.SetValue("")
	m.FormatInput.Model.Focus = true
	m.TextInput.Model.Focus = false
	m.TextInput.Model.Blur()
}

func MoveCursorWithinBounds(m *TuiModel) {
	defer func() {
		if recover() != nil {
			println("whoopsy")
		}
	}()
	offset := GetOffsetForLineNumber(m.Format.CursorY)
	l := len(*m.Format.EditSlices[m.Format.CursorY])

	end := l - 1 - offset
	if m.Format.CursorX > end {
		m.Format.CursorX = end
	}
}

func HandleEditInput(m *TuiModel, str, val string) (ret bool) {
	selectedInput := &m.TextInput.Model
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
		EditEnter(m)
		ret = true
	}

	return ret
}

func HandleEditMovement(m *TuiModel, str, val string) (ret bool) {
	selectedInput := &m.TextInput.Model
	if str == "home" {
		selectedInput.SetCursor(0)

		ret = true
	} else if str == "end" {
		if len(val) > 0 {
			selectedInput.SetCursor(len(val) - 1)
		}

		ret = true
	} else if str == "left" {
		cursorPosition := selectedInput.Cursor()

		if cursorPosition == selectedInput.Offset && cursorPosition != 0 {
			selectedInput.Offset--
			selectedInput.OffsetRight--
		}

		if cursorPosition != 0 {
			selectedInput.SetCursor(cursorPosition - 1)
		}

		ret = true
	} else if str == "right" {
		cursorPosition := selectedInput.Cursor()

		if cursorPosition == selectedInput.OffsetRight {
			selectedInput.Offset++
			selectedInput.OffsetRight++
		}

		selectedInput.SetCursor(cursorPosition + 1)

		ret = true
	}

	return ret
}

func HandleFormatMovement(m *TuiModel, str string) (ret bool) {
	lines := 0
	for _, v := range m.Format.EditSlices {
		if *v != "" {
			lines++
		}
	}
	switch str {
	case "pgdown":
		l := len(m.Format.Text) - 1
		for i := 0; i < m.Viewport.Height && m.Viewport.YOffset < l; i++ {
			ScrollDown(m)
		}
		ret = true
		break
	case "pgup":
		for i := 0; i <
			m.Viewport.Height && m.Viewport.YOffset > 0; i++ {
			ScrollUp(m)
		}
		ret = true
		break
	case "home":
		m.Viewport.YOffset = 0
		m.Format.CursorX = 0
		m.Format.CursorY = 0
		ret = true
		break
	case "end":
		m.Viewport.YOffset = len(m.Format.Text) - m.Viewport.Height
		m.Format.CursorY = Min(m.Viewport.Height-FooterHeight, strings.Count(m.Data().EditTextBuffer, "\n"))
		m.Format.CursorX = m.Format.RunningOffsets[len(m.Format.RunningOffsets)-1]
		ret = true
		break
	case "right":
		ret = true
		m.Format.CursorX++

		offset := GetOffsetForLineNumber(m.Format.CursorY)
		x := m.Format.CursorX + offset + 1 // for the space at the end
		l := len(*m.Format.EditSlices[m.Format.CursorY])
		maxY := lines - 1
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

			offset := GetOffsetForLineNumber(m.Format.CursorY)
			l := len(*m.Format.EditSlices[m.Format.CursorY])
			m.Format.CursorX = l - 1 - offset
		} else if m.Format.CursorX < 0 &&
			m.Format.CursorY == 0 &&
			m.Viewport.YOffset > 0 {
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
		} else if m.Viewport.YOffset > 0 {
			ScrollUp(m)
		}

		break
	case "down":
		ret = true
		if m.Format.CursorY < m.Viewport.Height-FooterHeight && m.Format.CursorY < lines-1 {
			m.Format.CursorY++
		} else {
			ScrollDown(m)
		}
	}

	return ret
}

func InsertCharacter(m *TuiModel, newlineOrTab string) {
	yOffset := Max(m.Viewport.YOffset, 0)
	cursor := m.Format.RunningOffsets[m.Format.CursorY+yOffset] + m.Format.CursorX
	runes := []rune(m.Data().EditTextBuffer)

	min := Max(cursor, 0)
	min = Min(min, len(m.Data().EditTextBuffer))
	first := runes[:min]
	last := runes[min:]
	f := string(first)
	l := string(last)
	m.Data().EditTextBuffer = f + newlineOrTab + l
	if len(last) == 0 { // for whatever reason, if you don't double up on newlines if appending to end, it gets removed
		m.Data().EditTextBuffer += newlineOrTab
	}
	numLines := 0
	for _, v := range m.Format.Text {
		if v != "" { // ignore padding
			numLines++
		}
	}
	if yOffset+m.Viewport.Height == numLines && newlineOrTab == "\n" {
		m.Viewport.YOffset++
	} else if newlineOrTab == "\n" {
		m.Format.CursorY++
	}

	m.Format.Text = GetFormattedTextBuffer(m)
	m.SetViewSlices()
	if newlineOrTab == "\n" {
		m.Format.CursorX = 0
	} else {
		m.Format.CursorX++
	}
}

func HandleFormatInput(m *TuiModel, str string) bool {
	switch str {
	case "tab":
		InsertCharacter(m, "\t")
		return true
	case "enter":
		InsertCharacter(m, "\n")
		return true
	case "backspace":
		cursor := m.Format.CursorX + FormatModeOffset
		input := m.Format.EditSlices[m.Format.CursorY]
		inputLen := len(*input)
		runes := []rune(*input)
		if m.Format.CursorX > 0 { // cursor in middle of line
			if cursor == inputLen && inputLen > 0 {
				*input = (*input)[0 : inputLen-1]
			} else if cursor > 0 {
				min := Max(cursor, 0)
				min = Min(min, inputLen-1)
				first := runes[:min-1]
				last := runes[min:]
				*input = string(first) + string(last)
			}

			return false
		} else if m.Format.CursorY > 0 && m.Format.CursorX == 0 { // beginning of line
			yOffset := Max(m.Viewport.YOffset, 0)
			cursor := m.Format.RunningOffsets[m.Format.CursorY+yOffset] + m.Format.CursorX
			runes := []rune(m.Data().EditTextBuffer)
			min := Max(cursor, 0)
			min = Min(min, len(m.Data().EditTextBuffer)-1)
			first := runes[:min-1]
			last := runes[min:]
			m.Data().EditTextBuffer = string(first) + string(last)
			if yOffset+m.Viewport.Height == len(m.Format.Text) && yOffset > 0 {
				m.Viewport.YOffset--
			} else {
				m.Format.CursorY--
			}
			m.Format.Text = GetFormattedTextBuffer(m)
			m.SetViewSlices()
		}

		return true
	}

	return false
}

func HandleFormatMode(m *TuiModel, str string) {
	var (
		val         string
		replacement string
	)

	inputReturn := HandleFormatInput(m, str)

	if HandleFormatMovement(m, str) {
		return
	}

	for _, v := range InputBlacklist {
		if strings.Contains(str, v) {
			return
		}
	}

	lineNumberOffset := GetOffsetForLineNumber(m.Format.CursorY)

	pString := m.Format.EditSlices[m.Format.CursorY]
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

	// if json special rules
	replacement = m.Data().EditTextBuffer
	cursor := m.Format.RunningOffsets[m.Viewport.YOffset+m.Format.CursorY]

	fIndex := Max(cursor, 0)
	lIndex := m.Viewport.YOffset + m.Format.CursorY + 1

	defer func() {
		if recover() != nil {
			println("whoopsy!") // bug happened once, debug...
		}
	}()

	first := replacement[:fIndex]
	middle := val[lineNumberOffset+1:]
	last := replacement[Min(m.Format.RunningOffsets[lIndex], len(replacement)):]

	if (first != "" || last != "") && last != "\n" {
		middle += "\n"
	}

	replacement = first + // replace the entire line the edit appears on
		middle + // insert the edit
		last // top the edit off with the rest of the string

	m.Data().EditTextBuffer = replacement
	if len(*pString) == FormatModeOffset && str != "backspace" { // insert on empty lines behaves funny
		*pString = *pString + str
	} else {
		*pString = val
	}

	m.Format.CursorX += delta

	if inputReturn {
		return
	}

	for i := m.Viewport.YOffset + m.Format.CursorY + 1; i < len(m.Format.RunningOffsets); i++ {
		m.Format.RunningOffsets[i] += delta
	}

}

// HandleEditMode implementation is kind of jank, but we can clean it up later
func HandleEditMode(m *TuiModel, str string) {
	var (
		input string
		val   string
	)
	selectedInput := &m.TextInput.Model
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

	if HandleEditMovement(m, str, val) || HandleEditInput(m, str, val) {
		return
	}

	for _, v := range InputBlacklist {
		if strings.Contains(str, v) {
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
	selectedInput.SetCursor(prePos + 1)
}
