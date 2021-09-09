package viewer

import (
	"fmt"
	"os"
)

// handleEditMode implementation is kind of jank, but we can clean it up later
func handleEditMode(m *TuiModel, str, first, last, input, val string) {
	if str == "esc" {
		m.textInput.SetValue("")
		return
	}

	for _, v := range inputBlacklist {
		if str == v {
			return
		}
	}

	/*
	if str == "left" {
			cursorPosition := m.textInput.Cursor()

			if cursorPosition == m.textInput.offset {
				m.textInput.offset--
				m.textInput.offsetRight--
			}

			m.textInput.SetCursor(cursorPosition - 1)
		} else if str == "right" {
			cursorPosition := m.textInput.Cursor()

			if cursorPosition == m.textInput.offsetRight {
				m.textInput.offset++
				m.textInput.offsetRight++
			}

			m.textInput.setCursor(cursorPosition + 1)
		} else
	*/
	if str == "backspace" {
		cursor := m.textInput.Cursor()
		if cursor == len(input) && len(input) > 0 {
			m.textInput.SetValue(input[0:len(input) - 1])
		} else if cursor > 0 {
			min := Max(m.textInput.Cursor(), 0)
			min = Min(min, len(input) - 1)
			first = input[:min - 1]
			last = input[min:]
			m.textInput.SetValue(first + last)
			m.textInput.SetCursor(m.textInput.Cursor() - 1)
		}
	} else if str == "enter" { // writes your selection
		if input == ":q" { // quit mod mode
			m.editModeEnabled = false
			m.textInput.SetValue("")
			return
		} else if input == ":s" { // saves copy, default filename + :s _____ will save with that filename in cwd
			m.editModeEnabled = false
			m.textInput.SetValue("")
			newFileName := m.Serialize()
			m.DisplayMessage(fmt.Sprintf("Wrote copy of database to filepath %s", newFileName))
			return
		} else if input == ":s!" { // overwrites original
			m.editModeEnabled = false
			m.textInput.SetValue("")
			m.SerializeOverwrite()
			return
		} else if input == ":h" {
			m.DisplayMessage(GetHelpText())
			return
		}

		raw, _, _ := m.GetSelectedOption()
		if *raw == input {
			// no update
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
		original, _, _ := m.GetSelectedOption()
		m.ProcessSqlQueryForDatabaseType(&Update{
			Update: GetInterfaceFromString(input, original),
		})

		m.editModeEnabled = false
		m.textInput.SetValue("")
		*raw = input
	} else {
		if val != "" {
			m.textInput.SetValue(val)
		} else {
			m.textInput.SetValue(str)
		}
		m.textInput.SetCursor(len(m.textInput.Value()))
	}

	return
}