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
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection

  # Email address for the Harbor user.
  email: "alice@example.com"

  # Optional real name / full name.
  realname: "Alice Example"

  creationPolicy: Create

  # Required reference to the Secret key containing the user's password.
  passwordSecretRef:
    name: harbor-user-alice-password
    key: password
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **metadata.name** (string, required)
  The Harbor username managed by this CR.

- **spec.email** (string, required by Harbor)
  Email associated with the user.

- **spec.realname** (string, optional)
  Full name / display name. Defaults to `metadata.name`.

- **spec.passwordSecretRef** (object, required)
  Reference containing the Secret `name` and `key` for the user's password.

- **spec.creationPolicy** (string, optional)
  Controls whether the user is created, adopted, or either. When omitted, uses the operator's default creation policy (`Create` unless configured otherwise).

## Common Fields

`User` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create**

  - Creates the Harbor user with the given username/email/password.
  - Applies `creationPolicy` when the user is not yet recorded in status.

- **Update**

  - Updates mutable fields such as email and real name to match the CR.

- **Delete**

  - Uses `spec.deletionPolicy` to delete or orphan the Harbor user.
  - If the user is already gone, deletion is considered successful.

- **Interaction with Member**

  - User CRs are typically referenced by `Member.spec.memberUser.userRef`.
