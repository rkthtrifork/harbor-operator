# Introduction

`harbor-operator` manages Harbor resources through Kubernetes custom resources.

This site is split into introduction material, task-oriented guides, reference
documentation, architecture notes, project knowledge, and contributor guidance.

## Start Here

<div class="grid cards" markdown>

-   :material-compass-outline: __Introduction__

    ---

    Install the operator, connect Harbor, and create your first resources.

    [Get started](quickstart.md)

-   :material-book-open-page-variant-outline: __Reference__

    ---

    Find resource behavior guides, operator configuration, and the generated schema reference.

    [Open Reference](reference/index.md)

-   :material-compass: __Guides__

    ---

    Learn connection patterns, multi-tenancy, lifecycle behavior, upgrades, and troubleshooting.

    [Open Guides](reference/connection-patterns.md)

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

1. [Quickstart](quickstart.md)
2. [Installation](introduction/installation.md)
3. [Concepts](introduction/concepts.md)
4. [Guides](reference/connection-patterns.md)
5. [Reference overview](reference/index.md)
6. [Examples](examples/index.md)
