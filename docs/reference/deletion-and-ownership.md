# Deletion and Ownership

This page summarizes the ownership and delete behaviors that are easy to miss when using the operator.

## deletionPolicy

Harbor-backed resources embed `spec.deletionPolicy`.

Available values:

- `Delete`
- `Orphan`

`Delete` is the default.

## Delete

With `deletionPolicy: Delete`, the operator tries to clean up the corresponding Harbor-side object before removing the Kubernetes finalizer.

This is the normal managed-lifecycle mode.

## Orphan

With `deletionPolicy: Orphan`, the operator removes the Kubernetes finalizer even if Harbor cleanup cannot be completed.

This is the break-glass mode when you need the Kubernetes object gone even though Harbor cleanup is impossible or undesirable.

## Connection Deleted First

If the referenced connection object disappears first:

- resources that still need Harbor-side cleanup can remain in `Terminating` under `Delete`
- setting `Orphan` allows them to be deleted from Kubernetes without Harbor cleanup

## Robot Secret Ownership

Robot credentials are operator-managed output:

- the operator writes the destination secret
- the secret is not used as an input password source
- unrelated pre-existing secrets are not silently adopted

## Singleton Ownership

The following map to singleton Harbor APIs:

- `Configuration`
- `GCSchedule`
- `PurgeAuditSchedule`
- `ScanAllSchedule`

Only one CR may own each singleton API per Harbor instance. If multiple CRs target the same Harbor instance for the same singleton API, the oldest CR remains owner and the others report a conflict.
