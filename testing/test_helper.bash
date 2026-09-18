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

teardown_fixture() {
  cd "$REPO_ROOT" || return 1
  [ -n "$TEST_PROJECT_DIR" ] && rm -rf "$TEST_PROJECT_DIR"
  [ -n "$ZIRRIC_HOME_DIR" ] && rm -rf "$ZIRRIC_HOME_DIR"
}
