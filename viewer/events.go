package viewer

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	getTableNamesQuery = "SELECT name FROM sqlite_master WHERE type='table'"
)

var (
	inputBlacklist = []string{
		"esc",
	}
)

// handleMouseEvents does that
func handleMouseEvents(m *TuiModel, msg *tea.MouseMsg) {
	if m.editModeEnabled {
		return
	}

	switch msg.Type {
	case tea.MouseWheelDown:
		scrollDown(m)
		break
	case tea.MouseWheelUp:
		scrollUp(m)
		break
	case tea.MouseLeft:
		selectOption(m)
		break
	default:
		if !m.renderSelection {
			m.mouseEvent = tea.MouseEvent(*msg)
		}
		break
	}
}

// handleWidowSizeEvents does that
func handleWidowSizeEvents(m *TuiModel, msg *tea.WindowSizeMsg) {
	verticalMargins := headerHeight + footerHeight

	if !m.ready {
		m.viewport = viewport.Model{
			Width:  msg.Width,
			Height: msg.Height - verticalMargins}
		m.viewport.YPosition = headerHeight
		m.viewport.HighPerformanceRendering = false // couldn't get this working
		m.ready = true
		m.tableStyle = m.GetBaseStyle()
		m.mouseEvent.Y = headerHeight
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMargins
	}
}

// handleKeyboardEvents does that
func handleKeyboardEvents(m *TuiModel, msg *tea.KeyMsg) {
	str := msg.String()
	val := m.textInput.Value() + str
	if m.editModeEnabled && val != ":q" {
		m.textInput.SetCursorMode(textinput.CursorBlink)
		for _, v := range inputBlacklist {
			if str == v {
				m.textInput.SetValue("")
				return
			}
		}

		if str == "backspace" {
			val := m.textInput.Value()
			if len(val) > 0 {
				m.textInput.SetValue(val[:len(val) - 1])
			}
		} else if str == "enter" { // writes your selection
			if len(m.actionStack) >= 10 {
				m.actionStack = m.actionStack[1:]
			}

			deepCopy := m.CopyMap()
			m.actionStack = append(m.actionStack, deepCopy)
			raw, _, _ := m.GetSelectedOption()
			*raw = m.textInput.Value()
			m.editModeEnabled = false
			m.textInput.SetValue("")
		} else {
			m.textInput.SetValue(m.textInput.Value() + msg.String())
		}

		return
	} else if m.editModeEnabled && val == ":q" { // quit mod mode
		m.editModeEnabled = false
		m.textInput.SetValue("")
		return
	} else if m.editModeEnabled && val == ":s" {
		m.Serialize()
	} else if m.editModeEnabled && val == ":!s" {
		m.SerializeOverwrite()
	}

	switch str {
	case "u": // undo
		if len(m.actionStack) > 0 { // TODO: make this a from/to swap as a function
			from := m.actionStack[len(m.actionStack) - 1]
			to := m.Table

			for k, v := range from {
				if copyValues, ok := v.(map[string][]interface{}); ok {
					columnNames := m.TableHeaders[k]
					columnValues := make(map[string][]interface{})
					// golang wizardry
					columns := make([]interface{}, len(columnNames))

					for i, _ := range columns {
						columns[i] = copyValues[columnNames[i]][0]
					}

					for i, colName := range columnNames {
						columnValues[colName] = columns[i].([]interface{})
					}

					to[k] = columnValues // data for schema, organized by column
				}
			}

			m.actionStack = m.actionStack[0:len(m.actionStack) - 1]
		}
		break
	case ":":
		raw, _, col := m.GetSelectedOption()
		if m.GetRow() >= len(col) {
			break
		}

		m.editModeEnabled = true
		m.textInput.SetValue(GetStringRepresentationOfInterface(m, *raw))
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
		break
	case "s": // manual keyboard control for row ++ (some weird behavior exists with the header height...)
		max := len(m.GetSchemaData()[m.GetHeaders()[m.GetColumn()]])

		if m.mouseEvent.Y-headerHeight < max-1 {
			m.mouseEvent.Y++
		} else {
			m.mouseEvent.Y = max
		}

		break
	case "w": // manual keyboard control for row --
		if m.mouseEvent.Y > headerHeight {
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
	case "esc": // exit full screen cell value view
		m.renderSelection = false
		m.expandColumn = -1
		m.mouseEvent.Y = m.preScrollYPosition
		m.viewport.YOffset = m.preScrollYOffset
		break
	}
}
