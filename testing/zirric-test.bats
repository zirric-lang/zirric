#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "test runs the stdlib runner when no task declares otherwise" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-test-fixtures/untasked"

  run_zirric test
  [ "$status" -eq 0 ]
  [[ "$output" == *"TAP version"* ]]
  [[ "$output" == *"untasked.thing._t.testAdd"* ]]
  [[ "$output" == *"PASSED"* ]]
}

@test "test exits non-zero when a test fails" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-test-fixtures/failing"

  run_zirric test
  [ "$status" -ne 0 ]
  [[ "$output" == *"FAILED"* ]]
}

@test "test runs a declared test task instead" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-test-fixtures/tasked"

  run_zirric test
  [ "$status" -eq 0 ]
  [[ "$output" == *"the task ran"* ]]
  [[ "$output" != *"TAP version"* ]]
}
