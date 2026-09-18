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
	want := "generate\tGenerates something\n"
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
