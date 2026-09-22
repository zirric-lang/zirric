#!/usr/bin/env bats

load 'test_helper'

teardown() {
  teardown_fixture
}

@test "cave new creates a Cavefile in an empty project" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/empty"

  run_zirric cave new --mod code.knabel.dev.example.my_app
  [ "$status" -eq 0 ]
  [[ "$output" == *"created Cavefile"* ]]

  [ -f Cavefile ]
  grep -q "^mod code.knabel.dev.example.my_app$" Cavefile
  grep -q "@cave.Package()" Cavefile
  grep -q "^data MyApp {$" Cavefile
}

@test "cave new needs something to name the package after" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/empty"

  run_zirric cave new
  [ "$status" -ne 0 ]
  [[ "$output" == *"--mod"* ]]
  [ ! -f Cavefile ]
}

@test "cave new names the package after the Git remote" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/empty"
  git init -q .
  git remote add origin git@code.knabel.dev:example/widgets.git

  run_zirric cave new
  [ "$status" -eq 0 ]

  grep -q "^mod code.knabel.dev.example.widgets$" Cavefile
  grep -q '@cave.Git("https://code.knabel.dev/example/widgets")' Cavefile
}

@test "cave new writes every attribute it is given" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/empty"

  run_zirric cave new \
    --mod myapp \
    --version 1.2.3 \
    --language-version '^0.1.0' \
    --git-url https://code.knabel.dev/example/myapp \
    --description 'Widgets and layout' \
    --documentation-url https://example.com/docs
  [ "$status" -eq 0 ]

  grep -q '@cave.Version("1.2.3")' Cavefile
  grep -q '@cave.LanguageVersion("\^0.1.0")' Cavefile
  grep -q '@cave.Git("https://code.knabel.dev/example/myapp")' Cavefile
  grep -q '@cave.Description("Widgets and layout")' Cavefile
  grep -q '@cave.Documentation("https://example.com/docs")' Cavefile
}

@test "what cave new writes is a Cavefile the tooling can read" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/empty"

  run_zirric cave new --mod myapp --description 'Widgets and layout'
  [ "$status" -eq 0 ]

  run_zirric cave describe
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: myapp"* ]]
  [[ "$output" == *"description: Widgets and layout"* ]]
}

@test "cv n is a shorthand for cave new" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/empty"

  run_zirric cv n --mod myapp
  [ "$status" -eq 0 ]
  [ -f Cavefile ]
}

@test "cave new refuses to overwrite an existing Cavefile" {
  setup_fixture "$BATS_TEST_DIRNAME/zirric-cave-new-fixtures/existing-cavefile"

  run_zirric cave new --mod myapp
  [ "$status" -ne 0 ]
  [[ "$output" == *"already exists"* ]]

  grep -q "mod existing" Cavefile
}
