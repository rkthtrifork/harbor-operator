#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src_file="$repo_root/config/rbac/role.yaml"
dst_file="$repo_root/charts/harbor-operator/templates/clusterrole.yaml"

python3 - <<'PY'
import yaml
from pathlib import Path

src_file = Path("config/rbac/role.yaml")
dst_file = Path("charts/harbor-operator/templates/clusterrole.yaml")

obj = yaml.safe_load(src_file.read_text())
rules = obj.get("rules", [])

lines = []
lines.append("apiVersion: rbac.authorization.k8s.io/v1")
lines.append("kind: ClusterRole")
lines.append("metadata:")
lines.append("  name: {{ include \"harbor-operator.fullname\" . }}-manager-role")
lines.append("  labels:")
lines.append("{{ include \"harbor-operator.labels\" . | indent 4 }}")
lines.append("rules:")

for r in rules:
    api_groups = r.get("apiGroups", [])
    resources = r.get("resources", [])
    verbs = r.get("verbs", [])

    lines.append('- apiGroups: [%s]' % ",".join('"%s"' % g for g in api_groups))
    lines.append('  resources: [%s]' % ",".join('"%s"' % res for res in resources))
    lines.append('  verbs: [%s]' % ",".join('"%s"' % v for v in verbs))

dst_file.write_text("\n".join(lines) + "\n")
PY

echo "Synced RBAC from $src_file to $dst_file"
