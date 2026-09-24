package runtime

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrCancelled unwinds a routine co.cancel stopped. Not a program failure, so wait reports it as Err(Cancelled()).
var ErrCancelled = errors.New("routine cancelled")

// errSendOnClosed and errCloseOfClosed are caller bugs rather than expected failures, so they stop the program instead of becoming results (ZE-008).
var (
	errSendOnClosed  = errors.New("send on a closed channel")
	errCloseOfClosed = errors.New("close of a closed channel")
)

type routineState int

const (
	routineReady routineState = iota
	routineRunning
	routineParked
	// routineInHost has given up the run lock, but unlike routineParked nothing in the program can hand it back: waking it would give the lock to a goroutine stuck in a syscall.
	routineInHost
	routineDone
)

// Routine is one running function, owned by the scope it was spawned into.
// Everything below caller is guarded by the scheduler's lock.
type Routine struct {
	id     int
	sched  *Scheduler
	caller VMCaller
	body   RuntimeValue
	// resume carries the run lock here. Buffered, so handing it over never waits for the receiving goroutine.
	resume chan struct{}

	state     routineState
	where     string
	cancelled bool
	done      bool
	result    RuntimeValue
	waiters   []*Routine
	// Stopping the routine cancels these, so a nested scope leaves no orphans.
	scopes []*RoutineScope
}

// Inspect implements RuntimeValue.
func (r *Routine) Inspect() string { return fmt.Sprintf("routine #%d", r.id) }

// Lookup implements RuntimeValue.
func (r *Routine) Lookup(name string) RuntimeValue { return nil }

// TypeConstantId implements RuntimeValue.
func (r *Routine) TypeConstantId() TypeId { return typeIdRoutine }

// RoutineScope owns the routines spawned into it. Only co.scope creates one.
type RoutineScope struct {
	id    int
	sched *Scheduler
	// owner runs the scope's body, and cancelling the scope never stops it.
	owner     *Routine
	routines  []*Routine
	cancelled bool
	finished  bool
}

// Inspect implements RuntimeValue.
func (scope *RoutineScope) Inspect() string { return fmt.Sprintf("scope #%d", scope.id) }

// Lookup implements RuntimeValue.
func (scope *RoutineScope) Lookup(name string) RuntimeValue { return nil }

// TypeConstantId implements RuntimeValue.
func (scope *RoutineScope) TypeConstantId() TypeId { return typeIdScope }

// Scheduler runs routines one at a time, handing a single run lock between them at switch points.
// There is no scheduler goroutine: whoever gives the lock up passes it on, so a switch only ever happens inside a switch point.
type Scheduler struct {
	mu sync.Mutex

	ready   []*Routine
	running *Routine
	// known holds the routines that have not ended, for deadlock reporting and shutdown.
	known []*Routine
	// parked counts routines waiting to be woken. Routines inside a host call are not among them: nothing in the program can wake one.
	parked int
	// external counts host calls and armed timers, which can still wake a parked routine.
	external int
	// fatal is the first failure that escaped a routine; every other routine unwinds with it.
	fatal      error
	deadlocked bool
	// Numbered apart from scopes and channels, so a deadlock message says "routine #1" for the first one spawned.
	nextRoutineId int
	nextId        int
}

// NewScheduler starts a scheduler whose run lock is already held by the routine the program itself runs in.
func NewScheduler(main VMCaller) *Scheduler {
	s := &Scheduler{nextRoutineId: 1, nextId: 1}
	root := &Routine{
		sched:  s,
		caller: main,
		resume: make(chan struct{}, 1),
		state:  routineRunning,
	}
	s.running = root
	s.known = []*Routine{root}
	return s
}

// Concurrent reports whether giving way is worth anything, meaning more than the program's own routine is alive.
func (s *Scheduler) Concurrent() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.known) > 1
}

// Current returns the routine holding the run lock, always the one asking.
func (s *Scheduler) Current() *Routine {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// dispatchLocked hands the run lock to the next ready routine, if it is free.
func (s *Scheduler) dispatchLocked() {
	if s.running != nil || len(s.ready) == 0 {
		return
	}
	next := s.ready[0]
	s.ready = s.ready[1:]
	next.state = routineRunning
	next.where = ""
	s.running = next
	next.resume <- struct{}{}
}

// releaseLocked passes the run lock on, reporting a deadlock when nothing can take it.
func (s *Scheduler) releaseLocked() {
	s.running = nil
	s.dispatchLocked()
	s.checkDeadlockLocked()
}

func (s *Scheduler) checkDeadlockLocked() {
	if s.running != nil || len(s.ready) > 0 || s.external > 0 || s.parked == 0 {
		return
	}
	s.reportDeadlockLocked()
}

// reportDeadlockLocked ends every parked routine with one failure, naming what each of them was waiting on.
func (s *Scheduler) reportDeadlockLocked() {
	if s.deadlocked {
		return
	}
	s.deadlocked = true
	parked := s.parkedLocked()
	blocked := make([]string, 0, len(parked))
	for _, r := range parked {
		blocked = append(blocked, fmt.Sprintf("routine #%d on %s", r.id, r.where))
	}
	if s.fatal == nil {
		s.fatal = fmt.Errorf("deadlock: every routine is waiting (%s)", strings.Join(blocked, ", "))
	}
	for _, r := range parked {
		s.wakeLocked(r)
	}
}

// parkedLocked lists the routines parked at a switch point. Only reached once a program has stalled, so a walk beats keeping an index.
func (s *Scheduler) parkedLocked() []*Routine {
	parked := make([]*Routine, 0, s.parked)
	for _, r := range s.known {
		if r.state == routineParked {
			parked = append(parked, r)
		}
	}
	return parked
}

// wakeLocked requeues a parked routine. Waking one that is not parked does nothing, which lets several channels hold the same select waiter.
func (s *Scheduler) wakeLocked(r *Routine) {
	if r.state != routineParked {
		return
	}
	s.parked--
	r.state = routineReady
	s.ready = append(s.ready, r)
	s.dispatchLocked()
}

// parkLocked gives the run lock up and waits for it back, holding the scheduler's lock on both sides.
// requeue distinguishes giving way from blocking, where something else has to wake the routine.
func (s *Scheduler) parkLocked(r *Routine, where string, requeue bool) error {
	if requeue {
		r.state = routineReady
		s.ready = append(s.ready, r)
	} else {
		r.state = routineParked
		r.where = where
		s.parked++
	}
	s.releaseLocked()
	s.mu.Unlock()
	<-r.resume
	s.mu.Lock()
	return s.stopReasonLocked(r)
}

// stopReasonLocked reports why a routine must unwind rather than carry on, if it must.
func (s *Scheduler) stopReasonLocked(r *Routine) error {
	if s.fatal != nil {
		return s.fatal
	}
	if r.cancelled {
		return ErrCancelled
	}
	return nil
}

// Yield gives the other ready routines a turn, so a stdlib call is a switch point even when a test fake answers instantly.
func (s *Scheduler) Yield(r *Routine) error {
	if r == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.stopReasonLocked(r); err != nil {
		return err
	}
	if len(s.ready) == 0 {
		return nil
	}
	return s.parkLocked(r, "", true)
}

// BeginHostCall releases the run lock for the length of a call that waits on the host.
func (s *Scheduler) BeginHostCall(r *Routine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.external++
	r.state = routineInHost
	r.where = "a host call"
	s.releaseLocked()
}

// EndHostCall takes the run lock back once the call the routine was waiting on has returned.
func (s *Scheduler) EndHostCall(r *Routine) error {
	s.mu.Lock()
	s.external--
	r.state = routineReady
	s.ready = append(s.ready, r)
	s.dispatchLocked()
	s.mu.Unlock()
	<-r.resume
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopReasonLocked(r)
}

// ArmExternal registers an event outside the program that can still wake a routine, so waiting for it is not mistaken for a deadlock.
func (s *Scheduler) ArmExternal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.external++
}

// DisarmExternal retires what ArmExternal registered.
func (s *Scheduler) DisarmExternal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.external--
	s.checkDeadlockLocked()
}

// Spawn starts body in a new routine owned by scope, and returns at once: it runs when the spawner reaches a switch point.
func (s *Scheduler) Spawn(scope *RoutineScope, body RuntimeValue, caller VMCaller) (*Routine, error) {
	s.mu.Lock()
	if scope.finished || scope.cancelled {
		ended := "finished"
		if scope.cancelled {
			ended = "been cancelled"
		}
		s.mu.Unlock()
		return nil, fmt.Errorf("co.spawn: the scope has already %s", ended)
	}
	r := &Routine{
		id:     s.nextRoutineId,
		sched:  s,
		caller: caller,
		body:   body,
		resume: make(chan struct{}, 1),
		state:  routineReady,
	}
	s.nextRoutineId++
	s.known = append(s.known, r)
	scope.routines = append(scope.routines, r)
	s.ready = append(s.ready, r)
	s.dispatchLocked()
	s.mu.Unlock()

	go r.loop()
	return r, nil
}

// loop runs the body once handed the run lock, and passes the lock on when done.
func (r *Routine) loop() {
	<-r.resume

	s := r.sched
	var result RuntimeValue
	var fatal error

	s.mu.Lock()
	stop := s.stopReasonLocked(r)
	s.mu.Unlock()

	if stop == nil {
		value, err := r.caller.CallFunction(r.body)
		switch {
		case err == nil:
			result = value
		case errors.Is(err, ErrCancelled):
			// Cancellation ends the routine and nothing else; wait reports it as Err(Cancelled()).
		default:
			fatal = err
		}
	}

	s.mu.Lock()
	if fatal != nil && s.fatal == nil {
		s.fatal = fatal
		s.abortAllLocked()
	}
	r.done = true
	r.state = routineDone
	r.result = result
	for _, w := range r.waiters {
		s.wakeLocked(w)
	}
	r.waiters = nil
	// A routine still holding scopes was stopped inside one; they end with it.
	for _, scope := range r.scopes {
		s.cancelScopeLocked(scope)
	}
	r.scopes = nil
	s.forgetLocked(r)
	s.releaseLocked()
	s.mu.Unlock()
}

func (s *Scheduler) forgetLocked(r *Routine) {
	for i, known := range s.known {
		if known == r {
			s.known = append(s.known[:i], s.known[i+1:]...)
			return
		}
	}
}

// abortAllLocked wakes every parked routine, so one routine's failure ends the program instead of stranding the rest.
func (s *Scheduler) abortAllLocked() {
	for _, r := range s.parkedLocked() {
		s.wakeLocked(r)
	}
}

// Wait blocks until the routine has ended. A nil result means it was cancelled.
func (s *Scheduler) Wait(waiter *Routine, target *Routine) (RuntimeValue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for !target.done {
		if err := s.stopReasonLocked(waiter); err != nil {
			return nil, err
		}
		target.waiters = append(target.waiters, waiter)
		if err := s.parkLocked(waiter, fmt.Sprintf("wait for routine #%d", target.id), false); err != nil {
			return nil, err
		}
	}
	return target.result, nil
}

// OpenScope creates a scope owned by the routine asking for one.
func (s *Scheduler) OpenScope(owner *Routine) *RoutineScope {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope := &RoutineScope{id: s.nextId, sched: s, owner: owner}
	s.nextId++
	owner.scopes = append(owner.scopes, scope)
	return scope
}

// CloseScope waits for every routine of the scope, so nothing outlives the block that started it.
func (s *Scheduler) CloseScope(scope *RoutineScope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope.finished = true
	owner := scope.owner
	var stop error
	for {
		// An unwinding owner cancels what it owns: waiting on routines nobody will feed is how a program hangs.
		if (owner.cancelled || s.fatal != nil) && !scope.cancelled {
			s.cancelScopeLocked(scope)
		}
		pending := scope.pendingLocked()
		if pending == nil {
			break
		}
		pending.waiters = append(pending.waiters, owner)
		if err := s.parkLocked(owner, fmt.Sprintf("the end of scope #%d", scope.id), false); err != nil && stop == nil {
			stop = err
		}
	}
	s.dropScopeLocked(owner, scope)
	return stop
}

// pendingLocked returns a routine of the scope that has not ended yet, or nil when all of them have.
func (scope *RoutineScope) pendingLocked() *Routine {
	for _, r := range scope.routines {
		if !r.done {
			return r
		}
	}
	return nil
}

func (s *Scheduler) dropScopeLocked(owner *Routine, scope *RoutineScope) {
	for i, open := range owner.scopes {
		if open == scope {
			owner.scopes = append(owner.scopes[:i], owner.scopes[i+1:]...)
			return
		}
	}
}

// Cancel stops every routine spawned into the scope. It never stops the scope's body, nor the routine that called it.
func (s *Scheduler) Cancel(scope *RoutineScope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelScopeLocked(scope)
}

func (s *Scheduler) cancelScopeLocked(scope *RoutineScope) {
	if scope.cancelled {
		return
	}
	scope.cancelled = true
	for _, r := range scope.routines {
		s.cancelRoutineLocked(r)
	}
}

// cancelRoutineLocked marks a routine to stop at its next switch point, taking its open scopes with it.
// One inside a host call is only marked: wakeLocked ignores it, and the call reads the mark when it returns.
func (s *Scheduler) cancelRoutineLocked(r *Routine) {
	if r.done || r.cancelled {
		return
	}
	r.cancelled = true
	for _, scope := range r.scopes {
		s.cancelScopeLocked(scope)
	}
	s.wakeLocked(r)
}

// Shutdown stops everything still running, so a program that failed leaves no routines behind.
func (s *Scheduler) Shutdown(reason error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fatal == nil {
		if reason == nil {
			reason = errors.New("the program ended")
		}
		s.fatal = reason
	}
	for _, r := range append([]*Routine(nil), s.known...) {
		s.cancelRoutineLocked(r)
	}
	s.abortAllLocked()
}
