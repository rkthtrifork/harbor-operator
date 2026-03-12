# Quota CRD

A **Quota** custom resource manages Harbor project quotas via `/api/v2.0/quotas`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Quota
metadata:
  name: project-quota
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  projectRef:
    name: my-project
  hard:
    storage: 1073741824
```

## Key Fields

- **spec.projectRef** / **spec.projectNameOrID** (one required)
  Project whose quota should be updated.

- **spec.hard** (map, optional)
  Hard limits for quota resources.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Update**
  Updates the project's hard quota limits in Harbor.
