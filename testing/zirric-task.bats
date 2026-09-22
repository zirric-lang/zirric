#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "task lists declared tasks" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task
  [ "$status" -eq 0 ]
  [[ "$output" == *"greet"* ]]
  [[ "$output" == *"Prints a greeting"* ]]
}

@test "task prints a message when no tasks are declared" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/no-tasks"

  run_zirric task
  [ "$status" -eq 0 ]
  [[ "$output" == *"no tasks"* ]]
}

@test "task executes a declared task by name" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task greet
  [ "$status" -eq 0 ]
}

@test "task fails cleanly for an unknown task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task nope
  [ "$status" -ne 0 ]
  [[ "$output" == *"unknown task"*"nope"* ]]
}

@test "a task answers by its bare name" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric greet
  [ "$status" -eq 0 ]
}

@test "a task answers by a bare alias" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric hi
  [ "$status" -eq 0 ]
}

@test "a declared task appears in the root help" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"greet"* ]]
}

@test "task ignores trailing arguments rather than failing" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task greet --dry extra-arg
  [ "$status" -eq 0 ]
}

@test "task passes typed flag values to a @tasks.Call task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric task build --target=release
  [ "$status" -eq 0 ]
  [[ "$output" == "release" ]]
}

@test "task rejects an unknown flag on a @tasks.Call task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric task build --nope
  [ "$status" -ne 0 ]
  [[ "$output" == *"unknown flag"* ]]
}

@test "task --help describes a @tasks.Call task's flags" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric task build --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"--target"* ]]
  [[ "$output" == *"-d, --dry"* ]]
}

@test "a bare alias dispatches to a @tasks.Call task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric b --target=bare
  [ "$status" -eq 0 ]
  [[ "$output" == "bare" ]]
}

@test "task fails cleanly when two tasks share a name" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/name-collision"

  run_zirric task dup
  [ "$status" -ne 0 ]
  [[ "$output" == *"both use the name \"dup\""* ]]
}
