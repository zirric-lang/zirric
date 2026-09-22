#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "cave install with no declared dependencies still installs the project" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-install-fixtures/no-deps"

  run_zirric cave install
  [ "$status" -eq 0 ]
  [[ "$output" == *"✓ install_no_deps"* ]]
}

@test "cave install prints each declared dependency by its alias" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-install-fixtures/stdlib-dep"

  run_zirric cave install
  [ "$status" -eq 0 ]
  [[ "$output" == *"✓ io ("* ]]
}

@test "cave install lists dependencies before the project itself" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-install-fixtures/stdlib-dep"

  run_zirric cave install
  [ "$status" -eq 0 ]

  # The project's own line must be the last one printed.
  last_line="${lines[${#lines[@]} - 1]}"
  [[ "$last_line" == *"✓ stdlib_dep"* ]]
}

@test "cv i is a shorthand for cave install" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-install-fixtures/no-deps"

  run_zirric cv i
  [ "$status" -eq 0 ]
  [[ "$output" == *"✓ install_no_deps"* ]]
}
