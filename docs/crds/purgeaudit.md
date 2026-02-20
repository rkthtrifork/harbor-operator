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
  harborConnectionRef: "my-harbor"
  schedule:
    type: Custom
    cron: "0 30 2 * * *"
  parameters:
    auditRetentionHour: 168
    includeEventTypes: "create_artifact,delete_artifact,pull_artifact"
    dryRun: false
```

## Key Fields

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

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

## Behavior

- **Create/Update**
  Applies the purge audit schedule to Harbor.

- **Delete**
  Removing the CR does not delete the Harbor schedule. The CR is simply removed.
