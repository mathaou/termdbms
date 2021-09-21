package viewer

import (
	"fmt"
	"os"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
)

type EnterFunction func(m *TuiModel, selectedInput *tuiutil.TextInputModel, input string)

type LineEdit struct {
	Model         tuiutil.TextInputModel
	Original      *interface{}
}

func ExitToDefaultView(m *TuiModel) {
	m.UI.EditModeEnabled = false
	m.UI.FormatModeEnabled = false
	m.UI.HelpDisplay = false
	m.UI.CanFormatScroll = false
	m.Format.CursorY = 0
	m.Format.CursorX = 0
	m.Format.EditSlices = nil
	m.Format.Text = nil
	m.Format.RunningOffsets = nil
	m.FormatInput.Model.Reset()
	m.TextInput.Model.Reset()
	m.Viewport.YOffset = 0
}

func EditEnter(m *TuiModel) {
	selectedInput := &m.TextInput.Model
	i := selectedInput.Value()
	var (
		original *interface{}
		input    string
	)

	if i == ":q" { // quit mod mode
		ExitToDefaultView(m)
		return
	}
	if !m.UI.FormatModeEnabled {
		input = i
		raw, _, _ := m.GetSelectedOption()
		original = raw
		if input == ":h" {
			m.UI.HelpDisplay = true
			m.DisplayMessage(GetHelpText())
			return
		} else if input == ":edit" {
			str := GetStringRepresentationOfInterface(*original)
			PrepareFormatMode(m)
			if conv, err := FormatJson(str); err == nil { // if json prettify
				m.Data.EditTextBuffer = conv
			} else {
				m.Data.EditTextBuffer = str
			}
			m.FormatInput.Original = original
			m.Format.Text = GetFormattedTextBuffer(m)
			m.SetViewSlices()
			m.FormatInput.Model.SetCursor(0)
			return
		} else if input == ":new" {
			PrepareFormatMode(m)
			m.Data.EditTextBuffer = "\n"
			m.FormatInput.Original = original
			m.Format.Text = GetFormattedTextBuffer(m)
			m.SetViewSlices()
			m.FormatInput.Model.SetCursor(0)
			return
		}
	} else {
		input = m.Data.EditTextBuffer
		original = m.FormatInput.Original
		if !(i == ":w" || i == ":wq" || i == ":s" || i == ":s!") {
			m.TextInput.Model.SetValue("")
			return
		}
	}

	if *original == input {
		ExitToDefaultView(m)
		return
	}

	if i == ":s" { // saves copy, default filename + :s _____ will save with that filename in cwd
		ExitToDefaultView(m)
		newFileName, err := Serialize(m)
		if err != nil {
			m.DisplayMessage(fmt.Sprintf("%v", err))
		} else {
			m.DisplayMessage(fmt.Sprintf("Wrote copy of database to filepath %s.", newFileName))
		}

		return
	} else if i == ":s!" { // overwrites original - should add confirmation dialog!
		ExitToDefaultView(m)
		err := SerializeOverwrite(m)
		if err != nil {
			m.DisplayMessage(fmt.Sprintf("%v", err))
		} else {
			m.DisplayMessage("Overwrote original database file with changes.")
		}

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
	case *database.SQLite:
		deepCopy := m.CopyMap()
		// THE GLOBALIST TAKEOVER
		deepState := TableState{
			Database: &database.SQLite{
				FileName: m.Table.Database.GetFileName(),
				Database: nil,
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

	if _, err := FormatJson(input); err == nil { // if json uglify
		input = strings.ReplaceAll(input, " ", "")
		input = strings.ReplaceAll(input, "\n", "")
		input = strings.ReplaceAll(input, "\t", "")
		input = strings.ReplaceAll(input, "\r", "")
	}

	database.ProcessSqlQueryForDatabaseType(&database.Update{
		Update: GetInterfaceFromString(input, original),
	}, m.GetRowData(), m.GetSchemaName(), m.GetSelectedColumnName(), &m.Table.Database)

	m.UI.EditModeEnabled = false
	m.Data.EditTextBuffer = ""
	m.FormatInput.Model.SetValue("")

	*original = input

	if m.UI.FormatModeEnabled && i == ":wq" {
		ExitToDefaultView(m)
	}
}
