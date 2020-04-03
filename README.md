<img src="data/logo/toolbox-logo-landscape.svg" alt="Toolbox logo landscape" width="800"/>

[![Build Status](https://zuul-ci.org/gated.svg)](https://softwarefactory-project.io/zuul/t/local/builds?project=containers/toolbox)

[Toolbox](https://github.com/containers/toolbox) is a tool that offers a
familiar package based environment for developing and debugging software that
runs fully unprivileged using [Podman](https://podman.io/).

**Ok, this sounds a bit criptic..**

Operating systems that are based on [OSTree](https://ostree.readthedocs.io/en/latest/)
like [CoreOS](https://coreos.fedoraproject.org/) and
[Fedora Silverblue](https://silverblue.fedoraproject.org/) discourage
installation of software on the host (they don't even have a classic package
manager), and instead install software as (or in) containers.

**Hmm, why is that so?**

There are several advantages to this approach including higher reliability,
easier upgrading and easier debugging. Fedora Magazine has nice articles about
both [CoreOS](https://fedoramagazine.org/introducing-fedora-coreos/) and
[Silverblue](https://fedoramagazine.org/what-is-silverblue/).

**OK, what is then Toolbox good for?**

Toolbox is basically a thin layer on top of [Podman](https://podman.io) (which
is a tool for managing [OCI](https://www.opencontainers.org/)-based containers)
providing sane default values when creating/running containers. These
containers then seamlessly integrate with the rest of the operating system
(environmental variables, file paths,..).

**Uh.., what?**

Let's say you want to try to use `ansible` but you don't want to bloat your
system (Fedora Workstation) or you can't really install it comfortably
(Silverblue).

Just create a container with `toolbox create`, enter it with `toolbox enter`
(you should see the '⬢' symbol) and just type `sudo dnf install ansible`.

When the installation is done you should have `ansible` prepared for use,
without affecting the base operating system (if you leave the container,
`ansible` will no longer be available until you return into it).

And because your home folder is automatically mounted in the container, you
can easily execute your already existing ansible playbooks.

When you're done, you can simply delete the container with `toolbox rm` and the
system should be in the same state when you started.

**Sounds good! So, to use Toolbox I need one of those mentioned OSs?**

No, you don't. This tool *doesn't require* using an OSTree based system — it
works equally well if you're running e.g. existing Fedora Workstation or
Server. But if you haven't given Silverblue a try then you definitely should.

Just, please, note that currently the only supported distribution is Fedora and projects around it (eg. Silverblue, CoreOS) but the ultimate goal is to make the tool work even on other distirbution.

## Requirements:

- podman
- flatpak-session-helper (monitoring of key system files)

## Installation

### Fedora Silverblue

Toolbox is installed by default on Fedora Silverblue.

### Fedora

`$ dnf install toolbox`

## How to build from source

To build the project you need to have [Go](https://golang.org/) installed on your system and have it [configured](https://golang.org/doc/install).

**To build Toolbox:**

```
$ git clone git@github.com:containers/toolbox.git / https://github.com/containers/toolbox.git
$ cd toolbox/src
$ go build
```

That's it! Go should pull and build all the necessarry dependencies and build the binary.

**To install Toolbox:**

While in the /src directory:

```
go install
```

Installation using `go build` moves the binary made by `go build` to GOBIN which can be configured to be part of PATH. More about GOBIN is [here](https://golang.org/cmd/go/#hdr-Compile_and_install_packages_and_dependencies).

**To uninstall Toolbox:**
```
go clean
```
This only removes the `toolbox` binary from GOBIN. This method does 
not apply to Toolbox from a package.

## Basic usage

### Create your toolbox container:

```
[user@hostname ~]$ toolbox create
Created container: fedora-toolbox-30
Enter with: toolbox enter
[user@hostname ~]$
```

This will create a container called `fedora-toolbox-<version-id>` based on the version of Fedora you are using.

### Enter the toolbox:

```
[user@hostname ~]$ toolbox enter
⬢[user@toolbox ~]$
```

Notice: The '⬢' symbol may not be present if you are using a custom shell theme.

### Remove a toolbox container:

```
[user@hostname ~]$ toolbox rm <container-name>
[user@hostname ~]$
```

## Distro support

By default, Toolbox creates the container using a **pre-prepared**
[OCI](https://www.opencontainers.org/) image called
`<ID>-toolbox:<VERSION-ID>`, where `<ID>` and `<VERSION-ID>` are taken from the
host's `/usr/lib/os-release`. For example, the default image on a Fedora 30
host would be `fedora-toolbox:30`.

This default can be overridden by the `--image` option in `toolbox create`,
but operating system distributors should provide an adequately configured
default image to ensure a smooth user experience.

Notice: When the '--image' option is used with a *valid* image *but* the image
is not a valid Toolbox image, then the creation of a container will *fail* due
to an unsupported image.

## Image requirements

Toolbox customizes newly created containers in a certain way. This requires
certain tools and paths to be present and have certain characteristics inside
the OCI image.

Tools:
* `getent(1)`
* `id(1)`
* `ln(1)`
* `mkdir(1)`: for hosts where `/home` is a symbolic link to `/var/home`
* `passwd(1)`
* `readlink(1)`
* `rm(1)`
* `rmdir(1)`: for hosts where `/home` is a symbolic link to `/var/home`
* `sleep(1)`
* `test(1)`
* `touch(1)`
* `unlink(1)`
* `useradd(8)`

Paths:
* `/etc/host.conf`: optional, if present not a bind mount
* `/etc/hosts`: optional, if present not a bind mount
* `/etc/krb5.conf.d`: directory, not a bind mount
* `/etc/localtime`: optional, if present not a bind mount
* `/etc/resolv.conf`: optional, if present not a bind mount
* `/etc/timezone`: optional, if present not a bind mount

The image should have `sudo(8)` enabled for users belonging to either the
`sudo` or `wheel` groups, and the group itself should exist. File an
[issue](https://github.com/containers/toolbox/issues/new) if you really need
support for a different group. However, it's preferable to keep this list as
short as possible.

Since Toolbox only works with OCI images that fulfill certain requirements,
it will refuse images that aren't tagged with
`com.github.containers.toolbox="true"` and
`com.github.debarshiray.toolbox="true"` labels. These labels are meant to be
used by the maintainer of the image to indicate that they have read this
document and tested that the image works with Toolbox. You can use the
following snippet in a Dockerfile for this:
```
LABEL com.github.containers.toolbox="true" \
      com.github.debarshiray.toolbox="true"
``` 

## Goals and Use Cases

### High Level Goals

- Provide a CLI convenience interface to run containers (via `podman`) easily
- Support for Developer and Debugging/Management use cases
- Support for multiple distros
    - toolbox package in multiple distros
    - toolbox containers for multiple distros

### Non-Goals - Anti Use Cases

- Supporting multiple container runtimes. `toolbox` will use `podman` exclusively
- Adding significant features on top of `podman`
	- Significant feature requests should be driven into `podman` upstream
- To run containers that aren't tightly integrated with the host
	- i.e. extremely sandboxed containers become specific to the user quickly

### Developer Use Cases

- I’m a developer hacking on source code and building/testing code
    - Most cases: user doesn't need root, rootless containers work fine
    - Some cases: user needs root for testing
- Desktop Development: 
    - developers need things like dbus, display, etc, to be forwarded into the toolbox
- Headless Development:
    - toolbox works properly in headless environments (no display, etc)
- Need development tools like gdb, strace, etc to work

### Debugging/System management Use Cases

- Inspecting Host Processes/Kernel
    - Typically need root access
    - Need bpftrace, strace on host processes to work
		- Ideally even do things like helping get kernel-debuginfo data for the host kernel
- Managing system services
    - systemctl restart foo.service
    - journalctl
- Managing updates to the host
    - rpm-ostree
    - dnf/yum (classic systems)

### Specific environments

- Fedora Silverblue
	- Silverblue comes with a subset of packages and discourages host software changes
		- Users need a toolbox container as a working environment
		- Future: use toolbox container by default when a user opens a shell
- Fedora CoreOS
	- Similar to silverblue, but non-graphical and smaller package set
- RHEL CoreOS
	- Similar to Fedora CoreOS. Based on RHEL content and the underlying OS for OpenShift
	- Need to [use default authfile on pull](https://github.com/coreos/toolbox/pull/58/commits/413f83f7240d3c31121b557bfd55e489fad24489)
    - Need to ensure compatibility with the rhel7/support-tools container 
		- currently not a toolbox image, opportunity for collaboration
	- Alignment with `oc debug node/` (OpenShift)
		- `oc debug node` opens a shell on a kubernetes node
		- Value in having a consistent environment for both `toolbox` in debugging mode and `oc debug node`