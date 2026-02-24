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
  harborConnectionRef: "my-harbor"
  schedule:
    type: Custom
    cron: "0 0 2 * * *"
  parameters:
    delete_untagged: true
    workers: 1
```

## Key Fields

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

- **spec.schedule.type** (string, required)
  One of: `Hourly`, `Daily`, `Weekly`, `Custom`, `Manual`, `None`, `Schedule`.

- **spec.schedule.cron** (string, optional)
  Cron expression. Harbor requires this for any scheduled run (all types except
  `Manual` and `None`).

- **spec.parameters** (map, optional)
  GC parameters passed through to Harbor (for example `delete_untagged` and
  `workers`).

## Behavior

- **Create/Update**
  Applies the GC schedule to Harbor.

- **Delete**
  Removing the CR does not delete the Harbor schedule. The CR is simply removed.
