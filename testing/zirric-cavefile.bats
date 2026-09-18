#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "cavefile prints yaml by default" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/no-deps"

  run_zirric cavefile
  [ "$status" -eq 0 ]
  [[ "$output" == "name:"* ]]
}

@test "cavefile -o json prints json" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/no-deps"

  run_zirric cavefile -o json
  [ "$status" -eq 0 ]
  [[ "$output" == "{"* ]]
  [[ "$output" == *'"name"'* ]]
}

@test "cavefile --output yaml is equivalent to the default" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/no-deps"

  run_zirric cavefile --output yaml
  [ "$status" -eq 0 ]
  [[ "$output" == "name:"* ]]
}

@test "cavefile rejects an unsupported output format" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/no-deps"

  run_zirric cavefile -o toml
  [ "$status" -ne 0 ]
  [[ "$output" == *"unsupported --output"* ]]
}

@test "cavefile lists declared dependencies" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/with-deps"

  run_zirric cavefile -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"name": "io"'* ]]
}

@test "cavefile shows the kind and version of each dependency" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/with-deps"

  run_zirric cavefile -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"kind": "stdlib"'* ]]
  [[ "$output" == *'"kind": "git"'* ]]
  [[ "$output" == *'"version": "latest"'* ]]
}

@test "cavefile shows a git dependency's declared version" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cavefile-fixtures/with-version"

  run_zirric cavefile -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"name": "example"'* ]]
  [[ "$output" == *'"kind": "git"'* ]]
  [[ "$output" == *'"version": "^1.2.3"'* ]]
}

@test "cavefile lists declared tasks" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric cavefile -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"name": "greet"'* ]]
  [[ "$output" == *'"kind": "exec"'* ]]
  [[ "$output" == *'"exec": "greet.zirr"'* ]]
  [[ "$output" == *'"aliases"'* ]]
}
