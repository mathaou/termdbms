package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"io/ioutil"
	"math"
	"strconv"
)

func GetNewModel() TuiModel {
	return TuiModel{
		Table:          make(map[string]interface{}),
		TableHeaders:   make(map[string][]string),
		TableIndexMap:  make(map[int]string),
		TableSelection: 0,
		ready:          false,
		renderSelection: false,
	}
}

func (m *TuiModel) NumHeaders() int {
	return len(m.GetHeaders())
}

func (m *TuiModel) CellWidth() int {
	return m.viewport.Width / m.NumHeaders()
}

func (m *TuiModel) GetBaseStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(m.CellWidth()).
		Align(lipgloss.Left).
		BorderRight(true).
		BorderLeft(true)
}

func (m *TuiModel) GetColumn() int {
	return m.mouseEvent.X / m.CellWidth()
}

func (m *TuiModel) GetRow() int {
	return int(math.Max(float64(m.mouseEvent.Y-headerHeight), 0)) + m.viewport.YOffset
}

func (m *TuiModel) GetTableName() string {
	return m.TableIndexMap[m.TableSelection]
}

func (m *TuiModel) GetHeaders() []string {
	return m.TableHeaders[m.GetTableName()]
}

func (m *TuiModel) GetTable() map[string][]interface{} {
	return m.Table[m.GetTableName()].(map[string][]interface{})
}


func selectOption(m *TuiModel, tbl map[string][]interface{}) {
	selectedColumn := m.GetHeaders()[m.GetColumn()]
	col := tbl[selectedColumn]
	row := m.GetRow()
	l := len(col)

	if row < l && l > 0 {
		m.renderSelection = true
	}
}

func scrollDown(m *TuiModel, tbl map[string][]interface{}) {
	max := 0
	for _, v := range tbl {
		if len(v) > max {
			max = len(v)
		}
	}
	if m.viewport.YOffset < max-1 {
		m.viewport.YOffset++
	}
}

func scrollUp(m *TuiModel) {
	if m.viewport.YOffset > 0 {
		m.viewport.YOffset--
	}
}


func PrettyString(str string) (string, error) {
	b := []byte(str)
	if !json.Valid(b) {
		return str, nil
	}
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, b, "", "    "); err != nil {
		return "", err
	}
	return prettyJSON.String(), nil
}

func displayTable(m *TuiModel) string {
	var builder []string
	columnNamesToInterfaceArray := m.GetTable()

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
				if int(max) == minVal {
					s = s[:len(s) - 3] + "..."
				}
				rowBuilder = append(rowBuilder, base.Render(s))
			} else if i, ok := val.(int64); ok {
				rowBuilder = append(rowBuilder, base.Render(fmt.Sprintf("%d", i)))
			}
		}

		builder = append(builder, lipgloss.JoinVertical(lipgloss.Left, rowBuilder...))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, builder...)
}

func displaySelection(m *TuiModel) string {
	selectedColumn := m.GetHeaders()[m.GetColumn()]
	col := m.GetTable()[selectedColumn]
	row := m.GetRow()
	if row >= len(col) {
		return displayTable(m)
	}
	raw := col[row]

	var prettyPrint string
	if _, ok := raw.(string); ok {
		p, _ := PrettyString(raw.(string))
		prettyPrint = p
	} else if _, ok := raw.(int64); ok {
		prettyPrint = strconv.Itoa(int(raw.(int64)))
	}
	if len(prettyPrint) > maximumRendererCharacters {
		fileName := m.GetTableName()+"_"+selectedColumn+"_"+strconv.Itoa(row)+".txt"
		e := ioutil.WriteFile(fileName, []byte(prettyPrint), 0777)
		if e != nil {
			return fmt.Sprintf("Error writing file %v", e)
		}
		return fmt.Sprintf("Selected string exceeds maximum limit of %d characters. \n" +
			"The file was written to your current working " +
			"directory for your convenience with the filename \n%s.", maximumRendererCharacters, fileName)
	} else {
		return prettyPrint
	}
}

func assembleTable(m *TuiModel) string {
	if m.renderSelection {
		return displaySelection(m)
	}

	return displayTable(m)
}
