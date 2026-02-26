#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src_dir="$repo_root/config/crd/bases"
dst_dir="$repo_root/charts/harbor-operator/crds"

mkdir -p "$dst_dir"

find "$dst_dir" -type f -name '*.yaml' -delete
find "$src_dir" -type f -name '*.yaml' -maxdepth 1 -exec cp {} "$dst_dir" \;

echo "Synced CRDs from $src_dir to $dst_dir"
