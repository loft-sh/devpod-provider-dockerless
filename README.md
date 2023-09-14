# Dockerless Provider for DevPod

Dockerless provider is useful for situations where you don't have Docker or Podman
installed, you're on Linux, and you still want to use Devcontainers.

Dockerless provider will use (RootlessKit)[https://github.com/rootless-containers/rootlesskit] for
rootless containers (and `unshare` in case you're already root) to create namespaces.
It will use (Crane)[https://github.com/google/go-containerregistry/#crane] to pull and manage images.

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
