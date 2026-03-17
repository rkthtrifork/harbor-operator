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
  Schedule type and cron expression. `Manual` is not supported for
  `ScanAllSchedule`. For all types except `None`, Harbor expects a cron
  expression.

- **spec.parameters** (map, optional)
  Additional scan-all parameters.

## Common Fields

`ScanAllSchedule` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Only one `ScanAllSchedule` may manage a given Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.
  Creates or updates the scan-all schedule in Harbor.
