# Scan All Schedule CRD

A **ScanAllSchedule** custom resource manages Harbor's scan-all schedule via
`/api/v2.0/system/scanAll/schedule`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ScanAllSchedule
metadata:
  name: scanall-daily
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  schedule:
    type: Daily
    cron: "0 0 0 * * *"
```

## Key Fields

- **spec.schedule** (object, required)
  Schedule type and cron expression.

- **spec.parameters** (map, optional)
  Additional scan-all parameters.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create / Update**
  Only one `ScanAllSchedule` may manage a given Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.
  Creates or updates the scan-all schedule in Harbor.
