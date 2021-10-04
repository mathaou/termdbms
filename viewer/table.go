package viewer

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
	"time"
)

type TableAssembly func(m *TuiModel, s *string, c *chan bool)

var (
	HeaderAssembly TableAssembly
	FooterAssembly TableAssembly
	Message        string
	mid            *string
	MIP            bool
)

func init() {
	tmp := ""
	MIP = false
	mid = &tmp
	HeaderAssembly = func(m *TuiModel, s *string, done *chan bool) {
		if m.UI.ShowClipboard {
			*done <- true
			return
		}

		var (
			builder []string
		)

		style := m.GetBaseStyle()

		if !tuiutil.Ascii {
			// for column headers
			style = style.Foreground(lipgloss.Color(tuiutil.HeaderForeground())).
				BorderBackground(lipgloss.Color(tuiutil.HeaderBorderBackground())).
				Background(lipgloss.Color(tuiutil.HeaderBackground()))
		}
		headers := m.Data().TableHeadersSlice
		for i, d := range headers { // write all headers
			if m.UI.ExpandColumn != -1 && i != m.UI.ExpandColumn {
				continue
			}

			text := " " + TruncateIfApplicable(m, d)
			builder = append(builder, style.
				Render(text))
		}

		{
			// schema name
			var headerTop string

			if m.UI.EditModeEnabled || m.UI.FormatModeEnabled {
				headerTop = m.TextInput.Model.View()
				if !m.TextInput.Model.Focused() {
					headerTop = HeaderStyle.Copy().Faint(true).Render(headerTop)
				}
			} else {
				headerTop = fmt.Sprintf(" %s (%d/%d) - %d record(s) + %d column(s)",
					m.GetSchemaName(),
					m.UI.CurrentTable,
					len(m.Data().TableHeaders), // look at how headers get rendered to get accurate record number
					len(m.GetColumnData()),
					len(m.GetHeaders())) // this will need to be refactored when filters get added
				headerTop = HeaderStyle.Render(headerTop)
			}

			headerMid := lipgloss.JoinHorizontal(lipgloss.Left, builder...)
			*s = lipgloss.JoinVertical(lipgloss.Left, headerTop, headerMid)
		}

		*done <- true
	}
	FooterAssembly = func(m *TuiModel, s *string, done *chan bool) {
		if m.UI.ShowClipboard {
			*done <- true
			return
		}
		var (
			row int
			col int
		)
		if !m.UI.FormatModeEnabled { // reason we flip is because it makes more sense to store things by column for data
			row = m.GetRow() + m.Viewport.YOffset
			col = m.GetColumn() + m.Scroll.ScrollXOffset
		} else { // but for format mode thats just a regular row/col situation
			row = m.Format.CursorX
			col = m.Format.CursorY + m.Viewport.YOffset
		}
		footer := fmt.Sprintf(" %d, %d", row, col)
		undoRedoInfo := fmt.Sprintf(" undo(%d) / redo(%d) ", len(m.UndoStack), len(m.RedoStack))
		switch m.Table().Database.(type) {
		case *database.SQLite:
			break
		default:
			undoRedoInfo = ""
			break
		}

		gapSize := m.Viewport.Width - lipgloss.Width(footer) - lipgloss.Width(undoRedoInfo) - 2

		if MIP {
			MIP = false
			if !tuiutil.Ascii {
				Message = FooterStyle.Render(Message)
			}
			go func() {
				newSize := gapSize-lipgloss.Width(Message)
				if newSize < 1 {
					newSize = 1
				}
				half := strings.Repeat("-", newSize/2)
				if lipgloss.Width(Message) > gapSize {
					Message = Message[0:gapSize-3] + "..."
				}
				*mid = half + Message + half
				time.Sleep(time.Second * 5)
				Message = ""
				go Program.Send(tea.KeyMsg{})
			}()
		} else if Message == "" {
			*mid = strings.Repeat("-", gapSize)
		}
		queryResultsFlag := "├"
		if m.QueryData != nil || m.QueryResult != nil {
			queryResultsFlag = "*"
		}
		footer = FooterStyle.Render(undoRedoInfo) + queryResultsFlag + *mid + "┤" + FooterStyle.Render(footer)
		*s = footer

		*done <- true
	}
}
