package viewer

import (
	"database/sql"
	"sync"
)

var (
	dbMutex sync.Mutex
	dbs     map[string]*sql.DB
	DriverString string
)

type Database interface {
	Update(q *Update)
	GenerateQuery(u *Update) (string, []string)
	GetPlaceholderForDatabaseType() string
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
	switch conv := q.(type) {
	case *Update:
		conv.v = m.GetRowData()
		conv.TableName = m.GetSchemaName()
		conv.Column = m.GetSelectedColumnName()
		m.Table.Database.Update(conv)
		break
	}
}


// GetDatabaseForFile does what you think it does
func GetDatabaseForFile(database string) *sql.DB {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	if db, ok := dbs[database]; ok {
		return db
	}
	db, err := sql.Open(DriverString, database)
	if err != nil {
		panic(err)
	}
	dbs[database] = db
	return db
}