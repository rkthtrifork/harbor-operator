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

`Quota` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Update**
  Updates the project's hard quota limits in Harbor.
