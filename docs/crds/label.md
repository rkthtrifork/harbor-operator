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
  harborConnectionRef: "my-harbor"
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

## Behavior

- **Create / Update**
  Creates or updates the label in Harbor.

- **Delete**
  Deletes the label in Harbor when the CR is deleted.
