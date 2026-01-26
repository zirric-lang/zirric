package world

import (
	"os"
)

type unfilteredOS struct{}

func LiveOS() OS {
	return unfilteredOS{}
}

// Exit implements OS.
func (unfilteredOS) Exit(code int) {
	os.Exit(code)
}
