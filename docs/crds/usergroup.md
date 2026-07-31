# User Group CRD

A **UserGroup** custom resource manages Harbor user groups via
`/api/v2.0/usergroups`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: UserGroup
metadata:
  name: platform-engineers
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  groupName: 56d1d2cb-0ab3-4c5f-b743-34a811d36abf
  groupType: 3
```

## Key Fields

- **metadata.name** (string, required)
  The Kubernetes identity used by references to this CR.

- **spec.groupName** (string, required)
  The exact user group name stored in Harbor. For OIDC groups, this is commonly
  the identity provider's group ID.

- **spec.groupType** (int, required)
  Group type: 1 = LDAP, 2 = HTTP, 3 = OIDC.

- **spec.ldapGroupDN** (string, optional)
  LDAP DN for LDAP groups.

- **spec.creationPolicy** (string, optional)
  Controls whether the group is created, adopted, or either. When omitted, uses the operator's default creation policy (`Create` unless configured otherwise).

## Common Fields

`UserGroup` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates the user group in Harbor.

- **Delete**
  Deletes the user group in Harbor when the CR is deleted.
