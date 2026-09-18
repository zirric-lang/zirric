package orchestra

import (
	"reflect"
	"testing"
)

func TestBuildArgv(t *testing.T) {
	got := buildArgv("scripts/build.zirr", []string{"--flag", "value"})
	want := []string{"scripts/build.zirr", "--flag", "value"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgv = %v, want %v", got, want)
	}
}

func TestBuildArgvNoExtraArgs(t *testing.T) {
	got := buildArgv("scripts/build.zirr", nil)
	want := []string{"scripts/build.zirr"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgv = %v, want %v", got, want)
	}
}
