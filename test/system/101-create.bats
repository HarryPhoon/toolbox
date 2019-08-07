#!/usr/bin/env bats

load helpers

@test "Create the default container." {
  run_toolbox -y create
}

@test "Create a container with a valid custom name (whole word)" {
  run_toolbox -y create -c "customname"
}

@test "Try to create a container with a bad custom name (with special characters)" {
  run_toolbox 1 -y create -c "ßpeci@lNam€"
  is "${lines[0]}" "toolbox: invalid argument for '--container'" "Toolbox reports invalid argument for --container"
  #is "${lines[1]}" "Container names must match '[a-zA-Z0-9][a-zA-Z0-9_.-]*'." "Toolbox shows required pattern for naming of containers"
}

@test "Try to create a container with a bad custom name (with a dot)" {
  run_toolbox 1 -y create -c ".custom.name."
  is "${lines[0]}" "toolbox: invalid argument for '--container'" "Toolbox reports invalid argument for --container"
  #is "${lines[1]}" "Container names must match '[a-zA-Z0-9][a-zA-Z0-9_.-]*'." "Toolbox shows required pattern for naming of containers"
}

@test "Try to create a container with a bad custom name (with a hyphen)" {
  run_toolbox 1 -y create -c "-custom-name-"
  is "${lines[0]}" "toolbox: invalid argument for '--container'" "Toolbox reports invalid argument for --container"
  #is "${lines[1]}" "Container names must match '[a-zA-Z0-9][a-zA-Z0-9_.-]*'." "Toolbox shows required pattern for naming of containers"
}

@test "Try to create a container with a bad custom name (with an underline)" {
  run_toolbox 1 -y create -c "_custom_name_"
  is "${lines[0]}" "toolbox: invalid argument for '--container'" "Toolbox reports invalid argument for --container"
  #is "${lines[1]}" "Container names must match '[a-zA-Z0-9][a-zA-Z0-9_.-]*'." "Toolbox shows required pattern for naming of containers"
}

@test "Try to create a container with no name (just a single space)" {
  run_toolbox 1 -y create -c " "
  is "${lines[0]}" "toolbox: invalid argument for '--container'" "Toolbox reports invalid argument for --container"
  #is "${lines[1]}" "Container names must match '[a-zA-Z0-9][a-zA-Z0-9_.-]*'." "Toolbox shows required pattern for naming of containers"
}

@test "Create a container with a custom image (f28)" {
  run_toolbox -y create -i $(get_image_name fedora 28)
}

@test "Create a container with a custom name and image (f29)" {
  run_toolbox -y create -c "customname" -i $(get_image_name fedora 29)
}