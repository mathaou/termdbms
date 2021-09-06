package viewer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"hash/fnv"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HiddenTmpDirectoryName = ".termdbms"
)

// non interface helper methods

func TruncateIfApplicable(m *TuiModel, conv string) (s string) {
	max := 0
	viewportWidth := m.viewport.Width
	cellWidth := m.CellWidth()
	if m.renderSelection || m.expandColumn > -1 {
		max = viewportWidth
	} else {
		max = cellWidth
	}
	textWidth := lipgloss.Width(conv)
	minVal := Min(textWidth, max)

	if max == minVal && textWidth >= max { // truncate
		s = conv[:minVal]
		s = s[:lipgloss.Width(s)-3] + "..."
	} else {
		s = conv
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
	from := m.Table.Data
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
		return len(m.GetSchemaData()[m.TableHeaders[m.GetSchemaName()][0]])
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

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func CopyFile(src string) (string, int64, error) {
	sourceFileStat, err := os.Stat(src)
	dst := fmt.Sprintf(".%d",
		Hash(fmt.Sprintf("%s%d",
			src,
			rand.Uint64())))
	if err != nil {
		return "", 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return "", 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer source.Close()

	destination, err := os.CreateTemp(HiddenTmpDirectoryName, dst)
	if err != nil {
		return "", 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	info, _ := destination.Stat()
	path, _ := filepath.Abs(
		fmt.Sprintf("%s/%s",
			HiddenTmpDirectoryName,
			info.Name())) // platform agnostic
	return path, nBytes, err
}

func Min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func Abs(a int) int {
	if a < 0 {
		return a * -1
	}

	return a
}