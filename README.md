# Build CRD

This repository implements `Build` and `BuildTemplate` custom resources
for Kubernetes, and a controller for making them work.

If you are interested in contributing, see [CONTRIBUTING.md](./CONTRIBUTING.md)
and [DEVELOPMENT.md](./DEVELOPMENT.md).

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
