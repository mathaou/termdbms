package viewer

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strconv"
	"strings"
	"time"
)

var (
	Program *tea.Program
)

// selectOption does just that
func selectOption(m *TuiModel) {
	if m.renderSelection || m.helpDisplay {
		return
	}

	m.renderSelection = true
	raw, _, col := m.GetSelectedOption()
	l := len(col)
	row := m.viewport.YOffset + m.mouseEvent.Y - headerHeight

	if row <= l && l > 0 &&
		m.mouseEvent.Y >= headerHeight &&
		m.mouseEvent.Y < m.viewport.Height + headerHeight &&
		m.mouseEvent.X < m.CellWidth() * (len(m.TableHeadersSlice)) {
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

	if m.viewport.YOffset < max-m.viewport.Height {
		m.viewport.YOffset++
		m.mouseEvent.Y = Min(m.mouseEvent.Y, m.viewport.YOffset)
	}

	if !m.renderSelection {
		m.preScrollYPosition = m.mouseEvent.Y
		m.preScrollYOffset = m.viewport.YOffset
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

	if !m.renderSelection {
		m.preScrollYPosition = m.mouseEvent.Y
		m.preScrollYOffset = m.viewport.YOffset
	}
}

// TABLE STUFF

// displayTable does some fancy stuff to get a table rendered in text
func displayTable(m *TuiModel) string {
	var (
		builder []string
	)

	// go through all columns
	for c, columnName := range m.TableHeadersSlice {
		if m.expandColumn > -1 && m.expandColumn != c {
			continue
		}

		var (
			rowBuilder []string
		)

		columnValues := m.DataSlices[columnName]
		for r, val := range columnValues {
			base := m.GetBaseStyle()
			// handle highlighting
			if c == m.GetColumn() && r == m.GetRow() {
				base.Foreground(lipgloss.Color(highlight))
			}
			// display text based on type
			s := GetStringRepresentationOfInterface(val)
			rowBuilder = append(rowBuilder, base.Render(TruncateIfApplicable(m, s)))
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
	col := m.GetColumnData()
	m.expandColumn = m.GetColumn()
	row := m.GetRow()
	if m.mouseEvent.Y >= m.viewport.Height+headerHeight && !m.renderSelection { // this is for when the selection is outside the bounds
		return displayTable(m)
	}

	base := m.GetBaseStyle()

	if m.selectionText != "" { // this is basically just if its a string follow these rules
		_, err := formatJson(m.selectionText)
		rows := SplitLines(m.selectionText)
		if err == nil && strings.Contains(m.selectionText, "{"){
			rows = rows[m.viewport.YOffset:m.viewport.Height + m.viewport.YOffset]
		}

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
