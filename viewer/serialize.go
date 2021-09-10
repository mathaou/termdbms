package viewer

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
)

var (
	serializationErrorString string
)

func init() {
	serializationErrorString = fmt.Sprintf("Database driver %s does not support serialization.", DriverString)
}

func (m *TuiModel) Serialize() (string, error) {
	switch m.Table.Database.(type) {
	case *SQLite:
		return SerializeSQLiteDB(m.Table.Database.(*SQLite), m), nil
	default:
		return "", errors.New(serializationErrorString)
	}
}

func (m *TuiModel) SerializeOverwrite() error {
	switch m.Table.Database.(type) {
	case *SQLite:
		SerializeOverwrite(m.Table.Database.(*SQLite), m)
		return nil
	default:
		return errors.New(serializationErrorString)
	}
}

// SQLITE

func SerializeSQLiteDB(db *SQLite, m *TuiModel) string {
	db.CloseDatabaseReference()
	source, err := os.ReadFile(db.GetFileName())
	if err != nil {
		panic(err)
	}
	ext := path.Ext(m.InitialFileName)
	newFileName := fmt.Sprintf("%s-%d%s", strings.TrimSuffix(m.InitialFileName, ext), rand.Intn(4), ext)
	err = os.WriteFile(newFileName, source, 0777)
	if err != nil {
		log.Fatal(err)
	}
	db.SetDatabaseReference(db.GetFileName())
	return newFileName
}

func SerializeOverwrite(db *SQLite, m *TuiModel) {
	db.CloseDatabaseReference()
	source, err := os.ReadFile(db.GetFileName())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(m.InitialFileName, source, 0777)
	if err != nil {
		log.Fatal(err)
	}
	db.SetDatabaseReference(db.GetFileName())
}