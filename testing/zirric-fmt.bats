#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

# The misformatted sources are written here rather than stored as fixtures, so
# that `zirric fmt` run over this repository cannot quietly format them away.
write_unformatted() {
  printf 'mod sample\n\nfn add(a: Int, b: Int) -> Int {\n      const sum = a+b\n  return sum\n}\n' > main.zirr
}

write_formatted() {
  printf 'mod sample\n\nfn add(a: Int, b: Int) -> Int {\n\tconst sum = a + b\n\treturn sum\n}\n' > main.zirr
}

@test "fmt rewrites unformatted sources in place" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  write_unformatted

  run_zirric fmt
  [ "$status" -eq 0 ]

  run grep -qF $'\tconst sum = a + b' main.zirr
  [ "$status" -eq 0 ]
}

@test "fmt also formats the Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'import cave\n\n@cave.Dependencies()\ndata Dependencies {\n      prelude\n}\n' > Cavefile

  run_zirric fmt
  [ "$status" -eq 0 ]

  run grep -qF $'\tprelude' Cavefile
  [ "$status" -eq 0 ]
}

@test "fmt --check exits non-zero before formatting and zero after" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  write_unformatted

  run_zirric fmt --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"main.zirr"* ]]

  run_zirric fmt
  [ "$status" -eq 0 ]

  run_zirric fmt --check
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "fmt --check leaves already formatted sources untouched" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  write_formatted

  run_zirric fmt --check
  [ "$status" -eq 0 ]
}

@test "fmt --list reports paths without writing" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  write_unformatted

  run_zirric fmt --list
  [ "$status" -eq 0 ]
  [[ "$output" == *"main.zirr"* ]]

  run grep -q 'const sum = a+b' main.zirr
  [ "$status" -eq 0 ]
}

@test "fmt rewrites the deprecated => arrow" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'fn f(a) => Int {\n\ta\n}\n' > main.zirr

  run_zirric fmt
  [ "$status" -eq 0 ]

  run grep -qF -- '-> Int' main.zirr
  [ "$status" -eq 0 ]
}

@test "fmt reads stdin and writes the result to stdout" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"

  run bash -c "printf 'fn f() {\n      const x = 1+2\n}\n' | '$ZIRRIC_BIN' fmt --stdin"
  [ "$status" -eq 0 ]
  [[ "$output" == *"const x = 1 + 2"* ]]
}

@test "fmt skips paths excluded by the Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'import cave\n\n@cave.FormattingExcludes(["vendor/**"])\ndata Formatting {}\n' > Cavefile
  mkdir -p vendor
  write_unformatted
  cp main.zirr vendor/dep.zirr

  run_zirric fmt
  [ "$status" -eq 0 ]

  run grep -qF $'\tconst sum = a + b' main.zirr
  [ "$status" -eq 0 ]

  run grep -q 'const sum = a+b' vendor/dep.zirr
  [ "$status" -eq 0 ]
}

@test "fmt --check ignores excluded paths" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'import cave\n\n@cave.FormattingExcludes(["vendor/**"])\ndata Formatting {}\n' > Cavefile
  mkdir -p vendor
  printf 'fn f() {\n      const x = 1+2\n}\n' > vendor/dep.zirr

  run_zirric fmt --check
  [ "$status" -eq 0 ]
}

@test "fmt --no-excludes formats excluded paths too" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'import cave\n\n@cave.FormattingExcludes(["vendor/**"])\ndata Formatting {}\n' > Cavefile
  mkdir -p vendor
  write_unformatted
  cp main.zirr vendor/dep.zirr

  run_zirric fmt --no-excludes
  [ "$status" -eq 0 ]

  run grep -qF $'\tconst sum = a + b' vendor/dep.zirr
  [ "$status" -eq 0 ]
}

@test "fmt refuses to run when the Cavefile is malformed" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'import cave\n\n@cave.Dependencies(\ndata Broken {\n' > Cavefile
  write_unformatted

  run_zirric fmt
  [ "$status" -eq 1 ]
  [[ "$output" == *"malformed"* ]]

  run grep -q 'const sum = a+b' main.zirr
  [ "$status" -eq 0 ]
}

@test "fmt --no-excludes still works with a malformed Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  printf 'import cave\n\n@cave.Dependencies(\ndata Broken {\n' > Cavefile
  write_unformatted

  run_zirric fmt --no-excludes main.zirr
  [ "$status" -eq 0 ]

  run grep -qF $'\tconst sum = a + b' main.zirr
  [ "$status" -eq 0 ]
}

@test "fmt works in a project with no Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-fmt-fixtures/project"
  rm -f Cavefile
  write_unformatted

  run_zirric fmt
  [ "$status" -eq 0 ]

  run grep -qF $'\tconst sum = a + b' main.zirr
  [ "$status" -eq 0 ]
}
