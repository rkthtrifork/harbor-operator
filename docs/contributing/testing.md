# Testing and Verification

## Main Test Targets

```sh
make test
make test-e2e
```

- `make test` runs the non-E2E Go test suite
- `make test-e2e` runs the live end-to-end suite against the current Kind cluster

## Generated Assets

When API types, Kubebuilder markers, RBAC, or docs reference content change, regenerate and verify the generated assets:

```sh
make manifests generate sync-chart generate-docs
```

## Docs Site

Build the docs locally with:

```sh
make docs-build
```

Serve them locally with:

```sh
make docs-serve
```

These commands run through the `squidfunk/mkdocs-material` container image for consistency with CI.
