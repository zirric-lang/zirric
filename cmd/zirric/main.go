package main

import "code.knabel.dev/zirric-lang/zirric/cmd/zirric/cmds"

func main() {
	if err := cmds.Execute(); err != nil {
		panic(err)
	}
}
