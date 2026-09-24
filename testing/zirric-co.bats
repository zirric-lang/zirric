#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "a program where every routine is waiting says so, and names them" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/deadlock"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  [[ "$output" == *"deadlock: every routine is waiting"* ]]
  [[ "$output" == *"routine #0"* ]]
}

@test "sending on a closed channel stops the program" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/send-on-closed"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  [[ "$output" == *"send on a closed channel"* ]]
}

@test "closing a closed channel stops the program" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/close-of-closed"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  [[ "$output" == *"close of a closed channel"* ]]
}

@test "spawning into a scope that has ended stops the program" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/spawn-after-scope"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  [[ "$output" == *"the scope has already finished"* ]]
}

@test "selecting over no channels stops the program" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/empty-select"

  run_zirric run main.zirr
  [ "$status" -ne 0 ]
  [[ "$output" == *"at least one channel"* ]]
}

@test "a routine waiting on standard input does not stop the timer" {
  # Standard input stays open and delivers nothing, so the ticks can only happen while the reader waits.
  # The program stops itself at the third one, so nothing here races its startup.
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/spinner"

  run bash -c "'$ZIRRIC_BIN' run main.zirr < <(sleep 5)"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ticked 3 times while the reader waited"* ]]
  [[ "$output" != *"input ended"* ]]
}

@test "cancelling a routine that is waiting for input loses none of it" {
  # A read already handed to the host cannot be taken back, so the stream owns its reads.
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/cancel-keeps-input"

  # The key is written only once the program reports the reader is gone, so nothing races its startup.
  # The fifo is opened read-write because opening one end alone blocks until the other appears.
  # fd 9 rather than the usual 3, which bats keeps for its own output.
  mkfifo keys
  exec 9<> keys
  "$ZIRRIC_BIN" run main.zirr < keys > out.txt 2>&1 &
  local pid=$!
  local waited=0
  until grep -q "scope returned" out.txt 2>/dev/null; do
    sleep 0.05
    waited=$((waited + 1))
    [ "$waited" -lt 200 ] || break
  done
  printf 'a' >&9
  exec 9>&-
  wait "$pid"
  status=$?
  output="$(cat out.txt)"

  [ "$status" -eq 0 ]
  [[ "$output" == *"main read: a"* ]]
  [[ "$output" != *"the cancelled reader kept going"* ]]
}

@test "cancelling a routine that is waiting for input ends its scope at once" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-co-fixtures/cancel-does-not-wait"

  # Process substitution rather than a pipeline, so what is timed is the program and not the shell waiting for the writer.
  run bash -c "start=\$SECONDS; '$ZIRRIC_BIN' run main.zirr < <(sleep 5; printf 'a'); echo \"elapsed=\$((SECONDS - start))s\""
  [ "$status" -eq 0 ]
  [[ "$output" == *"done"* ]]
  # It would take the full five seconds if the scope waited for the abandoned read.
  [[ "$output" == *"elapsed=0s"* || "$output" == *"elapsed=1s"* || "$output" == *"elapsed=2s"* ]]
}
