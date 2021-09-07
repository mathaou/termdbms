package viewer

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

const (
	getTableNamesQuery = "SELECT name FROM sqlite_master WHERE type='table'"
)

var (
	inputBlacklist = []string{
		"alt+[",
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
func handleWidowSizeEvents(m *TuiModel, msg *tea.WindowSizeMsg) tea.Cmd {
	verticalMargins := headerHeight + footerHeight

	if !m.ready {
		m.viewport = ViewportModel{
			Width:  msg.Width,
			Height: msg.Height - verticalMargins}
		m.viewport.YPosition = headerHeight
		m.viewport.HighPerformanceRendering = true
		m.ready = true
		m.tableStyle = m.GetBaseStyle()
		m.mouseEvent.Y = headerHeight

		m.SetViewSlices()
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMargins
	}

	if m.viewport.HighPerformanceRendering {
		return Sync(m.viewport)
	}

	return nil
}

// handleKeyboardEvents does that
func handleKeyboardEvents(m *TuiModel, msg *tea.KeyMsg) {
	str := msg.String()
	input := m.textInput.Value()
	val := input + str

	if m.editModeEnabled { // handle edit mode
		if str == "esc" {
			m.textInput.SetValue("")
			return
		}

		m.textInput.SetCursorMode(textinput.CursorBlink)
		for _, v := range inputBlacklist {
			if str == v {
				return
			}
		}

		if str == "backspace" {
			val := m.textInput.Value()
			// TODO: lipgloss.Width couldn't be used here because the width was sometimes 2, when I just need to go back one
			if len(val) > 0 {
				m.textInput.SetValue(val[:len(val)-1])
			}
		} else if str == "enter" { // writes your selection
			if m.editModeEnabled && input == ":q" { // quit mod mode
				m.editModeEnabled = false
				m.textInput.SetValue("")
				return
			} else if m.editModeEnabled && input == ":s" { // saves copy, default filename + :s _____ will save with that filename in cwd
				m.editModeEnabled = false
				m.textInput.SetValue("")
				m.Serialize()
				return
			} else if m.editModeEnabled && input == ":!s" { // overwrites original
				m.editModeEnabled = false
				m.textInput.SetValue("")
				m.SerializeOverwrite()
				return
			} else if m.editModeEnabled && input == ":h" {
				m.selectionText = GetHelpText()
				m.editModeEnabled = false
				m.renderSelection = true
				m.helpDisplay = true
				return
			}

			raw, _, _ := m.GetSelectedOption()
			if *raw == input {
				// no update
				return
			}

			// plain jane cell update
			if len(m.UndoStack) >= 10 {
				ref := m.UndoStack[len(m.UndoStack)-1]
				err := os.Remove(ref.Database.GetFileName())
				if err != nil {
					fmt.Printf("%v", err)
					os.Exit(1)
				}
				m.UndoStack = m.UndoStack[1:] // need some more complicated logic to handle dereferencing
			}

			deepCopy := m.CopyMap()
			// THE GLOBALIST TAKEOVER
			deepState := TableState{
				Database: &SQLite{
					FileName:          m.Table.Database.GetFileName(),
					DatabaseReference: nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.UndoStack = append(m.UndoStack, deepState)
			dst, _, _ := CopyFile(m.Table.Database.GetFileName())
			m.Table.Database.CloseDatabaseReference()
			m.Table.Database.SetDatabaseReference(GetDatabaseForFile(dst))
			m.ProcessSqlQueryForDatabaseType(&Update{
				Update: *raw,
			})

			m.SerializeOverwrite() // for testing
			m.editModeEnabled = false
			m.textInput.SetValue("")
			*raw = input
		} else {
			m.textInput.SetValue(val)
		}

		return
	}

	switch str {
	case "r": // redo
		if len(m.RedoStack) > 0 { // do this after you get undo working, basically just the same thing reversed
			// handle undo
			//deepCopy := m.CopyMap()
			//m.UndoStack = append(m.UndoStack, TableState{
			//	Filename: "test",
			//	Data:     deepCopy,
			//})
			// handle redo
			from := m.RedoStack[len(m.RedoStack)-1]
			to := m.Table
			swapTableValues(m, &from, &to)

			m.RedoStack = m.RedoStack[0 : len(m.RedoStack)-1] // pop
		}
		break
	case "u": // undo
		if len(m.UndoStack) > 0 {
			// handle redo
			//deepCopy := m.CopyMap()
			//m.RedoStack = append(m.RedoStack, TableState{
			//	Filename: "test",
			//	Data:     deepCopy,
			//})
			// handle undo
			from := m.UndoStack[len(m.UndoStack)-1]
			to := m.Table
			swapTableValues(m, &from, &to)

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
	case "right":
		if len(m.GetHeaders()) >= 12 {
			m.scrollXOffset++
		}
		break
	case "left":
		if m.scrollXOffset > 0 {
			m.scrollXOffset--
		}
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
	case "esc": // exit full screen cell value view, also brings back to top
		m.renderSelection = false
		m.helpDisplay = false
		m.selectionText = ""
		m.expandColumn = -1
		m.mouseEvent.Y = m.preScrollYPosition
		m.viewport.YOffset = m.preScrollYOffset
		break
	}
}
