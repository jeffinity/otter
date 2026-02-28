package tuix

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRenderUsageWithDescAndSections(t *testing.T) {
	t.Parallel()

	got := RenderUsage("otter new", "desc", []Section{
		{Title: "用法", Lines: []string{"new <module> <app>"}},
		{Title: "示例", Lines: []string{"otter new a/b/c demo"}},
	})

	for _, needle := range []string{"otter new", "desc", "用法", "示例", "demo"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("RenderUsage() output missing %q: %s", needle, got)
		}
	}
}

func TestRenderUsageWithoutDesc(t *testing.T) {
	t.Parallel()

	got := RenderUsage("title", "   ", []Section{
		{Title: "选项", Lines: []string{"-h, --help"}},
	})
	if strings.Contains(got, "desc") {
		t.Fatalf("unexpected desc content in output: %s", got)
	}
	if !strings.Contains(got, "title") || !strings.Contains(got, "选项") {
		t.Fatalf("RenderUsage() basic content missing: %s", got)
	}
}

func TestPrintStaticToBuffer(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := PrintStatic(&buf, "hello usage"); err != nil {
		t.Fatalf("PrintStatic() returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "hello usage") {
		t.Fatalf("expected content in output, got: %s", out)
	}
}

func TestNewSpinnerUsesMiniDot(t *testing.T) {
	t.Parallel()

	sp := NewSpinner("ignored-title")
	if len(sp.Spinner.Frames) == 0 {
		t.Fatal("spinner frames should not be empty")
	}
	if len(sp.Spinner.Frames) != 10 {
		t.Fatalf("expected MiniDot frames count=10, got=%d", len(sp.Spinner.Frames))
	}
	if sp.Spinner.Frames[0] != "⠋" {
		t.Fatalf("expected MiniDot first frame, got %q", sp.Spinner.Frames[0])
	}
}

func TestPrintStaticNilWriterUsesStdout(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	defer r.Close()

	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	if err := PrintStatic(nil, "stdout-content"); err != nil {
		t.Fatalf("PrintStatic(nil) returned error: %v", err)
	}
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout pipe) failed: %v", err)
	}
	if !strings.Contains(string(out), "stdout-content") {
		t.Fatalf("expected stdout content, got: %q", string(out))
	}
}

func TestStaticModelMethods(t *testing.T) {
	t.Parallel()

	m := staticModel{content: "abc"}
	if got := m.View(); got != "abc\n" {
		t.Fatalf("View() = %q, want %q", got, "abc\n")
	}

	if initCmd := m.Init(); initCmd == nil {
		t.Fatal("Init() should return quit cmd")
	}

	next, cmd := m.Update(tea.KeyMsg{})
	if cmd == nil {
		t.Fatal("Update() should return quit cmd")
	}
	nextModel, ok := next.(staticModel)
	if !ok {
		t.Fatalf("Update() model type = %T, want staticModel", next)
	}
	if nextModel.content != m.content {
		t.Fatalf("Update() model content changed: %q != %q", nextModel.content, m.content)
	}
}

var _ io.Writer = bytes.NewBuffer(nil)
