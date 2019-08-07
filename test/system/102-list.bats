#!/usr/bin/env bats

load helpers

function setup() {
  setup_with_one_container
}


@test "Run list with zero containers and zero images" {
  remove_images
  run_toolbox 1 list
  is "$output" "" "Output of list should be blank"
}

@test "Run list with zero containers (-c flag)" {
  run_toolbox 1 list -c
  is "$output" "" "Output of list should be blank"
}

@test "Run list with zero images (-i flag)" {
  remove_images
  run_toolbox 1 list -i
  is "$output" "" "Output of list should be blank"
}

@test "Run list with 1 default container and 1 default image" {
  run_toolbox list
  is "${lines[1]}" "IMAGE ID" "Header for images"
  is "${lines[2]}" ".*registry.fedoraproject.org" "Default image"
  is "${lines[5]}" "CONTAINER ID" "Header for containers"
  is "${lines[6]}" "registry.fedoraproject.org" "Default container"
}

@test "Run list with 3 containers (-c flag)" {
  create_toolbox 3 fedora
  run_toolbox list -c
  for i in $(seq 5 7); do
    is "${lines[$i]}" ".*fedora-[$i-4] \+" "One of the containers"
  done
}

@test "Run list with 3 images (-i flag)" {
  get_images 3
  run_toolbox list -i
  for i in $(seq 2 4); do
    is "${lines[$i]}" ".*registry.fedoraproject.org/f$[33-$i] \+" "One of the images"
  done
}