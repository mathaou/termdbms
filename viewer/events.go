package viewer

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	getTableNamesQuery = "SELECT name FROM sqlite_master WHERE type='table'"
)

var (
	inputBlacklist = []string{
		"alt+[",
		"up",
		"down",
		"tab",
		"end",
		"home",
		"pgdown",
		"pgup",
	}
)

// handleMouseEvents does that
func handleMouseEvents(m *TuiModel, msg *tea.MouseMsg) {
	switch msg.Type {
	case tea.MouseWheelDown:
		if !m.editModeEnabled {
			scrollDown(m)
		}
		break
	case tea.MouseWheelUp:
		if !m.editModeEnabled {
			scrollUp(m)
		}
		break
	case tea.MouseLeft:
		if !m.editModeEnabled {
			selectOption(m)
		}
		break
	default:
		if !m.renderSelection && !m.editModeEnabled && !m.helpDisplay {
			m.mouseEvent = tea.MouseEvent(*msg)
		}
		break
	}
}

// handleWidowSizeEvents does that
func handleWidowSizeEvents(m *TuiModel, msg *tea.WindowSizeMsg) tea.Cmd {
	verticalMargins := headerHeight + footerHeight

	if !m.ready {
		m.viewport = viewport.Model{
			Width:  msg.Width,
			Height: msg.Height - verticalMargins}
		m.viewport.YPosition = headerHeight
		m.viewport.HighPerformanceRendering = true
		m.ready = true
		m.mouseEvent.Y = headerHeight

		{ // race condition here on debug mode TODO
			m.tableStyle = m.GetBaseStyle()
			m.SetViewSlices()
		}
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
func handleKeyboardEvents(m *TuiModel, msg *tea.KeyMsg) {
	var (
		str string
		input string
		min int
		first string
		last string
		val string
	)
	str = msg.String()
	input = m.textInput.Value()
	if input != "" && m.textInput.Cursor() < len(input) - 1 {
		min = Max(m.textInput.Cursor(), 0)
		min = Min(min, len(input) - 1)
		first = input[:min]
		last = input[min:]
		val = first + str + last
	} else {
		val = input + str
	}

	if m.editModeEnabled { // handle edit mode
		handleEditMode(m, str, first, last, input, val)
		return
	}

	switch str {
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
		m.editModeEnabled = true
		raw, _, col := m.GetSelectedOption()
		if m.GetRow() >= len(col) {
			m.editModeEnabled = false
			break
		}

		str := GetStringRepresentationOfInterface(*raw)
		if lipgloss.Width(str + m.textInput.Prompt) > m.viewport.Width {
			m.formatModeEnabled = true
		}
		m.textInput.SetValue(str)
		break
	case "p":
		if m.renderSelection {
			WriteTextFile(m, m.selectionText)
		}
		break
	case "c":
		toggleColumn(m)
		break
	case "b":
		m.borderToggle = !m.borderToggle
		break
	case "up", "k": // toggle next schema + 1
		if m.TableSelection == len(m.TableIndexMap) {
			m.TableSelection = 1
		} else {
			m.TableSelection++
		}

		// fix spacing and whatnot
		m.tableStyle = m.tableStyle.Width(m.CellWidth())
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
		m.tableStyle = m.tableStyle.Width(m.CellWidth())
		m.viewport.YOffset = 0
		m.scrollXOffset = 0
		break
	case "right", "l":
		if len(m.GetHeaders()) > maxHeaders && m.scrollXOffset < len(m.GetHeaders())-1-maxHeaders {
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
		if m.mouseEvent.X+m.CellWidth() <= m.viewport.Width {
			m.mouseEvent.X += m.CellWidth()
		}
		break
	case "a": // manual keyboard control for column --
		if m.mouseEvent.X-m.CellWidth() >= 0 {
			m.mouseEvent.X -= m.CellWidth()
		}
		break
	case "enter": // manual trigger for select highlighted cell
		if !m.editModeEnabled {
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
		if !m.renderSelection && !m.helpDisplay {
			m.editModeEnabled = true
			break
		}
		m.renderSelection = false
		m.helpDisplay = false
		m.selectionText = ""
		m.expandColumn = -1
		m.mouseEvent.Y = m.preScrollYPosition
		m.viewport.YOffset = m.preScrollYOffset
		break
	}
}
