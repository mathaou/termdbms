package viewer

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"strings"
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
	i, ok := listItem.(SQLSnippet)
	if !ok {
		return
	}

	digits := len(fmt.Sprintf("%d", len(m.Items()))) + 1
	incomingDigits := len(fmt.Sprintf("%d", index+1))
	str := fmt.Sprintf("%d) %s%s", index+1, strings.Repeat(" ", digits-incomingDigits), i.Title())

	fn := style.Copy().PaddingLeft(4).Render
	if index == m.Index() {
		fn = func(s string) string {
			return style.Copy().PaddingLeft(2).Render("> " + s)
		}
	}

	fmt.Fprintf(w, fn(str))
}
