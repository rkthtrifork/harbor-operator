---
name: harbor-resource-change
description: Implement Harbor-backed custom resource changes end to end. Use when adding a CRD or changing a resource's Kubernetes API, Harbor client operations, reconciliation, ownership, adoption, deletion, status, documentation, samples, or generated manifests.
---

# Harbor Resource Change

Treat a Harbor resource as one contract spanning Kubernetes API shape, Harbor semantics, reconciliation, tests, documentation, and generated delivery assets.

## Establish the contract

Read `AGENTS.md`, the closest existing resource implementation, and the relevant operation in `hack/harbor-openapi.yaml`. Determine identity, scope, referenced objects, Harbor IDs, create/update support, adoption, deletion policy, secrets, and observable status before editing.

Flag backwards-compatibility implications before changing a CRD schema or established ownership/deletion behavior. Prefer Kubernetes references and referenced status over raw Harbor IDs. Use `metadata.name` for named Harbor identities and add `CreationPolicy` only when the controller can uniquely discover an existing Harbor resource for adoption.

## Implement the vertical slice

Change only the layers the contract requires, but check each layer deliberately:

1. Define API fields, validation/default markers, status, print columns, and registration under `api/v1alpha1`.
2. Add typed Harbor client behavior and request/response tests when Harbor operations change. Match the checked-in OpenAPI semantics; do not infer behavior from the UI.
3. Reconcile generation, connection selection, deletion/finalization, references, adoption/defaulting, create/update, status, and drift detection in a flat, legible flow. Surface operational failures through `setErrorStatus`.
4. Test observable create, update, adoption, deletion, error, and idempotency behavior that the change affects. Reuse nearby envtest and HTTP test patterns.
5. Update the CRD guide, examples or samples, and chart-facing configuration when users need them.

Do not hand-edit generated deepcopy code, CRDs, RBAC, chart copies of generated assets, or `docs/reference/api.md`.

## Verify

Run focused tests while iterating. Then run `make check`, inspect the generated diff, and confirm every generated change follows from an intentional source change.

When real Kubernetes or Harbor behavior remains material and unproven, start or reuse the local Kind stack and manually exercise the changed behavior. Prefer focused manual checks over the full E2E suite when they isolate the relevant risk more directly.

Inspect existing clusters and workloads before creating one because another task may be using a compatible shared stack. Reuse a healthy stack with unique test resources when practical, but avoid resets, teardown, or replacing shared operator and CRD state; use an isolated cluster when validation would disrupt concurrent work.

Report the contract chosen, compatibility implications, generated outputs, automated evidence, and any live behavior not exercised.
