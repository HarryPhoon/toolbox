% toolbox-create(1)

## NAME
toolbox\-create - Create a new toolbox container

## SYNOPSIS
**toolbox create** [*options*] *CONTAINER*

## DESCRIPTION

Creates a new toolbox container. You can then use the `toolbox enter` command
to interact with the container at any point.

A toolbox container is an OCI container created from an OCI image. On Fedora
the base image is known as `fedora-toolbox`. If the image is not present
locally, then it is pulled from `registry.fedoraproject.org`.

Toolbox containers and images are tagged with the version of the OS that
corresponds to the content inside them. The user-specific images and the
toolbox containers are prefixed with the name of the base image.

## OPTIONS ##

The following options are understood:

**--image** NAME, **-i** NAME

Change the NAME of the base image used to create the toolbox container. This
is useful for creating containers from custom-built base images.

**--release** RELEASE, **-r** RELEASE

Create a toolbox container for a different operating system RELEASE than the
host.

## EXAMPLES

### Create a toolbox container using the default image matching the host OS

```
$ toolbox create
```

### Create a toolbox container using the default image for Fedora 30

```
$ toolbox create --release f30
```

### Create a custom toolbox container from a custom image

```
$ toolbox create foo --image bar
```


## SEE ALSO

`buildah(1)`, `podman(1)`
