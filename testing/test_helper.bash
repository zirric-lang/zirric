REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ZIRRIC_BIN="${ZIRRIC_BIN:-$REPO_ROOT/zirric}"

# Runs the zirric binary under test, capturing $status/$output/$lines via bats' run.
run_zirric() {
  if [ ! -x "$ZIRRIC_BIN" ]; then
    echo "zirric binary not found at $ZIRRIC_BIN; build it first: go build -o zirric ./cmd/zirric" >&2
    return 1
  fi
  run "$ZIRRIC_BIN" "$@"
}

# Copies fixture_dir into a fresh temp project directory, cds into it, and points
# ZIRRIC_PATH at an isolated temp registry so tests never share state or touch
# the developer's real ~/.zirric.
setup_fixture() {
  local fixture_dir="$1"
  TEST_PROJECT_DIR="$(mktemp -d)"
  cp -R "$fixture_dir"/. "$TEST_PROJECT_DIR"/
  cd "$TEST_PROJECT_DIR" || return 1

  ZIRRIC_HOME_DIR="$(mktemp -d)"
  export ZIRRIC_PATH="$ZIRRIC_HOME_DIR"
}

# Tears down whatever setup_fixture made, and succeeds when it made nothing: a test that skipped before
# setting a fixture up would otherwise be reported as a teardown failure rather than as a skip.
teardown_fixture() {
  cd "$REPO_ROOT" || return 1
  if [ -n "$TEST_PROJECT_DIR" ]; then
    rm -rf "$TEST_PROJECT_DIR"
  fi
  if [ -n "$ZIRRIC_HOME_DIR" ]; then
    rm -rf "$ZIRRIC_HOME_DIR"
  fi
  return 0
}
