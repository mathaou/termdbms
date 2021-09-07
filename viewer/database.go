package viewer

import (
	"database/sql"
	"sync"
)

var (
	dbMutex sync.Mutex
	dbs     map[string]*sql.DB
)

type Query interface {
	GenerateQuery(db Database) (string, []string)
}

type Database interface {
	Update(q *Update)
	GetFileName() string
	GetDatabaseReference() *sql.DB
	CloseDatabaseReference()
	SetDatabaseReference(dbPath string)
}

func init() {
	// We keep one connection pool per database.
	dbMutex = sync.Mutex{}
	dbs = make(map[string]*sql.DB)
}

func (m *TuiModel) ProcessSqlQueryForDatabaseType(q Query) {
	switch q.(type) {
	case *Update:
		update, _ := q.(*Update)
		update.Values = m.GetRowData()
		update.TableName = m.GetSchemaName()
		update.Column = m.GetSelectedColumnName()
		m.Table.Database.Update(update)
		break
	}
}
