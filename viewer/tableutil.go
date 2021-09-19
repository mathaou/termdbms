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
			Slices:         nil,
			Text:           nil,
			RunningOffsets: nil,
			CursorX:        0,
			CursorY:        0,
		},
		TableHeaders:    make(map[string][]string),
		DataSlices:      make(map[string][]interface{}),
		TableIndexMap:   make(map[int]string),
		TableSelection:  0,
		expandColumn:    -1,
		ready:           false,
		renderSelection: false,
		editModeEnabled: false,
		textInput: LineEdit{
			Model:         NewModel(),
			EnterBehavior: HeaderLineEditEnterBehavior,
		},
		formatInput: LineEdit{
			Model:         NewModel(),
			EnterBehavior: BodyLineEditEnterBehavior,
		},
	}
	m.formatInput.Model.Prompt = ""
	return m
}

// NumHeaders gets the number of columns for the current schema
func (m *TuiModel) NumHeaders() int {
	headers := m.GetHeaders()
	l := len(headers)
	if m.expandColumn > -1 || l == 0 {
		return 1
	}

	maxHeaders = m.viewport.Width / 20 // seemed like a good number

	if l > maxHeaders {
		return maxHeaders
	}

	return l
}

// CellWidth gets the current cell width for schema
func (m *TuiModel) CellWidth() int {
	h := m.NumHeaders()
	return m.viewport.Width / h
}

// GetBaseStyle returns a new style that is used everywhere
func (m *TuiModel) GetBaseStyle() lipgloss.Style {
	cw := m.CellWidth()
	s := lipgloss.NewStyle().
		Foreground(lipgloss.Color(textColor())).
		Width(cw).
		Align(lipgloss.Left)

	if m.borderToggle && !Ascii {
		s = s.BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(borderColor()))
	}

	return s
}

// GetColumn gets the column the mouse cursor is in
func (m *TuiModel) GetColumn() int {
	baseVal := m.mouseEvent.X / m.CellWidth()
	if m.renderSelection || m.editModeEnabled || m.formatModeEnabled {
		return m.scrollXOffset + baseVal
	}

	return baseVal
}

// GetRow does math to get a valid row that's helpful
func (m *TuiModel) GetRow() int {
	baseVal := Max(m.mouseEvent.Y-headerHeight, 0)
	if m.renderSelection || m.editModeEnabled {
		return m.viewport.YOffset + baseVal
	} else if m.formatModeEnabled {
		return m.preScrollYOffset + baseVal
	}
	return baseVal
}

// GetSchemaName gets the current schema name
func (m *TuiModel) GetSchemaName() string {
	return m.TableIndexMap[m.TableSelection]
}

// GetHeaders does just that for the current schema
func (m *TuiModel) GetHeaders() []string {
	return m.TableHeaders[m.GetSchemaName()]
}

func (m *TuiModel) SetViewSlices() {
	if m.formatModeEnabled {
		var slices []*string
		for i := 0; i < m.viewport.Height; i++ {
			yOffset := Max(m.viewport.YOffset, 0)
			if yOffset+i > len(m.Format.Text)-1 {
				break
			}
			pStr := &m.Format.Text[Max(yOffset+i, 0)]
			slices = append(slices, pStr)
		}
		m.Format.Slices = slices
		m.CanFormatScroll = len(m.Format.Text)-m.viewport.YOffset-m.viewport.Height > 0
		if m.Format.CursorX < 0 {
			m.Format.CursorX = 0
		}
	} else {
		// header slices
		headers := m.TableHeaders[m.GetSchemaName()]
		headersLen := len(headers)

		if headersLen > maxHeaders {
			headers = headers[m.scrollXOffset : maxHeaders+m.scrollXOffset-1]
		}

		// data slices
		for _, columnName := range headers {
			interfaceValues := m.GetSchemaData()[columnName]
			if len(interfaceValues) >= m.viewport.Height {
				min := Min(m.viewport.YOffset, len(interfaceValues)-m.viewport.Height)
				m.DataSlices[columnName] = interfaceValues[min : m.viewport.Height+min]
			} else {
				m.DataSlices[columnName] = interfaceValues
			}
		}

		m.TableHeadersSlice = headers
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
	headers := m.GetHeaders()
	schema := m.GetSchemaData()
	data := make(map[string]interface{})
	for _, v := range headers {
		data[v] = schema[v][m.GetRow()]
	}

	return data
}

func (m *TuiModel) GetSelectedOption() (*interface{}, int, []interface{}) {
	if !m.formatModeEnabled {
		m.preScrollYOffset = m.viewport.YOffset
		m.preScrollYPosition = m.mouseEvent.Y
	}
	row := m.GetRow()
	col := m.GetColumnData()
	if row >= len(col) {
		return nil, row, col
	}
	return &col[row], row, col
}

func (m *TuiModel) DisplayMessage(msg string) {
	m.selectionText = msg
	m.editModeEnabled = false
	m.renderSelection = true
}

func (m *TuiModel) GetSelectedLineEdit() *LineEdit {
	if m.textInput.Model.Focused() {
		return &m.textInput
	}

	return &m.formatInput
}
