package viewer

import (
	"database/sql"
	"github.com/charmbracelet/lipgloss"
)

var maxHeaders int

// GetNewModel returns a TuiModel struct with some fields set
func GetNewModel(baseFileName string, db *sql.DB) TuiModel {
	m := TuiModel{
		Table: TableState{
			Database: &SQLite{
				FileName: baseFileName,
				db:       db,
			},
			Data: make(map[string]interface{}),
		},
		Format: FormatState{
			EditSlices:     nil,
			Text:           nil,
			RunningOffsets: nil,
			CursorX:        0,
			CursorY:        0,
		},
		UI: UIState{
			CanFormatScroll:   false,
			RenderSelection:   false,
			HelpDisplay:       false,
			EditModeEnabled:   false,
			FormatModeEnabled: false,
			BorderToggle:      false,
			CurrentTable:      0,
			ExpandColumn:      -1,
		},
		Scroll: ScrollData{},
		Data: UIData{
			TableHeaders:      make(map[string][]string),
			TableHeadersSlice: []string{},
			TableSlices:       make(map[string][]interface{}),
			TableIndexMap:     make(map[int]string),
		},
		TextInput: LineEdit{
			Model:         NewModel(),
			EnterBehavior: HeaderLineEditEnterBehavior,
		},
		FormatInput: LineEdit{
			Model:         NewModel(),
			EnterBehavior: BodyLineEditEnterBehavior,
		},
	}
	m.FormatInput.Model.Prompt = ""
	return m
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
		Foreground(lipgloss.Color(TextColor())).
		Width(cw).
		Align(lipgloss.Left)

	if m.UI.BorderToggle && !Ascii {
		s = s.BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(BorderColor()))
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
