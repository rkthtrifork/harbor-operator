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

- **spec.projectRef** (object, required)
  Project to attach the policy to.

- **metadata.name** (string, required)
  The Harbor webhook policy name managed by this CR.

- **spec.eventTypes** (array, required)
  Harbor webhook event types.

- **spec.targets** (array, required)
  Webhook targets (type, address, optional authHeader). Each target's
  `skipCertVerify` defaults to `false`; enabling it disables TLS certificate
  verification and is insecure.

- **spec.enabled** (bool, optional)
  Enables or disables the policy. Defaults to `true`.

- **spec.creationPolicy** (string, optional)
  Controls whether the policy is created, adopted, or either. When omitted, uses the operator's default creation policy (`Create` unless configured otherwise).

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
  A policy with the same name is adopted when `creationPolicy` permits adoption.
