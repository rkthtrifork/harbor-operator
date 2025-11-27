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
  harborConnectionRef: "my-harbor"

  # Project reference in Harbor (name or ID, depending on your usage).
  projectRef: "my-project"

  # Role within the project. The operator converts this to Harbor's role ID.
  role: "developer"

  memberUser:
    # Either an existing Harbor user ID...
    # userID: 123
    # ...or a username (the operator will resolve it as needed).
    username: "alice"
```

### Example: group member

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Member
metadata:
  name: my-project-dev-group
spec:
  harborConnectionRef: "my-harbor"
  projectRef: "my-project"
  role: "maintainer"

  memberGroup:
    # Group ID in Harbor, if already known.
    id: 42
    # Optional group name.
    groupName: "dev-team"
    # Group type / backend, e.g. LDAP (integer code as in Harbor API).
    groupType: 1
    # For LDAP groups, the group DN.
    ldapGroupDN: "cn=dev-team,ou=groups,dc=example,dc=com"
```

## Key Fields

- **spec.harborConnectionRef** (string, required)
  HarborConnection to use.

- **spec.projectRef** (string, required)
  Project identifier in Harbor:

  - Often a project name.
  - Depending on your client usage, may also be a numeric ID.

- **spec.role** (string, required)
  Human-readable role name. The operator maps this to a Harbor role ID, e.g.:

  - `admin` → 1
  - `developer` → 2
  - `guest` → 3
  - `maintainer` → 4

  (Exactly as implemented in your controller.)

- **spec.memberUser** (object, optional)
  Defines a user member:

  - **userID** (int, optional) – existing Harbor user ID.
  - **username** (string, optional) – Harbor username.

- **spec.memberGroup** (object, optional)
  Defines a group member:

  - **id** (int, optional) – existing Harbor group ID.
  - **groupName** (string, optional)
  - **groupType** (int, optional) – Harbor group type (e.g. internal, LDAP).
  - **ldapGroupDN** (string, optional) – DN for LDAP groups.

Exactly one of `memberUser` or `memberGroup` should be set.

## Behavior

- **Create**

  - Converts `role` to Harbor’s role ID.
  - Calls Harbor’s project membership API with either `member_user` or `member_group`.
  - Treats “already exists” responses as idempotent (depending on client handling).

- **Update**

  - Changing `role` will update the member’s role in Harbor (if supported by your client).
  - Changing member identity is typically treated as a delete+create scenario.

- **Delete**

  - On CR deletion, the operator attempts to remove the corresponding member from Harbor.

- **Error handling**

  - If Harbor returns an error (e.g. unknown user, unknown project), the operator
    logs the details so you can diagnose configuration issues.
