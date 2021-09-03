package viewer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

// GetNewModel returns a TuiModel struct with some fields set
func GetNewModel() TuiModel {
	return TuiModel{
		Table:           make(map[string]interface{}),
		TableHeaders:    make(map[string][]string),
		TableIndexMap:   make(map[int]string),
		TableSelection:  0,
		expandColumn:    -1,
		ready:           false,
		renderSelection: false,
		editModeEnabled: false,
		textInput: textinput.NewModel(),
	}
}

// NumHeaders gets the number of columns for the current schema
func (m *TuiModel) NumHeaders() int {
	if m.expandColumn > -1  || len(m.GetHeaders()) == 0{
		return 1
	}
	return len(m.GetHeaders())
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
	n := m.GetSchemaName()
	return m.Table[n].(map[string][]interface{})
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

// non interface helper methods

func TruncateIfApplicable(m *TuiModel, conv string) string {
	max := func() float64 { // this might be kind of hacky, but it works
		if m.renderSelection || m.expandColumn > -1 {
			return float64(m.viewport.Width)
		} else {
			return float64(m.CellWidth())
		}
	}()
	minVal := int(math.Min(float64(len(conv)), max))
	s := conv[:minVal]
	if int(max) == minVal { // truncate
		s = s[:len(s)-3] + "..."
	}

	return s
}

func GetStringRepresentationOfInterface(m *TuiModel, val interface{}) string {
	if str, ok := val.(string); ok {
		return TruncateIfApplicable(m, str)
	} else if i, ok := val.(int64); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := val.(float64); ok {
		return fmt.Sprintf("%.2f", i)
	} else if t, ok := val.(time.Time); ok {
		str := t.String()
		return TruncateIfApplicable(m, str)
	} else if val == nil {
		return "NULL"
	}

	return ""
}

func WriteTextFile(m *TuiModel, text string) (string, error) {
	fileName := m.GetSchemaName() + "_" + "renderView_" + fmt.Sprintf("%d", rand.Int()) + ".txt"
	e := os.WriteFile(fileName, []byte(text), 0777)
	return fileName, e
}

// IsUrl is some code I stole off stackoverflow to validate paths
func IsUrl(fp string) bool {
	// Check if file already exists
	if _, err := os.Stat(fp); err == nil {
		return true
	}

	// Attempt to create it
	var d []byte
	if err := os.WriteFile(fp, d, 0644); err == nil {
		os.Remove(fp) // And delete it
		return true
	}

	return false
}

func FileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	return true, err
}

func SplitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func (m *TuiModel) CopyMap() (to map[string]interface{}) {
	from := m.Table
	to = map[string]interface{}{}

	for k, v := range from {
		if copyValues, ok := v.(map[string][]interface{}); ok {
			columnNames := m.TableHeaders[k]
			columnValues := make(map[string][]interface{})
			// golang wizardry
			columns := make([]interface{}, len(columnNames))

			for i, _ := range columns {
				columns[i] = copyValues[columnNames[i]]
			}

			for i, colName := range columnNames {
				val := columns[i].([]interface{})
				copy := make([]interface{}, len(val))
				for k, _ := range val {
					copy[k] = val[k]
				}
				columnValues[colName] = append(columnValues[colName], copy)
			}

			to[k] = columnValues // data for schema, organized by column
		}
	}

	return to
}

// assembleTable shows either the selection text or the table
func assembleTable(m *TuiModel) string {
	if m.renderSelection {
		return displaySelection(m)
	}

	return displayTable(m)
}

func getScrollDownMaxForSelection(m *TuiModel) int {
	max := 0
	if m.renderSelection {
		conv, _ := formatJson(m.selectionText)
		lines := SplitLines(conv)
		max = len(lines)
	} else {
		for _, v := range m.GetSchemaData() {
			if len(v) > max {
				max = len(v)
			}
		}
	}

	return max
}

// formatJson is some more code I stole off stackoverflow
func formatJson(str string) (string, error) {
	b := []byte(str)
	if !json.Valid(b) { // return original string if not json
		return str, errors.New("this is not valid JSON")
	}
	var formattedJson bytes.Buffer
	if err := json.Indent(&formattedJson, b, "", "    "); err != nil {
		return "", err
	}
	return formattedJson.String(), nil
}