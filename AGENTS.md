# AGENTS.md

How to work in this repo. Keep it short and precise — this is not a changelog. See [§ Maintaining this file](#maintaining-this-file) before adding anything.

## Orientation

The public Kubernetes API originates in the Go types and Kubebuilder markers under `api/v1alpha1`. The Harbor OpenAPI document at `hack/harbor-openapi.yaml` is a checked-in reference for Harbor API semantics, not a generator input for this operator.

Generated outputs include `api/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases`, `config/rbac/role.yaml`, chart CRDs and RBAC, and `docs/reference/api.md`. Read them when generated shape is relevant, but do not hand-edit them. Change their source types or markers, then regenerate with `make generate` or the appropriate focused generation target.

## Operating principles

### Autonomy, keyed to blast radius

- **Just do it** when a change is reversible, localized, and does not alter a public contract. Do not ask permission for the obvious.
- **Propose first and wait** when a change touches architecture, CRD schema or compatibility, ownership or deletion semantics, release behavior, or is hard to undo — even if you are confident. "Obvious" is not sufficient on its own here.

### Direction

- When changing public contracts or persisted Kubernetes resource shape, flag backwards-compatibility implications before implementing so they can be weighed.
- When a task outgrows a simple existing design, a refactor is welcome: do it if it is clearly right and in the just-do-it category; propose it otherwise.

### Suggesting improvements

- Surface material design, infrastructure, or architecture improvements you spot while working, each with a one-line cost/benefit. Not nitpicks.
- If a fix is trivial and safe, make it. Otherwise report the improvement in the handoff without silently expanding the task. If a tool would materially help your work, say so.

### Testing

- Cover behavior and edge cases; do not test internal wiring. Assert contracts and observable behavior so implementation can change freely without breaking tests.
- Verification is evidence, not a checklist. Choose checks that match the changed behavior, explain what they prove when it matters, and add task-specific manual or end-to-end validation when automated checks do not cover the risk.
- Use the local Kind stack for bounded integration checks when reconciliation depends on real Kubernetes or Harbor behavior. Do not require full E2E validation for changes whose risk is already covered by focused tests and static checks.

## Operator contracts

- Specs and statuses embed `HarborSpecBase` and `HarborStatusBase`. Named Harbor identities use `metadata.name`; do not duplicate the identity in `spec`.
- Identity-based resources without Harbor IDs expose `AllowTakeover`. Prefer Kubernetes references and referenced status over raw Harbor IDs or `nameOrID` unions.
- Root CRDs include the root and status-subresource markers plus `Ready`, `Reason`, `Message`, and `Age` print columns.
- Reconciliation handles generation changes, client construction, deletion and finalization, adoption/defaulting, create/update, status, and drift detection in a legible order. Surface operational failures through `setErrorStatus`.
- Every CRD has a guide under `docs/crds/` with an example, field summary, and behavioral constraints.
- `--watch-namespaces` scopes watched namespaces. `--harbor-connection` selects one operator-wide `ClusterHarborConnection`; in that mode a resource connection reference may be omitted or must match it.

For a complete Harbor-backed resource change, use the shared workflow in [`.agents/skills/harbor-resource-change.md`](.agents/skills/harbor-resource-change.md).

## Commands

These commands are the baseline vocabulary for verification. Keep them aligned with the Makefile and CI.

| Scope | Command |
| --- | --- |
| Normal non-E2E baseline | `make check` |
| Generated assets | `make verify-generated` |
| Go tests | `make test` |
| Lint | `make lint` |
| Docs | `make docs-build` |
| Live Kind suite | `make test-e2e` |
| Local stack | `make kind-up`, `make kind-refresh`, `make kind-redeploy` |

`make verify-generated` regenerates code, manifests, chart assets, and API reference docs and fails only when regeneration adds to the existing generated-file diff, so it is safe to use on a dirty feature branch.

## Git conventions

Use Conventional Commits for commit messages and PR titles, for example `feat(api): add project metadata`. Branch names should use the same type prefix, for example `feat/project-metadata`.

Renovate must keep semantic commit titles and strict PR titles. Required pull-request workflows should start even when a change is irrelevant to their expensive work; use lightweight changed-file detection inside those workflows rather than trigger-level path filters.

## Pointers

- **Contributor contract** → [CONTRIBUTING.md](CONTRIBUTING.md)
- **Harbor API semantics** → [hack/harbor-openapi.yaml](hack/harbor-openapi.yaml)
- **Release process** → [docs/contributing/releases.md](docs/contributing/releases.md) and [`.agents/skills/harbor-release.md`](.agents/skills/harbor-release.md)
- **Shared agent workflows** → [`.agents/skills/`](.agents/skills/)

## Maintaining this file

This file is short on purpose, and bounded by strict membership, not a line limit. Before adding anything, route it:

- A **decision plus rationale** → an ADR under `docs/decisions/` when the repository needs to preserve that decision. Never record rationale here.
- A **contributor or operational procedure** → `CONTRIBUTING.md` or the relevant page under `docs/contributing/`.
- An **implementation detail** → record it nowhere. Code and tests are the truth, and they are free to change.
- A **reusable task workflow with a clear boundary** → a shared skill under `.agents/skills/`, linked into tool-specific surfaces when useful, with source-of-truth links instead of duplicated detail.
- An **improvement idea** → fix it when trivial and in scope; otherwise surface it in the handoff or the repository's established tracker.

Update guidance in place when reality changes; do not append exceptions. Remove stale or redundant guidance directly. Keep shared contributor rules aligned with `CONTRIBUTING.md`, but do not duplicate its detailed procedures here.
