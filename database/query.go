package database

import (
	"database/sql"
	"sync"
)

var (
	DBMutex      sync.Mutex
	Databases    map[string]*sql.DB
	DriverString string
)

func init() {
	// We keep one connection pool per database.
	DBMutex = sync.Mutex{}
	Databases = make(map[string]*sql.DB)
}

type Query interface {
	GetValues() map[string]interface{}
	SetValues(map[string]interface{})
}

type Update struct {
	v    map[string]interface{} // these are anchors to ensure the right row/col gets updated
	Column    string                 // this is the header
	Update    interface{}            // this is the new cell value
	TableName string
}

func (u *Update) GetValues() map[string]interface{} {
	return u.v
}

func (u *Update) SetValues(v map[string]interface{}) {
	u.v = v
}

// GetDatabaseForFile does what you think it does
func GetDatabaseForFile(database string) *sql.DB {
	DBMutex.Lock()
	defer DBMutex.Unlock()
	if db, ok := Databases[database]; ok {
		return db
	}
	db, err := sql.Open(DriverString, database)
	if err != nil {
		panic(err)
	}
	Databases[database] = db
	return db
}