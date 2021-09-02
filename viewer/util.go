package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"io/ioutil"
	"math"
	"os"
	"strconv"
)

// GetNewModel returns a TuiModel struct with some fields set
func GetNewModel() TuiModel {
	return TuiModel{
		Table:           make(map[string]interface{}),
		TableHeaders:    make(map[string][]string),
		TableIndexMap:   make(map[int]string),
		TableSelection:  0,
		ready:           false,
		renderSelection: false,
	}
}

// NumHeaders gets the number of columns for the current schema
func (m *TuiModel) NumHeaders() int {
	return len(m.GetHeaders())
}

// CellWidth gets the current cell width for schema
func (m *TuiModel) CellWidth() int {
	return m.viewport.Width / m.NumHeaders()
}

// GetBaseStyle returns a new style that is used everywhere
func (m *TuiModel) GetBaseStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(m.CellWidth()).
		Align(lipgloss.Left).
		BorderRight(true).
		BorderLeft(true)
}

// GetColumn gets the column the mouse cursor is in
func (m *TuiModel) GetColumn() int {
	return m.mouseEvent.X / m.CellWidth()
}

// GetRow does math to get a valid row that's helpful
func (m *TuiModel) GetRow() int {
	return int(math.Max(float64(m.mouseEvent.Y-headerHeight), 0)) + m.viewport.YOffset
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
	return m.Table[m.GetSchemaName()].(map[string][]interface{})
}

// selectOption does just that
func selectOption(m *TuiModel) {
	selectedColumn := m.GetHeaders()[m.GetColumn()]
	col := m.GetSchemaData()[selectedColumn]
	row := m.GetRow()
	l := len(col)

	if row < l && l > 0 {
		m.renderSelection = true
	}
}

// scrollDown is a simple function to move the viewport down
func scrollDown(m *TuiModel) {
	max := 0
	for _, v := range m.GetSchemaData() {
		if len(v) > max {
			max = len(v)
		}
	}
	if m.viewport.YOffset < max-1 {
		m.viewport.YOffset++
		m.mouseEvent.Y++
	}
}

// scrollUp is a simple function to move the viewport up
func scrollUp(m *TuiModel) {
	if m.viewport.YOffset > 0 {
		m.viewport.YOffset--
		m.mouseEvent.Y--
	} else {
		m.mouseEvent.Y = headerHeight
	}
}

// formatJson is some more code I stole off stackoverflow
func formatJson(str string) (string, error) {
	b := []byte(str)
	if !json.Valid(b) { // return original string if not json
		return str, nil
	}
	var formattedJson bytes.Buffer
	if err := json.Indent(&formattedJson, b, "", "    "); err != nil {
		return "", err
	}
	return formattedJson.String(), nil
}

// displayTable does some fancy stuff to get a table rendered in text
func displayTable(m *TuiModel) string {
	var builder []string
	columnNamesToInterfaceArray := m.GetSchemaData()

	// go through all columns
	for c, columnName := range m.GetHeaders() {
		var rowBuilder []string
		columnValues := columnNamesToInterfaceArray[columnName]
		for r, val := range columnValues {
			base := m.GetBaseStyle()
			if c == m.GetColumn() && r == m.GetRow() {
				base.Foreground(lipgloss.Color(highlight))
			}
			if str, ok := val.(string); ok {
				max := float64(m.CellWidth())
				minVal := int(math.Min(float64(len(str)), max))
				s := str[:minVal]
				if int(max) == minVal { // truncate
					s = s[:len(s)-3] + "..."
				}
				rowBuilder = append(rowBuilder, base.Render(s))
			} else if i, ok := val.(int64); ok {
				rowBuilder = append(rowBuilder, base.Render(fmt.Sprintf("%d", i)))
			}
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
	row := m.GetRow()
	if row >= len(col) {
		return displayTable(m)
	}
	raw := col[row]

	var prettyPrint string
	if _, ok := raw.(string); ok {
		p, _ := formatJson(raw.(string))
		prettyPrint = p
	} else if _, ok := raw.(int64); ok {
		prettyPrint = strconv.Itoa(int(raw.(int64)))
	}
	if len(prettyPrint) > maximumRendererCharacters {
		fileName := m.GetSchemaName() + "_" + selectedColumn + "_" + strconv.Itoa(row) + ".txt"
		e := ioutil.WriteFile(fileName, []byte(prettyPrint), 0777)
		if e != nil {
			return fmt.Sprintf("Error writing file %v", e)
		}
		return fmt.Sprintf("Selected string exceeds maximum limit of %d characters. \n"+
			"The file was written to your current working "+
			"directory for your convenience with the filename \n%s.", maximumRendererCharacters, fileName)
	} else {
		return prettyPrint
	}
}

// assembleTable shows either the selection text or the table
func assembleTable(m *TuiModel) string {
	if m.renderSelection {
		return displaySelection(m)
	}

	return displayTable(m)
}

// IsUrl is some code I stole off stackoverflow to validate paths
func IsUrl(fp string) bool {
	// Check if file already exists
	if _, err := os.Stat(fp); err == nil {
		return true
	}

	// Attempt to create it
	var d []byte
	if err := ioutil.WriteFile(fp, d, 0644); err == nil {
		os.Remove(fp) // And delete it
		return true
	}

	return false
}
