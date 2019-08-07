#!/usr/bin/env bats

load helpers

function setup() {
  setup_with_one_container
}

@test "Remove a specific container (called fedora-2)" {
  create_toolbox 2 fedora
  run_toolbox rm fedora-2
  [ "$output" = "" ]
}

@test "Try to remove a nonexistent container" {
  local todelete="nonexistentcontainer"
  run_toolbox 1 rm "$todelete"
  is "$output" "toolbox: failed to inspect $todelete" "Toolbox should fail with: no such container"
}

@test "Try to remove a running container (called fedora-1)" {
  create_toolbox 1 fedora
  run_toolbox enter -c fedora-1
  run exit
  run_toolbox 1 rm fedora-1
  is "$output" "toolbox: failed to remove container fedora-1" "Toolbox should fail to remove the container"
}

@test "Remove all containers (3 present)" {
  create_toolbox 3 fedora
  run_toolbox rm --all
  is "$output" "" ""
}

@test "Try to remove all containers (running containers)" {
  create_toolbox 2 fedora
  run_toolbox enter -c fedora-1
  run_toolbox exit
  run_toolbox enter -c fedora-2
  run_toolbox exit
  run_toolbox 1 rm --all
}

@test "Remove a specific image" {
  run_toolbox rmi "$TOOLBOX_DEFAULT_IMAGE"
}

@test "Try to remove a nonexistent image" {
  local todelete="nonexistentimage"
  run_toolbox rmi "$todelete"
}

@test "Remove all images" {
  get_images 3
  run_toolbox rmi --all
}

@test "Try to remove all images with present containers" {
  get_images 3
  create_toolbox 2 fedora
  run_toolbox 1 rmi --all
}