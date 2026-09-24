package runtime

import (
	"fmt"
	"io"
	"sync"
)

// hostReader owns every read of one host stream, so cancelling a routine that waits for input loses nothing (ZE-025).
// A read already issued cannot be taken back, and bytes cannot be asked for twice, so the read runs on the stream's own goroutine.
// Routines wait on a channel, which is a cancellable switch point, and bytes nobody is waiting for stay here for the next reader.
type hostReader struct {
	src io.Reader

	mu       sync.Mutex
	pending  []byte
	inflight bool
	// Sticky: a stream that ended or broke stays that way.
	ended   bool
	failure error
	// ready carries one value per completed read, built on first use because the stream predates the scheduler.
	ready *Channel
}

func newHostReader(src io.Reader) *hostReader {
	return &hostReader{src: src}
}

// read returns up to n bytes, waiting where the calling routine can be cancelled.
func (hr *hostReader) read(caller VMCaller, n int) (RuntimeValue, error) {
	if n < 0 {
		return nil, fmt.Errorf("read expects a length of zero or more, got %d", n)
	}
	if n == 0 {
		return Binary{}, nil
	}
	sched := caller.Routines()
	routine := sched.Current()

	for {
		taken, done, err := hr.takeLocally(n)
		if done {
			return taken, err
		}

		hr.mu.Lock()
		inflight := hr.inflight
		hr.mu.Unlock()

		// With nothing else to run, a goroutine would only add one; the program would wait either way.
		if !inflight && (routine == nil || !sched.Concurrent()) {
			return hr.readHere(n)
		}

		ready := hr.ensureReady(sched)
		hr.start(sched, ready, n)
		if _, _, waitErr := ready.Receive(routine); waitErr != nil {
			// Cancelled while waiting: what the running read brings in stays in pending for the next reader.
			return nil, waitErr
		}
	}
}

// takeLocally answers from what the stream already holds; done reports whether it could.
func (hr *hostReader) takeLocally(n int) (RuntimeValue, bool, error) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	if len(hr.pending) > 0 {
		count := min(n, len(hr.pending))
		taken := make([]byte, count)
		copy(taken, hr.pending[:count])
		hr.pending = hr.pending[count:]
		return Binary(taken), true, nil
	}
	if hr.failure != nil {
		return nil, true, fmt.Errorf("read: %w", hr.failure)
	}
	if hr.ended {
		return Binary{}, true, nil
	}
	return nil, false, nil
}

// readHere reads on the calling routine's own goroutine, for a program with nothing else to run.
func (hr *hostReader) readHere(n int) (RuntimeValue, error) {
	buf := make([]byte, n)
	count, err := hr.src.Read(buf)
	hr.mu.Lock()
	hr.note(err)
	hr.mu.Unlock()
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read: %w", err)
	}
	return Binary(buf[:count]), nil
}

// start puts a read in flight, unless one already is.
func (hr *hostReader) start(sched *Scheduler, ready *Channel, n int) {
	hr.mu.Lock()
	if hr.inflight {
		hr.mu.Unlock()
		return
	}
	hr.inflight = true
	hr.mu.Unlock()

	// Armed so that waiting for input is not mistaken for a deadlock.
	sched.ArmExternal()
	go func() {
		buf := make([]byte, n)
		count, err := hr.src.Read(buf)

		hr.mu.Lock()
		hr.pending = append(hr.pending, buf[:count]...)
		hr.note(err)
		hr.inflight = false
		hr.mu.Unlock()

		// Delivered before disarming, so a woken routine is already ready when the deadlock check runs.
		ready.Deliver(Void{})
		sched.DisarmExternal()
	}()
}

// note records how a read ended. Bytes alongside an end or a failure are kept: they were still read.
func (hr *hostReader) note(err error) {
	switch err {
	case nil:
	case io.EOF:
		hr.ended = true
	default:
		hr.failure = err
	}
}

// ensureReady returns the channel routines wait on, building it on first use.
// The scheduler's lock is taken with hr.mu released, so the two never nest.
func (hr *hostReader) ensureReady(sched *Scheduler) *Channel {
	hr.mu.Lock()
	existing := hr.ready
	hr.mu.Unlock()
	if existing != nil {
		return existing
	}

	created := sched.NewChannel(1)
	hr.mu.Lock()
	defer hr.mu.Unlock()
	if hr.ready == nil {
		hr.ready = created
	}
	return hr.ready
}
