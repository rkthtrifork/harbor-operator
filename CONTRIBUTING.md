# Contributing

Thanks for your interest in contributing! This repo is intentionally structured so that new CRDs and controllers stay consistent.

This guidance is kept in `CONTRIBUTING.md` because it is contributor-facing and meant to be read before making changes. If we later want a user-facing overview, we can add a short link in `docs/` that points here.

## Structure Contract

### CRD Types (`api/v1alpha1/*_types.go`)
Each CRD type file must include:
- A **Spec** and **Status** struct with `HarborSpecBase` and `HarborStatusBase` embedded.
- `AllowTakeover` on CRDs that represent named Harbor identities without IDs (Registry, Project, User, Robot, Member).
- Use `metadata.name` as the Harbor identity for named resources instead of adding duplicate name fields in `spec`.
- Prefer Kubernetes object references and referenced status over raw Harbor IDs or `nameOrID` selector fields.
- `// +kubebuilder:object:root=true` and `// +kubebuilder:subresource:status` on the root type.
- **Print columns** for `Ready`, `Reason`, `Message`, and `Age`.

Example:

```go
type ExampleSpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover should be set on identity-based CRDs (Registry/Project/User/Robot/Member).
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// TODO(user): add spec fields
}

type ExampleStatus struct {
	HarborStatusBase `json:",inline"`

	// TODO(user): add status fields
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,priority=1,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type Example struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExampleSpec   `json:"spec,omitempty"`
	Status ExampleStatus `json:"status,omitempty"`
}
```

### Controllers (`internal/controller/*_controller.go`)
Controllers should follow the same reconcile flow:
1. Load CR
2. Set Reconciling if generation changed
3. Build Harbor client
4. Delete path (`finalizeIfDeleting` + delete helper)
5. Ensure finalizer
6. Defaults / adoption
7. Create or Update
8. Status update via `setReadyStatus` / `markReady`
9. Return drift detection result

Errors must be surfaced with `setErrorStatus(...)` so users see the failure on the CR.

Example flow snippet:

```go
if cr.Status.ObservedGeneration != cr.Generation {
	if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "", ""); err != nil {
		return ctrl.Result{}, err
	}
}

hc, err := getHarborClient(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
if err != nil {
	return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
}

if done, err := finalizeIfDeleting(ctx, r.Client, &cr, func() error {
	return r.deleteThing(ctx, hc, &cr)
}); done {
	return ctrl.Result{}, err
}

if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
	return ctrl.Result{}, err
}

// create/update...
if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "..."); err != nil {
	return ctrl.Result{}, err
}
return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
```

### Docs (`docs/crds/*.md`)
Every CRD must have a doc file with:
- Short description
- Example YAML
- Spec field summary
- Notes about behavior/constraints

The documentation site is built with MkDocs Material. The hand-written guides live under `docs/crds/`, and the schema reference is generated into `docs/reference/api.md`.
GitHub Pages deployment is handled by the `docs` workflow on pushes to `main`.
The published site intentionally tracks `main` only. Historical docs are expected to be read from the repository at the relevant tag or commit.

## Common Tasks

```
make generate manifests
make generate-docs
```

## Automation Conventions

Pull request titles must follow conventional-commit format: `type(scope): summary` or `type: summary`.
The `pr-title` workflow enforces this on every PR.

Renovate is configured to emit semantic commit titles for dependency PRs and to use strict PR titles so base-branch suffixes like `(main)` are not appended.

When a GitHub Actions workflow only applies to a subset of the repository, prefer trigger-level `paths` filters unless the pull request check is required for merging.
For required PR checks, let the workflow start and use lightweight changed-file detection inside the workflow to skip unnecessary work.

To build the documentation site locally:

```sh
make docs-build
```

## Development Environment

Local host-based development is the supported workflow for this repository.
Use the tools in the `Makefile`, along with local installations of Docker, Go, Helm, `kubectl`, and Kind as needed.

This repository does not currently maintain a devcontainer setup.
If we later need a containerized development environment, we can add one back.

## Harbor OpenAPI Spec

The checked-in Harbor OpenAPI spec lives at:

```text
hack/harbor-openapi.yaml
```

Use it as the local reference when changing:
- `internal/harborclient`
- controller logic that depends on Harbor request/response behavior
- tests that verify Harbor API semantics

Refresh it from upstream with:

```sh
make update-harbor-openapi
```

This is a manual maintenance task. Update it when the change you are making depends on Harbor API details; it does not need to be refreshed on every contribution.

## Generated Assets

We treat generated outputs as source-of-truth for releases and keep Helm artifacts in sync:

- **RBAC**: `config/rbac/role.yaml` is canonical (generated by controller-gen).  
  Sync to the Helm chart via:
  ```
  make sync-chart-rbac
  ```
- **CRDs**: `config/crd/bases` are canonical.  
  Sync to the Helm chart via:
  ```
  make sync-chart-crds
  ```
- **Docs reference**: `docs/reference/api.md` is generated from the API types with `crd-ref-docs`.  
  Regenerate it via:
  ```
  make generate-docs
  ```

CI verifies that chart RBAC, CRDs, and generated API reference docs stay in sync with these sources.
Use `make sync-chart` to update both chart CRDs and RBAC together when a change affects generated chart assets.

## Helm Chart

We maintain the Helm chart under `charts/harbor-operator/`. Please keep these in mind:

### Operator vs Chart
- The **operator code** lives under `api/` and `internal/`.
- The **chart** is for packaging/deploying the operator.
- We consider Helm the primary install method; kustomize-based install manifests are no longer maintained.

### Chart Values & Schema
- Update `charts/harbor-operator/values.yaml` and `charts/harbor-operator/values.schema.json` together.
- Prefer additive changes to values to avoid breaking upgrades.
- Runtime flags exposed by the chart, such as `watchNamespaces` and `harborConnection`, should be documented in the chart README when added or changed.

### Chart RBAC & CRDs
- **RBAC**: `config/rbac/role.yaml` is canonical. Sync to the chart with:
  ```
  make sync-chart-rbac
  ```
- **CRDs**: `config/crd/bases` are canonical. Sync to the chart with:
  ```
  make sync-chart-crds
  ```

CI verifies both are in sync.

### Chart Versioning & Release Tags
- For manual chart releases, bump `charts/harbor-operator/Chart.yaml` `version` before release.
- Chart releases use tags `chart-vX.Y.Z` or `chart-vX.Y.Z-rc.N`.
- Any other suffix (e.g., `-test`) skips GitHub Release creation.

### Operator Release Tags
- Operator releases use tags `vX.Y.Z` or `vX.Y.Z-rc.N`.
- Any other suffix (e.g., `-test`) skips GitHub Release creation.

### Release Branches
- Release branches use the form `release/vX.Y` for supported operator minor lines.
- `main` remains the development branch; maintenance patch releases are cut from release branches.
- Support only the latest 3 release branches by semver for routine maintenance automation.
- Dependency-only patch releases may be tagged automatically from release branches on the scheduled patch train.
- Any non-dependency change on a release branch should be released manually.

### Chart Packaging on Release Branches
- The chart release workflow can package the chart with `helm package --version ... --app-version ...` using the release tags.
- This keeps the published chart artifact aligned with the operator image version without committing `Chart.yaml` patch bumps back to the release branch.
- Automated release-branch patch trains publish the operator tag first, wait for the matching GHCR image to exist, and only then create the chart tag.
- The scheduled patch train only processes the latest 3 supported release branches; you can still target an older branch explicitly through `workflow_dispatch` when needed.
- On release branches, dependency-only operator patch releases should also publish a new chart release so the chart default image tracks the newest operator patch.
- Chart-only patch releases remain a manual path and should set the intended chart/operator versions deliberately before tagging.
- GitHub's `latest` release is reserved for the highest stable operator tag (`vX.Y.Z`); chart releases (`chart-vX.Y.Z`) publish GitHub releases for assets/notes but do not mark themselves as `latest`.
- Auto-generated GitHub release notes are scoped by tag family so operator releases compare against earlier `v*` tags and chart releases compare against earlier `chart-v*` tags.
- RC release notes compare against the latest stable release on the same release branch (`X.Y` line); if that line has no stable release yet, the workflow falls back to the previous stable tag in the same tag family.
