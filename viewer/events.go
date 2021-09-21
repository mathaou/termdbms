package viewer

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
	"time"
)

// HandleMouseEvents does that
func HandleMouseEvents(m *TuiModel, msg *tea.MouseMsg) {
	switch msg.Type {
	case tea.MouseWheelDown:
		if !m.UI.EditModeEnabled {
			ScrollDown(m)
		}
		break
	case tea.MouseWheelUp:
		if !m.UI.EditModeEnabled {
			ScrollUp(m)
		}
		break
	case tea.MouseLeft:
		if !m.UI.EditModeEnabled && !m.UI.FormatModeEnabled && m.GetRow() < len(m.GetColumnData()) {
			SelectOption(m)
		}
		break
	default:
		if !m.UI.RenderSelection && !m.UI.EditModeEnabled && !m.UI.HelpDisplay && !m.UI.FormatModeEnabled {
			m.MouseData = tea.MouseEvent(*msg)
		}
		break
	}
}

// HandleWindowSizeEvents does that
func HandleWindowSizeEvents(m *TuiModel, msg *tea.WindowSizeMsg) tea.Cmd {
	verticalMargins := HeaderHeight + FooterHeight

	if !m.Ready {
		m.Viewport = viewport.Model{
			Width:  msg.Width,
			Height: msg.Height - verticalMargins}
		m.Viewport.YPosition = HeaderHeight
		m.Viewport.HighPerformanceRendering = true
		m.Ready = true
		m.MouseData.Y = HeaderHeight

		MaxInputLength = m.Viewport.Width
		m.TextInput.Model.CharLimit = -1
		m.TextInput.Model.Width = MaxInputLength - lipgloss.Width(m.TextInput.Model.Prompt)
		m.TextInput.Model.BlinkSpeed = time.Second
		m.TextInput.Model.SetCursorMode(tuiutil.CursorBlink)

		m.TableStyle = m.GetBaseStyle()
		m.SetViewSlices()
	} else {
		m.Viewport.Width = msg.Width
		m.Viewport.Height = msg.Height - verticalMargins
	}

	if m.Viewport.HighPerformanceRendering {
		return viewport.Sync(m.Viewport)
	}

	return nil
}

// HandleKeyboardEvents does that
func HandleKeyboardEvents(m *TuiModel, msg *tea.KeyMsg) tea.Cmd {
	var (
		cmd tea.Cmd
	)

	str := msg.String()
	cw := m.CellWidth()

	if m.UI.EditModeEnabled { // handle edit mode
		HandleEditMode(m, str)
		return cmd
	} else if m.UI.FormatModeEnabled {
		if str == "esc" { // cycle focus
			if m.TextInput.Model.Focused() {
				cmd = m.FormatInput.Model.FocusCommand()
				m.TextInput.Model.Blur()
			} else {
				cmd = m.TextInput.Model.FocusCommand()
				m.FormatInput.Model.Blur()
			}
			return cmd
		}

		if m.TextInput.Model.Focused() {
			HandleEditMode(m, str)
		} else {
			HandleFormatMode(m, str)
		}

		return cmd
	}

	// GLOBAL COMMANDS
	switch str {
	case "t":
		tuiutil.SelectedTheme = (tuiutil.SelectedTheme + 1) % len(tuiutil.ValidThemes)
		SetStyles()
		break
	case "pgdown":
		for i := 0; i < m.Viewport.Height; i++ {
			ScrollDown(m)
		}
		break
	case "pgup":
		for i := 0; i < m.Viewport.Height; i++ {
			ScrollUp(m)
		}
		break
	case "r": // redo
		if len(m.RedoStack) > 0 { // do this after you get undo working, basically just the same thing reversed
			// handle undo
			deepCopy := m.CopyMap()
			// THE GLOBALIST TAKEOVER
			deepState := TableState{
				Database: &database.SQLite{
					FileName: m.Table.Database.GetFileName(),
					Database:       nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.UndoStack = append(m.UndoStack, deepState)
			// handle redo
			from := m.RedoStack[len(m.RedoStack)-1]
			to := m.Table
			SwapTableValues(m, &from, &to)
			m.Table.Database.SetDatabaseReference(from.Database.GetFileName())

			m.RedoStack = m.RedoStack[0 : len(m.RedoStack)-1] // pop
		}
		break
	case "u": // undo
		if len(m.UndoStack) > 0 {
			// handle redo
			deepCopy := m.CopyMap()
			// THE GLOBALIST TAKEOVER
			deepState := TableState{
				Database: &database.SQLite{
					FileName: m.Table.Database.GetFileName(),
					Database:       nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.RedoStack = append(m.RedoStack, deepState)
			// handle undo
			from := m.UndoStack[len(m.UndoStack)-1]
			to := m.Table
			SwapTableValues(m, &from, &to)
			m.Table.Database.SetDatabaseReference(from.Database.GetFileName())

			m.UndoStack = m.UndoStack[0 : len(m.UndoStack)-1] // pop
		}
		break
	case ":": // edit mode or format mode depending on string length
		m.UI.EditModeEnabled = true
		raw, _, _ := m.GetSelectedOption()
		if raw == nil {
			m.UI.EditModeEnabled = false
			break
		}

		str := GetStringRepresentationOfInterface(*raw)
		// so if the selected text is wider than Viewport width or if it has newlines do format mode
		if lipgloss.Width(str+m.TextInput.Model.Prompt) > m.Viewport.Width ||
			strings.Count(str, "\n") > 0 { // enter format view
			PrepareFormatMode(m)
			cmd = m.FormatInput.Model.FocusCommand()       // get focus
			m.Scroll.PreScrollYOffset = m.Viewport.YOffset // store scrolling so state can be restored on exit
			m.Scroll.PreScrollYPosition = m.MouseData.Y
			if conv, err := FormatJson(str); err == nil { // if json prettify
				m.Data.EditTextBuffer = conv
			} else {
				m.Data.EditTextBuffer = str
			}
			m.FormatInput.Original = raw // pointer to original data
			m.Format.Text = GetFormattedTextBuffer(m)
			m.SetViewSlices()
			m.FormatInput.Model.SetCursor(0)
		} else { // otherwise, edit normally up top
			m.TextInput.Model.SetValue(str)
			m.FormatInput.Model.Focus = false
			m.TextInput.Model.Focus = true
		}
		break
	case "p":
		if m.UI.RenderSelection {
			WriteTextFile(m, m.Data.EditTextBuffer)
		}
		break
	case "c":
		ToggleColumn(m)
		break
	case "b":
		m.UI.BorderToggle = !m.UI.BorderToggle
		break
	case "up", "k": // toggle next schema + 1
		if m.UI.CurrentTable == len(m.Data.TableIndexMap) {
			m.UI.CurrentTable = 1
		} else {
			m.UI.CurrentTable++
		}

		// fix spacing and whatnot
		m.TableStyle = m.TableStyle.Width(cw)
		m.MouseData.Y = HeaderHeight
		m.MouseData.X = 0
		m.Viewport.YOffset = 0
		m.Scroll.ScrollXOffset = 0
		break
	case "down", "j": // toggle previous schema - 1
		if m.UI.CurrentTable == 1 {
			m.UI.CurrentTable = len(m.Data.TableIndexMap)
		} else {
			m.UI.CurrentTable--
		}

		// fix spacing and whatnot
		m.TableStyle = m.TableStyle.Width(cw)
		m.MouseData.Y = HeaderHeight
		m.MouseData.X = 0
		m.Viewport.YOffset = 0
		m.Scroll.ScrollXOffset = 0
		break
	case "right", "l":
		headers := m.GetHeaders()
		headersLen := len(headers)
		if headersLen > maxHeaders && m.Scroll.ScrollXOffset <= headersLen-maxHeaders {
			m.Scroll.ScrollXOffset++
		}
		break
	case "left", "h":
		if m.Scroll.ScrollXOffset > 0 {
			m.Scroll.ScrollXOffset--
		}
		break
	case "s": // manual keyboard control for row ++
		max := len(m.GetSchemaData()[m.GetHeaders()[m.GetColumn()]])

		if m.MouseData.Y-HeaderHeight+m.Viewport.YOffset < max-1 {
			m.MouseData.Y++
			ceiling := m.Viewport.Height+HeaderHeight-1
			tuiutil.Clamp(m.MouseData.Y, m.MouseData.Y + 1, ceiling)
			if m.MouseData.Y > ceiling {
				ScrollDown(m)
			}
		}

		break
	case "w": // manual keyboard control for row --
		pre := m.MouseData.Y
		if m.Viewport.YOffset > 0 && m.MouseData.Y == HeaderHeight {
			ScrollUp(m)
			m.MouseData.Y = pre
		} else if m.MouseData.Y > HeaderHeight {
			m.MouseData.Y--
		}
		break
	case "d": // manual keyboard control for column ++
		col := m.GetColumn()
		cols := len(m.Data.TableHeadersSlice) - 1
		if (m.MouseData.X-m.Viewport.Width) <= cw && m.GetColumn() < cols { // within tolerances
			m.MouseData.X += cw
		} else if col == cols {
			go Program.Send(tea.KeyMsg{
				Type: tea.KeyRight,
				Alt:  false,
			})
		}
		break
	case "a": // manual keyboard control for column --
		if m.MouseData.X-cw >= 0 {
			m.MouseData.X -= cw
		} else if m.GetColumn() == 0 {
			go Program.Send(tea.KeyMsg{
				Type: tea.KeyLeft,
				Alt:  false,
			})
		}
		break
	case "enter": // manual trigger for select highlighted cell
		if !m.UI.EditModeEnabled {
			SelectOption(m)
		}
		break
	case "m": // scroll up manually
		ScrollUp(m)
		break
	case "n": // scroll down manually
		ScrollDown(m)
		break
	case "esc": // exit full screen cell value view, also enabled edit mode
		if !m.UI.RenderSelection && !m.UI.HelpDisplay {
			m.UI.EditModeEnabled = true
			break
		}
		m.UI.RenderSelection = false
		m.UI.HelpDisplay = false
		m.Data.EditTextBuffer = ""
		cmd = m.TextInput.Model.FocusCommand()
		m.TextInput.Model.SetValue("")
		m.UI.ExpandColumn = -1
		m.MouseData.Y = m.Scroll.PreScrollYPosition
		m.Viewport.YOffset = m.Scroll.PreScrollYOffset
		break
	}

	return cmd
}
