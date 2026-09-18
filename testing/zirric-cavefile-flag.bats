#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "task lists tasks from the default Cavefile without the flag" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-flag-fixtures/with-alt"

  run_zirric task
  [ "$status" -eq 0 ]
  [[ "$output" == *"default-task"* ]]
  [[ "$output" != *"alt-task"* ]]
}

@test "--cavefile overrides which Cavefile is used" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-flag-fixtures/with-alt"

  run_zirric --cavefile Cavefile.alt task
  [ "$status" -eq 0 ]
  [[ "$output" == *"alt-task"* ]]
  [[ "$output" != *"default-task"* ]]
}

@test "--cavefile runs a task from the overridden Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-flag-fixtures/with-alt"

  run_zirric --cavefile Cavefile.alt task run alt-task
  [ "$status" -eq 0 ]
}

@test "--cavefile fails cleanly for a missing path" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-flag-fixtures/with-alt"

  run_zirric --cavefile does-not-exist.Cavefile task
  [ "$status" -ne 0 ]
  [[ "$output" == *"cavefile not found"* ]]
}

@test "--cavefile placed after the task name has no effect on task dispatch" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-flag-fixtures/with-alt"

  run_zirric task run alt-task --cavefile Cavefile.alt
  [ "$status" -ne 0 ]
  [[ "$output" == *"alt-task"*"not found"* ]]
}
