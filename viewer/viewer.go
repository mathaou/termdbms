package viewer

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"strings"
)

var (
	width        int
	height       int
	headerHeight = 3
	footerHeight = 3
)

const (
	highlight = "#0168B3"
	maximumRendererCharacters = 1024
	useHighPerformanceRenderer = false
)

type TuiModel struct {
	Table          map[string]interface{}
	TableHeaders   map[string][]string
	TableIndexMap  map[int]string
	TableSelection int
	ready          bool
	renderSelection bool
	viewport       viewport.Model
	tableStyle     lipgloss.Style
	mouseEvent     tea.MouseEvent
}

// INIT UPDATE AND RENDER

func (m TuiModel) Init() tea.Cmd {
	return nil
}

func (m TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if !m.renderSelection {
			m.mouseEvent = tea.MouseEvent(msg)
		}
		tbl := m.GetTable()
		if msg.Type == tea.MouseWheelDown {
			scrollDown(&m, tbl)
		} else if msg.Type == tea.MouseWheelUp {
			scrollUp(&m)
		} else if msg.Type == tea.MouseLeft {
			selectOption(&m, tbl)
		}
		break
	case tea.WindowSizeMsg:
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.Model{
				Width: msg.Width,
				Height: msg.Height - verticalMargins}
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer // find some way to fix this
			m.ready = true
			m.tableStyle = m.GetBaseStyle()
			m.mouseEvent.Y = headerHeight
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}


		if useHighPerformanceRenderer {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

		break
	case tea.KeyMsg:
		if m.renderSelection &&
			msg.String() != "esc" &&
			msg.String() != "ctrl+c" &&
			msg.String() != "q" {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.TableSelection == len(m.TableIndexMap) {
				m.TableSelection = 1
			} else {
				m.TableSelection++
			}

			m.tableStyle = m.tableStyle.Width(m.CellWidth())
			break
		case "down", "j":
			if m.TableSelection == 1 {
				m.TableSelection = len(m.TableIndexMap)
			} else {
				m.TableSelection--
			}

			m.tableStyle = m.tableStyle.Width(m.CellWidth())
			break
		case "s":
			max := len(m.GetTable()[m.GetHeaders()[m.GetColumn()]])

			if m.mouseEvent.Y - headerHeight < max-1 {
				m.mouseEvent.Y++
			}

			break
		case "w":
			if m.mouseEvent.Y > headerHeight && m.viewport.YOffset < m.mouseEvent.Y {
				m.mouseEvent.Y--
			}
			break
		case "d":
			if m.mouseEvent.X + m.CellWidth() <= m.viewport.Width {
				m.mouseEvent.X += m.CellWidth()
			}
			break
		case "a":
			if m.mouseEvent.X - m.CellWidth() >= 0 {
				m.mouseEvent.X -= m.CellWidth()
			}
			break
		case "enter":
			selectOption(&m, m.GetTable())
			break
		case "m":
			scrollUp(&m)
			break
		case "n":
			scrollDown(&m, m.GetTable())
			break
		case "esc":
			m.renderSelection = false
			break
		}
	}
	m.viewport, cmd = m.viewport.Update(msg)
	if useHighPerformanceRenderer {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TuiModel) View() string {
	if !m.ready || m.viewport.Width == 0 {
		return "\n  Initializing..."
	}

	done := make(chan bool, 3)

	var footer, header, content string

	// body
	go func(c *string) {
		*c = assembleTable(&m)
		done <- true
	}(&content)

	// header
	go func(h *string) {
		var builder strings.Builder
		style := m.tableStyle.
			Width(m.CellWidth()).
			Foreground(lipgloss.Color("#FFFFFF")).
			BorderBackground(lipgloss.Color("#231F20"))
		for _, d := range m.GetHeaders() {
			builder.WriteString(style.
				Render(d))
		}

		m.tableStyle = m.GetBaseStyle()

		headerTop := lipgloss.NewStyle().
			Underline(true).
			Faint(true).
			Render(fmt.Sprintf("%s (%d)",
				m.GetTableName(), m.TableSelection))
		headerBot := strings.Repeat(lipgloss.NewStyle().
			Align(lipgloss.Center).
			Faint(true).
			Render("-"),
			m.viewport.Width)
		headerMid := builder.String()
		headerMid = headerMid + strings.Repeat(" ", m.viewport.Width)
		*h = fmt.Sprintf("%s\n%s\n%s",
			headerTop,
			headerMid,
			headerBot)

		done <- true
	}(&header)

	// footer
	go func(f *string) {
		footerTop := "╭──────╮"
		footerMid := fmt.Sprintf("┤ %d, %d  ", m.GetRow(), m.GetColumn())
		footerBot := "╰──────╯"
		gapSize := m.viewport.Width - runewidth.StringWidth(footerMid)
		footerTop = strings.Repeat(" ", gapSize) + footerTop
		footerMid = strings.Repeat("─", gapSize) + footerMid
		footerBot = strings.Repeat(" ", gapSize) + footerBot
		*f = fmt.Sprintf("%s\n%s\n%s", footerTop, footerMid, footerBot)

		done <- true
	}(&footer)

	<-done
	<-done
	<-done

	close(done)

	m.viewport.SetContent(content)

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}
