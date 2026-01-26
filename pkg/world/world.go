package world

import "github.com/go-git/go-billy/v5"

type World struct {
	FS billy.Filesystem
	OS OS
}
