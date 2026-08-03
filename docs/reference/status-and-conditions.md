# Status and Conditions

Every managed resource reports reconciliation state through its Kubernetes
status. The `Ready` condition is the quickest way to tell whether the desired
Harbor object is available.

- `Ready=True` means the latest desired state was applied successfully.
- `Ready=False` means reconciliation needs attention. Read the condition's
  `reason` and `message`; they describe the failing step or dependency.
- `observedGeneration` tells you which resource generation the controller has
  processed. If it lags behind `metadata.generation`, reconciliation has not
  caught up yet.

Resources that use a Harbor connection may also report the resolved connection
identity in status. A changed or missing connection is surfaced as a condition
failure rather than silently switching Harbor instances.

During deletion, Kubernetes may keep the resource in `Terminating` while the
operator removes or verifies its Harbor object through its finalizer. If
deletion is blocked, inspect events and the resource's finalizers before
removing anything manually.

For condition-specific recovery steps, see
[Troubleshooting](../reference/troubleshooting.md). The [generated API
reference](api.md) contains the exact spec schema; this page explains the
shared status behavior that is useful during reconciliation.
