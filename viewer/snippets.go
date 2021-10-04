package viewer

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"strings"
	"termdbms/list"
	"termdbms/tuiutil"
)

var (
	style = lipgloss.NewStyle()
)

func (s SQLSnippet) Title() string {
	return s.Name
}

func (s SQLSnippet) Description() string {
	return s.Query
}

func (s SQLSnippet) FilterValue() string {
	return s.Name
}

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	localStyle := style.Copy()
	i, ok := listItem.(SQLSnippet)
	if !ok {
		return
	}

	digits := len(fmt.Sprintf("%d", len(m.Items()))) + 1
	incomingDigits := len(fmt.Sprintf("%d", index+1))

	if !tuiutil.Ascii {
		localStyle = style.Copy().Faint(true)
	}

	str := fmt.Sprintf("%d) %s%s | ", index+1, strings.Repeat(" ", digits-incomingDigits),
		i.Title())
	query := localStyle.Render(i.Query[0:Min(TUIWidth - 10, Max(len(i.Query) - 1, len(i.Query) - 1 - len(str)))]) // padding + tab + padding
	str += strings.ReplaceAll(query, "\n", "")

	localStyle = style.Copy().PaddingLeft(4)

	fn := localStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			localStyle = style.Copy().
				PaddingLeft(2)
			if !tuiutil.Ascii {
				localStyle = localStyle.
					Foreground(lipgloss.Color(tuiutil.HeaderTopForeground()))
			}

			return lipgloss.JoinHorizontal(lipgloss.Left,
				localStyle.
				Render("> "),
				style.Render(s))
		}
	}

	fmt.Fprintf(w, fn(str))
}
