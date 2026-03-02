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
  harborConnectionRef: "my-harbor"
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

## Behavior

- **Create / Update**
  Creates or updates the policy in Harbor.

- **Delete**
  Deletes the policy in Harbor when the CR is deleted.

- **Adoption**
  If `allowTakeover` is true, a policy with the same name is adopted.
