package viewer

import (
	"io"
	"os"
)

func (m *TuiModel) Serialize() {

}

func (m *TuiModel) SerializeOverwrite() {
	destination, err := os.Open(m.Table.Database.GetFileName())
	if err != nil {
		panic(err)
	}
	defer destination.Close()

	source, err := os.Open(m.InitialFileName)
	if err != nil {
		panic(err)
	}
	source.Close()

	io.Copy(destination, source)
}
