# Common Spec Fields

All Harbor-managed custom resources embed `HarborSpecBase`. That gives every CR
the same baseline controls for selecting a Harbor instance, handling deletion,
and forcing or scheduling reconciliation. Resource guides focus on
resource-specific behavior and link back here for the shared fields.

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
