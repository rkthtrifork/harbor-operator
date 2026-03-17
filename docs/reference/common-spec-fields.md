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

- **`spec.deletionPolicy`**
  Controls what happens when the Kubernetes object is deleted. `Delete`
  attempts Harbor-side cleanup before removing the finalizer. `Orphan` removes
  the finalizer even if Harbor cleanup cannot be completed.

- **`spec.driftDetectionInterval`**
  Enables periodic drift checks between the desired state in Kubernetes and the
  current state in Harbor. If omitted or set to `0`, periodic drift detection is
  disabled.

- **`spec.reconcileNonce`**
  Forces an immediate reconcile when the value changes. Use it when you want to
  trigger a refresh without changing any functional spec fields.
