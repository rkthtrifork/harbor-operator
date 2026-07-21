# Contributing

Changes should preserve the operator's Kubernetes API, Harbor semantics, generated assets, documentation, and delivery artifacts as one coherent contract.

## Before making changes

- Use the Go types and Kubebuilder markers under `api/v1alpha1` as the source of the public Kubernetes API.
- Use `hack/harbor-openapi.yaml` as the checked-in reference for Harbor API semantics.
- Refresh that reference with `make update-harbor-openapi` when a change depends on newer upstream Harbor API details.
- Do not hand-edit generated DeepCopy code, CRDs, RBAC, chart copies of generated assets, or `docs/reference/api.md`.
- Treat CRD schema, compatibility, ownership, deletion, architecture, and release behavior as wide-impact changes that require deliberate review.

## Harbor-backed resources

Keep resource implementations consistent across their API type, Harbor client behavior, reconciliation, tests, documentation, samples, and generated outputs.

- Embed `HarborSpecBase` and `HarborStatusBase` in specs and statuses.
- Use `metadata.name` as the Harbor identity for named resources.
- Add `CreationPolicy` only when an existing Harbor resource can be discovered uniquely.
- Prefer Kubernetes references and referenced status over raw Harbor IDs.
- Include the status subresource and the standard `Ready`, `Reason`, `Message`, and `Age` print columns.
- Reconcile generation changes, connection construction, deletion, finalization, adoption or defaulting, create or update, status, and drift detection in a legible order.
- Surface operational failures through `setErrorStatus`.
- Give every CRD a guide under `docs/crds/` with an example, field summary, and behavioral constraints.

## Verify changes

Run the normal non-E2E baseline:

```sh
make check
```

Use focused targets while iterating:

```sh
make test
make lint
make verify-generated
make docs-build
make test-e2e
```

`make verify-generated` checks generated code, manifests, chart assets, and the API reference without treating pre-existing feature-branch changes as newly introduced drift.

See the contributor guides for [local development](docs/contributing/local-development.md), [testing](docs/contributing/testing.md), [documentation](docs/contributing/documentation.md), and [releases](docs/contributing/releases.md).

## Chart changes

- Update `charts/harbor-operator/values.yaml` and `values.schema.json` together.
- Prefer additive values changes so existing installations remain upgradeable.
- Document new or changed runtime settings in the chart README.

## Pull requests

Use Conventional Commits for commit messages and PR titles, for example `feat(api): add project metadata`. Branch names use the same type prefix, for example `feat/project-metadata`.

When a required workflow has no relevant work, keep the check running and skip expensive steps through changed-file detection inside the workflow.
