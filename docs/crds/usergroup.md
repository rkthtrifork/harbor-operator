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
  groupName: developers
  groupType: 2
```

## Key Fields

- **spec.groupName** (string, optional)
  Group name. Defaults to metadata.name.

- **spec.groupType** (int, required)
  Group type: 1 = LDAP, 2 = HTTP, 3 = OIDC.

- **spec.ldapGroupDN** (string, optional)
  LDAP DN for LDAP groups.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create / Update**
  Creates or updates the user group in Harbor.

- **Delete**
  Deletes the user group in Harbor when the CR is deleted.
