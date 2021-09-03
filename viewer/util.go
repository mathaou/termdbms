package viewer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"math"
	"math/rand"
	"os"
	"strconv"
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

// selectOption does just that
func selectOption(m *TuiModel) {
	m.preScrollYOffset = m.viewport.YOffset
	m.preScrollYPosition = m.mouseEvent.Y
	selectedColumn := m.GetHeaders()[m.GetColumn()]
	col := m.GetSchemaData()[selectedColumn]
	row := m.GetRow()
	l := len(col)

	if row < l && l > 0 {
		m.renderSelection = true

		raw := col[row]
		if conv, ok := raw.(string); ok {
			if format, err := formatJson(conv); err == nil {
				m.selectionText = format
			} else {
				m.selectionText = TruncateIfApplicable(m, conv)
			}
		} else {
			m.selectionText = ""
		}
	}
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

// scrollDown is a simple function to move the viewport down
func scrollDown(m *TuiModel) {
	max := getScrollDownMaxForSelection(m)

	if m.viewport.YOffset < max-1 {
		m.viewport.YOffset++
		m.mouseEvent.Y = int(math.Min(float64(m.mouseEvent.Y), float64(m.viewport.YPosition)))
	}
}

// scrollUp is a simple function to move the viewport up
func scrollUp(m *TuiModel) {
	if m.viewport.YOffset > 0 {
		m.viewport.YOffset--
		m.mouseEvent.Y = int(math.Min(float64(m.mouseEvent.Y), float64(m.viewport.YPosition)))
	} else {
		m.mouseEvent.Y = headerHeight
	}
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

// displayTable does some fancy stuff to get a table rendered in text
func displayTable(m *TuiModel) string {
	var builder []string
	columnNamesToInterfaceArray := m.GetSchemaData()

	// go through all columns
	for c, columnName := range m.GetHeaders() {
		if m.expandColumn > -1 && m.expandColumn != c {
			continue
		}

		var rowBuilder []string
		columnValues := columnNamesToInterfaceArray[columnName]

		for r, val := range columnValues {
			base := m.GetBaseStyle()
			// handle highlighting
			if c == m.GetColumn() && r == m.GetRow() {
				base.Foreground(lipgloss.Color(highlight))
			}
			// display text based on type
			if str, ok := val.(string); ok {
				rowBuilder = append(rowBuilder, base.Render(TruncateIfApplicable(m, str)))
			} else if i, ok := val.(int64); ok {
				rowBuilder = append(rowBuilder, base.Render(fmt.Sprintf("%d", i)))
			} else if i, ok := val.(float64); ok {
				rowBuilder = append(rowBuilder, base.Render(fmt.Sprintf("%.2f", i)))
			} else if t, ok := val.(time.Time); ok {
				cw := m.CellWidth()
				str := t.String()
				minVal := int(math.Min(float64(len(str)), float64(cw)))
				s := str[:minVal]
				if len(s) == cw {
					s = s[:len(s)-3] + "..."
				}
				rowBuilder = append(rowBuilder, base.Render(s))
			} else if val == nil {
				rowBuilder = append(rowBuilder, base.Render("NULL"))
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
	m.expandColumn = m.GetColumn()
	row := m.GetRow()
	if m.mouseEvent.Y >= m.viewport.Height + headerHeight && !m.renderSelection { // this is for when the selection is outside the bounds
		return displayTable(m)
	}

	base := m.GetBaseStyle()

	if m.selectionText != "" {
		rows := SplitLines(m.selectionText)
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
	if len(prettyPrint) > maximumRendererCharacters {
		fileName, err := WriteText(m, prettyPrint)
		if err != nil {
			fmt.Printf("ERROR: could not write file %d", fileName)
		}
		return fmt.Sprintf("Selected string exceeds maximum limit of %d characters. \n"+
			"The file was written to your current working "+
			"directory for your convenience with the filename \n%s.", maximumRendererCharacters, fileName)
	}

	return prettyPrint
}

func WriteText(m *TuiModel, text string) (string, error) {
	fileName := m.GetSchemaName() + "_" + "renderView_" + fmt.Sprintf("%d", rand.Int()) + ".txt"
	e := os.WriteFile(fileName, []byte(text), 0777)
	return fileName, e
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