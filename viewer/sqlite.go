package viewer

import (
	"database/sql"
	"log"
)

type SQLite struct {
	FileName          string
	db *sql.DB
}

func (s *SQLite) Update(q *Update) {
	protoQuery, columnOrder := q.GenerateQuery(s)
	values := make([]interface{}, len(columnOrder))
	for i, v := range columnOrder {
		if i == 0 {
			values[i] = q.Update
		} else {
			values[i] = q.Values[v]
		}
	}
	tx, err := s.GetDatabaseReference().Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(protoQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	stmt.Exec(values...)
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func (s *SQLite) GetFileName() string {
	return s.FileName
}

func (s *SQLite) GetDatabaseReference() *sql.DB {
	return s.db
}

func (s *SQLite) CloseDatabaseReference() {
	s.GetDatabaseReference().Close()
	s.db = nil
}

func (s *SQLite) SetDatabaseReference(dbPath string) {
	db := GetDatabaseForFile(dbPath)
	s.FileName = dbPath
	s.db = db
}
