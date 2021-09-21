package viewer

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"
)

const (
	getTableNamesQuery = "SELECT name FROM sqlite_master WHERE type='table'"
)

// handleMouseEvents does that
func handleMouseEvents(m *TuiModel, msg *tea.MouseMsg) {
	switch msg.Type {
	case tea.MouseWheelDown:
		if !m.UI.EditModeEnabled {
			scrollDown(m)
		}
		break
	case tea.MouseWheelUp:
		if !m.UI.EditModeEnabled {
			scrollUp(m)
		}
		break
	case tea.MouseLeft:
		if !m.UI.EditModeEnabled && !m.UI.FormatModeEnabled && m.GetRow() < len(m.GetColumnData()) {
			selectOption(m)
		}
		break
	default:
		if !m.UI.RenderSelection && !m.UI.EditModeEnabled && !m.UI.HelpDisplay && !m.UI.FormatModeEnabled {
			m.mouseEvent = tea.MouseEvent(*msg)
		}
		break
	}
}

// handleWidowSizeEvents does that
func handleWidowSizeEvents(m *TuiModel, msg *tea.WindowSizeMsg) tea.Cmd {
	verticalMargins := headerHeight + footerHeight

	if !m.Ready {
		m.viewport = viewport.Model{
			Width:  msg.Width,
			Height: msg.Height - verticalMargins}
		m.viewport.YPosition = headerHeight
		m.viewport.HighPerformanceRendering = true
		m.Ready = true
		m.mouseEvent.Y = headerHeight

		maxInputLength = m.viewport.Width
		m.textInput.Model.CharLimit = -1
		m.textInput.Model.Width = maxInputLength - lipgloss.Width(m.textInput.Model.Prompt)
		m.textInput.Model.BlinkSpeed = time.Second
		m.textInput.Model.SetCursorMode(CursorBlink)

		m.tableStyle = m.GetBaseStyle()
		m.SetViewSlices()
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMargins
	}

	if m.viewport.HighPerformanceRendering {
		return viewport.Sync(m.viewport)
	}

	return nil
}

// handleKeyboardEvents does that
func handleKeyboardEvents(m *TuiModel, msg *tea.KeyMsg) tea.Cmd {
	var (
		cmd tea.Cmd
	)

	str := msg.String()
	cw := m.CellWidth()

	if m.UI.EditModeEnabled { // handle edit mode
		handleEditMode(m, str)
		return cmd
	} else if m.UI.FormatModeEnabled {
		if str == "esc" {
			if m.textInput.Model.Focused() {
				cmd = m.formatInput.Model.Focus()
				m.textInput.Model.Blur()
			} else {
				cmd = m.textInput.Model.Focus()
				m.formatInput.Model.Blur()
			}
			return cmd
		}

		if m.textInput.Model.Focused() {
			handleEditMode(m, str)
		} else {
			handleFormatMode(m, str)
		}

		return cmd
	}

	// GLOBAL COMMANDS
	switch str {
	case "t":
		SelectedTheme = (SelectedTheme + 1) % len(ValidThemes)
		setStyles()
		break
	case "pgdown":
		for i := 0; i < m.viewport.Height; i++ {
			scrollDown(m)
		}
		break
	case "pgup":
		for i := 0; i < m.viewport.Height; i++ {
			scrollUp(m)
		}
		break
	case "r": // redo
		if len(m.RedoStack) > 0 { // do this after you get undo working, basically just the same thing reversed
			// handle undo
			deepCopy := m.CopyMap()
			// THE GLOBALIST TAKEOVER
			deepState := TableState{
				Database: &SQLite{
					FileName: m.Table.Database.GetFileName(),
					db:       nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.UndoStack = append(m.UndoStack, deepState)
			// handle redo
			from := m.RedoStack[len(m.RedoStack)-1]
			to := m.Table
			swapTableValues(m, &from, &to)
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
				Database: &SQLite{
					FileName: m.Table.Database.GetFileName(),
					db:       nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.RedoStack = append(m.RedoStack, deepState)
			// handle undo
			from := m.UndoStack[len(m.UndoStack)-1]
			to := m.Table
			swapTableValues(m, &from, &to)
			m.Table.Database.SetDatabaseReference(from.Database.GetFileName())

			m.UndoStack = m.UndoStack[0 : len(m.UndoStack)-1] // pop
		}
		break
	case ":":
		m.UI.EditModeEnabled = true
		raw, _, col := m.GetSelectedOption()
		if m.GetRow() >= len(col) {
			m.UI.EditModeEnabled = false
			break
		}

		str := GetStringRepresentationOfInterface(*raw)
		// so if the selected text is wider than viewport width or if it has newlines do format mode
		if lipgloss.Width(str+m.textInput.Model.Prompt) > m.viewport.Width || strings.Count(str, "\n") > 0 { // enter format view
			prepareFormatMode(m)
			cmd = m.formatInput.Model.Focus()
			m.preScrollYOffset = m.viewport.YOffset
			m.preScrollYPosition = m.mouseEvent.Y
			if conv, err := formatJson(str); err == nil { // if json prettify
				m.selectionText = conv
			} else {
				m.selectionText = str
			}
			m.formatInput.Original = raw
			m.Format.Text = getFormattedTextBuffer(m)
			m.SetViewSlices()
			m.formatInput.Model.setCursor(0)
		} else { // otherwise, edit normally up top
			m.textInput.Model.SetValue(str)
			m.formatInput.Model.focus = false
			m.textInput.Model.focus = true
		}
		break
	case "p":
		if m.UI.RenderSelection {
			WriteTextFile(m, m.selectionText)
		}
		break
	case "c":
		toggleColumn(m)
		break
	case "b":
		m.UI.BorderToggle = !m.UI.BorderToggle
		break
	case "up", "k": // toggle next schema + 1
		if m.TableSelection == len(m.TableIndexMap) {
			m.TableSelection = 1
		} else {
			m.TableSelection++
		}

		// fix spacing and whatnot
		m.tableStyle = m.tableStyle.Width(cw)
		m.mouseEvent.Y = headerHeight
		m.mouseEvent.X = 0
		m.viewport.YOffset = 0
		m.scrollXOffset = 0
		break
	case "down", "j": // toggle previous schema - 1
		if m.TableSelection == 1 {
			m.TableSelection = len(m.TableIndexMap)
		} else {
			m.TableSelection--
		}

		// fix spacing and whatnot
		m.tableStyle = m.tableStyle.Width(cw)
		m.mouseEvent.Y = headerHeight
		m.mouseEvent.X = 0
		m.viewport.YOffset = 0
		m.scrollXOffset = 0
		break
	case "right", "l":
		headers := m.GetHeaders()
		headersLen := len(headers)
		if headersLen > maxHeaders && m.scrollXOffset <= headersLen-maxHeaders {
			m.scrollXOffset++
		}
		break
	case "left", "h":
		if m.scrollXOffset > 0 {
			m.scrollXOffset--
		}
		break
	case "s": // manual keyboard control for row ++
		max := len(m.GetSchemaData()[m.GetHeaders()[m.GetColumn()]])

		if m.mouseEvent.Y-headerHeight+m.viewport.YOffset < max-1 {
			m.mouseEvent.Y++
			if m.mouseEvent.Y > m.viewport.Height+headerHeight-1 {
				scrollDown(m)
				m.mouseEvent.Y = m.viewport.Height + headerHeight - 1
			}
		}

		break
	case "w": // manual keyboard control for row --
		pre := m.mouseEvent.Y
		if m.viewport.YOffset > 0 && m.mouseEvent.Y == headerHeight {
			scrollUp(m)
			m.mouseEvent.Y = pre
		} else if m.mouseEvent.Y > headerHeight {
			m.mouseEvent.Y--
		}
		break
	case "d": // manual keyboard control for column ++
		col := m.GetColumn()
		cols := len(m.TableHeadersSlice) - 1
		if (m.mouseEvent.X-m.viewport.Width) <= cw && m.GetColumn() < cols { // within tolerances
			m.mouseEvent.X += cw
		} else if col == cols {
			go Program.Send(tea.KeyMsg{
				Type: tea.KeyRight,
				Alt:  false,
			})
		}
		break
	case "a": // manual keyboard control for column --
		if m.mouseEvent.X-cw >= 0 {
			m.mouseEvent.X -= cw
		} else if m.GetColumn() == 0 {
			go Program.Send(tea.KeyMsg{
				Type: tea.KeyLeft,
				Alt:  false,
			})
		}
		break
	case "enter": // manual trigger for select highlighted cell
		if !m.UI.EditModeEnabled {
			selectOption(m)
		}
		break
	case "m": // scroll up manually
		scrollUp(m)
		break
	case "n": // scroll down manually
		scrollDown(m)
		break
	case "esc": // exit full screen cell value view, also enabled edit mode
		if !m.UI.RenderSelection && !m.UI.HelpDisplay {
			m.UI.EditModeEnabled = true
			break
		}
		m.UI.RenderSelection = false
		m.UI.HelpDisplay = false
		m.selectionText = ""
		cmd = m.textInput.Model.Focus()
		m.textInput.Model.SetValue("")
		m.expandColumn = -1
		m.mouseEvent.Y = m.preScrollYPosition
		m.viewport.YOffset = m.preScrollYOffset
		break
	}

	return cmd
}
