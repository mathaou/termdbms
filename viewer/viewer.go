package viewer

import (
	"database/sql"
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"strings"
)

var (
	HeaderHeight       = 3
	FooterHeight       = 1
	MaxInputLength     int
	HeaderStyle        lipgloss.Style
	FooterStyle        lipgloss.Style
	HeaderDividerStyle lipgloss.Style
	InitialModel       *TuiModel
)

type ScrollData struct {
	PreScrollYOffset   int
	PreScrollYPosition int
	ScrollXOffset      int
}

// TableState holds everything needed to save/serialize state
type TableState struct {
	Database Database
	Data     map[string]interface{}
}

type UIState struct {
	CanFormatScroll   bool
	RenderSelection   bool // render mode
	HelpDisplay       bool // help display mode
	EditModeEnabled   bool // edit mode
	FormatModeEnabled bool
	BorderToggle      bool
	ExpandColumn      int
	CurrentTable      int
}

type UIData struct {
	TableHeaders      map[string][]string // keeps track of which schema has which headers
	TableHeadersSlice []string
	TableSlices       map[string][]interface{}
	TableIndexMap     map[int]string // keeps the schemas in order
	EditTextBuffer    string
}

type FormatState struct {
	EditSlices     []*string // the bit to show
	Text           []string  // the master collection of lines to edit
	RunningOffsets []int     // this is a LUT for where in the original EditTextBuffer each line starts
	CursorX        int
	CursorY        int
}

// TuiModel holds all the necessary state for this app to work the way I designed it to
type TuiModel struct {
	Table           TableState // all non-destructive changes are TableStates getting passed around
	Format          FormatState
	UI              UIState
	Scroll          ScrollData
	Data            UIData
	Ready           bool
	InitialFileName string // used if saving destructively
	Viewport        viewport.Model
	TableStyle      lipgloss.Style
	MouseData       tea.MouseEvent
	TextInput       LineEdit
	FormatInput     LineEdit
	UndoStack       []TableState
	RedoStack       []TableState
}

func SetStyles() {
	HeaderStyle = lipgloss.NewStyle()
	FooterStyle = lipgloss.NewStyle()

	HeaderDividerStyle = lipgloss.NewStyle().
		Align(lipgloss.Center)

	if !Ascii {
		HeaderStyle = HeaderStyle.
			Foreground(lipgloss.Color(HeaderTopForeground()))

		FooterStyle = FooterStyle.
			Foreground(lipgloss.Color(FooterForeground()))

		HeaderDividerStyle = HeaderDividerStyle.
			Foreground(lipgloss.Color(HeaderBottom()))
	}
}

// INIT UPDATE AND RENDER

// Init currently doesn't do anything but necessary for interface adherence
func (m TuiModel) Init() tea.Cmd {
	SetStyles()

	return nil
}

// Update is where all commands and whatnot get processed
func (m TuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var (
		command  tea.Cmd
		commands []tea.Cmd
	)

	switch msg := message.(type) {
	case tea.MouseMsg:
		HandleMouseEvents(&m, &msg)
		m.SetViewSlices()
		break
	case tea.WindowSizeMsg:
		event := HandleWindowSizeEvents(&m, &msg)
		commands = append(commands, event)
		break
	case tea.KeyMsg:
		// when fullscreen selection viewing is in session, don't allow UI manipulation other than quit or exit
		s := msg.String()
		if m.UI.RenderSelection &&
			s != "esc" &&
			s != "ctrl+c" &&
			s != "q" &&
			s != "p" &&
			s != "m" &&
			s != "n" {
			break
		}
		if s == "ctrl+c" || (s == "q" && (!m.UI.EditModeEnabled && !m.UI.FormatModeEnabled)) {
			return m, tea.Quit
		}

		HandleKeyboardEvents(&m, &msg)
		if !m.UI.EditModeEnabled && m.Ready {
			m.SetViewSlices()
			if m.UI.FormatModeEnabled {
				MoveCursorWithinBounds(&m)
			}
		}

		break
	case error:
		return m, nil
	}

	if !m.UI.FormatModeEnabled {
		m.Viewport, _ = m.Viewport.Update(message)
	}

	if m.Viewport.HighPerformanceRendering {
		commands = append(commands, command)
	}

	return m, tea.Batch(commands...)
}

// View is where all rendering happens
func (m TuiModel) View() string {
	if !m.Ready || m.Viewport.Width == 0 {
		return "\n\tInitializing..."
	}

	// this ensures that all 3 parts can be worked on concurrently(ish)
	done := make(chan bool, 3)

	var footer, header, content string

	// body
	go func(c *string) {
		*c = AssembleTable(&m)
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
			style = style.Foreground(lipgloss.Color(HeaderForeground())).
				BorderBackground(lipgloss.Color(HeaderBorderBackground())).
				Background(lipgloss.Color(HeaderBackground())).
				PaddingLeft(1)
		}
		headers := m.Data.TableHeadersSlice
		for i, d := range headers { // write all headers
			if m.UI.ExpandColumn != -1 && i != m.UI.ExpandColumn {
				continue
			}

			text := TruncateIfApplicable(&m, d)
			builder = append(builder, style.
				Render(text))
		}

		{
			// schema name
			var headerTop string

			if m.UI.EditModeEnabled || m.UI.FormatModeEnabled {
				headerTop = m.TextInput.Model.View()
				if !m.TextInput.Model.Focused() {
					headerTop = HeaderStyle.Copy().Faint(true).Render(headerTop)
				}
			} else {
				headerTop = fmt.Sprintf("%s (%d/%d) - %d record(s) + %d column(s)",
					m.GetSchemaName(),
					m.UI.CurrentTable,
					len(m.Data.TableHeaders), // look at how headers get rendered to get accurate record number
					len(m.GetColumnData()),
					len(m.GetHeaders())) // this will need to be refactored when filters get added
				headerTop += strings.Repeat(" ", m.Viewport.Width-len(headerTop))
				headerTop = HeaderStyle.Render(headerTop)
			}

			// separator
			headerBot := strings.Repeat(
				HeaderDividerStyle.
					Render("¯"),
				m.Viewport.Width)
			headerMid := strings.Join(builder, "")
			//headerMid = headerMid + strings.Repeat(" ", m.Viewport.Width)
			*h = fmt.Sprintf("%s\n%s\n%s",
				headerTop,
				headerMid,
				headerBot)
		}

		done <- true
	}(&header)

	// footer (shows row/col for now)
	go func(f *string) {
		var (
			row int
			col int
		)
		if !m.UI.FormatModeEnabled { // reason we flip is because it makes more sense to store things by column for data
			row = m.GetRow() + m.Viewport.YOffset
			col = m.GetColumn() + m.Scroll.ScrollXOffset
		} else { // but for format mode thats just a regular row/col situation
			row = m.Format.CursorX
			col = m.Format.CursorY + m.Viewport.YOffset
		}
		footer := fmt.Sprintf(" %d, %d ", row, col)
		undoRedoInfo := fmt.Sprintf("undo(%d) / redo(%d) ", len(m.UndoStack), len(m.RedoStack))
		switch m.Table.Database.(type) {
		case *SQLite:
			break
		default:
			undoRedoInfo = ""
			break
		}
		gapSize := m.Viewport.Width - lipgloss.Width(footer) - lipgloss.Width(undoRedoInfo) - 2
		footer = FooterStyle.Render(undoRedoInfo) + "├" + strings.Repeat("─", gapSize) + "┤" + FooterStyle.Render(footer)
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
	rows, err := db.Query(GetTableNamesQuery)
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
		m.Table.Data[schemaName] = columnValues       // data for schema, organized by column
		m.Data.TableHeaders[schemaName] = columnNames // headers for the schema, for later reference
		// mapping between schema and an int ( since maps aren't deterministic), for later reference
		m.Data.TableIndexMap[indexMap] = schemaName
	}

	// set the first table to be initial view
	m.UI.CurrentTable = 1
}
