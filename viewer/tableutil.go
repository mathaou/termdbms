package viewer

import (
	"errors"
	"github.com/charmbracelet/lipgloss"
	"termdbms/tuiutil"
)

var maxHeaders int

// AssembleTable shows either the selection text or the table
func AssembleTable(m *TuiModel) string {
	if m.UI.ShowClipboard {
		return ShowClipboard(m)
	}
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

	maxHeaders = 7

	if l > maxHeaders { // this just looked the best after some trial and error
		if l%5 == 0 {
			return 5
		} else if l%4 == 0 {
			return 4
		} else if l%3 == 0 {
			return 3
		} else {
			return 6 // primes and shiiiii
		}
	}

	return l
}

// CellWidth gets the current cell width for schema
func (m *TuiModel) CellWidth() int {
	h := m.NumHeaders()
	return m.Viewport.Width/h + 2
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
	return m.Data().TableIndexMap[m.UI.CurrentTable]
}

// GetHeaders does just that for the current schema
func (m *TuiModel) GetHeaders() []string {
	schema := m.GetSchemaName()
	d := m.Data()
	return d.TableHeaders[schema]
}

func (m *TuiModel) SetViewSlices() {
	d := m.Data()
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
		headers := d.TableHeaders[m.GetSchemaName()]
		headersLen := len(headers)

		if headersLen > maxHeaders {
			headers = headers[m.Scroll.ScrollXOffset : maxHeaders+m.Scroll.ScrollXOffset-1]
		}
		// data slices
		defer func() {
			if recover() != nil {
				panic(errors.New("adsf"))
			}
		}()

		for _, columnName := range headers {
			interfaceValues := m.GetSchemaData()[columnName]
			if len(interfaceValues) >= m.Viewport.Height {
				min := Min(m.Viewport.YOffset, len(interfaceValues)-m.Viewport.Height)

				d.TableSlices[columnName] = interfaceValues[min : m.Viewport.Height+min]
			} else {
				d.TableSlices[columnName] = interfaceValues
			}
		}

		d.TableHeadersSlice = headers
	}
	// format slices
}

// GetSchemaData is a helper function to get the data of the current schema
func (m *TuiModel) GetSchemaData() map[string][]interface{} {
	n := m.GetSchemaName()
	t := m.Table()
	d := t.Data
	return d[n].(map[string][]interface{})
}

func (m *TuiModel) GetSelectedColumnName() string {
	col := m.GetColumn()
	headers := m.GetHeaders()
	index := Min(m.NumHeaders()-1, col)
	return headers[index]
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
	m.Data().EditTextBuffer = msg
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
