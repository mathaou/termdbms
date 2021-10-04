package viewer

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
)

type Command func(m *TuiModel) tea.Cmd

var (
	GlobalCommands = make(map[string]Command)
)

func init() {
	// GLOBAL COMMANDS
	GlobalCommands["t"] = func(m *TuiModel) tea.Cmd {
		tuiutil.SelectedTheme = (tuiutil.SelectedTheme + 1) % len(tuiutil.ValidThemes)
		SetStyles()
		themeName := tuiutil.ValidThemes[tuiutil.SelectedTheme]
		m.WriteMessage(fmt.Sprintf("Changed themes to %s", themeName))
		return nil
	}
	GlobalCommands["pgdown"] = func(m *TuiModel) tea.Cmd {
		for i := 0; i < m.Viewport.Height; i++ {
			ScrollDown(m)
		}

		return nil
	}
	GlobalCommands["pgup"] = func(m *TuiModel) tea.Cmd {
		for i := 0; i < m.Viewport.Height; i++ {
			ScrollUp(m)
		}

		return nil
	}
	GlobalCommands["r"] = func(m *TuiModel) tea.Cmd {
		if len(m.RedoStack) > 0 && m.QueryResult == nil && m.QueryData == nil { // do this after you get undo working, basically just the same thing reversed
			// handle undo
			deepCopy := m.CopyMap()
			// THE GLOBALIST TAKEOVER
			deepState := TableState{
				Database: &database.SQLite{
					FileName: m.Table().Database.GetFileName(),
					Database: nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.UndoStack = append(m.UndoStack, deepState)
			// handle redo
			from := m.RedoStack[len(m.RedoStack)-1]
			to := m.Table()
			m.SwapTableValues(&from, to)
			m.Table().Database.CloseDatabaseReference()
			m.Table().Database.SetDatabaseReference(from.Database.GetFileName())

			m.RedoStack = m.RedoStack[0 : len(m.RedoStack)-1] // pop
		}

		return nil
	}
	GlobalCommands["u"] = func(m *TuiModel) tea.Cmd {
		if len(m.UndoStack) > 0 && m.QueryResult == nil && m.QueryData == nil {
			// handle redo
			deepCopy := m.CopyMap()
			t := m.Table()
			// THE GLOBALIST TAKEOVER
			deepState := TableState{
				Database: &database.SQLite{
					FileName: t.Database.GetFileName(),
					Database: nil,
				}, // placeholder for now while testing database copy
				Data: deepCopy,
			}
			m.RedoStack = append(m.RedoStack, deepState)
			// handle undo
			from := m.UndoStack[len(m.UndoStack)-1]
			to := t
			m.SwapTableValues(&from, to)
			t.Database.CloseDatabaseReference()
			t.Database.SetDatabaseReference(from.Database.GetFileName())

			m.UndoStack = m.UndoStack[0 : len(m.UndoStack)-1] // pop
		}

		return nil
	}
	GlobalCommands[":"] = func(m *TuiModel) tea.Cmd {
		var (
			cmd tea.Cmd
		)
		if m.QueryData != nil || m.QueryResult != nil { // editing not allowed in query view mode
			return nil
		}
		m.UI.EditModeEnabled = true
		raw, _, _ := m.GetSelectedOption()
		if raw == nil {
			m.UI.EditModeEnabled = false
			return nil
		}

		str := GetStringRepresentationOfInterface(*raw)
		// so if the selected text is wider than Viewport width or if it has newlines do format mode
		if lipgloss.Width(str+m.TextInput.Model.Prompt) > m.Viewport.Width ||
			strings.Count(str, "\n") > 0 { // enter format view
			PrepareFormatMode(m)
			cmd = m.FormatInput.Model.FocusCommand()       // get focus
			m.Scroll.PreScrollYOffset = m.Viewport.YOffset // store scrolling so state can be restored on exit
			m.Scroll.PreScrollYPosition = m.MouseData.Y
			d := m.Data()
			if conv, err := FormatJson(str); err == nil { // if json prettify
				d.EditTextBuffer = conv
			} else {
				d.EditTextBuffer = str
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

		return cmd
	}
	GlobalCommands["p"] = func(m *TuiModel) tea.Cmd {
		if m.UI.RenderSelection {
			fn, _ := WriteTextFile(m, m.Data().EditTextBuffer)
			m.WriteMessage(fmt.Sprintf("Wrote selection to %s", fn))
		} else if m.QueryData != nil || m.QueryResult != nil {
			WriteCSV(m)
		}
		go Program.Send(tea.KeyMsg{})
		return nil
	}
	GlobalCommands["c"] = func(m *TuiModel) tea.Cmd {
		ToggleColumn(m)

		return nil
	}
	GlobalCommands["b"] = func(m *TuiModel) tea.Cmd {
		m.UI.BorderToggle = !m.UI.BorderToggle

		return nil
	}
	GlobalCommands["up"] = func(m *TuiModel) tea.Cmd {
		if m.UI.CurrentTable == len(m.Data().TableIndexMap) {
			m.UI.CurrentTable = 1
		} else {
			m.UI.CurrentTable++
		}

		// fix spacing and whatnot
		m.TableStyle = m.TableStyle.Width(m.CellWidth())
		m.MouseData.Y = HeaderHeight
		m.MouseData.X = 0
		m.Viewport.YOffset = 0
		m.Scroll.ScrollXOffset = 0

		return nil
	}
	GlobalCommands["down"] = func(m *TuiModel) tea.Cmd {
		if m.UI.CurrentTable == 1 {
			m.UI.CurrentTable = len(m.Data().TableIndexMap)
		} else {
			m.UI.CurrentTable--
		}

		// fix spacing and whatnot
		m.TableStyle = m.TableStyle.Width(m.CellWidth())
		m.MouseData.Y = HeaderHeight
		m.MouseData.X = 0
		m.Viewport.YOffset = 0
		m.Scroll.ScrollXOffset = 0

		return nil
	}
	GlobalCommands["right"] = func(m *TuiModel) tea.Cmd {
		headers := m.GetHeaders()
		headersLen := len(headers)
		if headersLen > maxHeaders && m.Scroll.ScrollXOffset <= headersLen-maxHeaders {
			m.Scroll.ScrollXOffset++
		}

		return nil
	}
	GlobalCommands["left"] = func(m *TuiModel) tea.Cmd {
		if m.Scroll.ScrollXOffset > 0 {
			m.Scroll.ScrollXOffset--
		}

		return nil
	}
	GlobalCommands["s"] = func(m *TuiModel) tea.Cmd {
		max := len(m.GetSchemaData()[m.GetHeaders()[m.GetColumn()]])

		if m.MouseData.Y-HeaderHeight+m.Viewport.YOffset < max-1 {
			m.MouseData.Y++
			ceiling := m.Viewport.Height + HeaderHeight - 1
			tuiutil.Clamp(m.MouseData.Y, m.MouseData.Y+1, ceiling)
			if m.MouseData.Y > ceiling {
				ScrollDown(m)
				m.MouseData.Y = ceiling
			}
		}

		return nil
	}
	GlobalCommands["w"] = func(m *TuiModel) tea.Cmd {
		pre := m.MouseData.Y
		if m.Viewport.YOffset > 0 && m.MouseData.Y == HeaderHeight {
			ScrollUp(m)
			m.MouseData.Y = pre
		} else if m.MouseData.Y > HeaderHeight {
			m.MouseData.Y--
		}

		return nil
	}
	GlobalCommands["d"] = func(m *TuiModel) tea.Cmd {
		cw := m.CellWidth()
		col := m.GetColumn()
		cols := len(m.Data().TableHeadersSlice) - 1
		if (m.MouseData.X-m.Viewport.Width) <= cw && m.GetColumn() < cols { // within tolerances
			m.MouseData.X += cw
		} else if col == cols {
			return func() tea.Msg {
				return tea.KeyMsg{
					Type: tea.KeyRight,
					Alt:  false,
				}
			}
		}

		return nil
	}
	GlobalCommands["a"] = func(m *TuiModel) tea.Cmd {
		cw := m.CellWidth()
		if m.MouseData.X-cw >= 0 {
			m.MouseData.X -= cw
		} else if m.GetColumn() == 0 {
			return func() tea.Msg {
				return tea.KeyMsg{
					Type: tea.KeyLeft,
					Alt:  false,
				}
			}
		}
		return nil
	}
	GlobalCommands["enter"] = func(m *TuiModel) tea.Cmd {
		if !m.UI.EditModeEnabled {
			SelectOption(m)
		}

		return nil
	}
	GlobalCommands["esc"] = func(m *TuiModel) tea.Cmd {
		m.TextInput.Model.SetValue("")
		if !m.UI.RenderSelection &&
			!m.UI.HelpDisplay {
			m.UI.EditModeEnabled = true
			return nil
		}

		m.UI.RenderSelection = false
		m.UI.HelpDisplay = false
		m.Data().EditTextBuffer = ""
		cmd := m.TextInput.Model.FocusCommand()
		m.UI.ExpandColumn = -1
		m.MouseData.Y = m.Scroll.PreScrollYPosition
		m.Viewport.YOffset = m.Scroll.PreScrollYOffset

		return cmd
	}

	GlobalCommands["k"] = GlobalCommands["up"]    // dual bind of up/k
	GlobalCommands["j"] = GlobalCommands["down"]  // dual bind of down/j
	GlobalCommands["l"] = GlobalCommands["right"] // dual bind of right/l
	GlobalCommands["h"] = GlobalCommands["left"]  // dual bind of left/h
	GlobalCommands["m"] = func(m *TuiModel) tea.Cmd {
		ScrollUp(m)
		return nil
	}
	GlobalCommands["n"] = func(m *TuiModel) tea.Cmd {
		ScrollDown(m)
		return nil
	}
	GlobalCommands["?"] = func(m *TuiModel) tea.Cmd {
		m.UI.HelpDisplay = true
		help := GetHelpText()
		m.DisplayMessage(help)
		return nil
	}
}
