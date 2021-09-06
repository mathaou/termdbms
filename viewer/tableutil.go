package viewer

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// GetNewModel returns a TuiModel struct with some fields set
func GetNewModel(baseFileName string) TuiModel {
	return TuiModel{
		Table: TableState{
			Filename: baseFileName,
			Data:     make(map[string]interface{}),
		},
		TableHeaders:    make(map[string][]string),
		TableIndexMap:   make(map[int]string),
		TableSelection:  0,
		expandColumn:    -1,
		ready:           false,
		renderSelection: false,
		editModeEnabled: false,
		textInput:       textinput.NewModel(),
	}
}

// NumHeaders gets the number of columns for the current schema
func (m *TuiModel) NumHeaders() int {
	headers := m.GetHeaders()
	l := len(headers)
	if m.expandColumn > -1 || l == 0 {
		return 1
	}

	if l > 12 {
		return 12
	}

	return l
}

// CellWidth gets the current cell width for schema
func (m *TuiModel) CellWidth() int {
	return m.viewport.Width / m.NumHeaders()
}

// GetBaseStyle returns a new style that is used everywhere
func (m *TuiModel) GetBaseStyle() lipgloss.Style {
	s := lipgloss.NewStyle().
		Width(m.CellWidth()).
		MaxWidth(m.CellWidth()).
		Align(lipgloss.Left).
		Padding(0).
		Margin(0)

	if m.borderToggle {
		s = s.BorderRight(true).
			BorderLeft(true).
			BorderStyle(lipgloss.RoundedBorder())
	}

	return s
}

// GetColumn gets the column the mouse cursor is in
func (m *TuiModel) GetColumn() int {
	baseVal := m.mouseEvent.X / m.CellWidth()
	if m.renderSelection || m.editModeEnabled {
		return m.scrollXOffset + baseVal
	}

	return baseVal
}

// GetRow does math to get a valid row that's helpful
func (m *TuiModel) GetRow() int {
	baseVal := Max(m.mouseEvent.Y-headerHeight, 0)
	if m.renderSelection || m.editModeEnabled {
		return m.viewport.YOffset + baseVal
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

// GetSchemaData is a helper function to get the data of the current schema
func (m *TuiModel) GetSchemaData() map[string][]interface{} {
	n := m.GetSchemaName()
	return m.Table.Data[n].(map[string][]interface{})
}

func (m *TuiModel) GetSelectedOption() (*interface{}, int, []interface{}) {
	m.preScrollYOffset = m.viewport.YOffset
	m.preScrollYPosition = m.mouseEvent.Y
	selectedColumn := m.GetHeaders()[m.GetColumn()]
	col := m.GetSchemaData()[selectedColumn]
	row := m.GetRow()
	if row >= len(col) {
		return nil, row, col
	}
	return &col[row], row, col
}