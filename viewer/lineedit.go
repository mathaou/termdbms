package viewer

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
)

type EnterFunction func(m *TuiModel, selectedInput *tuiutil.TextInputModel, input string)

type LineEdit struct {
	Model    tuiutil.TextInputModel
	Original *interface{}
}

func ExitToDefaultView(m *TuiModel) {
	m.UI.EditModeEnabled = false
	m.UI.FormatModeEnabled = false
	m.UI.SQLEdit = false
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

func CreateEmptyBuffer(m *TuiModel, original *interface{}) {
	PrepareFormatMode(m)
	m.Data().EditTextBuffer = "\n"
	m.FormatInput.Original = original
	m.Format.Text = GetFormattedTextBuffer(m)
	m.SetViewSlices()
	m.FormatInput.Model.SetCursor(0)
	return
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
	if !m.UI.FormatModeEnabled && !m.UI.SQLEdit {
		input = i
		raw, _, _ := m.GetSelectedOption()
		original = raw
		if input == ":d" && m.QueryData != nil {
			m.DefaultTable.Database.SetDatabaseReference(m.QueryResult.Database.GetFileName())
			m.QueryData = nil
			m.QueryResult = nil
			var c *sql.Rows
			defer func() {
				if c != nil {
					c.Close()
				}
			}()
			err := SetModel(m, c, m.DefaultTable.Database.GetDatabaseReference())
			if err != nil {
				m.DisplayMessage(fmt.Sprintf("%v", err))
			}
			ExitToDefaultView(m)
			return
		}
		if m.QueryData != nil {
			m.TextInput.Model.SetValue("Cannot manipulate database through UI while query results are being displayed.")
			return
		}
		if input == ":h" {
			m.UI.HelpDisplay = true
			m.DisplayMessage(GetHelpText())
			return
		} else if input == ":edit" {
			str := GetStringRepresentationOfInterface(*original)
			PrepareFormatMode(m)
			if conv, err := FormatJson(str); err == nil { // if json prettify
				m.Data().EditTextBuffer = conv
			} else {
				m.Data().EditTextBuffer = str
			}
			m.FormatInput.Original = original
			m.Format.Text = GetFormattedTextBuffer(m)
			m.SetViewSlices()
			m.FormatInput.Model.SetCursor(0)
			return
		} else if input == ":new" {
			CreateEmptyBuffer(m, original)
			return
		} else if input == ":sql" {
			CreateEmptyBuffer(m, original)
			m.UI.SQLEdit = true
			return
		}
	} else {
		input = m.Data().EditTextBuffer
		original = m.FormatInput.Original
		if (m.UI.FormatModeEnabled &&
			!(i == ":w" || i == ":wq" || i == ":s" || i == ":s!")) &&
			(m.UI.SQLEdit && !(i == ":exec")) {
			m.TextInput.Model.SetValue("")
			return
		}
	}

	if *original == input || "" == strings.TrimSpace(input) {
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

	if m.UI.SQLEdit { // if it gets here an its SQLEdit, then :exec was the command
		if m.QueryResult != nil {
			m.QueryResult = nil
		}
		m.QueryResult = &TableState{ // perform query
			Database: m.Table().Database,
			Data:     make(map[string]interface{}),
		}
		m.QueryData = &UIData{}

		firstword := strings.ToLower(strings.Split(input, " ")[0])
		// TODO finish exec vs query
		if exec := firstword == "input" ||
			firstword == "update" ||
			firstword == "delete"; exec {
			_, err := m.QueryResult.Database.GetDatabaseReference().Exec(input)

			if err != nil {
				m.QueryResult = nil
				m.QueryData = nil
				ExitToDefaultView(m)
				m.DisplayMessage(fmt.Sprintf("%v", err))
				return
			}

			// reset initial model, carry on to undo (undo might have to happen before this)
		} else { // query

			c, err := m.QueryResult.Database.GetDatabaseReference().Query(input)
			defer func() {
				if c != nil {
					c.Close()
				}
			}()
			if err != nil {
				m.QueryResult = nil
				m.QueryData = nil
				ExitToDefaultView(m)
				m.DisplayMessage(fmt.Sprintf("%v", err))
				return
			}

			i := 0

			m.QueryData.TableHeaders = make(map[string][]string)
			m.QueryData.TableIndexMap = make(map[int]string)
			m.QueryData.TableSlices = make(map[string][]interface{})
			m.QueryData.TableHeadersSlice = []string{}

			PopulateDataForResult(m, c, &i, "results")
			ExitToDefaultView(m)
			m.UI.EditModeEnabled = false
			m.UI.CurrentTable = 1
			m.Data().EditTextBuffer = ""
			m.FormatInput.Model.SetValue("")
		}
		return
	}

	if len(m.UndoStack) >= 10 {
		ref := m.UndoStack[len(m.UndoStack)-1]
		err := os.Remove(ref.Database.GetFileName())
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
		m.UndoStack = m.UndoStack[1:] // need some more complicated logic to handle dereferencing
	}

	switch m.Table().Database.(type) {
	case *database.SQLite:
		deepCopy := m.CopyMap()
		// THE GLOBALIST TAKEOVER
		deepState := TableState{
			Database: &database.SQLite{
				FileName: m.Table().Database.GetFileName(),
				Database: nil,
			},
			Data: deepCopy,
		}
		m.UndoStack = append(m.UndoStack, deepState)
		dst, _, _ := CopyFile(m.Table().Database.GetFileName())
		m.Table().Database.CloseDatabaseReference()
		m.Table().Database.SetDatabaseReference(dst)
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
	}, m.GetRowData(), m.GetSchemaName(), m.GetSelectedColumnName(), &m.Table().Database)

	m.UI.EditModeEnabled = false
	m.Data().EditTextBuffer = ""
	m.FormatInput.Model.SetValue("")

	*original = input

	if m.UI.FormatModeEnabled && i == ":wq" {
		ExitToDefaultView(m)
	}
}
