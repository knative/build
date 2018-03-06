# Build CRD

This repository implements `Build` and `BuildTemplate` custom resources
for Kubernetes, and a controller for making them work.

If you are interested in contributing, see [CONTRIBUTING.md](./CONTRIBUTING.md)
and [DEVELOPMENT.md](./DEVELOPMENT.md).

## Objective

Kubernetes is emerging as the predominant (if not de facto) container
orchestration layer.  It is also quickly becoming the foundational layer on top
of which folks are building higher-level compute abstractions (PaaS, FaaS).
However, many of these higher-level compute abstractions don't take containers
(the atom of Kubernetes), many start from the user's source and have managed
build processes.

The aim of this project isn't to be a complete standalone product that folks use
directly (e.g. as a CI/CD replacement), but a building block to facilitate the
expression of builds to be run on-cluster.

## Getting Started

You can install the latest release of the Build CRD by running:
```shell
kubectl create -f https://storage.googleapis.com/build-crd/latest/release.yaml
```

## Terminology and Conventions

* [Builds](./builds.md)
* [Build Templates](./build-templates.md)
* [Builders](./builder-contract.md)
* [Authentication](./cmd/creds-init/README.md)
