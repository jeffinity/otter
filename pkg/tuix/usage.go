package tuix

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	descStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	headStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("31")).Padding(0, 1)
	itemStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("39")).Padding(1, 2)
)

type Section struct {
	Title string
	Lines []string
}

func RenderUsage(title, desc string, sections []Section) string {
	parts := make([]string, 0, len(sections)+2)
	parts = append(parts, titleStyle.Render(title))
	if strings.TrimSpace(desc) != "" {
		parts = append(parts, descStyle.Render(desc))
	}
	for _, sec := range sections {
		items := make([]string, 0, len(sec.Lines)+1)
		items = append(items, headStyle.Render(sec.Title))
		for _, line := range sec.Lines {
			items = append(items, itemStyle.Render(line))
		}
		parts = append(parts, strings.Join(items, "\n"))
	}
	return boxStyle.Render(strings.Join(parts, "\n\n"))
}

func PrintStatic(out io.Writer, content string) error {
	target := out
	if target == nil {
		target = os.Stdout
	}
	model := staticModel{content: content}
	p := tea.NewProgram(model, tea.WithOutput(target), tea.WithInput(nil), tea.WithoutSignalHandler())
	if _, err := p.Run(); err != nil {
		_, _ = fmt.Fprintln(target, content)
		return err
	}
	return nil
}

func NewSpinner(title string) spinner.Model {
	_ = title
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("81"))),
	)
}

type staticModel struct {
	content string
}

func (m staticModel) Init() tea.Cmd {
	return tea.Quit
}

func (m staticModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m staticModel) View() string {
	return m.content + "\n"
}
