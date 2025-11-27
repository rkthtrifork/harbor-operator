#!/usr/bin/env bash
set -euo pipefail

GROUP="${HARBOR_API_GROUP:-harbor.harbor-operator.io}"

echo "Deleting CRDs for API group: ${GROUP}"

CRDS="$(kubectl get crd -o name 2>/dev/null | grep "\.${GROUP}$" || true)"

if [[ -z "${CRDS}" ]]; then
  echo "No CRDs found for group ${GROUP}, nothing to delete."
  exit 0
fi

echo "Found CRDs:"
echo "${CRDS}"

kubectl delete ${CRDS} --ignore-not-found=true || true

echo "Done deleting CRDs for group ${GROUP}."
