# FAQ

## Does the operator install Harbor?

No. The operator assumes Harbor already exists. The local Kind workflow installs Harbor only as a development convenience.

## When should I use `HarborConnection` vs `ClusterHarborConnection`?

Use:

- `HarborConnection` for namespaced, tenant-local access
- `ClusterHarborConnection` for a shared Harbor instance managed centrally

## What happens if the referenced connection changes?

Harbor-backed resources that reference that connection are reconciled again. This applies to both `HarborConnection` and `ClusterHarborConnection`.

## What happens if I delete the connection first?

Resources that still need Harbor-side cleanup can remain in `Terminating` when `deletionPolicy` is `Delete`. If you explicitly want Kubernetes deletion to proceed without Harbor cleanup, use `deletionPolicy: Orphan`.

## Why do some resources conflict instead of overwriting each other?

`Configuration`, `GCSchedule`, `PurgeAuditSchedule`, and `ScanAllSchedule` map to singleton Harbor APIs. Letting multiple CRs target the same Harbor instance would cause silent overwrites, so the operator keeps the oldest CR as owner and marks later CRs as conflicting.

## Where should I look for exact field definitions?

Use the generated [API Reference](../reference/resources.md) for exact schema, defaults, enums, and validation markers. Use the resource guides under [API](../reference/index.md) for behavior and examples.
