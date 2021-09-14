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
	FormatSlices       []string
	DataSlices         map[string][]interface{}
	TableIndexMap      map[int]string // keeps the schemas in order
	TableSelection     int
	InitialFileName    string // used if saving destructively
	FormatText         []string
	CanFormatScroll    bool
	ready              bool
	renderSelection    bool // render mode
	helpDisplay        bool // help display mode
	editModeEnabled    bool // edit mode
	formatModeEnabled  bool
	selectionText      string
	preScrollYOffset   int
	preScrollYPosition int
	formatCursorX      int
	formatCursorY      int
	scrollXOffset      int
	borderToggle       bool
	expandColumn       int
	viewport           viewport.Model
	tableStyle         lipgloss.Style
	mouseEvent         tea.MouseEvent
	textInput          LineEdit
	formatInput        LineEdit
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
	//if !m.formatModeEnabled {
	//	m.formatModeEnabled = true
	//	m.editModeEnabled = false
	//	m.selectionText = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed commodo, elit at scelerisque consequat, lectus ex semper turpis, a posuere mauris neque a odio. Nam at placerat elit. Suspendisse potenti. Nullam lorem felis, fringilla vitae commodo at, vestibulum quis lacus. Phasellus iaculis elementum enim, eu lobortis lacus imperdiet at. Praesent a hendrerit nisl. Mauris faucibus, mi non posuere porta, turpis risus posuere dolor, at tempor dolor sapien vitae eros. Ut non efficitur enim, eu pretium tortor. In et laoreet magna. Etiam dignissim viverra convallis. Suspendisse ac nibh velit. Nulla facilisis vestibulum nibh vitae venenatis. Vivamus ornare, justo hendrerit blandit ultrices, metus diam aliquet urna, et sollicitudin ante odio quis ex. Duis non luctus augue, ac fringilla eros.\n\nNunc at dolor arcu. Nullam quis velit id purus bibendum tincidunt bibendum vel nunc. Maecenas imperdiet aliquam mauris a tincidunt. Praesent faucibus sapien nec massa posuere, ac placerat enim viverra. Quisque a condimentum velit, id feugiat lectus. Vivamus iaculis magna ante. Nulla interdum tristique justo, ac blandit dolor rutrum vel. Maecenas id tristique leo.\n\nProin lobortis finibus nibh, vitae porttitor tortor. Duis rutrum, eros ac fringilla scelerisque, nisi velit tristique odio, facilisis fermentum quam enim et risus. In tempus ipsum a erat posuere, quis varius ex hendrerit. Donec suscipit nec nulla sed dictum. Aenean venenatis augue quam. In nunc leo, fringilla vel justo et, sodales tempor libero. Cras sit amet nulla vel elit aliquet facilisis ac nec leo. Quisque interdum, enim et porttitor condimentum, nunc erat efficitur orci, id mattis diam arcu porttitor urna. Ut consectetur mi eu urna gravida lacinia a vel diam. Sed posuere, ante ac scelerisque sollicitudin, nisl dolor mollis nulla, quis commodo massa eros ut neque. Nam tempus dui et est congue blandit.\n\nIn quis posuere diam, at efficitur orci. Nullam vulputate, tortor sed fringilla pretium, massa massa congue justo, quis consectetur justo sapien non odio. Praesent pulvinar non magna a mattis. Cras efficitur mauris eu pretium eleifend. Vestibulum fringilla scelerisque neque ac blandit. Ut sagittis congue tellus et viverra. Sed cursus augue id lobortis accumsan.\n\nNulla et justo eu ligula blandit volutpat. Sed cursus nunc elit, id consequat enim fringilla ac. Fusce sagittis fermentum magna at cursus. Donec metus est, vestibulum vel porttitor vitae, imperdiet non purus. Vestibulum aliquet scelerisque lobortis. Nullam sed dolor id libero interdum dignissim. Fusce porttitor id est in pretium. Donec non faucibus ipsum. Ut mattis orci tincidunt sapien commodo, quis euismod mi condimentum. Phasellus non eros felis. Etiam pellentesque ut massa id suscipit. Praesent viverra mauris in ultrices semper. Nam ante sem, sollicitudin nec mauris ut, euismod hendrerit justo. Curabitur placerat luctus lorem sit amet scelerisque. Suspendisse luctus tellus vitae felis fermentum luctus. Morbi dolor est, convallis ac ex sit amet, viverra dapibus ante."
	//	m.formatInput.Model.focus = true
	//	m.textInput.Model.focus = false
	//}
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
		if s == "ctrl+c" || (s == "q" && (!m.editModeEnabled && !m.formatModeEnabled)) {
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
				headerTop = m.textInput.Model.View()
				if !m.textInput.Model.Focused() {
					headerTop = headerStyle.Copy().Faint(true).Render(headerTop)
				}
			} else {
				headerTop = fmt.Sprintf("%s (%d/%d) - %d record(s) + %d column(s)",
					m.GetSchemaName(),
					m.TableSelection,
					len(m.TableHeaders), // look at how headers get rendered to get accurate record number
					len(m.GetColumnData()),
					len(m.GetHeaders())) // this will need to be refactored when filters get added
				headerTop += strings.Repeat(" ", m.viewport.Width-len(headerTop))
				headerTop = headerStyle.Render(headerTop)
			}

			// separator
			headerBot := strings.Repeat(headerBottomStyle.
				Render("¯"),
				m.viewport.Width)
			headerMid := strings.Join(builder, "")
			headerMid = headerMid + strings.Repeat(" ", m.viewport.Width)
			*h = fmt.Sprintf("%s\n%s\n%s",
				headerTop,
				headerMid,
				headerBot)
		}

		done <- true
	}(&header)

	// footer (shows row/col for now)
	go func(f *string) {
		footer := fmt.Sprintf(" %d, %d ", m.GetRow()+m.viewport.YOffset, m.GetColumn()+m.scrollXOffset)
		undoRedoInfo := fmt.Sprintf("undo(%d) / redo(%d) ", len(m.UndoStack), len(m.RedoStack))
		switch m.Table.Database.(type) {
		case *SQLite:
			break
		default:
			undoRedoInfo = ""
			break
		}
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
