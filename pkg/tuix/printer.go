package tuix

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	commandStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	commentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type Printer struct {
	out     io.Writer
	NoColor bool
}

func NewPrinter(out io.Writer, noColor bool) *Printer {
	if out == nil {
		out = os.Stdout
	}
	return &Printer{out: out, NoColor: noColor}
}

func (p *Printer) Println(a ...any) error {
	_, err := fmt.Fprintln(p.out, a...)
	return err
}

func (p *Printer) Printf(format string, a ...any) error {
	_, err := fmt.Fprintf(p.out, format, a...)
	return err
}

func (p *Printer) Commandln(command string) error {
	return p.Println(p.render(commandStyle, command))
}

func (p *Printer) Headerln(text string) error {
	return p.Println(p.render(headerStyle, text))
}

func (p *Printer) Commentln(text string) error {
	return p.Println(p.render(commentStyle, text))
}

func (p *Printer) render(style lipgloss.Style, text string) string {
	if p.NoColor {
		return text
	}
	renderer := lipgloss.NewRenderer(p.out)
	renderer.SetColorProfile(termenv.TrueColor)
	return style.Renderer(renderer).Render(text)
}
