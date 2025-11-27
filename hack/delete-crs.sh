#!/usr/bin/env bash
set -euo pipefail

GROUP="${HARBOR_API_GROUP:-harbor.harbor-operator.io}"

echo "Cleaning custom resources in API group: ${GROUP}"

# Get all resource kinds (plural names) in the group, e.g. projects, users, harborconnections, ...
RESOURCES="$(kubectl api-resources --api-group="${GROUP}" -o name 2>/dev/null || true)"

if [[ -z "${RESOURCES}" ]]; then
  echo "No API resources found for group ${GROUP}, nothing to delete."
  exit 0
fi

echo "Found resources in group ${GROUP}:"
echo "${RESOURCES}"

# First delete everything except harborconnections
for r in ${RESOURCES}; do
  if [[ "${r}" == "harborconnections" ]]; then
    continue
  fi
  echo "Deleting all ${r} in all namespaces..."
  kubectl delete "${r}" --all-namespaces --ignore-not-found=true || true
done

# Then delete harborconnections last (but only if the type exists)
if echo "${RESOURCES}" | grep -qx "harborconnections"; then
  echo "Deleting all harborconnections in all namespaces..."
  kubectl delete harborconnections --all-namespaces --ignore-not-found=true || true
else
  echo "Resource type 'harborconnections' not found in group ${GROUP}, skipping."
fi

echo "Done cleaning custom resources for group ${GROUP}."
