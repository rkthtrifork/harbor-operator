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

- **spec.projectRef** (object, required)
  Project to attach the rule to.

- **spec.action**, **spec.template** (string, optional)
  Defines the immutable rule behavior.

- **spec.tagSelectors** / **spec.scopeSelectors** (optional)
  Selector definitions for tag and scope matching.

- **spec.disabled** (bool, optional)
  Disable the rule without deleting it.

## Common Fields

`ImmutableTagRule` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates the rule in Harbor.

- **Delete**
  Deletes the rule in Harbor when the CR is deleted.
