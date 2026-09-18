#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "install with no declared dependencies still installs the project" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-install-fixtures/no-deps"

  run_zirric install
  [ "$status" -eq 0 ]
  [[ "$output" == *"✓ $(basename "$TEST_PROJECT_DIR")"* ]]
}

@test "install prints each declared dependency by its alias" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-install-fixtures/stdlib-dep"

  run_zirric install
  [ "$status" -eq 0 ]
  [[ "$output" == *"✓ io ("* ]]
}

@test "install lists dependencies before the project itself" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-install-fixtures/stdlib-dep"

  run_zirric install
  [ "$status" -eq 0 ]

  # The project's own line must be the last one printed.
  last_line="${lines[${#lines[@]} - 1]}"
  [[ "$last_line" == *"✓ $(basename "$TEST_PROJECT_DIR")"* ]]
}
