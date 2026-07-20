# Project CRD

A **Project** custom resource represents a Harbor project and its configuration:
visibility, security settings, auto-scan, and more.

The operator ensures that the project exists in Harbor and matches the desired spec.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: my-project
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection

  creationPolicy: Create

  # Make project public? (Harbor uses "public" metadata under the hood.)
  public: false

  # Enable security scanning and related metadata settings.
  metadata:
    auto_scan: "true"
    severity: high
    prevent_vul: "true"

  # Optional: drift detection for the project configuration.
  driftDetectionInterval: 5m
  reconcileNonce: "bump-1"
```

To create a proxy-cache project, reference a `Registry` when creating the Project:

```yaml
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  public: false
  registryRef:
    name: upstream-registry
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.public** (bool, required)
  Controls whether the project is public or private.

- **metadata.name** (string, required)
  The Harbor project name managed by this CR.

- **spec.creationPolicy** (string, optional)
  Controls whether the project is created, adopted, or either. Defaults to `Create`.

- **spec.metadata** (object, optional)
  These map to Harbor’s project metadata fields, controlling:

  - automatic scanning of images,
  - vulnerability blocking behavior,
  - minimum severity threshold, etc.

- **spec.registryRef** (object, optional, immutable)
  References the `Registry` used to create a Harbor proxy-cache project. The referenced resource must exist and have a Harbor registry ID before the Project can be created. Harbor cannot convert an existing project to or from a proxy-cache project, so this reference cannot be added, removed, or changed after creation. Recreate the Project to select a different proxy-cache mode or registry.

- **spec.driftDetectionInterval** (duration, optional)
  Periodic check for drift between Harbor’s project config and the CR.

- **spec.reconcileNonce** (string, optional)
  Update this to force a reconcile on demand.

## Common Fields

`Project` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**

  - Ensures the project exists in Harbor.
  - Updates metadata to match your spec.
  - Creates a proxy-cache project when `registryRef` is set at creation time.
  - Applies `creationPolicy` when the project is not yet recorded in status.

- **Delete**

  - Via finalizer, attempts to delete the project in Harbor when the CR is deleted.
  - If the project no longer exists, deletion is considered successful.

- **Drift detection**

  - Optional periodic reconciliation to keep Harbor’s project settings aligned
    with the CR.
