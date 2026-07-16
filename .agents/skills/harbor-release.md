---
name: harbor-release
description: Prepare, publish, or recover Harbor Operator and Helm chart releases. Use for release branches, operator releases, chart-only releases, release candidates, release metadata, dependency patch-train behavior, or incomplete release recovery.
---

# Harbor Release

Read `AGENTS.md` and `docs/contributing/releases.md` before changing release state. Treat the release guide and checked-in workflows/scripts as the source of truth; do not reconstruct the process from old tags or prior runs.

## Classify and inspect

Classify the operation as a new minor branch, dependency-only patch train, manual operator release, chart-only release, release candidate, or recovery. Inspect the target branch, `charts/harbor-operator/Chart.yaml`, required checks, relevant tag family, GitHub releases, and chart assets before acting.

Keep operator and chart versions independent. Reject a stable chart version that selects a prerelease operator. Existing tags do not prove that GitHub releases and chart assets were published.

## Use the workflow path

- Create new minor branches with `create-release-branch.yaml` and deliberate operator/chart metadata.
- Let eligible dependency-only release-branch changes use the patch train.
- Prepare manual or chart-only metadata with `prepare-release-metadata.yaml`.
- Wait for the required checks on the exact metadata commit.
- Publish operator and chart tags with their separate dispatch workflows. Never push release tags by hand.
- For recovery, schedule only the missing publication step; do not move or recreate existing tags.

Changing release automation requires focused tests for the affected scripts plus the normal repository checks. Run `python3 hack/test_release_branch_patch_train.py` for patch-train changes and keep required-check policy centralized in `hack/required_checks.py`.

Before any dispatch or other external release mutation, state the target branch, operator version, chart version, intended workflows, current check state, and recovery implications. Afterward, report the exact commit and which operator release, chart release, and assets now exist.
