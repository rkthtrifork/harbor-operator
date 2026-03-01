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
  harborConnectionRef: "my-harbor"
  schedule:
    type: Daily
    cron: "0 0 0 * * *"
```

## Key Fields

- **spec.schedule** (object, required)
  Schedule type and cron expression.

- **spec.parameters** (map, optional)
  Additional scan-all parameters.

## Behavior

- **Create / Update**
  Creates or updates the scan-all schedule in Harbor.
