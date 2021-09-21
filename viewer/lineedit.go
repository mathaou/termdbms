package viewer

import (
	"fmt"
	"os"
	"strings"
)

type EnterFunction func(m *TuiModel, selectedInput *TextInputModel, input string)

type LineEdit struct {
	Model         TextInputModel
	EnterBehavior EnterFunction
	Original      *interface{}
}

func exitToDefaultView(m *TuiModel) {
	m.UI.EditModeEnabled = false
	m.UI.FormatModeEnabled = false
	m.UI.HelpDisplay = false
	m.UI.CanFormatScroll = false
	m.Format.CursorY = 0
	m.Format.CursorX = 0
	m.Format.Slices = nil
	m.Format.Text = nil
	m.Format.RunningOffsets = nil
	m.formatInput.Model.Reset()
	m.textInput.Model.Reset()
	m.viewport.YOffset = 0
}

func BodyLineEditEnterBehavior(m *TuiModel, selectedInput *TextInputModel, input string) {
	// UNUSED, newlines handled manually
}

func HeaderLineEditEnterBehavior(m *TuiModel, selectedInput *TextInputModel, i string) {
	var (
		original *interface{}
		input    string
	)

	if i == ":q" { // quit mod mode
		exitToDefaultView(m)
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
			prepareFormatMode(m)
			if conv, err := formatJson(str); err == nil { // if json prettify
				m.selectionText = conv
			} else {
				m.selectionText = str
			}
			m.formatInput.Original = original
			m.Format.Text = getFormattedTextBuffer(m)
			m.SetViewSlices()
			m.formatInput.Model.setCursor(0)
			return
		} else if input == ":new" {
			prepareFormatMode(m)
			m.selectionText = "\n"
			m.formatInput.Original = original
			m.Format.Text = getFormattedTextBuffer(m)
			m.SetViewSlices()
			m.formatInput.Model.setCursor(0)
			return
		}
	} else {
		input = m.selectionText
		original = m.formatInput.Original
		if !(i == ":w" || i == ":wq" || i == ":s" || i == ":s!") {
			m.textInput.Model.SetValue("")
			return
		}
	}

	if *original == input {
		exitToDefaultView(m)
		return
	}

	if i == ":s" { // saves copy, default filename + :s _____ will save with that filename in cwd
		exitToDefaultView(m)
		newFileName, err := m.Serialize()
		if err != nil {
			m.DisplayMessage(fmt.Sprintf("%v", err))
		} else {
			m.DisplayMessage(fmt.Sprintf("Wrote copy of database to filepath %s.", newFileName))
		}

		return
	} else if i == ":s!" { // overwrites original - should add confirmation dialog!
		exitToDefaultView(m)
		err := m.SerializeOverwrite()
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

	if _, err := formatJson(input); err == nil { // if json prettify
		input = strings.ReplaceAll(input, " ", "")
		input = strings.ReplaceAll(input, "\n", "")
		input = strings.ReplaceAll(input, "\t", "")
		input = strings.ReplaceAll(input, "\r", "")
	}

	m.ProcessSqlQueryForDatabaseType(&Update{
		Update: GetInterfaceFromString(input, original),
	})

	m.UI.EditModeEnabled = false
	m.selectionText = ""
	m.formatInput.Model.SetValue("")

	*original = input

	if m.UI.FormatModeEnabled && i == ":wq" {
		exitToDefaultView(m)
	}
}
