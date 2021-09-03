package viewer

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"math"
	"runtime"
	"strings"
)

var (
	width        int
	height       int
	headerHeight = 3
	footerHeight = 3
	newline      string
)

const (
	highlight                 = "#0168B3" // change to whatever
	headerForeground          = "#231F20"
	headerBorderBackground    = "#AAAAAA"
	maximumRendererCharacters = math.MaxInt64 // this is kind of arbitrary
)

// TuiModel holds all the necessary state for this app to work the way I designed it to
type TuiModel struct {
	Table              map[string]interface{}
	TableHeaders       map[string][]string
	TableIndexMap      map[int]string
	TableSelection     int
	ready              bool
	renderSelection    bool
	selectionText      string
	preScrollYOffset   int
	preScrollYPosition int
	borderToggle       bool
	expandColumn       int
	viewport           viewport.Model
	tableStyle         lipgloss.Style
	mouseEvent         tea.MouseEvent
}

// INIT UPDATE AND RENDER

// Init currently doesn't do anything but necessary for interface adherence
func (m TuiModel) Init() tea.Cmd {
	newline = "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}
	return nil
}

// Update is where all commands and whatnot get processed
func (m TuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.MouseMsg:
		handleMouseEvents(&m, &msg)
		break
	case tea.WindowSizeMsg:
		handleWidowSizeEvents(&m, &msg)
		break
	case tea.KeyMsg:
		// when fullscreen selection viewing is in session, don't allow UI manipulation other than quit or exit
		s := msg.String()
		if m.renderSelection &&
			s != "esc" &&
			s != "ctrl+c" &&
			s != "q" &&
			s != "p"{
			break
		}
		if s == "ctrl+c" || s == "q" {
			return m, tea.Quit
		}

		handleKeyboardEvents(&m, &msg)
	}
	m.viewport, _ = m.viewport.Update(message)

	return m, nil
}

// View is where all rendering happens
func (m TuiModel) View() string {
	if !m.ready || m.viewport.Width == 0 {
		return "\n  Initializing..."
	}

	// this ensures that all 3 parts can be worked on concurrently(ish)
	done := make(chan bool, 3)

	var footer, header, content string

	// body
	go func(c *string) {
		*c = assembleTable(&m)
		done <- true
	}(&content)

	// header
	go func(h *string) {
		var (
			builder []string
		)

		style := m.GetBaseStyle().
			Width(m.CellWidth()).
			Foreground(lipgloss.Color(headerForeground)).
			Background(lipgloss.Color(headerBorderBackground))
		headers := m.GetHeaders()
		for i, d := range headers { // write all headers
			if m.expandColumn != -1 && i != m.expandColumn {
				continue
			}
			builder = append(builder, style.
				Render(TruncateIfApplicable(&m, d)))
		}

		{
			// schema name
			headerTop := lipgloss.NewStyle().
				Underline(true).
				Faint(true).
				Render(fmt.Sprintf("%s (%d)",
					m.GetSchemaName(), m.TableSelection))
			// separator
			headerBot := strings.Repeat(lipgloss.NewStyle().
				Align(lipgloss.Center).
				Faint(true).
				Render("-"),
				m.viewport.Width)
			headerMid := strings.Join(builder, "")
			headerMid = headerMid + strings.Repeat(" ", m.viewport.Width)
			*h = fmt.Sprintf("%s\n%s\n%s",
				headerTop,
				headerMid,
				headerBot)
		}

		done <- true
	}(&header)

	// footer (shows row/col for now)
	go func(f *string) {
		{
			footerTop := "╭──────╮"
			footerMid := fmt.Sprintf("┤ %d, %d ", m.GetRow(), m.GetColumn())
			footerBot := "╰──────╯"
			gapSize := m.viewport.Width - runewidth.StringWidth(footerMid)
			footerTop = strings.Repeat(" ", gapSize) + footerTop
			footerMid = strings.Repeat("─", gapSize) + footerMid
			footerBot = strings.Repeat(" ", gapSize) + footerBot
			*f = fmt.Sprintf("%s\n%s\n%s", footerTop, footerMid, footerBot)
		}

		done <- true
	}(&footer)

	// block until all 3 done
	<-done
	<-done
	<-done

	close(done) // close

	m.viewport.SetContent(content)

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer) // render
}
