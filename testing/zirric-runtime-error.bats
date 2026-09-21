#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "a failure inside a stdlib function says where it happened" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-runtime-error-fixtures/plugin-error"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  # A real position, not just a file name.
  [[ "$output" == *"main.zirr:4:"* ]]
}

@test "a runtime failure prints the stack that led there" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-runtime-error-fixtures/plugin-error"

  run_zirric run main.zirr
  [[ "$output" == *"at pick"* ]]
  [[ "$output" == *"at main"* ]]
}

@test "a runtime failure names types the way they are written" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-runtime-error-fixtures/plugin-error"

  run_zirric run main.zirr
  [[ "$output" == *"got String"* ]]
  [[ "$output" != *"runtime."* ]]
}

@test "a stack trace carries no frame whose position is only a file" {
  # A script run on its own, with no Cavefile, takes its package name from the file. A module's own token names a file but no line, and a frame built from one used to appear as "at main (main)" with a file where a position belongs.
  setup_fixture "$BATS_TEST_DIRNAME/zirric-runtime-error-fixtures/loose-script"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  [[ "$output" == *"main.zirr:4:"* ]]
  [[ "$output" != *"at main (main)"* ]]
  [[ "$output" != *"Error: main: "* ]]
}
