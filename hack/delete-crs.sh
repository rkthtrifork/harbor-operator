#!/usr/bin/env bash
set -euo pipefail

GROUP="${HARBOR_API_GROUP:-harbor.harbor-operator.io}"

echo "Cleaning custom resources in API group: ${GROUP}"

# Get resource kinds (plural) in the group and strip the group suffix so we get:
#   harborconnections.harbor.harbor-operator.io -> harborconnections
#   projects.harbor.harbor-operator.io          -> projects
RESOURCES="$(kubectl api-resources --api-group="${GROUP}" -o name 2>/dev/null \
  | sed 's/\..*$//' || true)"

if [[ -z "${RESOURCES}" ]]; then
  echo "No API resources found for group ${GROUP}, nothing to delete."
  exit 0
fi

echo "Found resources in group ${GROUP}:"
echo "${RESOURCES}"

ordered=(
  members
  robots
  retentionpolicies
  users
  projects
  registries
  configurations
  gcschedules
  purgeauditschedules
)

declare -A seen
for r in ${RESOURCES}; do
  seen["$r"]=1
done

delete_and_wait() {
  local r="$1"
  echo "Deleting all ${r}.${GROUP} in all namespaces..."
  kubectl delete "${r}.${GROUP}" \
    --all \
    --all-namespaces \
    --ignore-not-found=true || true
  kubectl wait --for=delete "${r}.${GROUP}" \
    --all \
    --all-namespaces \
    --timeout=120s || true
}

# First delete ordered resources (except harborconnections).
for r in "${ordered[@]}"; do
  if [[ -n "${seen[$r]:-}" ]]; then
    delete_and_wait "$r"
    unset "seen[$r]"
  fi
done

# Then delete any remaining resource kinds except harborconnections.
for r in "${!seen[@]}"; do
  if [[ "${r}" == "harborconnections" ]]; then
    continue
  fi
  delete_and_wait "$r"
done

# Then delete harborconnections last (if it exists)
if grep -qx "harborconnections" <<< "${RESOURCES}"; then
  delete_and_wait "harborconnections"
else
  echo "Resource type 'harborconnections' not found in group ${GROUP}, skipping."
fi

echo "Done cleaning custom resources for group ${GROUP}."
