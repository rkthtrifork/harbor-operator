# UserGroupClaim CRD

A **UserGroupClaim** is a namespaced, non-owning claim that an external group
must be registered in a Harbor instance. Harbor's `UserGroup` is a global
Harbor object; this CRD deliberately does not model its deletion lifecycle.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: UserGroupClaim
metadata:
  name: platform-engineers
  namespace: tenant-a
spec:
  harborConnectionRef:
    name: my-harbor
    kind: ClusterHarborConnection
  groupName: 56d1d2cb-0ab3-4c5f-b743-34a811d36abf
  groupType: 3
```

Reference the claim from a `Member`:

```yaml
spec:
  projectRef:
    name: tenant-a-apps
  role: developer
  memberGroup:
    groupClaimRef:
      name: platform-engineers
```

## Key Fields

- **metadata.name** is the Kubernetes reference identity. It is intentionally
  separate from the external `spec.groupName`, which may be a long OIDC group
  ID or another provider-specific identity.
- **spec.groupName** is the exact external group name stored in Harbor.
- **spec.groupType** is the Harbor group type: `1` = LDAP, `2` = HTTP, `3` =
  OIDC.
- **spec.ldapGroupDN** is the LDAP DN when `groupType` is `1`.

The identity fields are immutable. Delete and recreate the claim to request a
different external group.

## Behavior and ownership

- The operator searches Harbor for the exact group identity and creates it when
  it is absent. A concurrent create conflict is resolved by searching again.
- Multiple claims may resolve to the same Harbor UserGroup and all report the
  same Harbor group ID.
- Claims never update identity attributes on an existing group and never delete
  the Harbor UserGroup. Deleting a global Harbor UserGroup also deletes every
  project membership for that group, including memberships owned by other
  claims.
- A claim cannot be deleted while an active `Member` still references it. This
  finalizer is dependency ordering only; it is not Harbor ownership.
- If the Harbor UserGroup is removed out of band, the next claim reconciliation
  recreates it; referenced `Member` resources then restore their project
  memberships. Set `spec.driftDetectionInterval` (or the operator's default
  drift interval) when the operator should detect that change without another
  Kubernetes event.

`UserGroupClaim` embeds `HarborClaimSpecBase`, which provides the connection,
drift-detection, and reconcile-nonce fields. See [Connection Patterns](../reference/connection-patterns.md)
and [Multi-Tenancy](../reference/multi-tenancy.md) for the trust-boundary
implications.
