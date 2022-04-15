package viewer

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mathaou/termdbms/database"
	"github.com/mathaou/termdbms/list"
)

type SQLSnippet struct {
	Query string `json:"Query"`
	Name  string `json:"Name"`
}

type ScrollData struct {
	PreScrollYOffset   int
	PreScrollYPosition int
	ScrollXOffset      int
}

// TableState holds everything needed to save/serialize state
type TableState struct {
	Database database.Database
	Data     map[string]interface{}
}

type UIState struct {
	CanFormatScroll   bool
	RenderSelection   bool // render mode
	EditModeEnabled   bool // edit mode
	FormatModeEnabled bool
	BorderToggle      bool
	SQLEdit           bool
	ShowClipboard     bool
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
	DefaultTable    TableState // all non-destructive changes are TableStates getting passed around
	DefaultData     UIData
	QueryResult     *TableState
	QueryData       *UIData
	Format          FormatState
	UI              UIState
	Scroll          ScrollData
	Ready           bool
	InitialFileName string // used if saving destructively
	Viewport        viewport.Model
	ClipboardList   list.Model
	Clipboard       []list.Item
	TableStyle      lipgloss.Style
	MouseData       tea.MouseEvent
	TextInput       LineEdit
	FormatInput     LineEdit
	UndoStack       []TableState
	RedoStack       []TableState
}
