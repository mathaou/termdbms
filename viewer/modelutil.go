package viewer

import (
	"database/sql"
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
			EnterBehavior: HeaderLineEditEnterBehavior,
		},
		FormatInput: LineEdit{
			Model:         tuiutil.NewModel(),
			EnterBehavior: BodyLineEditEnterBehavior,
		},
	}
	m.FormatInput.Model.Prompt = ""
	return m
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
