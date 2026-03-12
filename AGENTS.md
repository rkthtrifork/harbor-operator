# AGENTS

This repo has strict structure expectations. If you expand the operator, follow this.
Contributor-facing guidance lives in [`CONTRIBUTING.md`](./CONTRIBUTING.md). Keep this file aligned with it.

## Required Structure

### CRD Types
Location: `api/v1alpha1/*_types.go`
- Must embed `HarborSpecBase` in Spec and `HarborStatusBase` in Status.
- Add `AllowTakeover` on identity-based CRDs that represent named Harbor identities without IDs.
- Must include printcolumns: `Ready`, `Reason`, `Message` (priority=1), `Age`.
- Add CRD-specific printcolumns (see existing types).

### Controllers
Location: `internal/controller/*_controller.go`
Follow the standard reconcile order:
1. Load CR
2. Set Reconciling if generation changed
3. Build Harbor client
4. Delete path (`finalizeIfDeleting` + delete helper)
5. Ensure finalizer
6. Defaults / adoption
7. Create/Update
8. Status update (`setReadyStatus`/`markReady`) + `setErrorStatus` on failures
9. Return drift detection result

Errors must be surfaced through `setErrorStatus`.

### Docs
Location: `docs/crds/*.md`
Each CRD requires a doc file with:
- Heading
- Example YAML (code block)
- Field summary/notes

## Harbor API Reference
- `hack/harbor-openapi.yaml` is the checked-in Harbor OpenAPI spec.
- Use it when changing `internal/harborclient`, Harbor-specific controller behavior, or tests that depend on Harbor API semantics.
- Refresh it with `make update-harbor-openapi` when needed.

## Generated Assets
- `config/crd/bases` is canonical for CRDs.
- `config/rbac/role.yaml` is canonical for RBAC.
- Sync Helm chart assets with `make sync-chart`.

## Verification
Run:
- `make manifests generate sync-chart`
