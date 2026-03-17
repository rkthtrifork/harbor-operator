# Label CRD

A **Label** custom resource manages Harbor labels via `/api/v2.0/labels`.
Labels can be global or project-scoped.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Label
metadata:
  name: team-blue
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  name: team-blue
  color: "#3366ff"
  scope: g
```

## Key Fields

- **spec.scope** (string, optional)
  `g` for global labels or `p` for project labels.

- **spec.projectRef** (object, optional)
  Required when using `scope: p`.

- **spec.name** (string, optional)
  Label name. Defaults to metadata.name.

## Common Fields

`Label` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates the label in Harbor.

- **Delete**
  Deletes the label in Harbor when the CR is deleted.
