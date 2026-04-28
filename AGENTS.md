# AGENTS

This repo has strict structure expectations. If you expand the operator, follow this.
Contributor-facing guidance lives in [`CONTRIBUTING.md`](./CONTRIBUTING.md). Keep this file aligned with it.

## Self-Maintenance

- Treat `AGENTS.md` as a living repo contract, not a static note.
- When an agent learns a durable repo expectation, constraint, or convention
  while doing work here, update `AGENTS.md` in the same change when practical.
- If guidance in `AGENTS.md` is outdated, redundant, misleading, or no longer enforced by the repo, remove or replace it directly instead of leaving stale instructions behind.
- Keep `AGENTS.md` and [`CONTRIBUTING.md`](./CONTRIBUTING.md) aligned on shared project rules. If one changes, check whether the other should change too.
- Prefer documenting durable goals, invariants, and decision criteria. Do not
  turn the current implementation into policy unless the repo truly depends on
  that exact shape.
- Current implementation paths are not sacred. If a simpler, safer, or more
  maintainable approach better preserves the durable goals, prefer that change
  and update the affected docs, tests, and automation together.
- Question existing choices when there is a concrete improvement, but avoid
  churn that only swaps one valid style for another.
- Do not add one-off task notes, temporary incident details, or personal working
  preferences.

## Required Structure

### CRD Types
Location: `api/v1alpha1/*_types.go`
- Must embed `HarborSpecBase` in Spec and `HarborStatusBase` in Status.
- Add `AllowTakeover` on identity-based CRDs that represent named Harbor identities without IDs.
- Use `metadata.name` as the Harbor-side identity for named resources. Do not add duplicate `spec.name` / `spec.username` / `spec.groupName` style fields for the same identity.
- Prefer Kubernetes object references plus referenced status over raw Harbor IDs or `nameOrID` union fields.
- Root CRDs must include `+kubebuilder:object:root=true` and `+kubebuilder:subresource:status`.
- Must include printcolumns: `Ready`, `Reason`, `Message` (priority=1), `Age`.
- Add CRD-specific printcolumns (see existing types).

### Controllers
Location: `internal/controller/*_controller.go`
Follow the standard reconcile order:
1. Load CR
2. Set Reconciling if generation changed
3. Build Harbor client
4. Delete path (`finalizeIfDeleting` + delete helper)
5. Ensure finalizer
6. Defaults / adoption
7. Create/Update
8. Status update (`setReadyStatus`/`markReady`) + `setErrorStatus` on failures
9. Return drift detection result

Errors must be surfaced through `setErrorStatus`.

### Docs
Location: `docs/crds/*.md`
Each CRD requires a doc file with:
- Short description
- Example YAML (code block)
- Spec field summary
- Notes about behavior and constraints

The docs site is built with MkDocs Material. Hand-written guides live under `docs/crds/`. Keep the generated schema reference in `docs/reference/api.md` up to date with `make generate-docs`.

## Harbor API Reference
- `hack/harbor-openapi.yaml` is the checked-in Harbor OpenAPI spec.
- Use it when changing `internal/harborclient`, Harbor-specific controller behavior, or tests that depend on Harbor API semantics.
- Refresh it with `make update-harbor-openapi` when needed.

## Generated Assets
- `config/crd/bases` is canonical for CRDs.
- `config/rbac/role.yaml` is canonical for RBAC.
- `docs/reference/api.md` is canonical for the generated API reference.
- Sync chart CRDs with `make sync-chart-crds`.
- Sync chart RBAC with `make sync-chart-rbac`.
- Sync Helm chart assets with `make sync-chart`.

## Operator Runtime Flags
- `--watch-namespaces` scopes the operator to a fixed namespace set when needed.
- `--harbor-connection` points Harbor-backed resources at one operator-wide `ClusterHarborConnection`; in that mode `spec.harborConnectionRef` may be omitted or must match the configured cluster connection.

## Automation Conventions
- Pull request titles must follow conventional-commit format (`type(scope): summary` or `type: summary`) because the `pr-title` workflow enforces it.
- Renovate PRs must keep semantic commit titles enabled and use strict PR titles so branch suffixes like `(main)` do not get appended.
- GitHub Actions workflows should use trigger-level `paths` filters for clearly scoped automation, but not on pull-request workflows whose checks are required for merging. For required PR checks, let the workflow start and use lightweight changed-file detection inside jobs or steps.

## Verification
Run:
- `make manifests generate sync-chart generate-docs`

Useful local docs target:
- `make docs-build`

## Development Environment
- Local host-based development is the supported workflow.
- Use the `Makefile` with local installations of Docker, Go, Helm, `kubectl`, and Kind as needed.
- This repository does not currently maintain a devcontainer setup.

## Release Principles
- Release branches use the form `release/vX.Y` for supported operator minor lines.
- `main` remains the development branch; maintenance patch releases are cut from release branches.
- Keep release operations reproducible, auditable, and hard to perform
  partially. Prefer one clear release path over parallel mechanisms.
- Release tags should be treated as records of an intentional release, not as an
  ad hoc control surface.
- Release automation must account for repository rulesets and required
  permissions.
- New minor release branches should start with `charts/harbor-operator/Chart.yaml`
  `version` and `appVersion` aligned to `X.Y.0`.
- Chart release artifacts should clearly identify both the chart version and the
  operator image version they install.
- Routine release-branch maintenance automation should only process the latest 3
  supported release branches; older branches require explicit release intent.
- Only dependency-only patch releases should be eligible for automatic
  merge/release on release branches.
- Automated release-branch patch releases should publish the operator image and
  matching chart together so chart defaults track the newest operator patch.
- Minor, major, and non-dependency changes on release branches require manual
  review and explicit release intent.
- Chart-only patch releases are manual exceptions and should set intended chart
  and operator versions deliberately.
- GitHub's `latest` release should track the highest stable operator tag (`vX.Y.Z`); chart releases (`chart-vX.Y.Z`) must not mark themselves as `latest`.
- Auto-generated GitHub release notes must be scoped to the same tag family: operator releases diff from earlier `v*` tags, and chart releases diff from earlier `chart-v*` tags.
- RC release notes should compare against the latest stable release on the same release branch (`X.Y` line); if that line has no stable release yet, fall back to the previous stable tag in the same tag family.
