# Immutable Tag Rule CRD

An **ImmutableTagRule** custom resource manages Harbor immutable tag rules via
`/api/v2.0/projects/{project}/immutabletagrules`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ImmutableTagRule
metadata:
  name: immutable-tags
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  projectRef:
    name: my-project

  action: immutable
  template: repoMatches
  tagSelectors:
    - kind: doublestar
      decoration: matches
      pattern: "**"
```

## Key Fields

- **spec.projectRef** / **spec.projectNameOrID** (one required)
  Project to attach the rule to.

- **spec.action**, **spec.template** (string, optional)
  Defines the immutable rule behavior.

- **spec.tagSelectors** / **spec.scopeSelectors** (optional)
  Selector definitions for tag and scope matching.

- **spec.disabled** (bool, optional)
  Disable the rule without deleting it.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create / Update**
  Creates or updates the rule in Harbor.

- **Delete**
  Deletes the rule in Harbor when the CR is deleted.
