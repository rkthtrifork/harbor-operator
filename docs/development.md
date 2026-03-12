# Development

## Documentation Structure

The documentation is split into two layers:

- hand-written guides in `docs/crds/`
- generated schema reference in `docs/reference/api.md`

The hand-written pages explain operator behavior and Harbor-specific semantics. The generated page reflects the API types and Kubebuilder markers.

## Reference Generator

The generated API reference uses `crd-ref-docs`.

Regenerate it with:

```sh
make generate-docs
```

## Local Docs Tooling

Local docs commands:

```sh
make docs-build
make docs-serve
```

These targets run through the `squidfunk/mkdocs-material` image, so the docs workflow is the same locally and in CI.

Material features enabled in this repo include:

- light and dark mode palette toggle
- edit and view actions linked to GitHub
- git-based last updated dates
- tabbed top-level navigation

## Drift Verification

Generated assets are verified in CI. That includes:

- generated code
- CRDs
- synced chart RBAC and chart CRDs
- generated API reference docs

The verify workflow effectively checks:

```sh
make manifests generate sync-chart generate-docs
```

## GitHub Pages

The site is built with MkDocs Material and published from `main` with the `docs` workflow.

Repository settings should use GitHub Pages with `GitHub Actions` as the source.
