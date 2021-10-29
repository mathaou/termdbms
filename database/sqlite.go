package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type SQLite struct {
	FileName string
	Database *sql.DB
}

func (db *SQLite) Update(q *Update) {
	protoQuery, columnOrder := db.GenerateQuery(q)
	values := make([]interface{}, len(columnOrder))
	updateValues := q.GetValues()
	for i, v := range columnOrder {
		var u interface{}
		if i == 0 {
			u = q.Update
		} else {
			u = updateValues[v]
		}

		if u == nil {
			u = "NULL"
		}

		values[i] = u
	}
	tx, err := db.GetDatabaseReference().Begin()
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

func (db *SQLite) GetFileName() string {
	return db.FileName
}

func (db *SQLite) GetDatabaseReference() *sql.DB {
	return db.Database
}

func (db *SQLite) CloseDatabaseReference() {
	db.GetDatabaseReference().Close()
	db.Database = nil
}

func (db *SQLite) SetDatabaseReference(dbPath string) {
	database := GetDatabaseForFile(dbPath)
	db.FileName = dbPath
	db.Database = database
}

func (db SQLite) GetPlaceholderForDatabaseType() string {
	return "?"
}

func (db SQLite) GetTableNamesQuery() string {
	val := "SELECT name FROM "
	val += "sqlite_master"
	val += " WHERE type='table'"

	return val
}

func (db *SQLite) GenerateQuery(u *Update) (string, []string) {
	var (
		query         string
		querySkeleton string
		valueOrder    []string
	)

	placeholder := db.GetPlaceholderForDatabaseType()

	querySkeleton = fmt.Sprintf("UPDATE %s"+
		" SET %s=%s ", u.TableName, u.Column, placeholder)
	valueOrder = append(valueOrder, u.Column)

	whereBuilder := strings.Builder{}
	whereBuilder.WriteString(" WHERE ")
	uLen := len(u.GetValues())
	i := 0
	for k := range u.GetValues() { // keep track of order since maps aren't deterministic
		assertion := fmt.Sprintf("%s=%s ", k, placeholder)
		valueOrder = append(valueOrder, k)
		whereBuilder.WriteString(assertion)
		if uLen > 1 && i < uLen-1 {
			whereBuilder.WriteString("AND ")
		}
		i++
	}
	query = querySkeleton + strings.TrimSpace(whereBuilder.String()) + ";"
	return query, valueOrder
}
