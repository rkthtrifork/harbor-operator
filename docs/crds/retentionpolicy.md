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
  harborConnectionRef: "my-harbor"
  algorithm: or
  scope:
    level: project
    ref: 1
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

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

- **spec.algorithm** (string, optional)
  Retention algorithm, e.g. `or`.

- **spec.rules** (array, required)
  Retention rules matching the Harbor retention API schema.

- **spec.scope** (object, optional)
  Scope for the policy. Use `level` and `ref` to target a specific project.

- **spec.trigger** (object, required)
  Defines when the policy runs. Harbor requires a trigger for creation (for
  scheduled execution, supply `kind: Schedule` and `settings.cron`).

## Behavior

- **Create/Update**
  Creates or updates the retention policy in Harbor.

- **Delete**
  Deletes the retention policy in Harbor if the ID is known.
