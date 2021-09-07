package viewer

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
)

func (m *TuiModel) Serialize() string {
	m.Table.Database.CloseDatabaseReference()
	source, err := os.ReadFile(m.Table.Database.GetFileName())
	if err != nil {
		panic(err)
	}
	ext := path.Ext(m.InitialFileName)
	newFileName := fmt.Sprintf("%s-%d%s", strings.TrimSuffix(m.InitialFileName, ext), rand.Intn(4), ext)
	err = os.WriteFile(newFileName, source, 0777)
	if err != nil {
		log.Fatal(err)
	}
	m.Table.Database.SetDatabaseReference(m.Table.Database.GetFileName())

	return newFileName
}

func (m *TuiModel) SerializeOverwrite() {
	m.Table.Database.CloseDatabaseReference()
	source, err := os.ReadFile(m.Table.Database.GetFileName())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(m.InitialFileName, source, 0777)
	if err != nil {
		log.Fatal(err)
	}
	m.Table.Database.SetDatabaseReference(m.Table.Database.GetFileName())
}
