package viewer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"os"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
)

func (m *TuiModel) CopyMap() (to map[string]interface{}) {
	from := m.Table().Data
	to = map[string]interface{}{}

	for k, v := range from {
		if copyValues, ok := v.(map[string][]interface{}); ok {
			columnNames := m.Data().TableHeaders[k]
			columnValues := make(map[string][]interface{})
			// golang wizardry
			columns := make([]interface{}, len(columnNames))

			for i := range columns {
				columns[i] = copyValues[columnNames[i]]
			}

			for i, colName := range columnNames {
				val := columns[i].([]interface{})
				buffer := make([]interface{}, len(val))
				for k := range val {
					buffer[k] = val[k]
				}
				columnValues[colName] = append(columnValues[colName], buffer)
			}

			to[k] = columnValues // data for schema, organized by column
		}
	}

	return to
}

// GetNewModel returns a TuiModel struct with some fields set
func GetNewModel(baseFileName string, db *sql.DB) TuiModel {
	m := TuiModel{
		DefaultTable: TableState{
			Database: &database.SQLite{
				FileName: baseFileName,
				Database: db,
			},
			Data: make(map[string]interface{}),
		},
		Format: FormatState{
			EditSlices:     nil,
			Text:           nil,
			RunningOffsets: nil,
			CursorX:        0,
			CursorY:        0,
		},
		UI: UIState{
			CanFormatScroll:   false,
			RenderSelection:   false,
			HelpDisplay:       false,
			EditModeEnabled:   false,
			FormatModeEnabled: false,
			BorderToggle:      false,
			CurrentTable:      0,
			ExpandColumn:      -1,
		},
		Scroll: ScrollData{},
		DefaultData: UIData{
			TableHeaders:      make(map[string][]string),
			TableHeadersSlice: []string{},
			TableSlices:       make(map[string][]interface{}),
			TableIndexMap:     make(map[int]string),
		},
		TextInput: LineEdit{
			Model: tuiutil.NewModel(),
		},
		FormatInput: LineEdit{
			Model: tuiutil.NewModel(),
		},
		Clipboard: []list.Item{},
	}
	m.FormatInput.Model.Prompt = ""

	snippetsFile := fmt.Sprintf("%s/%s", HiddenTmpDirectoryName, SQLSnippetsFile)

	exists, _ := Exists(snippetsFile)
	if exists {
		contents, _ := os.ReadFile(snippetsFile)
		var c []SQLSnippet
		json.Unmarshal(contents, &c)
		for _, v := range c {
			m.Clipboard = append(m.Clipboard, v)
		}
	}

	m.ClipboardList = list.NewModel(m.Clipboard, itemDelegate{}, 0, 0)

	m.ClipboardList.SetFilteringEnabled(true)
	m.ClipboardList.SetShowPagination(true)
	m.ClipboardList.SetShowTitle(false)

	return m
}

// SetModel creates a model to be used by bubbletea using some golang wizardry
func SetModel(m *TuiModel, c *sql.Rows, db *sql.DB) error {
	var err error

	indexMap := 0

	// gets all the schema names of the database
	rows, err := db.Query(m.Table().Database.GetTableNamesQuery())
	if err != nil {
		return err
	}

	defer rows.Close()

	// for each schema
	for rows.Next() {
		var schemaName string
		rows.Scan(&schemaName)

		// couldn't get prepared statements working and gave up because it was very simple
		var statement strings.Builder
		statement.WriteString("select * from ")
		statement.WriteString(schemaName)

		if c != nil {
			c.Close()
			c = nil
		}
		c, err = db.Query(statement.String())
		if err != nil {
			panic(err)
		}

		PopulateDataForResult(m, c, &indexMap, schemaName)
	}

	// set the first table to be initial view
	m.UI.CurrentTable = 1

	return nil
}

func PopulateDataForResult(m *TuiModel, c *sql.Rows, indexMap *int, schemaName string) {
	columnNames, _ := c.Columns()
	columnValues := make(map[string][]interface{})

	for c.Next() { // each row of the table
		// golang wizardry
		columns := make([]interface{}, len(columnNames))
		columnPointers := make([]interface{}, len(columnNames))
		// init interface array
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		c.Scan(columnPointers...)

		for i, colName := range columnNames {
			val := columnPointers[i].(*interface{})
			columnValues[colName] = append(columnValues[colName], *val)
		}
	}

	// onto the next schema
	*indexMap++
	if m.QueryResult != nil && m.QueryData != nil {
		m.QueryResult.Data[schemaName] = columnValues
		m.QueryData.TableHeaders[schemaName] = columnNames // headers for the schema, for later reference
		m.QueryData.TableIndexMap[*indexMap] = schemaName
		return
	}
	m.Table().Data[schemaName] = columnValues       // data for schema, organized by column
	m.Data().TableHeaders[schemaName] = columnNames // headers for the schema, for later reference
	// mapping between schema and an int ( since maps aren't deterministic), for later reference
	m.Data().TableIndexMap[*indexMap] = schemaName
}

func SwapTableValues(m *TuiModel, f, t *TableState) {
	from := &f.Data
	to := &t.Data
	for k, v := range *from {
		if copyValues, ok := v.(map[string][]interface{}); ok {
			columnNames := m.Data().TableHeaders[k]
			columnValues := make(map[string][]interface{})
			// golang wizardry
			columns := make([]interface{}, len(columnNames))

			for i := range columns {
				columns[i] = copyValues[columnNames[i]][0]
			}

			for i, colName := range columnNames {
				columnValues[colName] = columns[i].([]interface{})
			}

			(*to)[k] = columnValues // data for schema, organized by column
		}
	}
}
