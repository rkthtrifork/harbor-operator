# Quota CRD

A **Quota** custom resource manages Harbor project quotas via `/api/v2.0/quotas`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Quota
metadata:
  name: project-quota
spec:
  harborConnectionRef: "my-harbor"
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

## Behavior

- **Update**
  Updates the project's hard quota limits in Harbor.
