#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "init creates a Cavefile in an empty project" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-init-fixtures/empty"

  run_zirric init
  [ "$status" -eq 0 ]
  [[ "$output" == *"created Cavefile"* ]]

  [ -f Cavefile ]
  grep -q "@cave.Dependencies()" Cavefile
}

@test "init refuses to overwrite an existing Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-init-fixtures/existing-cavefile"

  run_zirric init
  [ "$status" -ne 0 ]
  [[ "$output" == *"already exists"* ]]

  grep -q "mod existing" Cavefile
}
