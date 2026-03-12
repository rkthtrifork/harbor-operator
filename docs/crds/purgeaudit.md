# Purge Audit Schedule CRD

A **PurgeAuditSchedule** custom resource manages Harbor audit log purge scheduling
via `/api/v2.0/system/purgeaudit/schedule`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: PurgeAuditSchedule
metadata:
  name: harbor-purge-audit
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  schedule:
    type: Custom
    cron: "0 30 2 * * *"
  parameters:
    auditRetentionHour: 168
    includeEventTypes: "create_artifact,delete_artifact,pull_artifact"
    dryRun: false
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.schedule.type** (string, required)
  One of: `Hourly`, `Daily`, `Weekly`, `Custom`, `Manual`, `None`, `Schedule`.

- **spec.schedule.cron** (string, optional)
  Cron expression. Harbor requires this for any scheduled run (all types except
  `Manual` and `None`).

- **spec.parameters.auditRetentionHour** (int, optional)
  Retention period in hours.

- **spec.parameters.includeEventTypes** (string, optional)
  Comma-separated event types to include.

- **spec.parameters.dryRun** (bool, optional)
  Run purge in dry-run mode.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create/Update**
  Only one `PurgeAuditSchedule` may manage a given Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.
  Applies the purge audit schedule to Harbor.

- **Delete**
  Removing the CR does not delete the Harbor schedule. The CR is simply removed.
