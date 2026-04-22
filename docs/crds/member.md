# Member CRD

A **Member** custom resource represents membership of a **user** or **group**
in a Harbor project with a specific role (admin, developer, guest, etc.).

The operator ensures that the corresponding project member exists in Harbor.

## Quick Start

### Example: user member

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Member
metadata:
  name: my-project-alice
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  allowTakeover: false

  projectRef:
    name: my-project

  # Role within the project. The operator converts this to Harbor's role ID.
  role: "developer"

  memberUser:
    userRef:
      name: alice
```

### Example: group member

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Member
metadata:
  name: my-project-dev-group
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  projectRef:
    name: my-project
  role: "maintainer"

  memberGroup:
    groupRef:
      name: dev-team
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind`
  (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.projectRef** (object, required)
  Project custom resource reference.

- **spec.role** (string, required)
  Human-readable role name. The operator maps this to a Harbor role ID, e.g.:

  - `admin` → 1
  - `developer` → 2
  - `guest` → 3
  - `maintainer` → 4

  (Exactly as implemented in your controller.)

- **spec.memberUser** (object, optional)
  References the `User` custom resource to grant membership to.

- **spec.memberGroup** (object, optional)
  References the `UserGroup` custom resource to grant membership to.

Exactly one of `memberUser` or `memberGroup` should be set.

- **spec.allowTakeover** (bool, optional)
  If `true`, the operator will adopt an existing Harbor membership for the same
  identity (user/group + project).

## Common Fields

`Member` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create**

  - Converts `role` to Harbor’s role ID.
  - Calls Harbor’s project membership API with either `member_user` or `member_group`.
  - Treats “already exists” responses as idempotent (depending on client handling).
  - If `allowTakeover` is `true` and a membership already exists, it is adopted.

- **Update**

  - Changing `role` will update the member’s role in Harbor (if supported by your client).
  - Changing member identity is typically treated as a delete+create scenario.

- **Delete**

  - On CR deletion, the operator attempts to remove the corresponding member from Harbor.

- **Error handling**

  - If Harbor returns an error (e.g. unknown user, unknown project), the operator
    logs the details so you can diagnose configuration issues.
