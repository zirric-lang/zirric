package vm

import (
	"fmt"
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/debuginfo"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// RuntimeError is a failure met while running, with the frames that were live when it happened.
//
// The frames are worked out at the moment the error is made, by looking each one's instruction pointer up in the program's debug table. Nothing is carried while a program is working: this is assembled once, on the way out.
type RuntimeError struct {
	Reason error
	Frames []TraceFrame
}

// TraceFrame is one entry of a stack trace.
type TraceFrame struct {
	Function string
	Source   *token.Source
}

// Error implements error.
func (e *RuntimeError) Error() string {
	if e.Reason == nil {
		return ""
	}
	if top := e.topSource(); top != nil {
		return fmt.Sprintf("%s: %s", top.String(), e.Reason.Error())
	}
	return e.Reason.Error()
}

// Unwrap gives access to the failure itself.
func (e *RuntimeError) Unwrap() error { return e.Reason }

// Position implements diag.Positioned, so the line the error happened on can be shown.
func (e *RuntimeError) Position() *token.Source { return e.topSource() }

func (e *RuntimeError) topSource() *token.Source {
	for _, frame := range e.Frames {
		// A source with no line points nowhere, so it is not a position a reader can follow.
		if frame.Source != nil && frame.Source.Line > 0 {
			return frame.Source
		}
	}
	return nil
}

// StackTrace renders the frames that were live, innermost first.
func (e *RuntimeError) StackTrace() string {
	if len(e.Frames) == 0 {
		return ""
	}
	var out strings.Builder
	for i, frame := range e.Frames {
		if i > 0 {
			out.WriteString("\n")
		}
		name := frame.Function
		if name == "" {
			// A frame with no function name is a file's top level, so it is named after the file rather than given a name it does not have.
			name = topLevelName(frame.Source)
		}
		fmt.Fprintf(&out, "  at %s", name)
		if frame.Source != nil && frame.Source.Line > 0 {
			fmt.Fprintf(&out, " (%s)", frame.Source.String())
		}
	}
	return out.String()
}

// fail wraps a failure with the frames that were live when it happened.
//
// An error that already carries frames is passed through, so the innermost failure keeps the trace it was made with rather than being re-traced at every level it unwinds through.
func (vm *VM) fail(err error) error {
	if err == nil {
		return nil
	}
	if _, already := err.(*RuntimeError); already {
		return err
	}
	return &RuntimeError{Reason: err, Frames: vm.trace()}
}

// trace reads the live frames out of the stack, innermost first.
func (vm *VM) trace() []TraceFrame {
	if vm.debug == nil {
		return nil
	}
	frames := make([]TraceFrame, 0, vm.framesIdx)
	// framesIdx counts the live frames rather than indexing the top one, so the walk starts below it. Starting at framesIdx reads whatever the last returning call left behind, since popFrame only decrements — a frame that is no longer running, reported as the one that failed.
	for i := vm.framesIdx - 1; i >= 0; i-- {
		frame := vm.frames[i]
		if frame == nil {
			continue
		}
		// The instruction pointer has already moved past the operands of the instruction being run, so the one that failed starts before it.
		position, ok := vm.debug.Lookup(frame.ins, frame.ip-1)
		if !ok {
			continue
		}
		frames = append(frames, TraceFrame{Function: nameOfFrame(frame, position), Source: position.Source})
	}
	return frames
}

// nameOfFrame is what to call a frame: what the debug table recorded, or the function it is running.
func nameOfFrame(frame *Frame, position debuginfo.Position) string {
	if position.Function != "" {
		return position.Function
	}
	if frame.closure != nil && frame.closure.Fn != nil && frame.closure.Fn.Symbol != nil {
		return frame.closure.Fn.Symbol.Name
	}
	return ""
}

// topLevelName names a frame that belongs to no function, using the file it is running.
func topLevelName(source *token.Source) string {
	if source == nil || source.File == "" || source.Line <= 0 {
		return "<top level>"
	}
	name := source.File
	if at := strings.LastIndex(name, "/"); at >= 0 {
		name = name[at+1:]
	}
	return name
}
