# Concepts

## Connection Objects

Every Harbor-backed custom resource points at a connection object.

- `HarborConnection` is namespaced and suited for tenant-local usage
- `ClusterHarborConnection` is cluster-scoped and suited for shared Harbor instances

The connection object carries the base URL, optional credentials, and optional CA material used to construct the Harbor client.

## Desired State Model

The operator treats Kubernetes custom resources as desired state and Harbor as the external system being reconciled.

That means:

- edits in Kubernetes drive Harbor changes
- Harbor-side drift is corrected on reconcile
- connection changes also trigger reconcile of dependent resources

## Resource Ownership

Some resources map cleanly to independently managed Harbor objects:

- `Project`
- `Registry`
- `User`
- `Robot`
- `Member`
- `Label`

Others map to singleton-style Harbor APIs:

- `Configuration`
- `GCSchedule`
- `PurgeAuditSchedule`
- `ScanAllSchedule`

Singleton resources are unique per Harbor instance. If multiple CRs target the same Harbor instance for the same singleton API, the oldest CR keeps ownership and later CRs report a conflict.

## Deletion Semantics

`spec.deletionPolicy` controls what happens when a Kubernetes object is deleted:

- `Delete` is the default and attempts Harbor cleanup before removing the finalizer
- `Orphan` removes the Kubernetes object even if Harbor cleanup cannot be completed

This matters mainly when Harbor is unreachable or the referenced connection object has already been removed.

## Secret Ownership

Robot credentials are operator-managed output, not input. The operator writes the destination secret and manages password generation and rotation rather than consuming an arbitrary existing password secret.
