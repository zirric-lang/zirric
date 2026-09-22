package cmds

import (
	"strings"
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/cavefile"
)

func TestPrintTasks(t *testing.T) {
	var buf strings.Builder
	tasks := []cavefile.Task{
		{Name: "generate", Help: "Generates something"},
	}
	if err := printTasks(&buf, tasks); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "generate  Generates something\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Help text lines up in a column, however long the names are, and a task without any leaves no padding behind it.
func TestPrintTasks_AlignsHelpAndTrimsPadding(t *testing.T) {
	var buf strings.Builder
	tasks := []cavefile.Task{
		{Name: "generate-everything", Help: "Long name"},
		{Name: "g", Help: "Short name"},
		{Name: "quiet"},
	}
	if err := printTasks(&buf, tasks); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "generate-everything  Long name\n" +
		"g                    Short name\n" +
		"quiet\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrintTasks_None(t *testing.T) {
	var buf strings.Builder
	if err := printTasks(&buf, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no tasks") {
		t.Errorf("expected a no-tasks message, got %q", buf.String())
	}
}
