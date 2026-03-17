# Webhook Policy CRD

A **WebhookPolicy** custom resource manages Harbor project webhook policies via
`/api/v2.0/projects/{project}/webhook/policies`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: WebhookPolicy
metadata:
  name: build-webhooks
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  projectRef:
    name: my-project

  eventTypes:
    - PUSH_ARTIFACT

  targets:
    - type: http
      address: https://hooks.example.com/harbor
      payloadFormat: CloudEvents
```

## Key Fields

- **spec.projectRef** / **spec.projectNameOrID** (one required)
  Project to attach the policy to.

- **spec.eventTypes** (array, required)
  Harbor webhook event types.

- **spec.targets** (array, required)
  Webhook targets (type, address, optional authHeader).

- **spec.enabled** (bool, optional)
  Enables or disables the policy.

## Common Fields

`WebhookPolicy` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates the policy in Harbor.

- **Delete**
  Deletes the policy in Harbor when the CR is deleted.

- **Adoption**
  If `allowTakeover` is true, a policy with the same name is adopted.
