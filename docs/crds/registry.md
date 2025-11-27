


# Registry CRD

A **Registry** custom resource represents an external registry configured in Harbor
(e.g. GHCR, AWS ECR) and managed by the operator.

It references a HarborConnection and supports:

- Creation / update of the registry in Harbor
- Optional adoption of an existing registry
- Optional periodic drift detection

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Registry
metadata:
  name: my-registry
spec:
  # Reference to the HarborConnection resource.
  harborConnectionRef: "my-harbor"

  # The registry type, e.g. "github-ghcr".
  type: github-ghcr

  # Optional explicit registry name in Harbor.
  # Recommendation: leave empty to default to metadata.name.
  name: ""

  # The registry URL.
  url: "https://registry.example.com"

  # Set to true to bypass certificate verification.
  insecure: false

  # Allow adoption of an existing Harbor registry with the same name.
  allowTakeover: true

  # Periodic drift detection (e.g. "5m" for five minutes). 0 = disabled.
  driftDetectionInterval: 5m

  # Bump this to force a manual reconcile.
  reconcileNonce: "update-123"
```

> [!CAUTION]
> If `allowTakeover` is `true` and a registry with the same name already exists
> in Harbor, the operator will take control of it and update its configuration
> to match the CR.

## Key Fields

- **spec.harborConnectionRef** (string, required)
  The name of the HarborConnection to use.

- **spec.type** (string, required)
  The Harbor registry type (e.g. `github-ghcr`). Must be one of the supported types.

- **spec.name** (string, optional)
  Name of the registry in Harbor.

  - If empty, the operator uses `metadata.name`.
  - Using the CR name keeps the cluster as the single source of truth.

- **spec.url** (string, required)
  Registry URL. Validated as a URL.

- **spec.insecure** (bool, optional)
  If `true`, skips TLS verification when Harbor connects to this registry.

- **spec.allowTakeover** (bool, optional)
  If `true`, and a registry with the same name already exists in Harbor, the
  operator will:

  - adopt it,
  - store its Harbor ID in status,
  - and reconcile its configuration.

- **spec.driftDetectionInterval** (duration, optional)
  How often to re-check that Harbor’s config still matches the CR.
  `"0"` or omitted → drift detection disabled.

- **spec.reconcileNonce** (string, optional)
  Changing this value forces an immediate reconcile, even if nothing else changed.

## Behavior

- **Create**

  - Lists registries and checks for one with the desired name.
  - Creates a new registry via Harbor’s API if none exists.
  - If `allowTakeover` is `true` and a registry exists, it is adopted.

- **Update**

  - Compares desired spec with the Harbor registry.
  - Applies changes via Harbor’s update APIs.

- **Delete**

  - A finalizer ensures Harbor’s registry is deleted (if possible) on CR deletion.
  - If the stored Harbor registry ID is not found, deletion is treated as successful
    (assumed already removed).

- **Drift detection**

  - If `driftDetectionInterval` > 0, the controller requeues periodically to:

    - fetch the current registry configuration from Harbor
    - compare against the CR
    - update Harbor if drift is detected.



