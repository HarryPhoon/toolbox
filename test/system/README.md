# System tests

These tests are built on BATS (Bash Automated Testing System). They are strongly influenced by libpod project.

The tests are meant to ensure that toolbox's functionality remains stable throughout updates of both Toolbox and Podman/libpod.

## Structure

The tests are divided to sections according to the number of required steps to achieve the goal and the overall difficulty to get the task done.

- **Basic Tests**
  - [x] output version number (Toolbox + Podman)
  - [x] show help screen when no command is given
  - [x] show help screen when full flag is given
  - [x] show help screen when short flag is given
  - [x] show help screen when called
  - [x] list containers (no present)
  - [x] list default container and default image
  - [x] list containers (some present; different name patterns)
  - [x] list images (no present)
  - [x] list images (some present; different name patterns)
  - [x] create the default container
  - [ ] execute a full cycle of toolbox with default settings (create, list, run, enter, exit, delete)

- **Advanced Tests**
  - [x] create a container with a custom name
  - [x] create a container from a custom image
  - [x] create a container from a custom image with a custom name
  - [x] remove a specific container
  - [x] try to remove nonexistent container
  - [x] try to remove a running container
  - [x] remove all containers
  - [x] try to remove all containers (running)
  - [x] remove a specific image
  - [x] remove all images
  - [ ] run a command inside of an existing image

- **Complex Tests**
  - [ ] create several containers with various configuration and then list them
  - [ ] create several containers and hop between them (series of enter/exit)
  - [ ] create a container, enter it, run a series of basic commands (id, whoami, dnf, top, systemctl,..)
  - [ ] enter a container and test basic set of networking tools (ping, traceroute,..)

The list of tests is stil rather basic. We **welcome** PRs with test suggestions or even their implementation.

## Convention

- All tests that start with *Try to..* expect non-zero return value.