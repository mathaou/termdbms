package viewer

import (
	"fmt"
	"os"
)

type EnterFunction func(m *TuiModel, selectedInput *TextInputModel, input string)

type LineEdit struct {
	Model         TextInputModel
	EnterBehavior EnterFunction
}

func BodyLineEditEnterBehavior(m *TuiModel, selectedInput *TextInputModel, input string) {

}

func HeaderLineEditEnterBehavior(m *TuiModel, selectedInput *TextInputModel, input string) {
	raw, _, col := m.GetSelectedOption()
	if input == ":q" { // quit mod mode
		m.editModeEnabled = false
		m.formatModeEnabled = false
		m.helpDisplay = false
		selectedInput.SetValue("")
		return
	} else if input == ":s" { // saves copy, default filename + :s _____ will save with that filename in cwd
		m.editModeEnabled = false
		selectedInput.SetValue("")
		newFileName, err := m.Serialize()
		if err != nil {
			m.DisplayMessage(fmt.Sprintf("%v", err))
		} else {
			m.DisplayMessage(fmt.Sprintf("Wrote copy of database to filepath %s.", newFileName))
		}
		return
	} else if input == ":s!" { // overwrites original - should add confirmation dialog!
		m.editModeEnabled = false
		selectedInput.SetValue("")
		err := m.SerializeOverwrite()
		if err != nil {
			m.DisplayMessage(fmt.Sprintf("%v", err))
		} else {
			m.DisplayMessage("Overwrite original database file with changes.")
		}
		return
	} else if input == ":h" {
		m.helpDisplay = true
		m.DisplayMessage(GetHelpText())
		return
	} else if input == ":edit" {
		m.formatModeEnabled = true
		m.editModeEnabled = false
		if m.GetRow() >= len(col) {
			m.editModeEnabled = false
			return
		}

		m.selectionText = GetStringRepresentationOfInterface(*raw)
		m.formatInput.Model.focus = true
		m.textInput.Model.focus = false
		m.textInput.Model.SetValue("")
		return
	}

	if *raw == input {
		m.editModeEnabled = false
		selectedInput.SetValue("")
		return
	}

	// plain jane cell update
	if len(m.UndoStack) >= 10 {
		ref := m.UndoStack[len(m.UndoStack)-1]
		err := os.Remove(ref.Database.GetFileName())
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
		m.UndoStack = m.UndoStack[1:] // need some more complicated logic to handle dereferencing
	}

	switch m.Table.Database.(type) {
	case *SQLite:
		deepCopy := m.CopyMap()
		// THE GLOBALIST TAKEOVER
		deepState := TableState{
			Database: &SQLite{
				FileName: m.Table.Database.GetFileName(),
				db:       nil,
			},
			Data: deepCopy,
		}
		m.UndoStack = append(m.UndoStack, deepState)
		dst, _, _ := CopyFile(m.Table.Database.GetFileName())
		m.Table.Database.CloseDatabaseReference()
		m.Table.Database.SetDatabaseReference(dst)
		break
	default:
		break
	}

	original, _, _ := m.GetSelectedOption()
	m.ProcessSqlQueryForDatabaseType(&Update{
		Update: GetInterfaceFromString(input, original),
	})

	m.editModeEnabled = false
	selectedInput.SetValue("")

	*raw = input
}
