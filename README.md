# Dockerless Provider for DevPod

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/devpod-provider-dockerless)

Dockerless provider is useful for situations where you don't have Docker or Podman
installed, you're on Linux, and you still want to use Devcontainers.

Dockerless provider will use [RootlessKit](https://github.com/rootless-containers/rootlesskit) for
rootless containers (and `unshare` in case you're already root) to create namespaces.
It will use [Crane](https://github.com/google/go-containerregistry/#crane) to pull and manage images.

All dependencies are self-contained in the provider binary.

The target is to have simple, reasonably isolated and easy to use container alternatives, without
external runtimes.

Useful use-cases, are easy nested containers or slim, no-deps images.

## Getting started

The provider is available for auto-installation using 

```sh
devpod provider add dockerless
devpod provider use dockerless
```

Follow the on-screen instructions to complete the setup.

Needed variables will be:

- TARGET_DIR

`TARGET_DIR` is where all the images, containers and state are stored.

You only need to specify where you want all of the `dockerless` data will be stored.

## Run it

After the initial setup, just use:

```sh
devpod up .
```

## Run in a container

To run in a container, we need CAP_SYS_ADMIN (needed for the unshare, mount and pivot_root syscalls)
To have custom networking we also need access to /dev/net/tun

We DO NOT require root
A nice way to run it is:

`docker run --rm -ti --cap-add CAP_SYS_ADMIN --device /dev/net/tun --user 1000:1000 build-alpine:latest`

Using the image in the `image` folder.
