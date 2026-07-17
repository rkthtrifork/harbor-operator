# Testing and Verification

## Main Test Targets

```sh
make check
make test
make test-e2e
```

- `make check` runs the normal non-E2E CI baseline: generated drift, lint, tests, and the docs build
- `make test` runs the non-E2E Go test suite
- `make test-e2e` runs the live end-to-end suite against the current Kind cluster

## Generated Assets

When API types, Kubebuilder markers, RBAC, or docs reference content change, regenerate and verify the generated assets:

```sh
make verify-generated
```

This preserves the generated-file diff that existed before the command and fails only when regeneration changes it further.

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
