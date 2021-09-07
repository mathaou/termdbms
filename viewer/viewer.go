package viewer

import (
	"database/sql"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"math"
	"os"
	"runtime"
	"strings"
)

var (
	width          int
	height         int
	headerHeight   = 3
	footerHeight   = 1
	newline        string
	maxInputLength int
	headerStyle    lipgloss.Style
)

const (
	highlight                 = "#0168B3" // change to whatever
	headerForeground          = "#231F20"
	headerBorderBackground    = "#AAAAAA"
	maximumRendererCharacters = math.MaxInt32
)

type TableState struct {
	Database Database
	Data     map[string]interface{}
}

// TuiModel holds all the necessary state for this app to work the way I designed it to
type TuiModel struct {
	Table              TableState          // all non destructive changes are TableStates getting passed around
	TableHeaders       map[string][]string // keeps track of which schema has which headers
	TableHeadersSlice  []string
	DataSlices         map[string][]interface{}
	TableIndexMap      map[int]string // keeps the schemas in order
	TableSelection     int
	InitialFileName    string // used if saving destructively
	ready              bool
	renderSelection    bool
	helpDisplay        bool
	editModeEnabled    bool
	selectionText      string
	preScrollYOffset   int
	preScrollYPosition int
	scrollXOffset      int
	borderToggle       bool
	expandColumn       int
	viewport           ViewportModel
	tableStyle         lipgloss.Style
	mouseEvent         tea.MouseEvent
	textInput          textinput.Model
	UndoStack          []TableState
	RedoStack          []TableState
	err                error
}

// INIT UPDATE AND RENDER

// Init currently doesn't do anything but necessary for interface adherence
func (m TuiModel) Init() tea.Cmd {
	newline = "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}

	maxInputLength = m.viewport.Width
	m.textInput.CharLimit = -1
	m.textInput.Width = maxInputLength

	headerStyle = lipgloss.NewStyle().
		Faint(true)

	return nil
}

// Update is where all commands and whatnot get processed
func (m TuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := message.(type) {
	case tea.MouseMsg:
		handleMouseEvents(&m, &msg)
		m.SetViewSlices()
		break
	case tea.WindowSizeMsg:
		event := handleWidowSizeEvents(&m, &msg)
		cmds = append(cmds, event)
		break
	case tea.KeyMsg:
		// when fullscreen selection viewing is in session, don't allow UI manipulation other than quit or exit
		s := msg.String()
		if m.renderSelection &&
			s != "esc" &&
			s != "ctrl+c" &&
			s != "q" &&
			s != "p" &&
			s != "m" &&
			s != "n" {
			break
		}
		if s == "ctrl+c" || (s == "q" && !m.editModeEnabled) {
			return m, tea.Quit
		}

		handleKeyboardEvents(&m, &msg)
		if !m.editModeEnabled {
			m.SetViewSlices()
		}

		break
	case error:
		m.err = msg
		return m, nil
	}

	m.viewport, _ = m.viewport.Update(message)

	if m.viewport.HighPerformanceRendering {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View is where all rendering happens
func (m TuiModel) View() string {
	if !m.ready || m.viewport.Width == 0 {
		return "\n  Initializing..."
	}

	// this ensures that all 3 parts can be worked on concurrently(ish)
	done := make(chan bool, 3)

	var footer, header, content string

	// body
	go func(c *string) {
		*c = assembleTable(&m)
		done <- true
	}(&content)

	// header
	go func(h *string) {
		var (
			builder []string
		)

		style := m.GetBaseStyle().
			Width(m.CellWidth()).
			Foreground(lipgloss.Color(headerForeground)).
			Background(lipgloss.Color(headerBorderBackground))
		headers := m.TableHeadersSlice
		for i, d := range headers { // write all headers
			if m.expandColumn != -1 && i != m.expandColumn {
				continue
			}

			text := TruncateIfApplicable(&m, d)
			builder = append(builder, style.
				Render(text))
		}

		{
			// schema name
			var headerTop string

			if m.editModeEnabled {
				var (
					min int
					max int
				)

				view := m.textInput.View()
				viewLen := len(view)
				outOfRange := m.viewport.Width < viewLen

				if outOfRange {
					min = Abs(m.viewport.Width - viewLen)
					max = m.viewport.Width + min
				} else {
					min = 0
					max = viewLen
				}

				headerTop = view[min:max]
				headerTop += strings.Repeat(" ", m.viewport.Width-len(headerTop))
			} else {
				headerTop = fmt.Sprintf("%s (%d/%d) - %d record(s) + %d column(s)",
					m.GetSchemaName(),
					m.TableSelection,
					len(m.TableHeaders),                // look at how headers get rendered to get accurate record number
					len(m.GetColumnData()),
					len(m.GetHeaders())) // this will need to be refactored when filters get added

				headerTop += strings.Repeat(" ", m.viewport.Width-len(headerTop))
			}

			//strings.Repeat(" ", m.viewport.Width-len(headerTop))

			// separator
			headerBot := strings.Repeat(lipgloss.NewStyle().
				Align(lipgloss.Center).
				Faint(true).
				Render("-"),
				m.viewport.Width)
			headerMid := strings.Join(builder, "")
			headerMid = headerMid + strings.Repeat(" ", m.viewport.Width)
			*h = fmt.Sprintf("%s\n%s\n%s",
				headerStyle.Render(headerTop),
				headerMid,
				headerBot)
		}

		done <- true
	}(&header)

	// footer (shows row/col for now)
	go func(f *string) {
		{
			footer := fmt.Sprintf(" %d, %d ", m.GetRow()+m.viewport.YOffset, m.GetColumn()+m.scrollXOffset)
			undoRedoInfo := fmt.Sprintf("undo(%d) / redo(%d) ", len(m.UndoStack), len(m.RedoStack))
			gapSize := m.viewport.Width - lipgloss.Width(footer) - lipgloss.Width(undoRedoInfo) - 2
			footer = headerStyle.Render(undoRedoInfo) + "├" + strings.Repeat("─", gapSize) + "┤" + headerStyle.Render(footer)
			*f = footer
		}

		done <- true
	}(&footer)

	// block until all 3 done
	<-done
	<-done
	<-done

	close(done) // close

	if m.helpDisplay {
		return content
	}

	if content == "" { // race condition TODO
		m.SetViewSlices()
		return m.View()
	} else {
		return fmt.Sprintf("%s\n%s\n%s", header, content, footer) // render
	}
}

// SetModel creates a model to be used by bubbletea using some golang wizardry
func (m *TuiModel) SetModel(c *sql.Rows, db *sql.DB) {
	var err error
	indexMap := 0

	// gets all the schema names of the database
	rows, err := db.Query(getTableNamesQuery)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	defer rows.Close()

	// for each schema
	for rows.Next() {
		var schemaName string
		rows.Scan(&schemaName)

		// couldn't get prepared statements working and gave up because it was very simple
		var statement strings.Builder
		statement.WriteString("select * from ")
		statement.WriteString(schemaName)

		if c != nil {
			c.Close()
			c = nil
		}
		c, err = db.Query(statement.String())
		if err != nil {
			panic(err)
		}

		columnNames, _ := c.Columns()
		columnValues := make(map[string][]interface{})

		for c.Next() { // each row of the table
			// golang wizardry
			columns := make([]interface{}, len(columnNames))
			columnPointers := make([]interface{}, len(columnNames))
			// init interface array
			for i, _ := range columns {
				columnPointers[i] = &columns[i]
			}

			c.Scan(columnPointers...)

			for i, colName := range columnNames {
				val := columnPointers[i].(*interface{})
				columnValues[colName] = append(columnValues[colName], *val)
			}
		}

		// onto the next schema
		indexMap++
		m.Table.Data[schemaName] = columnValues  // data for schema, organized by column
		m.TableHeaders[schemaName] = columnNames // headers for the schema, for later reference
		// mapping between schema and an int ( since maps aren't deterministic), for later reference
		m.TableIndexMap[indexMap] = schemaName
	}

	// set the first table to be initial view
	m.TableSelection = 1
}
