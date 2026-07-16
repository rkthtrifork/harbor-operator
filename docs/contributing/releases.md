# Releases

Operator and chart releases are independent but coordinated. Release branches are operator source lines; `charts/harbor-operator/Chart.yaml` is their release metadata, with `appVersion` selecting the operator image and `version` selecting the chart package.

## Principles

- Operator tags use `vX.Y.Z` or `vX.Y.Z-rc.N`; chart tags use `chart-vX.Y.Z` or `chart-vX.Y.Z-rc.N`.
- Release branches use `release/vX.Y`. `main` remains the development branch, and maintenance releases are cut from release branches.
- Create release tags through dispatch workflows, not by pushing tags manually. Tags record an intentional release; they are not an ad hoc control surface.
- New minor release branches start with deliberate chart and operator versions even when their semver lines differ.
- Stable chart releases must not reference prerelease operator versions. Release assets must identify both the chart version and installed operator version.
- Routine release-branch automation processes only the latest three supported release branches. Older branches require explicit release intent.
- GitHub's `latest` release tracks the highest stable operator tag. Chart releases must not mark themselves as latest.
- Generated release notes stay within their tag family. Release-candidate notes compare with the latest stable release on the same operator minor line, falling back to the previous stable tag in the same family when that line has no stable release.

## Create a release branch

Create the branch and initialize both versions through the workflow:

```sh
gh workflow run create-release-branch.yaml \
  -f operator_version=0.7.0 \
  -f chart_version=0.5.0 \
  -f source_ref=main
```

The workflow derives the `release/vX.Y` branch name from the operator version and commits the initial `Chart.yaml` metadata.

## Dependency-only patch train

Only dependency-only patch changes are eligible for automatic merge and release on supported release branches. Merge the Renovate PR and allow the patch train to run; it also runs weekly as a recovery backstop.

The patch train bumps `Chart.yaml`, waits for the required checks on that commit, publishes the operator image, and publishes a matching chart so chart defaults follow the newest operator patch. Existing tags alone do not mean the release is complete: missing GitHub releases or chart package assets must cause the corresponding publish job to be scheduled again.

Unpublished chart changes block the automated patch train. Chart changes already published through a chart-only release do not block a later dependency-only operator patch.

The implementation lives in `hack/release_branch_patch_train.py`; `hack/release-branch-patch-train.sh` is its workflow compatibility wrapper. Required check names live in `hack/required_checks.py`. Run `python3 hack/test_release_branch_patch_train.py` when changing patch-train behavior.

## Manual and release-candidate releases

Minor, major, and non-dependency release-branch changes require review and explicit release intent.

Prepare release metadata without editing it locally:

```sh
gh workflow run prepare-release-metadata.yaml \
  -f release_branch=release/v0.7 \
  -f operator_version=0.7.1 \
  -f chart_version=0.5.1
```

Wait for `docs`, `lint`, `verify-generated`, `test`, and `test-e2e` on the resulting release-branch commit. Then publish the operator and chart separately:

```sh
gh workflow run create-operator-release.yaml \
  -f release_branch=release/v0.7

gh workflow run create-chart-release.yaml \
  -f release_branch=release/v0.7
```

For a normal operator patch that should become the Helm default, prepare both versions, wait for checks, publish the operator, and then publish the chart. For a release candidate, use RC versions in the same metadata and publication workflows.

## Chart-only releases

For a chart-only patch, set only `chart_version` through `prepare-release-metadata.yaml`, leave `appVersion` on the intended stable operator version, wait for the required checks, and run only `create-chart-release.yaml`.

## Recovery

Before retrying a release, inspect the target branch commit, `Chart.yaml`, tags, GitHub releases, and chart assets. Resume only the missing publication step. Do not move or recreate existing release tags to force automation forward.
