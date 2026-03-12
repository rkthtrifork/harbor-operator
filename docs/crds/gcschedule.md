# GC Schedule CRD

A **GCSchedule** custom resource manages Harbor garbage collection scheduling via
`/api/v2.0/system/gc/schedule`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: GCSchedule
metadata:
  name: harbor-gc-schedule
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  schedule:
    type: Custom
    cron: "0 0 2 * * *"
  parameters:
    delete_untagged: true
    workers: 1
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.schedule.type** (string, required)
  One of: `Hourly`, `Daily`, `Weekly`, `Custom`, `Manual`, `None`, `Schedule`.

- **spec.schedule.cron** (string, optional)
  Cron expression. Harbor requires this for any scheduled run (all types except
  `Manual` and `None`).

- **spec.parameters** (map, optional)
  GC parameters passed through to Harbor (for example `delete_untagged` and
  `workers`).

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create/Update**
  Only one `GCSchedule` may manage a given Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.
  Applies the GC schedule to Harbor.

- **Delete**
  Removing the CR does not delete the Harbor schedule. The CR is simply removed.
