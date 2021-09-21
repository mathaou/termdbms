package viewer

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
)

func (m *TuiModel) CopyMap() (to map[string]interface{}) {
	from := m.Table.Data
	to = map[string]interface{}{}

	for k, v := range from {
		if copyValues, ok := v.(map[string][]interface{}); ok {
			columnNames := m.Data.TableHeaders[k]
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
		Table: TableState{
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
		Data: UIData{
			TableHeaders:      make(map[string][]string),
			TableHeadersSlice: []string{},
			TableSlices:       make(map[string][]interface{}),
			TableIndexMap:     make(map[int]string),
		},
		TextInput: LineEdit{
			Model:         tuiutil.NewModel(),
		},
		FormatInput: LineEdit{
			Model:         tuiutil.NewModel(),
		},
	}
	m.FormatInput.Model.Prompt = ""
	return m
}

// SetModel creates a model to be used by bubbletea using some golang wizardry
func SetModel(m *TuiModel, c *sql.Rows, db *sql.DB, query string) {
	var err error

	indexMap := 0

	// gets all the schema names of the database
	rows, err := db.Query(query)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
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
		indexMap++
		m.Table.Data[schemaName] = columnValues       // data for schema, organized by column
		m.Data.TableHeaders[schemaName] = columnNames // headers for the schema, for later reference
		// mapping between schema and an int ( since maps aren't deterministic), for later reference
		m.Data.TableIndexMap[indexMap] = schemaName
	}

	// set the first table to be initial view
	m.UI.CurrentTable = 1
}


func SwapTableValues(m *TuiModel, f, t *TableState) {
	from := &f.Data
	to := &t.Data
	for k, v := range *from {
		if copyValues, ok := v.(map[string][]interface{}); ok {
			columnNames := m.Data.TableHeaders[k]
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
