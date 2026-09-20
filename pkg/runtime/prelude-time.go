package runtime

import (
	"fmt"
	"strings"
	"time"
)

var (
	_ RuntimeValue = Duration(0)
	_ RuntimeValue = Instant(0)
	_ RuntimeValue = Timestamp(0)
)

// Duration is a span of time in nanoseconds.
// It is a primitive rather than a data type so that the VM can give it arithmetic, which is what keeps `a + b` and `d * 2` readable in a language without operator overloading.
type Duration int64

// Instant is a reading from a monotonic clock, in nanoseconds from an arbitrary origin.
// The origin is meaningless, so an Instant is only ever useful relative to another: subtracting two gives a Duration, and that is the only arithmetic the VM allows.
type Instant int64

// Timestamp is a point on the wall clock, in nanoseconds since the Unix epoch.
// Kept distinct from Instant so that a value which can jump backwards, as wall clocks do, cannot be used where elapsed time is meant.
type Timestamp int64

// Inspect implements RuntimeValue.
func (d Duration) Inspect() string {
	return formatDuration(int64(d))
}

// Lookup implements RuntimeValue.
func (d Duration) Lookup(name string) RuntimeValue { return nil }

// TypeConstantId implements RuntimeValue.
func (d Duration) TypeConstantId() TypeId { return typeIdDuration }

// Inspect implements RuntimeValue.
func (i Instant) Inspect() string {
	return fmt.Sprintf("instant %s", formatDuration(int64(i)))
}

// Lookup implements RuntimeValue.
func (i Instant) Lookup(name string) RuntimeValue { return nil }

// TypeConstantId implements RuntimeValue.
func (i Instant) TypeConstantId() TypeId { return typeIdInstant }

// Inspect implements RuntimeValue.
func (t Timestamp) Inspect() string {
	return t.Time().UTC().Format(time.RFC3339Nano)
}

// Time returns the Go time this Timestamp denotes.
func (t Timestamp) Time() time.Time {
	return time.Unix(0, int64(t))
}

// Lookup implements RuntimeValue.
func (t Timestamp) Lookup(name string) RuntimeValue { return nil }

// TypeConstantId implements RuntimeValue.
func (t Timestamp) TypeConstantId() TypeId { return typeIdTimestamp }

// formatDuration renders a span the way a person reads it, so that a printed duration says 1.5s rather than 1500000000.
// The compact form is only used when it reads back as the same span, so that formatting never quietly loses precision; anything else falls back to Go's own rendering, which is exact but noisier.
func formatDuration(nanos int64) string {
	compact := compactDuration(nanos)
	if parsed, err := time.ParseDuration(compact); err == nil && int64(parsed) == nanos {
		return compact
	}
	return time.Duration(nanos).String()
}

func compactDuration(nanos int64) string {
	if nanos == 0 {
		return "0s"
	}
	sign := ""
	if nanos < 0 {
		sign = "-"
		nanos = -nanos
	}
	switch {
	case nanos < 1000:
		return fmt.Sprintf("%s%dns", sign, nanos)
	case nanos < 1000*1000:
		return sign + trimZeros(float64(nanos)/1000) + "µs"
	case nanos < 1000*1000*1000:
		return sign + trimZeros(float64(nanos)/(1000*1000)) + "ms"
	case nanos < 60*1000*1000*1000:
		return sign + trimZeros(float64(nanos)/(1000*1000*1000)) + "s"
	case nanos < 60*60*1000*1000*1000:
		return sign + trimZeros(float64(nanos)/(60*1000*1000*1000)) + "m"
	default:
		return sign + trimZeros(float64(nanos)/(60*60*1000*1000*1000)) + "h"
	}
}

func trimZeros(v float64) string {
	s := fmt.Sprintf("%.3f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
