package viewer

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"strconv"
	"strings"
	"termdbms/tuiutil"
	"time"
)

var (
	Program          *tea.Program
	FormatModeOffset int
)

func GetOffsetForLineNumber(a int) int {
	return FormatModeOffset - len(strconv.Itoa(a))
}

func SelectOption(m *TuiModel) {
	if m.UI.RenderSelection || m.UI.HelpDisplay {
		return
	}

	m.UI.RenderSelection = true
	raw, _, col := m.GetSelectedOption()
	if raw == nil {
		return
	}
	l := len(col)
	row := m.Viewport.YOffset + m.MouseData.Y - HeaderHeight

	if row <= l && l > 0 &&
		m.MouseData.Y >= HeaderHeight &&
		m.MouseData.Y < m.Viewport.Height+HeaderHeight &&
		m.MouseData.X < m.CellWidth()*(len(m.Data().TableHeadersSlice)) {
		if conv, ok := (*raw).(string); ok {
			m.Data().EditTextBuffer = conv
		} else {
			m.Data().EditTextBuffer = ""
		}
	} else {
		m.UI.RenderSelection = false
	}
}

// ScrollDown is a simple function to move the Viewport down
func ScrollDown(m *TuiModel) {
	if m.UI.FormatModeEnabled && m.UI.CanFormatScroll && m.Viewport.YPosition != 0 {
		m.Viewport.YOffset++
		return
	}

	max := GetScrollDownMaximumForSelection(m)

	if m.Viewport.YOffset < max-m.Viewport.Height {
		m.Viewport.YOffset++
		m.MouseData.Y = Min(m.MouseData.Y, m.Viewport.YOffset)
	}

	if !m.UI.RenderSelection {
		m.Scroll.PreScrollYPosition = m.MouseData.Y
		m.Scroll.PreScrollYOffset = m.Viewport.YOffset
	}
}

// ScrollUp is a simple function to move the Viewport up
func ScrollUp(m *TuiModel) {
	if m.UI.FormatModeEnabled && m.UI.CanFormatScroll && m.Viewport.YOffset > 0 && m.Viewport.YPosition != 0 {
		m.Viewport.YOffset--
		return
	}

	if m.Viewport.YOffset > 0 {
		m.Viewport.YOffset--
		m.MouseData.Y = Min(m.MouseData.Y, m.Viewport.YOffset)
	} else {
		m.MouseData.Y = HeaderHeight
	}

	if !m.UI.RenderSelection {
		m.Scroll.PreScrollYPosition = m.MouseData.Y
		m.Scroll.PreScrollYOffset = m.Viewport.YOffset
	}
}

// TABLE STUFF

// DisplayTable does some fancy stuff to get a table rendered in text
func DisplayTable(m *TuiModel) string {
	var (
		builder []string
	)

	// go through all columns
	for c, columnName := range m.Data().TableHeadersSlice {
		if m.UI.ExpandColumn > -1 && m.UI.ExpandColumn != c {
			continue
		}

		var (
			rowBuilder []string
		)

		columnValues := m.Data().TableSlices[columnName]
		for r, val := range columnValues {
			base := m.GetBaseStyle().
				UnsetBorderLeft().
				UnsetBorderStyle().
				UnsetBorderForeground()
			s := GetStringRepresentationOfInterface(val)
			s = " " + s
			// handle highlighting
			if c == m.GetColumn() && r == m.GetRow() {
				if !tuiutil.Ascii {
					base.Foreground(lipgloss.Color(tuiutil.Highlight()))
				} else if tuiutil.Ascii {
					s = "|" + s
				}
			}
			// display text based on type
			rowBuilder = append(rowBuilder, base.Render(TruncateIfApplicable(m, s)))
		}

		for len(rowBuilder) < m.Viewport.Height { // fix spacing issues
			rowBuilder = append(rowBuilder, "")
		}

		column := lipgloss.JoinVertical(lipgloss.Left, rowBuilder...)
		// get a list of columns
		builder = append(builder, m.GetBaseStyle().Render(column))
	}

	// join them into rows
	return lipgloss.JoinHorizontal(lipgloss.Left, builder...)
}

func GetFormattedTextBuffer(m *TuiModel) []string {
	v := m.Data().EditTextBuffer

	lines := SplitLines(v)
	FormatModeOffset = len(strconv.Itoa(len(lines))) + 1 // number of characters in the numeric string

	var ret []string
	m.Format.RunningOffsets = []int{}

	total := 0
	strlen := 0
	for i, v := range lines {
		xOffset := len(strconv.Itoa(i))
		totalOffset := Max(FormatModeOffset-xOffset, 0)
		//wrap := wordwrap.String(v, m.Viewport.Width-totalOffset)

		right := tuiutil.Indent(
			v,
			fmt.Sprintf("%d%s", i, strings.Repeat(" ", totalOffset)),
			false)
		ret = append(ret, right)
		m.Format.RunningOffsets = append(m.Format.RunningOffsets, total)

		strlen = len(v)

		total += strlen + 1
	}

	lineLength := len(ret)
	// need to add this so that the last line can be edited
	m.Format.RunningOffsets = append(m.Format.RunningOffsets,
		m.Format.RunningOffsets[lineLength-1]+
			len(ret[len(ret)-1][FormatModeOffset:]))

	for i := len(ret); i < m.Viewport.Height; i++ {
		ret = append(ret, "")
	}

	return ret
}

func DisplayFormatText(m *TuiModel) string {
	cpy := make([]string, len(m.Format.EditSlices))
	for i, v := range m.Format.EditSlices {
		cpy[i] = *v
	}
	newY := ""
	line := &cpy[Min(m.Format.CursorY, len(cpy)-1)]
	x := 0
	offset := FormatModeOffset - 1
	for _, r := range *line {
		newY += string(r)
		if x == m.Format.CursorX+offset {
			x++
			break
		}
		x++
	}

	*line += " " // space at the end

	highlight := string((*line)[x])
	if tuiutil.Ascii {
		highlight = "|" + highlight
		newY += highlight
	} else {
		newY += lipgloss.NewStyle().Background(lipgloss.Color("#ffffff")).Render(highlight)
	}

	newY += (*line)[x+1:]
	*line = newY

	ret := strings.Join(
		cpy,
		"\n")

	return ret
}

// DisplaySelection does that or writes it to a file if the selection is over a limit
func DisplaySelection(m *TuiModel) string {
	col := m.GetColumnData()
	row := m.GetRow()
	m.UI.ExpandColumn = m.GetColumn()
	if m.MouseData.Y >= m.Viewport.Height+HeaderHeight &&
		!m.UI.RenderSelection { // this is for when the selection is outside the bounds
		return DisplayTable(m)
	}

	base := m.GetBaseStyle()

	if m.Data().EditTextBuffer != "" { // this is basically just if its a string follow these rules
		conv := m.Data().EditTextBuffer
		if c, err := FormatJson(m.Data().EditTextBuffer); err == nil {
			conv = c
		}
		rows := SplitLines(wordwrap.String(conv, m.Viewport.Width))
		min := 0
		if len(rows) > m.Viewport.Height {
			min = m.Viewport.YOffset
		}
		max := min + m.Viewport.Height
		rows = rows[min:Min(len(rows), max)]

		for len(rows) < m.Viewport.Height {
			rows = append(rows, "")
		}
		return base.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	var prettyPrint string
	raw := col[row]

	if conv, ok := raw.(int64); ok {
		prettyPrint = strconv.Itoa(int(conv))
	} else if i, ok := raw.(float64); ok {
		prettyPrint = base.Render(fmt.Sprintf("%.2f", i))
	} else if t, ok := raw.(time.Time); ok {
		str := t.String()
		prettyPrint = base.Render(str)
	} else if raw == nil {
		prettyPrint = base.Render("NULL")
	}

	lines := SplitLines(prettyPrint)
	for len(lines) < m.Viewport.Height {
		lines = append(lines, "")
	}

	prettyPrint = " " + base.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return wordwrap.String(prettyPrint, m.Viewport.Width)
}
