% toolbox-run(1)

## NAME
toolbox\-run - Run a command in an existing toolbox container

## SYNOPSIS
**toolbox run** [*options*] *CONTAINER* [*COMMAND* [*args* ...]]

## DESCRIPTION

Runs a command inside an existing toolbox container. The container should have
been created using the `toolbox create` command.

If the **--release | -r** option is specified, then the CONTAINER does not have
to be specified.

A toolbox container is an OCI container. Therefore, `toolbox run` is analogous
to a `podman start` followed by a `podman exec`.

On Fedora the toolbox containers are tagged with the version of the OS that
corresponds to the content inside them. Their names are prefixed with the name
of the base image.

## OPTIONS ##

The following options are understood:

**--default**, **-d**

Run command inside a toolbox container using the default image matching the
host OS.

**--release** RELEASE, **-r** RELEASE

Run command inside a toolbox container for a different operating system
RELEASE than the host.

**--notty** **-n**

Run command inside a toolbox container but do not allocate a tty. This
is mostly useful for running commands from dmenu, rofi and so on.

## EXAMPLES

### Run ls inside a toolbox container using the default image matching the host OS

```
$ toolbox run -d ls -la
```

### Run emacs inside a toolbox container using the default image for Fedora 30

```
$ toolbox run --release f30 emacs
```

### Run uptime inside a custom toolbox container using a custom image

```
$ toolbox run foo uptime
```

## SEE ALSO

`buildah(1)`, `podman(1)`, `podman-exec(1)`, `podman-start(1)`
