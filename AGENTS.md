# AGENTS

This repo has strict structure expectations. If you expand the operator, follow this.

## Required Structure

### CRD Types
Location: `api/v1alpha1/*_types.go`
- Must embed `HarborSpecBase` in Spec and `HarborStatusBase` in Status.
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

## Verification
Run:
- `make generate manifests`
