#!/usr/bin/env bash
set -euo pipefail

GROUP="${HARBOR_API_GROUP:-harbor.harbor-operator.io}"

echo "Cleaning custom resources in API group: ${GROUP}"

RESOURCES="$(kubectl api-resources --api-group="${GROUP}" -o name \
  | sed 's/\..*$//')"

if [[ -z "${RESOURCES}" ]]; then
  echo "No API resources found for group ${GROUP}, nothing to delete."
  exit 0
fi

echo "Found resources in group ${GROUP}:"
echo "${RESOURCES}"

leaf_resources=(
  members
  immutabletagrules
  labels
  quotas
  retentionpolicies
  robots
  webhookpolicies
  replicationpolicies
  configurations
  gcschedules
  purgeauditschedules
  scanallschedules
  scannerregistrations
)
referenced_resources=(users usergroups projects)
registry_resources=(registries)
connection_resources=(harborconnections clusterharborconnections)

declare -A seen
for r in ${RESOURCES}; do
  seen["$r"]=1
done

declare -A known
for r in \
  "${leaf_resources[@]}" \
  "${referenced_resources[@]}" \
  "${registry_resources[@]}" \
  "${connection_resources[@]}"; do
  known["$r"]=1
done

delete_phase() {
  local r
  local joined
  local -a targets=()

  for r in "$@"; do
    if [[ -n "${seen[$r]:-}" ]]; then
      targets+=("${r}.${GROUP}")
      unset "seen[$r]"
    fi
  done

  if [[ "${#targets[@]}" -eq 0 ]]; then
    return
  fi

  printf -v joined '%s,' "${targets[@]}"
  joined="${joined%,}"

  echo "Deleting resource phase: ${joined}"
  kubectl delete "${joined}" \
    --all \
    --all-namespaces \
    --ignore-not-found=true \
    --timeout=120s
}

unknown_resources=()
for r in "${!seen[@]}"; do
  if [[ -z "${known[$r]:-}" ]]; then
    unknown_resources+=("$r")
  fi
done

# Each phase waits before removing resources needed by the next phase.
delete_phase "${leaf_resources[@]}" "${unknown_resources[@]}"
delete_phase "${referenced_resources[@]}"
delete_phase "${registry_resources[@]}"
delete_phase "${connection_resources[@]}"

echo "Done cleaning custom resources for group ${GROUP}."
