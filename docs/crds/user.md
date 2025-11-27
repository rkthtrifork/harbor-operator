# User CRD

A **User** custom resource represents a Harbor user. The operator can:

- Create the user in Harbor
- Optionally manage its lifecycle (updates / deletion)
- Use it in combination with Member CRs for project roles

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: User
metadata:
  name: alice
spec:
  harborConnectionRef: "my-harbor"

  # Harbor username (defaults to metadata.name if omitted).
  username: "alice"

  # Email address for the Harbor user.
  email: "alice@example.com"

  # Optional real name / full name.
  realname: "Alice Example"

  # Optional admin flag.
  admin: false

  # Optional: reference to a secret with the initial password.
  passwordSecretRef: "harbor-user-alice-password"

  # Optional: if true, delete the user in Harbor when the CR is deleted.
  manageLifecycle: true
```

> Exact field names depend on your CRD schema; adjust the example to match your spec.

## Key Fields

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

- **spec.username** (string, optional)
  Username in Harbor.

  - If omitted, the operator may default to `metadata.name`.

- **spec.email** (string, required by Harbor)
  Email associated with the user.

- **spec.realname** (string, optional)
  Full name / display name.

- **spec.admin** (bool, optional)
  Whether this user should be a Harbor system admin.

- **spec.passwordSecretRef** (string, optional)
  Secret that contains the user’s password (key name depends on your schema,
  e.g. `password`).

- **spec.manageLifecycle** (bool, optional)
  If `true`, the operator will delete the Harbor user when the CR is deleted.
  If `false`, deletion of the CR leaves the Harbor user intact.

## Behavior

- **Create**

  - Creates the Harbor user with the given username/email/password.
  - Optionally sets admin flag.

- **Update**

  - Updates mutable fields (e.g. email, realname, admin) to match the CR.

- **Delete**

  - If `manageLifecycle` is `true`, attempts to delete the user in Harbor.
  - If the user is already gone, deletion is considered successful.

- **Interaction with Member**

  - User CRs are typically referenced indirectly by Member CRs (via username
    or user ID) to assign roles in projects.
