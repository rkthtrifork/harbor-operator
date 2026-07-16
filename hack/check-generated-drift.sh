#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
generated_paths=(
  api/v1alpha1/zz_generated.deepcopy.go
  config/crd/bases
  config/rbac/role.yaml
  charts/harbor-operator/crds
  charts/harbor-operator/templates/clusterrole.yaml
  docs/reference/api.md
)

before="$(mktemp)"
after="$(mktemp)"
trap 'rm -f "$before" "$after"' EXIT

snapshot() {
  git -C "$repo_root" diff --binary -- "${generated_paths[@]}"
  while IFS= read -r file; do
    printf 'UNTRACKED %s ' "$file"
    git -C "$repo_root" hash-object -- "$file"
  done < <(git -C "$repo_root" ls-files --others --exclude-standard -- "${generated_paths[@]}")
}

snapshot >"$before"
make -C "$repo_root" manifests generate sync-chart generate-docs
snapshot >"$after"

if cmp -s "$before" "$after"; then
  echo "Generated assets are up to date."
  exit 0
fi

echo "Generated asset drift detected. Regeneration updated the working tree."
echo "Review the changes and commit intentional generated outputs."
exit 1
