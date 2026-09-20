#!/usr/bin/env bats

# Runs against the repository's own Cavefile at the repo root, not an
# isolated fixture — unlike the other testing/*.bats suites, which each
# operate on a throwaway copy of a small fixture project via setup_fixture.

load 'test_helper'

teardown() {
  cd "$REPO_ROOT" || return 1
}

@test "task run test succeeds against the repo's own Cavefile" {
  cd "$REPO_ROOT" || return 1

  run_zirric task run test
  [ "$status" -eq 0 ]
}
