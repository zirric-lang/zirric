#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "a .zirr file is run by naming it" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"

  run_zirric main.zirr
  [ "$status" -eq 0 ]
  [[ "$output" == *"main.zirr"* ]]
}

@test "arguments after the file reach the program" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"

  run_zirric main.zirr a b
  [ "$status" -eq 0 ]
  [[ "$output" == *"a"* ]]
  [[ "$output" == *"b"* ]]
}

@test "run still takes a file explicitly" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"

  run_zirric run main.zirr
  [ "$status" -eq 0 ]
  [[ "$output" == *"main.zirr"* ]]
}

@test "naming a file that does not exist reports the file, not an unknown command" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"

  run_zirric nope.zirr
  [ "$status" -ne 0 ]
  [[ "$output" != *"unknown command"* ]]
}

@test "a name that is neither a command nor a .zirr file is still an unknown command" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"

  run_zirric nonsense
  [ "$status" -ne 0 ]
  [[ "$output" == *"unknown command"* ]]
}

@test "a directory is run as a module by naming it" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"
  mkdir -p sub
  printf 'mod runscript.sub\n\nimport scripts\n\nscripts.println("the sub module ran")\n' > sub/m.zirr

  run_zirric sub
  [ "$status" -eq 0 ]
  [[ "$output" == *"the sub module ran"* ]]
}

@test "a task wins over a directory of the same name" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-run-fixtures/script"
  mkdir -p sub
  printf 'mod runscript.sub\n\nimport scripts\n\nscripts.println("the sub module ran")\n' > sub/m.zirr
  cat >> Cavefile <<'ZF'

import tasks

@tasks.Name("sub")
@tasks.Exec("main.zirr")
data SubTask {}
ZF

  run_zirric sub
  [ "$status" -eq 0 ]
  [[ "$output" != *"the sub module ran"* ]]
}
