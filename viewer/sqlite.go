package viewer

import "database/sql"

type SQLite struct {
	FileName          string
	DatabaseReference *sql.DB
}

func (s *SQLite) Update(q *Update) {
	query := q.GenerateQuery()
	tx, _ := s.DatabaseReference.Begin()
	//if err != nil {
	//	panic(err)
	//}
	//defer tx.Rollback()
	result, _ := tx.Exec(query)
	println(result.RowsAffected())
}

func (s *SQLite) GetFileName() string {
	return s.FileName
}

func (s *SQLite) GetDatabaseReference() *sql.DB {
	return s.DatabaseReference
}

func (s *SQLite) CloseDatabaseReference() {
	s.DatabaseReference.Close()
	s.DatabaseReference = nil
}

func (s *SQLite) SetDatabaseReference(db *sql.DB) {
	s.DatabaseReference = db
}
