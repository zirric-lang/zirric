package runtime

import "fmt"

// Channel is a queue between routines. Its state is only touched under the scheduler's lock, so a send and a receive never interleave halfway.
type Channel struct {
	id       int
	sched    *Scheduler
	capacity int

	buf    []RuntimeValue
	closed bool
	// senders each hold the value they could not place. An unbuffered channel keeps them here until a receiver takes it.
	senders []*sendWaiter
	// receivers are woken to look again rather than handed a value, so one waiter can sit on several channels.
	receivers []*recvWaiter
}

type sendWaiter struct {
	routine *Routine
	value   RuntimeValue
	taken   bool
}

type recvWaiter struct {
	routine  *Routine
	channels []*Channel
}

// Inspect implements RuntimeValue.
func (ch *Channel) Inspect() string { return fmt.Sprintf("channel #%d", ch.id) }

// Lookup implements RuntimeValue.
func (ch *Channel) Lookup(name string) RuntimeValue { return nil }

// TypeConstantId implements RuntimeValue.
func (ch *Channel) TypeConstantId() TypeId { return typeIdChannel }

// NewChannel creates a channel holding up to capacity values; zero makes every send wait for a receiver.
func (s *Scheduler) NewChannel(capacity int) *Channel {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := &Channel{id: s.nextId, sched: s, capacity: capacity}
	s.nextId++
	return ch
}

// Send blocks until there is room, then puts value in.
func (ch *Channel) Send(r *Routine, value RuntimeValue) error {
	s := ch.sched
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopReasonLocked(r); err != nil {
		return err
	}
	if ch.closed {
		return errSendOnClosed
	}
	if ch.capacity > 0 && len(ch.buf) < ch.capacity {
		ch.buf = append(ch.buf, value)
		ch.notifyLocked()
		return nil
	}

	waiter := &sendWaiter{routine: r, value: value}
	ch.senders = append(ch.senders, waiter)
	ch.notifyLocked()
	for {
		err := s.parkLocked(r, fmt.Sprintf("send on channel #%d", ch.id), false)
		if waiter.taken {
			// Already with a receiver, so the send happened whether or not the routine may carry on.
			return err
		}
		ch.dropSenderLocked(waiter)
		if err != nil {
			return err
		}
		if ch.closed {
			return errSendOnClosed
		}
		if ch.capacity > 0 && len(ch.buf) < ch.capacity {
			ch.buf = append(ch.buf, value)
			ch.notifyLocked()
			return nil
		}
		ch.senders = append(ch.senders, waiter)
	}
}

// Receive blocks until a value arrives. The second result is false once the channel is closed and drained.
func (ch *Channel) Receive(r *Routine) (RuntimeValue, bool, error) {
	_, value, ok, err := ch.sched.Select(r, []*Channel{ch})
	return value, ok, err
}

// Select takes from the first ready channel in list order.
// Order is priority, which keeps a select deterministic; an always-ready channel early in the list starves the later ones.
func (s *Scheduler) Select(r *Routine, channels []*Channel) (*Channel, RuntimeValue, bool, error) {
	if len(channels) == 0 {
		return nil, nil, false, fmt.Errorf("co.select needs at least one channel")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopReasonLocked(r); err != nil {
		return nil, nil, false, err
	}
	waiter := &recvWaiter{routine: r, channels: channels}
	for {
		for _, ch := range channels {
			if !ch.readyLocked() {
				continue
			}
			value, ok := ch.takeLocked()
			return ch, value, ok, nil
		}
		for _, ch := range channels {
			ch.receivers = append(ch.receivers, waiter)
		}
		err := s.parkLocked(r, selectDescription(channels), false)
		for _, ch := range channels {
			ch.dropReceiverLocked(waiter)
		}
		if err != nil {
			return nil, nil, false, err
		}
	}
}

func selectDescription(channels []*Channel) string {
	if len(channels) == 1 {
		return fmt.Sprintf("receive on channel #%d", channels[0].id)
	}
	return fmt.Sprintf("select over %d channels", len(channels))
}

// Close says no more values will be sent. The values already in the channel are still received.
func (ch *Channel) Close() error {
	s := ch.sched
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch.closed {
		return errCloseOfClosed
	}
	ch.closed = true
	ch.notifyLocked()
	for _, waiter := range ch.senders {
		s.wakeLocked(waiter.routine)
	}
	return nil
}

// readyLocked reports whether a receive would return without waiting. A closed, drained channel is always ready, which is how a select learns to stop watching it.
func (ch *Channel) readyLocked() bool {
	return len(ch.buf) > 0 || len(ch.senders) > 0 || ch.closed
}

// takeLocked removes the next value, pulling a blocked sender's value in behind it.
func (ch *Channel) takeLocked() (RuntimeValue, bool) {
	if len(ch.buf) > 0 {
		value := ch.buf[0]
		ch.buf = ch.buf[1:]
		ch.refillLocked()
		return value, true
	}
	if len(ch.senders) > 0 {
		waiter := ch.senders[0]
		ch.senders = ch.senders[1:]
		waiter.taken = true
		ch.sched.wakeLocked(waiter.routine)
		return waiter.value, true
	}
	return nil, false
}

// refillLocked moves a waiting sender's value into the freed slot and releases that sender.
func (ch *Channel) refillLocked() {
	if len(ch.senders) == 0 || len(ch.buf) >= ch.capacity {
		return
	}
	waiter := ch.senders[0]
	ch.senders = ch.senders[1:]
	waiter.taken = true
	ch.buf = append(ch.buf, waiter.value)
	ch.sched.wakeLocked(waiter.routine)
}

// notifyLocked wakes every waiting receiver to look again.
func (ch *Channel) notifyLocked() {
	for _, waiter := range ch.receivers {
		ch.sched.wakeLocked(waiter.routine)
	}
}

func (ch *Channel) dropSenderLocked(waiter *sendWaiter) {
	for i, candidate := range ch.senders {
		if candidate == waiter {
			ch.senders = append(ch.senders[:i], ch.senders[i+1:]...)
			return
		}
	}
}

func (ch *Channel) dropReceiverLocked(waiter *recvWaiter) {
	for i, candidate := range ch.receivers {
		if candidate == waiter {
			ch.receivers = append(ch.receivers[:i], ch.receivers[i+1:]...)
			return
		}
	}
}

// Deliver puts a value in from outside any routine, which is how a real timer fires.
// Capacity is ignored: the only sender is the host event the channel was made for.
func (ch *Channel) Deliver(value RuntimeValue) {
	s := ch.sched
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch.closed {
		return
	}
	ch.buf = append(ch.buf, value)
	ch.notifyLocked()
}
