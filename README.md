# Tyk Gateway Custom Go Plugins

### Description

This project is an environment for writing, compiling and bundling Golang plugins for the Tyk Gateway.

### Quickstart

Follow these [instructions](https://tyk.io/docs/nightly/plugins/get-started-plugins/).

Alternatively, you can watch our video Quickstart [here](https://www.youtube.com/watch?v=2AsSWZRZW24).

### Dependencies

- Golang
- Make
- Docker
- Docker Compose

### Relevant Documentation

- [Native Golang Plugins](https://pkg.go.dev/plugin)
- [Tyk Custom Plugins](https://tyk.io/docs/plugins/)
- [Tyk Golang Plugins](https://tyk.io/docs/plugins/supported-languages/golang/)
- [Tyk Authentication Plugins](https://tyk.io/docs/plugins/auth-plugins/)
- [Tyk Authentication Plugin ID Extractor](https://tyk.io/docs/plugins/auth-plugins/id-extractor/)
- [Tyk OAuth 2.0](https://tyk.io/docs/basic-config-and-security/security/authentication-authorization/oauth-2-0/)
- [Tyk Plugin Bundles](https://tyk.io/docs/plugins/how-to-serve-plugins/plugin-bundles/)
- [Tyk Docker Pro Demo](https://tyk.io/docs/tyk-on-premises/docker/docker-pro-demo/)

## Getting Started

To get started, make sure you have Go installed locally on your machine. Visit https://go.dev/doc/install to download
the latest version of Go and for instructions how to install it for your operating system.

Alternatively if on Ubuntu >= 21.04:

```shell
$ sudo snap install go --classic
```
or if on MacOS with [Homebrew](https://brew.sh/):
```shell
$ brew install go
```
Verify Go is installed on your machine by running in a terminal:
```shell
$ go version
go version go1.17.4 linux/amd64
```
You will also need `make` to run project commands.

On Ubuntu:
```shell
$ sudo apt-get install -y build-essential
```

On MacOS with Homebrew:
```shell
$ brew install make
```

Verify `make` is installed on your machine by running in a terminal:
```shell
$ make --version
GNU Make 4.3
Built for x86_64-pc-linux-gnu
Copyright (C) 1988-2020 Free Software Foundation, Inc.
License GPLv3+: GNU GPL version 3 or later <http://gnu.org/licenses/gpl.html>
This is free software: you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.
```

This project uses [tyk-pro-docker-demo](https://github.com/TykTechnologies/tyk-pro-docker-demo) 
as a local development environment to test and validate the Go authentication plugin, so we will also require 
[Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) 
to be installed on your machine.

Verify Docker and Docker Compose are installed by running in a terminal:
```shell
$ docker --version
Docker version 20.10.11, build dea9396
$ docker-compose --version
docker-compose version 1.29.2, build 5becea4c
```

## Building the Go Plugin

A specific of Tyk Golang plugins is that they need to be built using exactly the same Tyk binary as the one to be 
installed. In order to make it work, we provide a special Docker image, which we internally use for building our
official binaries too. These Docker images can be found at https://hub.docker.com/r/tykio/tyk-plugin-compiler.

Therefore, it is imperative that the version of the `tyk-plugin-compiler` that you use must match the version of 
Tyk Gateway you are using, e.g., `tykio/tyk-plugin-compiler:v4.0.0` for `tykio/tyk-gateway:v4.0.0`

You can set version, by setting TYK_VERSION environment variable, like: `TYK_VERSION=v4.0.0`

To build the plugin using the `tyk-plugin-compiler`, run the following command in a terminal:
```shell
$ TYK_VERSION=v4.2.1 make build
```

This command will run the plugin compiler and create a Go plugin called `CustomGoPlugin.so` 
which can be found in `tyk/middleware/CustomGoPlugin.so` after it successfully builds. This `.so` file can be loaded 
into Tyk Gateway as a custom plugin directly from the filesystem, but in a production setting, it is strongly recommended to 
load the plugin as a [plugin bundle](https://tyk.io/docs/plugins/how-to-serve-plugins/plugin-bundles/).

The `make build` command will also restart
Tyk Gateway if the container is running so that any changes made to the plugin will be applied after being built. See below
for more background on updating Go plugins.

## Deploying the Go Plugin

In production environments, it is strongly recommended to deploy your Tyk custom plugin
as a [plugin bundle](https://tyk.io/docs/plugins/how-to-serve-plugins/plugin-bundles/).

A plugin bundle is a ZIP file that contains your custom middleware files and its associated configuration block
(the `custom_middleware` block). The idea behind plugin bundles is to enhance the process of attaching and loading custom
middleware. It allows you to avoid duplicating the content of the `custom_middleware` section onto each of your APIs definitions,
which is still possible if you do not want to support a bundle server within your global Tyk setup.

Tyk provides a bundler tool to generate plugin bundles. Please note that the generated bundles must be served using your
own web server.
See [Downloading and Updating Bundles](https://tyk.io/docs/plugins/how-to-serve-plugins/plugin-bundles/#downloading-and-updating-bundles)
for more documentation.

To run the bundler tool and generate a plugin bundle, run the following command in a terminal:
```shell
$ make bundle
```

This will create a production-ready plugin bundle that can be found at `tyk/bundle/bundle.zip`.

## Updating the Go Plugin

Loading an updated version of your plugin require one of the following actions:

- An API reload with a NEW path or file name of your `.so` file with the plugin. You will need to update the API spec
- section `"custom_middleware"`, specifying a new value for the `"path"` field of the plugin you need to reload.
- Tyk main process reload. This will force a reload of all Golang plugins for all APIs.

In this project, we will be loading the plugin through the filesystem for development purposes, but it is strongly
recommended to use the plugin bundles for production environments.

If a plugin is loaded as a bundle and you need to update it you will need to update your API spec with new `.zip` file
name in the `"custom_middleware_bundle"` field. Make sure the new `.zip` file is uploaded and available via the bundle
HTTP endpoint before you update your API spec.

### Open Telemetry and Troubleshooting
The **Custom Go Plugin repository** now deploys [Open Telemetry with Jaeger](https://tyk.io/docs/product-stack/tyk-gateway/advanced-configurations/distributed-tracing/open-telemetry/otel_jaeger/) by default to enhance API observability and troubleshooting experience. You can visit the **Jaeger Dashboard** at [http://localhost:16686/](http://localhost:16686/). **It is very important to note that the support for Open Telemetry with the Tyk Gateway is only avaiable for versions `v5.2.0+`.**

To get started, please review our documentation on [Open Telemetry Overview](https://tyk.io/docs/product-stack/tyk-gateway/advanced-configurations/distributed-tracing/open-telemetry/open-telemetry-overview/). 

You can refer to our official documentation on [How to instrument Custom Go Plugins with OpenTelemetry](https://tyk.io/docs/product-stack/tyk-gateway/advanced-configurations/plugins/otel-plugins/).

To stand up, oTel example: 
```shell
$ make up-otel build
```

To stand up, oTel example in OSS:
```shell
$ make up-oss-otel build
```

### Examples
- [Open Telemetry Instrumentation](plugins/otel-instrumentation/)

## Project Lifecycle Makefile Commands

To build the project and bring up your local instance of Tyk, run in a terminal:
```shell
$ make
```

To build the Go plugin and restart the Tyk Gateway if its currently running, run in a terminal:
```shell
$ make build
```

To run the Tyk bundler tool and generate a production plugin bundle, run in a terminal:
```shell
$ make bundle
```

To clean ephemeral project files (including built plugins), run in a terminal:
```shell
$ make clean
```

To bring up the Docker containers running Tyk, run in a terminal:
```shell
$ make up
```

To bring down the Docker containers running Tyk, run in a terminal:
```shell
$ make down
```

To get logs from the Docker containers running Tyk, run in a terminal:
```shell
$ make logs
```

To get the current status of the Docker containers running Tyk, run in a terminal:
```shell
$ make status
```


## Dockerfile

To create a custom Docker image with the custom plugin loaded, run:
```bash
$ docker buildx build --platform linux/amd64 -t sedkis/tyk-gateway-jwt-custom-auth:v0.5 --load .
```

This will create an AMD image and load it into local Docker repo.

To build and push to remote:

```bash
$ docker buildx build --platform linux/amd64 -t sedkis/tyk-gateway-jwt-custom-auth:v0.5 --push .
```
