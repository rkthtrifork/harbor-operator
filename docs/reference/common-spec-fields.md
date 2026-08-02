# Common Spec Fields

Most Harbor-managed custom resources embed `HarborSpecBase`. Non-owning claims
such as `UserGroupClaim` embed `HarborClaimSpecBase`, which intentionally omits
Harbor-side deletion policy. Both bases provide the same connection, drift, and
reconcile controls; resource guides focus on resource-specific behavior.

For the exact generated schema, defaults, and validation markers, see
[HarborSpecBase](api.md#harborspecbase) in the API reference.

## Shared Fields

- **`spec.harborConnectionRef`**
  Selects the Harbor connection object to use. Set `name` and, when needed,
  `kind` to choose between `HarborConnection` and `ClusterHarborConnection`.
  The `kind` defaults to `HarborConnection`.
  When the operator is started with `--harbor-connection`, this field may be
  omitted and the operator-wide `ClusterHarborConnection` is used instead.

- **`spec.deletionPolicy`**
  Controls what happens when the Kubernetes object is deleted. `Delete`
  attempts Harbor-side cleanup before removing the finalizer. `Orphan` skips
  Harbor-side deletion and removes the finalizer so the Kubernetes object can
  be deleted immediately. Defaults to `Delete`.

- **`spec.driftDetectionInterval`**
  Enables periodic drift checks between the desired state in Kubernetes and the
  current state in Harbor. If omitted, `--default-drift-detection-interval` is
  used. An explicit value of `0` disables periodic drift detection even when the
  operator has a non-zero default.

- **`spec.reconcileNonce`**
  Forces an immediate reconcile when the value changes. Use it when you want to
  trigger a refresh without changing any functional spec fields.

## Creation Policy

Resources that can uniquely discover an existing Harbor resource expose
`spec.creationPolicy`:

- `Create` creates a new resource and reports a conflict if a match already exists.
- `Adopt` requires a matching resource and reports an error instead of creating one when no match exists.
- `CreateOrAdopt` adopts a matching resource when present and creates one otherwise.

When `spec.creationPolicy` is omitted, the operator uses
`--default-creation-policy`, whose default is `Create`. An explicit resource
value always takes precedence.

After creation or adoption, the operator fully reconciles the Harbor resource.
`spec.deletionPolicy` independently controls whether deleting the Kubernetes object
also deletes the managed Harbor resource.

The status of every Harbor-backed resource records `resolvedHarborConnection` once
the first connection is selected. The binding contains the connection kind,
name, namespace, and Kubernetes UID. A later change to the effective connection
is reported as `HarborConnectionChanged` and does not mutate or delete Harbor
state.
