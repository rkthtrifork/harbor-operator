# Documentation

## Sources and generated reference

Hand-written pages explain operator behavior, Harbor semantics, examples, and operational constraints. The schema reference in `docs/reference/api.md` is generated from the Go API types and Kubebuilder markers with `crd-ref-docs`; do not edit it directly.

Regenerate the API reference with:

```sh
make generate-api-reference
```

## Build locally

Build the complete site, including a regenerated API reference:

```sh
make docs-build
```

Serve it locally with:

```sh
make docs-serve
```

Both targets use the pinned MkDocs Material container configured in the Makefile. Navigation and site configuration live in `hack/mkdocs.yml`.

## Publishing

The `docs` workflow builds documentation changes on pull requests and publishes the site from `main` through GitHub Pages. Published documentation follows `main`; use the Markdown at a release tag or commit for historical behavior.

Generated reference drift is part of `make verify-generated` and the normal CI baseline. See [Testing and Verification](testing.md) for the complete check vocabulary.
