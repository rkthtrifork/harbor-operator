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

With `deletionPolicy: Orphan`, the operator skips Harbor-side deletion and removes the Kubernetes finalizer so the CR can disappear while the Harbor object remains.

This is the mode to use when Harbor cleanup is undesirable or when you need Kubernetes deletion to proceed without waiting on Harbor-side deletion.

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
