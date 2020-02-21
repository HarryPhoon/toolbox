#!/usr/bin/env bats

load helpers

@test "Show usage screen when no command is given" {
  run_toolbox 1
  is "${lines[-1]}" `Use "toolbox [command] --help" for more information about a command.` "Last line of the usage output"
}
