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
  allowTakeover: false
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
  project name for `kind: project`.

- **spec.duration** (int, optional)
  Duration in days. Use `-1` for never expires. If omitted, it defaults to `-1`.

- **spec.secretRef** (object, optional)
  Reference to the operator-managed secret where the generated robot secret is written.
  If omitted, the operator creates `<metadata.name>-secret` with key `secret`.
  If the Secret already exists, it must already be managed by the same `Robot`.

- **spec.allowTakeover** (bool, optional)
  If `true`, the operator will adopt an existing Harbor robot with the same name.

Robot secrets are rotated automatically once Harbor reports that the robot
credential has expired (based on `expires_at`). The operator then refreshes the
secret and stores it in the referenced Secret.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create**

  - Creates the robot account with the requested permissions.
  - Uses `spec.name` or defaults to `metadata.name`.
  - If `allowTakeover` is `true` and a robot already exists, it is adopted.

- **Update**

  - Updates description, permissions, disabled state, and duration.
  - Rotates the Harbor credential when Harbor reports the current secret as expired.
  - Writes the rotated value back to the operator-managed Secret.

- **Delete**

  - Deletes the robot account in Harbor.

## Notes

- `spec.secretRef` is a destination for operator-managed output, not an input source.
- The controller does not adopt or overwrite unrelated existing Secrets.
