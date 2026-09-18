package main

import (
	"os"

	"code.knabel.dev/zirric-lang/zirric/cmd/zirric/cmds"
)

func main() {
	if err := cmds.Execute(); err != nil {
		os.Exit(1)
	}
}
