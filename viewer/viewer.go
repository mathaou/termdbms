package viewer

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mathaou/termdbms/list"
	"github.com/mathaou/termdbms/tuiutil"
)

var (
	HeaderHeight       = 2
	FooterHeight       = 1
	MaxInputLength     int
	HeaderStyle        lipgloss.Style
	FooterStyle        lipgloss.Style
	HeaderDividerStyle lipgloss.Style
	InitialModel       *TuiModel
)

func (m *TuiModel) Data() *UIData {
	if m.QueryData != nil {
		return m.QueryData
	}

	return &m.DefaultData
}

func (m *TuiModel) Table() *TableState {
	if m.QueryResult != nil {
		return m.QueryResult
	}

	return &m.DefaultTable
}

func SetStyles() {
	HeaderStyle = lipgloss.NewStyle()
	FooterStyle = lipgloss.NewStyle()

	HeaderDividerStyle = lipgloss.NewStyle().
		Align(lipgloss.Center)

	if !tuiutil.Ascii {
		HeaderStyle = HeaderStyle.
			Foreground(lipgloss.Color(tuiutil.HeaderTopForeground()))

		FooterStyle = FooterStyle.
			Foreground(lipgloss.Color(tuiutil.FooterForeground()))

		HeaderDividerStyle = HeaderDividerStyle.
			Foreground(lipgloss.Color(tuiutil.HeaderBottom()))
	}
}

// INIT UPDATE AND RENDER

// Init currently doesn't do anything but necessary for interface adherence
func (m TuiModel) Init() tea.Cmd {
	SetStyles()

	return nil
}

// Update is where all commands and whatnot get processed
func (m TuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var (
		command  tea.Cmd
		commands []tea.Cmd
	)

	if !m.UI.FormatModeEnabled {
		m.Viewport, _ = m.Viewport.Update(message)
	}

	switch msg := message.(type) {
	case list.FilterMatchesMessage:
		m.ClipboardList, command = m.ClipboardList.Update(msg)
		break
	case tea.MouseMsg:
		HandleMouseEvents(&m, &msg)
		m.SetViewSlices()
		break
	case tea.WindowSizeMsg:
		event := HandleWindowSizeEvents(&m, &msg)
		if event != nil {
			commands = append(commands, event)
		}
		break
	case tea.KeyMsg:
		str := msg.String()
		if m.UI.ShowClipboard {
			HandleClipboardEvents(&m, str, &command, msg)
			break
		}

		// when fullscreen selection viewing is in session, don't allow UI manipulation other than quit or exit
		s := msg.String()
		invalidRenderCommand := m.UI.RenderSelection &&
			s != "esc" &&
			s != "ctrl+c" &&
			s != "q" &&
			s != "p" &&
			s != "m" &&
			s != "n"
		if invalidRenderCommand {
			break
		}

		if s == "ctrl+c" || (s == "q" && (!m.UI.EditModeEnabled && !m.UI.FormatModeEnabled)) {
			return m, tea.Quit
		}

		event := HandleKeyboardEvents(&m, &msg)
		if event != nil {
			commands = append(commands, event)
		}
		if !m.UI.EditModeEnabled && m.Ready {
			m.SetViewSlices()
			if m.UI.FormatModeEnabled {
				MoveCursorWithinBounds(&m)
			}
		}

		break
	case error:
		return m, nil
	}

	if m.Viewport.HighPerformanceRendering {
		commands = append(commands, command)
	}

	return m, tea.Batch(commands...)
}

// View is where all rendering happens
func (m TuiModel) View() string {
	if !m.Ready || m.Viewport.Width == 0 {
		return "\n\tInitializing..."
	}

	// this ensures that all 3 parts can be worked on concurrently(ish)
	done := make(chan bool, 3)
	defer close(done) // close

	var footer, header, content string

	// body
	go func(c *string) {
		*c = AssembleTable(&m)
		done <- true
	}(&content)

	if m.UI.ShowClipboard {
		<-done
		return content
	}

	// header
	go HeaderAssembly(&m, &header, &done)
	// footer (shows row/col for now)
	go FooterAssembly(&m, &footer, &done)

	// block until all 3 done
	<-done
	<-done
	<-done

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer) // render
}
