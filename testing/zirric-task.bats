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

@test "task run executes a declared task by name" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task run greet
  [ "$status" -eq 0 ]
}

@test "task run fails cleanly for an unknown task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task run nope
  [ "$status" -ne 0 ]
  [[ "$output" == *"\"nope\" not found"* ]]
}

@test "x is a shorthand for task run" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric x greet
  [ "$status" -eq 0 ]
}

@test "x resolves task aliases" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric x hi
  [ "$status" -eq 0 ]
}

@test "task run ignores trailing arguments rather than failing" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric task run greet --dry extra-arg
  [ "$status" -eq 0 ]
}

@test "task run passes typed flag values to a @tasks.Call task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric task run build --target=release
  [ "$status" -eq 0 ]
  [[ "$output" == "release" ]]
}

@test "task run rejects an unknown flag on a @tasks.Call task" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric task run build --nope
  [ "$status" -ne 0 ]
  [[ "$output" == *"unknown flag"* ]]
}

@test "task run --help describes a @tasks.Call task's flags" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric task run build --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"--target"* ]]
  [[ "$output" == *"-d, --dry"* ]]
}

@test "x dispatches to a @tasks.Call task by alias" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-call-task"

  run_zirric x b --target=via-x
  [ "$status" -eq 0 ]
  [[ "$output" == "via-x" ]]
}

@test "task run fails cleanly when two tasks share a name" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/name-collision"

  run_zirric task run dup
  [ "$status" -ne 0 ]
  [[ "$output" == *"both use the name \"dup\""* ]]
}
