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
make manifests generate sync-chart generate-docs
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

### Release Versioning
- Operator releases use tags `vX.Y.Z` or `vX.Y.Z-rc.N`.
- Chart releases use tags `chart-vX.Y.Z` or `chart-vX.Y.Z-rc.N`.
- Operator and chart versions are independent. `release/vX.Y` branches track
  the operator `X.Y` source line; chart versions on that branch may be on a
  different semver line.
- `charts/harbor-operator/Chart.yaml` is the release metadata source on release
  branches: `appVersion` is the operator image version and `version` is the
  chart version.
- New minor release branches should start with `Chart.yaml` metadata set
  deliberately, for example `version: 0.5.0` and `appVersion: "0.7.0"` on
  `release/v0.7`.
- Keep releases reproducible, auditable, and hard to perform partially. Prefer
  one clear release path over parallel mechanisms.
- Treat release tags as records of an intentional release, not as an ad hoc
  control surface.
- Release automation must account for repository rulesets and required
  permissions.
- Create release tags through the dispatch workflows instead of pushing tags by
  hand. The workflows validate the branch, required checks, and tag target
  before publishing.

### Release Branches
- Release branches use the form `release/vX.Y` for supported operator minor lines.
- `main` remains the development branch; maintenance patch releases are cut from release branches.
- Support only the latest 3 release branches by semver for routine maintenance automation.
- Only dependency-only patch releases should be eligible for automatic
  merge/release on release branches.
- Minor, major, and non-dependency changes on release branches require manual
  review and explicit release intent.

### Chart Packaging on Release Branches
- Published chart artifacts should clearly identify both the chart version and
  the operator image version they install.
- Stable chart releases must not reference prerelease operator versions.
- Chart-only patch releases are supported. Change the chart files on the
  relevant operator release branch, bump only `Chart.yaml` `version`, keep
  `appVersion` on the intended operator version, and run the chart release
  workflow.
- Operator patch releases should update `Chart.yaml` `appVersion` to the new
  operator version. Bump `Chart.yaml` `version` when publishing a matching chart
  package.
- Automated release-branch patch releases are allowed only for dependency-only
  changes. The patch train bumps `Chart.yaml`, waits for required checks on that
  commit, publishes the operator image, and then publishes a matching chart so
  chart defaults track the newest operator patch.
- The patch train is recoverable. If release tags already exist but the GitHub
  release or chart package asset is missing, the next run schedules the missing
  publish job again.
- Unpublished chart changes block automated dependency patch releases. Chart
  changes that were already published by a chart-only release do not block a
  later dependency-only operator patch.
- Release automation behavior is covered by `hack/test_release_branch_patch_train.py`;
  run it when changing release automation.
- GitHub's `latest` release is reserved for the highest stable operator tag (`vX.Y.Z`); chart releases (`chart-vX.Y.Z`) publish GitHub releases for assets/notes but do not mark themselves as `latest`.
- Auto-generated GitHub release notes are scoped by tag family so operator releases compare against earlier `v*` tags and chart releases compare against earlier `chart-v*` tags.
- RC release notes compare against the latest stable release on the same release branch (`X.Y` line); if that line has no stable release yet, the workflow falls back to the previous stable tag in the same tag family.

### Release Workflows
Use the workflows below instead of pushing release tags by hand.

- Create a new operator release branch. The branch name is derived from the
  operator version, and the initial `Chart.yaml` metadata is committed to the
  branch:
  ```
  gh workflow run create-release-branch.yaml \
    -f operator_version=0.7.0 \
    -f chart_version=0.5.0 \
    -f source_ref=main
  ```
- For dependency-only Renovate changes on supported release branches, merge the
  PR and let the patch train run. It also runs weekly as a backstop. No manual
  release command is needed. If a publish job fails after tags are created, the
  next patch train run will schedule the missing publish job again.
- If the release branch already exists and the release is not handled by the
  patch train, prepare the next release metadata without editing files locally:
  ```
  gh workflow run prepare-release-metadata.yaml \
    -f release_branch=release/v0.7 \
    -f operator_version=0.7.1 \
    -f chart_version=0.5.1
  ```
- Wait for `docs`, `lint`, `verify-generated`, `test`, and `test-e2e` to pass on
  the release branch commit before publishing.
- Publish the operator version in `Chart.yaml` `appVersion`:
  ```
  gh workflow run create-operator-release.yaml \
    -f release_branch=release/v0.7
  ```
- Publish the chart version in `Chart.yaml` `version`:
  ```
  gh workflow run create-chart-release.yaml \
    -f release_branch=release/v0.7
  ```
- For a normal manual operator patch that should become the Helm default, prepare
  both versions, wait for checks, run the operator release workflow, then run the
  chart release workflow.
- For release candidates, use the same metadata workflow with RC versions, for
  example `operator_version=0.7.0-rc.1` and `chart_version=0.5.0-rc.1`, then run
  the same separate operator and chart release workflows.
- For chart-only releases, set only `chart_version` in the metadata workflow,
  keep `appVersion` on the intended operator version, wait for checks, and run
  only `create-chart-release.yaml`.
