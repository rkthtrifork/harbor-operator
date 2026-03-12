# Singleton APIs

Some Harbor APIs are singleton-style and therefore cannot be safely managed by multiple CRs for the same Harbor instance.

Examples:

- `Configuration`
- `GCSchedule`
- `PurgeAuditSchedule`
- `ScanAllSchedule`

## Example: ScanAllSchedule

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ScanAllSchedule
metadata:
  name: scanall-sample
spec:
  harborConnectionRef:
    name: harborconnection-sample
    kind: HarborConnection
  schedule:
    type: Daily
    cron: "0 0 0 * * *"
```

## Conflict Behavior

If two singleton CRs target the same Harbor instance for the same singleton API:

- the oldest CR remains owner
- the later CR reports a conflict

This is true even if the CRs reference different connection objects that ultimately point at the same Harbor base URL.
