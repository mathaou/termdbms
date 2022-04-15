package viewer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	HiddenTmpDirectoryName = ".termdbms"
	SQLSnippetsFile        = "snippets.termdbms"
)

func TruncateIfApplicable(m *TuiModel, conv string) (s string) {
	max := 0
	viewportWidth := m.Viewport.Width
	cellWidth := m.CellWidth()
	if m.UI.RenderSelection || m.UI.ExpandColumn > -1 {
		max = viewportWidth
	} else {
		max = cellWidth
	}

	if strings.Count(conv, "\n") > 0 {
		conv = SplitLines(conv)[0]
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

func GetInterfaceFromString(str string, original *interface{}) interface{} {
	switch (*original).(type) {
	case bool:
		bVal, _ := strconv.ParseBool(str)
		return bVal
	case int64:
		iVal, _ := strconv.ParseInt(str, 10, 64)
		return iVal
	case int32:
		iVal, _ := strconv.ParseInt(str, 10, 64)
		return iVal
	case float64:
		fVal, _ := strconv.ParseFloat(str, 64)
		return fVal
	case float32:
		fVal, _ := strconv.ParseFloat(str, 64)
		return fVal
	case time.Time:
		t := (*original).(time.Time)
		return t // TODO figure out how to handle things like time and date
	case string:
		return str
	}

	return nil
}

func GetStringRepresentationOfInterface(val interface{}) string {
	if str, ok := val.(string); ok {
		return str
	} else if i, ok := val.(int64); ok { // these default to int64 so not sure how this would affect 32 bit systems TODO
		return fmt.Sprintf("%d", i)
	} else if i, ok := val.(int32); ok { // these default to int32 so not sure how this would affect 32 bit systems TODO
		return fmt.Sprintf("%d", i)
	} else if i, ok := val.(float64); ok {
		return fmt.Sprintf("%.2f", i)
	} else if i, ok := val.(float32); ok {
		return fmt.Sprintf("%.2f", i)
	} else if t, ok := val.(time.Time); ok {
		str := t.String()
		return str
	} else if val == nil {
		return "NULL"
	}

	return ""
}

func WriteCSV(m *TuiModel) { // basically display table but without any styling
	if m.QueryData == nil || m.QueryResult == nil {
		return // should never happen but just making sure
	}

	var (
		builder [][]string
		buffer  strings.Builder
	)

	d := m.Data()

	// go through all columns
	for _, columnName := range d.TableHeaders[QueryResultsTableName] {
		var (
			rowBuilder []string
		)

		columnValues := m.GetSchemaData()[columnName]
		rowBuilder = append(rowBuilder, columnName)
		for _, val := range columnValues {
			s := GetStringRepresentationOfInterface(val)
			// display text based on type
			rowBuilder = append(rowBuilder, s)
		}
		builder = append(builder, rowBuilder)
	}

	depth := len(builder[0])
	headers := len(builder)

	for i := 0; i < depth; i++ {
		var r []string
		for x := 0; x < headers; x++ {
			r = append(r, builder[x][i])
		}
		buffer.WriteString(strings.Join(r, ","))
		buffer.WriteString("\n")
	}

	WriteTextFile(m, buffer.String())
}

func WriteTextFile(m *TuiModel, text string) (string, error) {
	rand.Seed(time.Now().Unix())
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
	if strings.Count(s, "\n") == 0 {
		return append(lines, s)
	}

	reader := strings.NewReader(s)
	sc := bufio.NewScanner(reader)

	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func GetScrollDownMaximumForSelection(m *TuiModel) int {
	max := 0
	if m.UI.RenderSelection {
		conv, _ := FormatJson(m.Data().EditTextBuffer)
		lines := SplitLines(conv)
		max = len(lines)
	} else if m.UI.FormatModeEnabled {
		max = len(SplitLines(DisplayFormatText(m)))
	} else {
		return len(m.GetColumnData())
	}

	return max
}

// FormatJson is some more code I stole off stackoverflow
func FormatJson(str string) (string, error) {
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
	rand.Seed(time.Now().UnixNano())
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

// MATH YO

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
