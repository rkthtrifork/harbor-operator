# Introduction

`harbor-operator` manages Harbor resources through Kubernetes custom resources.

This site is split into introduction material, API documentation, worked examples, and contributor guidance.

## Start Here

<div class="grid cards" markdown>

-   :material-compass-outline: __Introduction__

    ---

    Start locally, understand the model, and find the main operator concepts.

    [Open introduction](index.md)

-   :material-book-open-page-variant-outline: __API__

    ---

    Read resource guides and generated schema reference in one place.

    [Open API section](reference/index.md)

-   :material-test-tube: __Examples__

    ---

    Start from sample manifests for common tasks such as connections, projects, and robot accounts.

    [Open examples](examples/index.md)

-   :material-source-pull: __Contributing__

    ---

    Find the local workflow, testing expectations, and docs/publishing notes.

    [Open contributing docs](contributing/index.md)

</div>

## Documentation Model

The hand-written pages explain how the operator behaves:

- how a resource maps to Harbor
- ownership and deletion semantics
- examples and operational notes

The generated API reference documents the schema from the Go API types and Kubebuilder markers:

- fields and types
- defaults
- enums
- validation rules

## Documentation Versioning

The published docs site tracks the current `main` branch only.

If you need docs for an older release or historical behavior, check out the relevant git tag or commit in the repository and read the Markdown files directly or run:

```sh
make docs-build
```

## Suggested Reading Order

1. [Getting Started](quickstart.md)
2. [Installation](introduction/installation.md)
3. [Concepts](introduction/concepts.md)
4. [API overview](reference/index.md)
5. [Examples](examples/index.md)
