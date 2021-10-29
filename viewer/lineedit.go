package viewer

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"termdbms/database"
	"termdbms/tuiutil"
	"time"
)

const (
	QueryResultsTableName = "results"
)

type EnterFunction func(m *TuiModel, selectedInput *tuiutil.TextInputModel, input string)

type LineEdit struct {
	Model    tuiutil.TextInputModel
	Original *interface{}
}

func ExitToDefaultView(m *TuiModel) {
	m.UI.RenderSelection = false
	m.UI.EditModeEnabled = false
	m.UI.FormatModeEnabled = false
	m.UI.SQLEdit = false
	m.UI.ShowClipboard = false
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

func CreatePopulatedBuffer(m *TuiModel, original *interface{}, str string) {
	PrepareFormatMode(m)
	m.Data().EditTextBuffer = str
	m.FormatInput.Original = original
	m.Format.Text = GetFormattedTextBuffer(m)
	m.SetViewSlices()
	m.FormatInput.Model.SetCursor(0)
	return
}

func EditEnter(m *TuiModel) {
	selectedInput := &m.TextInput.Model
	i := selectedInput.Value()

	d := m.Data()
	t := m.Table()

	var (
		original *interface{}
		input    string
	)

	if i == ":q" { // quit mod mode
		ExitToDefaultView(m)
		return
	}
	if !m.UI.FormatModeEnabled && !m.UI.SQLEdit && !m.UI.ShowClipboard {
		input = i
		raw, _, _ := m.GetSelectedOption()
		original = raw
		if input == ":d" && m.QueryData != nil && m.QueryResult != nil {
			m.DefaultTable.Database.SetDatabaseReference(m.QueryResult.Database.GetFileName())
			m.QueryData = nil
			m.QueryResult = nil
			var c *sql.Rows
			defer func() {
				if c != nil {
					c.Close()
				}
			}()
			err := m.SetModel(c, m.DefaultTable.Database.GetDatabaseReference())
			if err != nil {
				m.DisplayMessage(fmt.Sprintf("%v", err))
			}
			ExitToDefaultView(m)
			return
		}
		if m.QueryData != nil {
			m.TextInput.Model.SetValue("")
			m.WriteMessage("Cannot manipulate database through UI while query results are being displayed.")
			return
		}
		if input == ":h" {
			m.DisplayMessage(GetHelpText())
			return
		} else if input == ":edit" {
			str := GetStringRepresentationOfInterface(*original)
			PrepareFormatMode(m)
			if conv, err := FormatJson(str); err == nil { // if json prettify
				d.EditTextBuffer = conv
			} else {
				d.EditTextBuffer = str
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
		} else if input == ":clip" {
			ExitToDefaultView(m)
			if len( m.ClipboardList.Items()) == 0 {
				return
			}
			m.UI.ShowClipboard = true
			return
		}
	} else {
		input = d.EditTextBuffer
		original = m.FormatInput.Original
		sqlFlags := m.UI.SQLEdit && !(i == ":exec" || strings.HasPrefix(i, ":stow"))
		formatFlags := m.UI.FormatModeEnabled && !(i == ":w" || i == ":wq" || i == ":s" || i == ":s!")
		if formatFlags && sqlFlags {
			m.TextInput.Model.SetValue("")
			return
		}
	}

	if original != nil && *original == input {
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

	if m.UI.SQLEdit {
		if i == ":exec" {
			handleSQLMode(m, input)
		} else if strings.HasPrefix(i, ":stow") {
			if len(input) > 0 {
				split := strings.Split(i, " ")
				rand.Seed(time.Now().UnixNano())
				r := rand.Int()
				title := fmt.Sprintf("%d", r) // if no title given then just call it random string
				if len(split) == 2 {
					title = split[1]
				}
				m.Clipboard = append(m.Clipboard, SQLSnippet{
					Query: input,
					Name: title,
				})
				b, _ := json.Marshal(m.Clipboard)
				snippetsFile := fmt.Sprintf("%s/%s", HiddenTmpDirectoryName, SQLSnippetsFile)
				f, _ := os.OpenFile(snippetsFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0775)
				f.Write(b)
				f.Close()
				m.WriteMessage(fmt.Sprintf("Wrote SQL snippet %s to %s. Total count is %d", title, snippetsFile, len(m.ClipboardList.Items()) + 1))
			}
			m.TextInput.Model.SetValue("")
		}
		return
	}

	old, n := populateUndo(m)
	if old == n || n != m.DefaultTable.Database.GetFileName() {
		panic(errors.New("could not get database file name"))
	}

	if _, err := FormatJson(input); err == nil { // if json uglify
		input = strings.ReplaceAll(input, " ", "")
		input = strings.ReplaceAll(input, "\n", "")
		input = strings.ReplaceAll(input, "\t", "")
		input = strings.ReplaceAll(input, "\r", "")
	}

	u := GetInterfaceFromString(input, original)
	database.ProcessSqlQueryForDatabaseType(&database.Update{
		Update: u,
	}, m.GetRowData(), m.GetSchemaName(), m.GetSelectedColumnName(), &t.Database)

	m.UI.EditModeEnabled = false
	d.EditTextBuffer = ""
	m.FormatInput.Model.SetValue("")

	*original = input

	if m.UI.FormatModeEnabled && i == ":wq" {
		ExitToDefaultView(m)
	}
}

func handleSQLMode(m *TuiModel, input string) {
	if m.QueryResult != nil {
		m.QueryResult = nil
	}
	m.QueryResult = &TableState{ // perform query
		Database: m.Table().Database,
		Data:     make(map[string]interface{}),
	}
	m.QueryData = &UIData{}

	firstword := strings.ToLower(strings.Split(input, " ")[0])
	if exec := firstword == "update" ||
		firstword == "delete" ||
		firstword == "insert"; exec {
		m.QueryData = nil
		m.QueryResult = nil
		populateUndo(m)
		_, err := m.DefaultTable.Database.GetDatabaseReference().Exec(input)
		if err != nil {
			ExitToDefaultView(m)
			m.DisplayMessage(fmt.Sprintf("%v", err))
			return
		}
		var c *sql.Rows
		defer func() {
			if c != nil {
				c.Close()
			}
		}()
		err = m.SetModel(c, m.DefaultTable.Database.GetDatabaseReference())
		if err != nil {
			m.DisplayMessage(fmt.Sprintf("%v", err))
		} else {
			ExitToDefaultView(m)
		}
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

		m.PopulateDataForResult(c, &i, QueryResultsTableName)
		ExitToDefaultView(m)
		m.UI.EditModeEnabled = false
		m.UI.CurrentTable = 1
		m.Data().EditTextBuffer = ""
		m.FormatInput.Model.SetValue("")
	}
}

func populateUndo(m *TuiModel) (old string, new string) {
	if len(m.UndoStack) >= 10 {
		ref := m.UndoStack[len(m.UndoStack)-1]
		err := os.Remove(ref.Database.GetFileName())
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
		m.UndoStack = m.UndoStack[1:] // need some more complicated logic to handle dereferencing?
	}

	switch m.DefaultTable.Database.(type) {
	case *database.SQLite:
		deepCopy := m.CopyMap()
		// THE GLOBALIST TAKEOVER
		deepState := TableState{
			Database: &database.SQLite{
				FileName: m.DefaultTable.Database.GetFileName(),
				Database: nil,
			},
			Data: deepCopy,
		}
		m.UndoStack = append(m.UndoStack, deepState)
		old = m.DefaultTable.Database.GetFileName()
		dst, _, _ := CopyFile(old)
		new = dst
		m.DefaultTable.Database.CloseDatabaseReference()
		m.DefaultTable.Database.SetDatabaseReference(dst)
		break
	default:
		break
	}

	return old, new
}
