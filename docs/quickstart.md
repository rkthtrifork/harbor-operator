# Quickstart

This operator assumes Harbor already exists. The local development flow uses Kind only as a convenient way to run Harbor and the operator together.

## Start a Local Stack

For the fastest local cluster:

```sh
make kind-up
```

For a local cluster with Cilium and Hubble:

```sh
make kind-up-cilium
```

Both targets:

- create a Kind cluster
- install Traefik
- install Harbor from the official Helm chart
- build and deploy the operator

## Core Workflow

1. Start the stack with `make kind-up` or `make kind-up-cilium`
2. Change code or CRD types
3. Redeploy with `make kind-refresh`
4. Exercise the operator with sample CRs or focused manifests

## Refresh the Operator

After changing controller code, CRD types, generated assets, or chart wiring:

```sh
make kind-refresh
```

That rebuilds the image, reloads it into Kind, regenerates assets, reapplies CRDs, and redeploys the operator.

## Apply Sample Resources

```sh
make apply-samples
```

To remove Harbor custom resources again:

```sh
make delete-crs
```

## Useful Commands

```sh
make kind-refresh
make kind-redeploy
make test
make test-e2e
```

- `kind-refresh` redeploys the operator onto the current Kind cluster
- `kind-redeploy` resets operator-managed state and deploys again
- `test` runs the non-E2E test suite
- `test-e2e` runs the live end-to-end suite against the current Kind cluster

## Build the Docs

Generate the CRD API reference:

```sh
make generate-docs
```

Build the site:

```sh
make docs-build
```

Serve it locally:

```sh
make docs-serve
```

The docs site always runs through the `squidfunk/mkdocs-material` container image.
