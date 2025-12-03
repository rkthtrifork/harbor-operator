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

# First delete everything except harborconnections
for r in ${RESOURCES}; do
  if [[ "${r}" == "harborconnections" ]]; then
    # Delete this kind later, after everything else
    continue
  fi

  echo "Deleting all ${r}.${GROUP} in all namespaces..."
  kubectl delete "${r}.${GROUP}" \
    --all \
    --all-namespaces \
    --ignore-not-found=true || true
done

# Then delete harborconnections last (if it exists)
if grep -qx "harborconnections" <<< "${RESOURCES}"; then
  echo "Deleting all harborconnections.${GROUP} in all namespaces..."
  kubectl delete "harborconnections.${GROUP}" \
    --all \
    --all-namespaces \
    --ignore-not-found=true || true
else
  echo "Resource type 'harborconnections' not found in group ${GROUP}, skipping."
fi

echo "Done cleaning custom resources for group ${GROUP}."
