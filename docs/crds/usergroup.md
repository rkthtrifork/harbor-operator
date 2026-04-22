# User Group CRD

A **UserGroup** custom resource manages Harbor user groups via
`/api/v2.0/usergroups`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: UserGroup
metadata:
  name: developers
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  groupType: 2
```

## Key Fields

- **metadata.name** (string, required)
  The Harbor user group name managed by this CR.

- **spec.groupType** (int, required)
  Group type: 1 = LDAP, 2 = HTTP, 3 = OIDC.

- **spec.ldapGroupDN** (string, optional)
  LDAP DN for LDAP groups.

## Common Fields

`UserGroup` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates the user group in Harbor.

- **Delete**
  Deletes the user group in Harbor when the CR is deleted.
