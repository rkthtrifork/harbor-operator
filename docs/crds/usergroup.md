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
  harborConnectionRef: "my-harbor"
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

## Behavior

- **Create / Update**
  Creates or updates the user group in Harbor.

- **Delete**
  Deletes the user group in Harbor when the CR is deleted.
