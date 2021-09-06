package viewer

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strconv"
	"time"
)

// selectOption does just that
func selectOption(m *TuiModel) {
	m.renderSelection = true
	raw, row, col := m.GetSelectedOption()
	l := len(col)

	if row < l && l > 0 {
		if conv, ok := (*raw).(string); ok {
			if format, err := formatJson(conv); err == nil {
				m.selectionText = format
			} else {
				m.selectionText = TruncateIfApplicable(m, conv)
			}
		} else {
			m.selectionText = ""
		}
	} else {
		m.renderSelection = false
	}
}

func swapTableValues(m *TuiModel, f, t *TableState) {
	from := &f.Data
	to := &t.Data
	for k, v := range *from {
		if copyValues, ok := v.(map[string][]interface{}); ok {
			columnNames := m.TableHeaders[k]
			columnValues := make(map[string][]interface{})
			// golang wizardry
			columns := make([]interface{}, len(columnNames))

			for i, _ := range columns {
				columns[i] = copyValues[columnNames[i]][0]
			}

			for i, colName := range columnNames {
				columnValues[colName] = columns[i].([]interface{})
			}

			(*to)[k] = columnValues // data for schema, organized by column
		}
	}
}

func toggleColumn(m *TuiModel) {
	if m.expandColumn > -1 {
		m.expandColumn = -1
	} else {
		m.expandColumn = m.GetColumn()
	}
}

// scrollDown is a simple function to move the viewport down
func scrollDown(m *TuiModel) {
	max := getScrollDownMaxForSelection(m)

	if m.viewport.YOffset < max-1 {
		m.viewport.YOffset++
		m.mouseEvent.Y = Min(m.mouseEvent.Y, m.viewport.YOffset)
	}
}

// scrollUp is a simple function to move the viewport up
func scrollUp(m *TuiModel) {
	if m.viewport.YOffset > 0 {
		m.viewport.YOffset--
		m.mouseEvent.Y = Min(m.mouseEvent.Y, m.viewport.YOffset)
	} else {
		m.mouseEvent.Y = headerHeight
	}
}


// TABLE STUFF

// displayTable does some fancy stuff to get a table rendered in text
func displayTable(m *TuiModel) string {
	var (
		builder []string
		headersSlice []string
	)
	columnNamesToInterfaceArray := m.GetSchemaData()

	headers := m.GetHeaders() // TODO: maybe hot load the next slice in update
	if len(headers) > 12 {
		headersSlice = headers[m.scrollXOffset:12 + m.scrollXOffset]
	} else {
		headersSlice = headers
	}
	// go through all columns
	for c, columnName := range headersSlice {
		if m.expandColumn > -1 && m.expandColumn != c {
			continue
		}

		var (
			rowBuilder []string
			columnValues []interface{}
		)

		interfaceValues := columnNamesToInterfaceArray[columnName]
		if len (interfaceValues) >= m.viewport.Height {
			min := Min(m.viewport.YOffset, len(interfaceValues) - m.viewport.Height)
			columnValues = interfaceValues[min:m.viewport.Height + min]
		} else {
			columnValues = interfaceValues
		}

		for r, val := range columnValues {
			base := m.GetBaseStyle()
			// handle highlighting
			if c == m.GetColumn() && r == m.GetRow() {
				base.Foreground(lipgloss.Color(highlight))
			}
			// display text based on type
			s := GetStringRepresentationOfInterface(m, val)
			rowBuilder = append(rowBuilder, base.Render(s))
		}

		for len(rowBuilder) < m.viewport.Height {
			rowBuilder = append(rowBuilder, "")
		}

		// get a list of columns
		builder = append(builder, lipgloss.JoinVertical(lipgloss.Left, rowBuilder...))
	}

	// join them into rows
	return lipgloss.JoinHorizontal(lipgloss.Left, builder...)
}

// displaySelection does that or writes it to a file if the selection is over a limit
func displaySelection(m *TuiModel) string {
	selectedColumn := m.GetHeaders()[m.GetColumn()]
	col := m.GetSchemaData()[selectedColumn]
	m.expandColumn = m.GetColumn()
	row := m.GetRow()
	if m.mouseEvent.Y >= m.viewport.Height + headerHeight && !m.renderSelection { // this is for when the selection is outside the bounds
		return displayTable(m)
	}

	base := m.GetBaseStyle()

	if m.selectionText != "" { // this is basically just if its a string follow these rules
		rows := SplitLines(m.selectionText)
		for len(rows) < m.viewport.Height {
			rows = append(rows, "")
		}
		return base.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	var prettyPrint string
	raw := col[row]

	if conv, ok := raw.(int64); ok {
		prettyPrint = strconv.Itoa(int(conv))
	} else if i, ok := raw.(float64); ok {
		prettyPrint = base.Render(fmt.Sprintf("%.2f", i))
	} else if t, ok := raw.(time.Time); ok {
		str := t.String()
		prettyPrint = base.Render(TruncateIfApplicable(m, str))
	} else if raw == nil {
		prettyPrint = base.Render("NULL")
	}
	if lipgloss.Width(prettyPrint) > maximumRendererCharacters {
		fileName, err := WriteTextFile(m, prettyPrint)
		if err != nil {
			fmt.Printf("ERROR: could not write file %d", fileName)
		}
		return fmt.Sprintf("Selected string exceeds maximum limit of %d characters. \n"+
			"The file was written to your current working "+
			"directory for your convenience with the filename \n%s.", maximumRendererCharacters, fileName)
	}

	lines := SplitLines(prettyPrint)
	for len(lines) < m.viewport.Height {
		lines = append(lines, "")
	}

	prettyPrint = base.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return prettyPrint
}