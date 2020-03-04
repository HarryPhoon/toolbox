#!/usr/bin/env bats

load helpers

@test "Create the default container" {
  run_toolbox -y create
}

@test "Create a container with a valid custom name ('not-running')" {
  run_toolbox -y create "not-running"
}

@test "Create a container with a custom image and name ('running';f29)" {
  run_toolbox -y create "running" -i fedora-toolbox:29
}

@test "Try to create a container with invalid custom name" {
  run_toolbox 1 -y create "ßpeci@l.Nam€"
  is "${lines[0]}" "level=fatal msg=\"Container names must match ^[a-zA-Z0-9][a-zA-Z0-9_.-]*$\"" "Toolbox reports invalid name for a container"
}
