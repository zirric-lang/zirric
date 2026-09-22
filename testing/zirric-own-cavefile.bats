#!/usr/bin/env bats

# Runs against the repository's own Cavefile at the repo root, not an
# isolated fixture — unlike the other testing/*.bats suites, which each
# operate on a throwaway copy of a small fixture project via setup_fixture.

load 'test_helper'

teardown() {
  cd "$REPO_ROOT" || return 1
}

@test "test succeeds against the repo's own package" {
  cd "$REPO_ROOT" || return 1

  run_zirric test
  [ "$status" -eq 0 ]
}
