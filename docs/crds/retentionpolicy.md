# Retention Policy CRD

A **RetentionPolicy** custom resource manages Harbor retention policies via
`/api/v2.0/retentions`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: RetentionPolicy
metadata:
  name: harbor-retention
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  algorithm: or
  projectRef:
    name: project-sample
  trigger:
    kind: Schedule
    settings:
      cron: "0 0 0 * * *"
  rules:
    - action: retain
      template: latestPushedK
      params:
        latestPushedK:
          value: 10
      tagSelectors:
        - kind: doublestar
          decoration: matches
          pattern: "**"
      scopeSelectors:
        repository:
          - kind: doublestar
            decoration: repoMatches
            pattern: "**"
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.algorithm** (string, optional)
  Retention algorithm, e.g. `or`.

- **spec.rules** (array, required)
  Retention rules matching the Harbor retention API schema.

- **spec.projectRef** (object, optional)
  Reference to a Project CR. When set, the controller derives the Harbor project
  ID and applies `scope.level=project` and `scope.ref=<projectID>`.

- **spec.scope** (object, optional)
  Scope for the policy. Use `level` and `ref` to target a specific project.
  Must not be set with `projectRef`.

- **spec.trigger** (object, required)
  Defines when the policy runs. Harbor requires a trigger for creation (for
  scheduled execution, supply `kind: Schedule` and `settings.cron`).

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create/Update**
  Creates or updates the retention policy in Harbor.

- **Delete**
  Deletes the retention policy in Harbor if the ID is known.
