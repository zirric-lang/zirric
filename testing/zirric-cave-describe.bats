#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "cave describe prints yaml by default" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/no-deps"

  run_zirric cave describe
  [ "$status" -eq 0 ]
  [[ "$output" == "package:"* ]]
}

@test "cave describe -o json prints json" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/no-deps"

  run_zirric cave describe -o json
  [ "$status" -eq 0 ]
  [[ "$output" == "{"* ]]
  [[ "$output" == *'"name"'* ]]
}

@test "cave describe --output yaml is equivalent to the default" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/no-deps"

  run_zirric cave describe --output yaml
  [ "$status" -eq 0 ]
  [[ "$output" == "package:"* ]]
}

@test "cave describe rejects an unsupported output format" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/no-deps"

  run_zirric cave describe -o toml
  [ "$status" -ne 0 ]
  [[ "$output" == *"unsupported --output"* ]]
}

@test "cave describe lists declared dependencies" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/with-deps"

  run_zirric cave describe -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"name": "io"'* ]]
}

@test "cave describe shows the kind and version of each dependency" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/with-deps"

  run_zirric cave describe -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"kind": "stdlib"'* ]]
  [[ "$output" == *'"kind": "git"'* ]]
  [[ "$output" == *'"version": "latest"'* ]]
}

@test "cave describe shows a git dependency's declared version" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/with-version"

  run_zirric cave describe -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"name": "example"'* ]]
  [[ "$output" == *'"kind": "git"'* ]]
  [[ "$output" == *'"version": "^1.2.3"'* ]]
}

@test "cave describe lists declared tasks" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-task-fixtures/with-task"

  run_zirric cave describe -o json
  [ "$status" -eq 0 ]
  [[ "$output" == *'"name": "greet"'* ]]
  [[ "$output" == *'"kind": "exec"'* ]]
  [[ "$output" == *'"exec": "greet.zirr"'* ]]
  [[ "$output" == *'"aliases"'* ]]
}

@test "cv is a shorthand for cave" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-describe-fixtures/no-deps"

  run_zirric cv describe
  [ "$status" -eq 0 ]
  [[ "$output" == "package:"* ]]
}
