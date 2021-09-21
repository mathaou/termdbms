package viewer

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"termdbms/database"
)

type Database interface {
	Update(q *database.Update)
	GenerateQuery(u *database.Update) (string, []string)
	GetPlaceholderForDatabaseType() string
	GetFileName() string
	GetDatabaseReference() *sql.DB
	CloseDatabaseReference()
	SetDatabaseReference(dbPath string)
}

func ProcessSqlQueryForDatabaseType(m *TuiModel, q database.Query) {
	switch conv := q.(type) {
	case *database.Update:
		conv.SetValues(m.GetRowData())
		conv.TableName = m.GetSchemaName()
		conv.Column = m.GetSelectedColumnName()
		m.Table.Database.Update(conv)
		break
	}
}

// SetModel creates a model to be used by bubbletea using some golang wizardry
func SetModel(m *TuiModel, c *sql.Rows, db *sql.DB) {
	var err error

	indexMap := 0

	// gets all the schema names of the database
	rows, err := db.Query(GetTableNamesQuery)
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
