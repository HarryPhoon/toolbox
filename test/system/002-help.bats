#!/usr/bin/env bats

load helpers

@test "Show usage screen when no command is given" {
  run_toolbox 1
  is "${lines[-1]}" "Error: subcommand is required" "Last line of the usage output"
}
