package viewer

import (
	"database/sql"
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"math"
	"os"
	"strings"
)

var (
	headerHeight      = 3
	footerHeight      = 1
	maxInputLength    int
	headerStyle       lipgloss.Style
	footerStyle       lipgloss.Style
	headerBottomStyle lipgloss.Style
	InitialModel      *TuiModel
)

const (
	maximumRendererCharacters = math.MaxInt32
)

var (
	highlight = func() string {
		return ThemesMap[SelectedTheme][highlightKey]
	} // change to whatever
	headerBackground = func() string {
		return ThemesMap[SelectedTheme][headerBackgroundKey]
	}
	headerBorderBackground = func() string {
		return ThemesMap[SelectedTheme][headerBorderBackgroundKey]
	}
	headerForeground = func() string {
		return ThemesMap[SelectedTheme][headerForegroundKey]
	}
	footerForegroundColor = func() string {
		return ThemesMap[SelectedTheme][footerForegroundColorKey]
	}
	headerBottomColor = func() string {
		return ThemesMap[SelectedTheme][headerBottomColorKey]
	}
	headerTopForegroundColor = func() string {
		return ThemesMap[SelectedTheme][headerTopForegroundColorKey]
	}
	borderColor = func() string {
		return ThemesMap[SelectedTheme][borderColorKey]
	}
	textColor = func() string {
		return ThemesMap[SelectedTheme][textColorKey]
	}
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
	renderSelection    bool // render mode
	helpDisplay        bool // help display mode
	editModeEnabled    bool // edit mode
	formatModeEnabled  bool
	selectionText      string
	preScrollYOffset   int
	preScrollYPosition int
	scrollXOffset      int
	borderToggle       bool
	expandColumn       int
	viewport           viewport.Model
	tableStyle         lipgloss.Style
	mouseEvent         tea.MouseEvent
	textInput          TextInputModel
	formatInput        TextInputModel
	UndoStack          []TableState
	RedoStack          []TableState
	err                error
}

func setStyles() {
	headerStyle = lipgloss.NewStyle()
	footerStyle = lipgloss.NewStyle()

	headerBottomStyle = lipgloss.NewStyle().
		Align(lipgloss.Center)

	if !Ascii {
		headerStyle = headerStyle.
			Foreground(lipgloss.Color(headerTopForegroundColor()))

		footerStyle = footerStyle.
			Foreground(lipgloss.Color(footerForegroundColor()))

		headerBottomStyle = headerBottomStyle.
			Foreground(lipgloss.Color(headerBottomColor()))
	}
}

// INIT UPDATE AND RENDER

// Init currently doesn't do anything but necessary for interface adherence
func (m TuiModel) Init() tea.Cmd {
	setStyles()

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
		if !m.editModeEnabled && m.ready {
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

		style := m.GetBaseStyle()

		if !Ascii {
			// for column headers
			style = style.Foreground(lipgloss.Color(headerForeground())).
				BorderBackground(lipgloss.Color(headerBorderBackground())).
				Background(lipgloss.Color(headerBackground())).
				PaddingLeft(1)
		}
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

			if m.editModeEnabled || m.formatModeEnabled {
				headerTop = m.textInput.View()
			} else {
				headerTop = fmt.Sprintf("%s (%d/%d) - %d record(s) + %d column(s)",
					m.GetSchemaName(),
					m.TableSelection,
					len(m.TableHeaders), // look at how headers get rendered to get accurate record number
					len(m.GetColumnData()),
					len(m.GetHeaders())) // this will need to be refactored when filters get added
				headerTop += strings.Repeat(" ", m.viewport.Width-len(headerTop))
			}

			// separator
			headerBot := strings.Repeat(headerBottomStyle.
				Render("¯"),
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
		footer := fmt.Sprintf(" %d, %d ", m.GetRow()+m.viewport.YOffset, m.GetColumn()+m.scrollXOffset)
		undoRedoInfo := fmt.Sprintf("undo(%d) / redo(%d) ", len(m.UndoStack), len(m.RedoStack))
		gapSize := m.viewport.Width - lipgloss.Width(footer) - lipgloss.Width(undoRedoInfo) - 2
		footer = footerStyle.Render(undoRedoInfo) + "├" + strings.Repeat("─", gapSize) + "┤" + footerStyle.Render(footer)
		*f = footer

		done <- true
	}(&footer)

	// block until all 3 done
	<-done
	<-done
	<-done

	close(done) // close

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer) // render
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
			for i := range columns {
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
	m.TableSelection = 3
}
