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
  harborConnectionRef: "my-harbor"
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

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

- **spec.level** (string, required)
  Robot scope. Must be `system` or `project`.

- **spec.permissions** (array, required)
  Permissions granted to the robot. Each permission includes a `kind`, optional
  `namespace`, and one or more access rules. The `namespace` is the Harbor
  project name for `kind: project`.

- **spec.duration** (int, optional)
  Duration in days. Use `-1` for never expires. If omitted, it defaults to `-1`.

- **spec.secretRef** (object, optional)
  Reference to a secret where the operator writes the generated robot secret.
  If omitted, the operator creates `<metadata.name>-secret` with key `secret`.

Robot secrets are rotated automatically once Harbor reports that the robot
credential has expired (based on `expires_at`). The operator then refreshes the
secret and stores it in the referenced Secret.

## Behavior

- **Create**

  - Creates the robot account with the requested permissions.
  - Uses `spec.name` or defaults to `metadata.name`.

- **Update**

  - Updates description, permissions, disabled state, and duration.
  - Refreshes the robot secret when the referenced secret changes.

- **Delete**

  - Deletes the robot account in Harbor.
