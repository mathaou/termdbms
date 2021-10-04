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

func (d itemDelegate) Height() int  { return 2 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	var localStyle lipgloss.Style
	i, ok := listItem.(SQLSnippet)
	if !ok {
		return
	}

	digits := len(fmt.Sprintf("%d", len(m.Items()))) + 1
	incomingDigits := len(fmt.Sprintf("%d", index+1))

	if tuiutil.Ascii {
		localStyle = style.Copy().Faint(true)
	}

	str := fmt.Sprintf("%d) %s%s\n\t%s", index+1, strings.Repeat(" ", digits-incomingDigits), i.Title(),
		localStyle.Render(i.Query[0:Min(TUIWidth - 10, len(i.Query) - 1)])) // padding + tab + padding
	if tuiutil.Ascii {
		localStyle = style.Copy().PaddingLeft(4)
	}
	fn := localStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			if tuiutil.Ascii {
				localStyle = style.Copy().
					Foreground(lipgloss.Color(tuiutil.HeaderTopForeground())).
					PaddingLeft(2)
			}
			if tuiutil.Ascii {
				s = style.Render(s)
			}
			return lipgloss.JoinHorizontal(lipgloss.Left,
				localStyle.
				Render("> "),
				s)
		}
	}

	fmt.Fprintf(w, fn(str))
}
