package viewer

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		width := msg.Width
		height := msg.Height
		m.Viewport = viewport.Model{
			Width:  width,
			Height: height - verticalMargins}

		m.ClipboardList.SetWidth(width)
		m.ClipboardList.SetHeight(height)
		TUIWidth = width
		TUIHeight = height
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

	if m.UI.EditModeEnabled { // handle edit mode
		HandleEditMode(m, str)
		return nil
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

		return nil
	}

	for k := range GlobalCommands {
		if str == k {
			return GlobalCommands[str](m)
		}
	}

	return nil
}
