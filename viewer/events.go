package viewer

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"termdbms/list"
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
		if !m.UI.RenderSelection && !m.UI.EditModeEnabled && !m.UI.FormatModeEnabled {
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

func HandleClipboardEvents(m *TuiModel, str string, command *tea.Cmd, msg tea.Msg) {
	state := m.ClipboardList.FilterState()
	if (str == "q" || str == "esc" || str == "enter") && state != list.Filtering {
		switch str {
		case "enter":
			i, ok := m.ClipboardList.SelectedItem().(SQLSnippet)
			if ok {
				ExitToDefaultView(m)
				CreatePopulatedBuffer(m, nil, i.Query)
				m.UI.SQLEdit = true
			}
			break
		default:
			ExitToDefaultView(m)
		}
		m.ClipboardList.ResetFilter()
	} else {
		tmpItems := len(m.ClipboardList.Items())
		m.ClipboardList, *command = m.ClipboardList.Update(msg)
		if len(m.ClipboardList.Items()) != tmpItems { // if item removed
			m.Clipboard = m.ClipboardList.Items()
			b, _ := json.Marshal(m.Clipboard)
			snippetsFile := fmt.Sprintf("%s/%s", HiddenTmpDirectoryName, SQLSnippetsFile)
			f, _ := os.OpenFile(snippetsFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0775)
			f.Write(b)
			f.Close()
		}
	}
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
