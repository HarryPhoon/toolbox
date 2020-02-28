#!/usr/bin/env bats

load helpers

@test "Run list with zero containers and two images" {
  run_toolbox list
  is "${#lines[@]}" "0" "Expected number of lines of the output is 0 (Img: 0 + Spc: 0 + Cont: 0)"
}

@test "Run list with zero containers (-c flag)" {
  run_toolbox list -c
  is "$output" "" "Output of list should be blank"
}

@test "Run list with two images (-i flag)" {
  run_toolbox list -i
  is "${#lines[@]}" "0" "Expected number of lines of the output is 0"
}
