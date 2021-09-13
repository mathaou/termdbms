package viewer

import (
	"fmt"
	"os"
)

var (
	inputBlacklist = []string{
		"up",
		"down",
		"tab",
		"pgdown",
		"pgup",
	}
)

// handleEditMode implementation is kind of jank, but we can clean it up later
func handleEditMode(m *TuiModel, str, input, val string) {
	inputLen := len(input)
	if str == "esc" {
		m.textInput.SetValue("")
		return
	}

	for _, v := range inputBlacklist {
		if str == v {
			return
		}
	}

	if str == "home" {
		m.textInput.setCursor(0)
	} else if str == "end" {
		if len(val) > 0 {
			m.textInput.setCursor(len(val) - 1)
		}
	} else if str == "left" {
		cursorPosition := m.textInput.Cursor()

		if cursorPosition == m.textInput.offset && cursorPosition != 0 {
			m.textInput.offset--
			m.textInput.offsetRight--
		}

		if cursorPosition != 0 {
			m.textInput.SetCursor(cursorPosition - 1)
		}
	} else if str == "right" {
		cursorPosition := m.textInput.Cursor()

		if cursorPosition == m.textInput.offsetRight {
			m.textInput.offset++
			m.textInput.offsetRight++
		}

		m.textInput.setCursor(cursorPosition + 1)
	} else if str == "backspace" {
		cursor := m.textInput.Cursor()
		if cursor == inputLen && inputLen > 0 {
			m.textInput.SetValue(input[0 : inputLen-1])
		} else if cursor > 0 {
			min := Max(m.textInput.Cursor(), 0)
			min = Min(min, inputLen-1)
			first := input[:min-1]
			lastRune := first[len(first) - 1]
			if lastRune > 127 {
				first = first[0:len(first) - 1]
			}
			last := input[min:]
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
			newFileName, err := m.Serialize()
			if err != nil {
				m.DisplayMessage(fmt.Sprintf("%v", err))
			} else {
				m.DisplayMessage(fmt.Sprintf("Wrote copy of database to filepath %s.", newFileName))
			}
			return
		} else if input == ":s!" { // overwrites original - should add confirmation dialog!
			m.editModeEnabled = false
			m.textInput.SetValue("")
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
		}

		raw, _, _ := m.GetSelectedOption()
		if *raw == input {
			m.editModeEnabled = false
			m.textInput.SetValue("")
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
		m.textInput.SetValue("")

		*raw = input
	} else {
		prePos := m.textInput.Cursor()
		if val != "" {
			m.textInput.SetValue(val)
		} else {
			m.textInput.SetValue(str)
		}

		if prePos != 0 {
			prePos = m.textInput.Cursor()
		}
		m.textInput.setCursor(prePos + 1)
	}
}
