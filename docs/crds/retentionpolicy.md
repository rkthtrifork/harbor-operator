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

`RetentionPolicy` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create/Update**
  Creates or updates the retention policy in Harbor.

- **Delete**
  Deletes the retention policy in Harbor if the ID is known.
