#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src_file="$repo_root/config/rbac/role.yaml"
dst_file="$repo_root/charts/harbor-operator/templates/clusterrole.yaml"

grep -q '^rules:$' "$src_file"

{
  cat <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "harbor-operator.fullname" . }}-manager-role
  labels:
{{ include "harbor-operator.labels" . | indent 4 }}
EOF
  sed -n '/^rules:$/,$p' "$src_file"
} >"$dst_file"

echo "Synced RBAC from $src_file to $dst_file"
