package viewer

import (
	"github.com/charmbracelet/lipgloss"
	"termdbms/tuiutil"
)

var maxHeaders int

// AssembleTable shows either the selection text or the table
func AssembleTable(m *TuiModel) string {
	if m.UI.HelpDisplay {
		return GetHelpText()
	}
	if m.UI.RenderSelection {
		return DisplaySelection(m)
	}
	if m.UI.FormatModeEnabled {
		return DisplayFormatText(m)
	}

	return DisplayTable(m)
}

// NumHeaders gets the number of columns for the current schema
func (m *TuiModel) NumHeaders() int {
	headers := m.GetHeaders()
	l := len(headers)
	if m.UI.ExpandColumn > -1 || l == 0 {
		return 1
	}

	maxHeaders = m.Viewport.Width / 20 // seemed like a good number

	if l > maxHeaders {
		return maxHeaders
	}

	return l
}

// CellWidth gets the current cell width for schema
func (m *TuiModel) CellWidth() int {
	h := m.NumHeaders()
	return m.Viewport.Width / h
}

// GetBaseStyle returns a new style that is used everywhere
func (m *TuiModel) GetBaseStyle() lipgloss.Style {
	cw := m.CellWidth()
	s := lipgloss.NewStyle().
		Foreground(lipgloss.Color(tuiutil.TextColor())).
		Width(cw).
		Align(lipgloss.Left)

	if m.UI.BorderToggle && !tuiutil.Ascii {
		s = s.BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(tuiutil.BorderColor()))
	}

	return s
}

// GetColumn gets the column the mouse cursor is in
func (m *TuiModel) GetColumn() int {
	baseVal := m.MouseData.X / m.CellWidth()
	if m.UI.RenderSelection || m.UI.EditModeEnabled || m.UI.FormatModeEnabled {
		return m.Scroll.ScrollXOffset + baseVal
	}

	return baseVal
}

// GetRow does math to get a valid row that's helpful
func (m *TuiModel) GetRow() int {
	baseVal := Max(m.MouseData.Y-HeaderHeight, 0)
	if m.UI.RenderSelection || m.UI.EditModeEnabled {
		return m.Viewport.YOffset + baseVal
	} else if m.UI.FormatModeEnabled {
		return m.Scroll.PreScrollYOffset + baseVal
	}
	return baseVal
}

// GetSchemaName gets the current schema name
func (m *TuiModel) GetSchemaName() string {
	return m.Data.TableIndexMap[m.UI.CurrentTable]
}

// GetHeaders does just that for the current schema
func (m *TuiModel) GetHeaders() []string {
	return m.Data.TableHeaders[m.GetSchemaName()]
}

func (m *TuiModel) SetViewSlices() {
	if m.UI.FormatModeEnabled {
		var slices []*string
		for i := 0; i < m.Viewport.Height; i++ {
			yOffset := Max(m.Viewport.YOffset, 0)
			if yOffset+i > len(m.Format.Text)-1 {
				break
			}
			pStr := &m.Format.Text[Max(yOffset+i, 0)]
			slices = append(slices, pStr)
		}
		m.Format.EditSlices = slices
		m.UI.CanFormatScroll = len(m.Format.Text)-m.Viewport.YOffset-m.Viewport.Height > 0
		if m.Format.CursorX < 0 {
			m.Format.CursorX = 0
		}
	} else {
		// header slices
		headers := m.Data.TableHeaders[m.GetSchemaName()]
		headersLen := len(headers)

		if headersLen > maxHeaders {
			headers = headers[m.Scroll.ScrollXOffset : maxHeaders+m.Scroll.ScrollXOffset-1]
		}

		// data slices
		for _, columnName := range headers {
			interfaceValues := m.GetSchemaData()[columnName]
			if len(interfaceValues) >= m.Viewport.Height {
				min := Min(m.Viewport.YOffset, len(interfaceValues)-m.Viewport.Height)
				m.Data.TableSlices[columnName] = interfaceValues[min : m.Viewport.Height+min]
			} else {
				m.Data.TableSlices[columnName] = interfaceValues
			}
		}

		m.Data.TableHeadersSlice = headers
	}
	// format slices
}

// GetSchemaData is a helper function to get the data of the current schema
func (m *TuiModel) GetSchemaData() map[string][]interface{} {
	n := m.GetSchemaName()
	return m.Table.Data[n].(map[string][]interface{})
}

func (m *TuiModel) GetSelectedColumnName() string {
	return m.GetHeaders()[m.GetColumn()]
}

func (m *TuiModel) GetColumnData() []interface{} {
	return m.GetSchemaData()[m.GetSelectedColumnName()]
}

func (m *TuiModel) GetRowData() map[string]interface{} {
	defer func() {
		if recover() != nil {
			println("Whoopsy!") // TODO, this happened once
		}
	}()
	headers := m.GetHeaders()
	schema := m.GetSchemaData()
	data := make(map[string]interface{})
	for _, v := range headers {
		data[v] = schema[v][m.GetRow()]
	}

	return data
}

func (m *TuiModel) GetSelectedOption() (*interface{}, int, []interface{}) {
	if !m.UI.FormatModeEnabled {
		m.Scroll.PreScrollYOffset = m.Viewport.YOffset
		m.Scroll.PreScrollYPosition = m.MouseData.Y
	}
	row := m.GetRow()
	col := m.GetColumnData()
	if row >= len(col) {
		return nil, row, col
	}
	return &col[row], row, col
}

func (m *TuiModel) DisplayMessage(msg string) {
	m.Data.EditTextBuffer = msg
	m.UI.EditModeEnabled = false
	m.UI.RenderSelection = true
}

func (m *TuiModel) GetSelectedLineEdit() *LineEdit {
	if m.TextInput.Model.Focused() {
		return &m.TextInput
	}

	return &m.FormatInput
}

func ToggleColumn(m *TuiModel) {
	if m.UI.ExpandColumn > -1 {
		m.UI.ExpandColumn = -1
	} else {
		m.UI.ExpandColumn = m.GetColumn()
	}
}
