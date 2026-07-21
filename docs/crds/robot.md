# Robot CRD

A **Robot** custom resource manages Harbor robot accounts using the
`/api/v2.0/robots` API. It supports both system- and project-level robots.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Robot
metadata:
  name: ci
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  creationPolicy: Create
  level: project
  permissions:
  - kind: project
    namespace: "library"
    access:
    - resource: repository
      action: pull
      effect: allow
    - resource: repository
      action: push
      effect: allow
  secretRef:
    name: harbor-robot-ci
    key: secret
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.level** (string, required)
  Robot scope. Must be `system` or `project`.

- **spec.permissions** (array, required)
  Permissions granted to the robot. Each permission includes a `kind`, optional
  `namespace`, and one or more access rules. The `namespace` is the Harbor
  project name for `kind: project`. Each access rule's `effect` defaults to `allow`.

- **spec.disable** (bool, optional)
  Controls whether the robot is disabled. When omitted, the operator leaves the
  value unset during creation and preserves the current value during updates.

- **spec.duration** (int, optional)
  Duration in days. Use `-1` for never expires. If omitted, it defaults to `-1`.

- **spec.secretRef** (object, optional)
  Reference to the operator-managed secret where the generated robot secret is written.
  If omitted, the operator creates `<metadata.name>-secret` with key `secret`.
  If the Secret already exists, it must already be managed by the same `Robot`.

- **spec.creationPolicy** (string, optional)
  Controls whether the robot is created, adopted, or either. When omitted, uses the operator's default creation policy (`Create` unless configured otherwise).

Robot secrets are rotated automatically once Harbor reports that the robot
credential has expired (based on `expires_at`). The operator then refreshes the
secret and stores it in the referenced Secret.

## Common Fields

`Robot` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create**

  - Creates the robot account with the requested permissions.
  - Uses `metadata.name` as the Harbor robot name.
  - Applies `creationPolicy` when the robot is not yet recorded in status.

- **Update**

  - Updates description, permissions, disabled state, and duration.
  - Rotates the Harbor credential when Harbor reports the current secret as expired.
  - Writes the rotated value back to the operator-managed Secret.

- **Delete**

  - Deletes the robot account in Harbor.

## Notes

- `spec.secretRef` is a destination for operator-managed output, not an input source.
- The controller does not adopt or overwrite unrelated existing Secrets.
