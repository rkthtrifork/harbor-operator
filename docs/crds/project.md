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
  harborConnectionRef: "my-harbor"

  # Optional explicit name for the Harbor project.
  # If empty, defaults to metadata.name.
  name: ""

  # Make project public? (Harbor uses "public" metadata under the hood.)
  public: false

  # Enable security scanning and related metadata settings.
  autoScan: true
  severity: "high"
  preventVul: true

  # Optional: drift detection for the project configuration.
  driftDetectionInterval: 5m
  reconcileNonce: "bump-1"
```

> The exact metadata set (auto-scan, severity, etc.) depends on your CRD schema,
> but the operator maps those fields into Harbor’s project metadata.

## Key Fields

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

- **spec.name** (string, optional)
  Name of the Harbor project.

  - If empty, `metadata.name` is used.

- **spec.public** (bool, optional)
  Controls whether the project is public or private.

- **spec.autoScan**, **spec.preventVul**, **spec.severity**, etc. (optional)
  These map to Harbor’s project metadata fields, controlling:

  - automatic scanning of images,
  - vulnerability blocking behavior,
  - minimum severity threshold, etc.

- **spec.driftDetectionInterval** (duration, optional)
  Periodic check for drift between Harbor’s project config and the CR.

- **spec.reconcileNonce** (string, optional)
  Update this to force a reconcile on demand.

## Behavior

- **Create / Update**

  - Ensures the project exists in Harbor.
  - Updates metadata to match your spec.

- **Delete**

  - Via finalizer, attempts to delete the project in Harbor when the CR is deleted.
  - If the project no longer exists, deletion is considered successful.

- **Drift detection**

  - Optional periodic reconciliation to keep Harbor’s project settings aligned
    with the CR.
