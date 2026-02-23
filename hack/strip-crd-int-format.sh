#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CRD_DIR="${ROOT_DIR}/config/crd/bases"

if [[ ! -d "${CRD_DIR}" ]]; then
  echo "CRD dir not found: ${CRD_DIR}" >&2
  exit 1
fi

shopt -s nullglob
for f in "${CRD_DIR}"/*.yaml; do
  if rg -q "format: int(32|64)" "${f}"; then
    sed -i.bak -E '/format: int(32|64)$/d' "${f}"
    rm -f "${f}.bak"
  fi
done

echo "Stripped int32/int64 format fields from CRD schemas in ${CRD_DIR}"
