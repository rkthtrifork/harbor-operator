# Technical Architecture

`harbor-operator` is a controller-runtime application that translates Kubernetes desired state into Harbor API operations. Kubernetes stores the declared and observed state; Harbor remains the external system being reconciled.

## Runtime shape

```text
Kubernetes API
  ├─ custom resources and connection objects
  ├─ referenced Secrets
  └─ status and finalizers
          ↕
controller-runtime manager
  ├─ cache, watches, field indexes, and work queues
  └─ one reconciler per custom resource kind
          ↓
connection resolution and typed Harbor client
          ↓
Harbor API
```

The binary in `cmd/main.go` validates operator-wide settings, configures the manager cache, registers every reconciler, and exposes health and optional metrics endpoints. `--watch-namespaces` limits the namespaced objects held by the manager cache. Secret reads use the manager's direct API reader so credentials are not served from that cache.

## Component boundaries

| Area | Responsibility |
| --- | --- |
| `api/v1alpha1` | Public Kubernetes API types, validation, defaults, status shape, and Kubebuilder markers. |
| `cmd` | Process configuration and controller-runtime manager wiring. |
| `internal/controller` | Kubernetes watches, connection and reference resolution, reconciliation, finalization, status, and drift detection. |
| `internal/harborclient` | Typed Harbor request and response models, HTTP transport, pagination, error classification, and Harbor API operations. |
| `internal/metrics` | Harbor request observations exposed through controller-runtime metrics. |
| `charts/harbor-operator` | Installation, runtime configuration, RBAC, and packaged CRDs. |

Controllers depend on the Kubernetes API and `internal/harborclient`; the Harbor client has no Kubernetes reconciliation responsibilities. This keeps Harbor protocol behavior independently testable and leaves ownership and lifecycle policy in the controllers.

## Reconciliation model

A Harbor-backed reconciler follows the same lifecycle:

1. Load the custom resource and mark a new generation as reconciling.
2. Resolve its `HarborConnection` or `ClusterHarborConnection`, then construct a client from the referenced credentials and CA material.
3. If deletion is in progress, apply the deletion policy and remove the finalizer when the Harbor-side obligation is complete.
4. Ensure the finalizer for an active resource.
5. Apply defaults and, when permitted, discover and adopt an existing Harbor identity.
6. Create the Harbor object when no remote identity is recorded, or compare and update the existing object.
7. Write the remote identity and conditions to status, then schedule drift detection when configured.

Operational failures are written to status and returned to controller-runtime for retry. When a controller discovers that a recorded Harbor object no longer exists, it clears the stale remote identity so a later reconciliation can recreate or readopt it according to policy.

Connection objects have their own reconcilers that validate the URL and check anonymous reachability or authenticated access. Harbor-backed controllers index connection references, so a connection change enqueues its dependent resources. With `--harbor-connection`, all Harbor-backed resources use one named `ClusterHarborConnection`; changes to that object fan out across all Harbor-backed resources in the watched namespaces.

See [Common Spec Fields](reference/common-spec-fields.md), [Connection Patterns](reference/connection-patterns.md), and [Deletion and Ownership](reference/deletion-and-ownership.md) for the user-visible contracts.

## State and ownership

The custom resource spec is desired state. Status records observations needed to continue reconciliation, including Harbor identifiers and standard conditions. Named Harbor resources use Kubernetes `metadata.name` as their external identity, while relationships between managed resources use Kubernetes references and referenced status.

Finalizers keep a custom resource present while Harbor-side deletion is required. `deletionPolicy: Orphan` removes that obligation. Singleton Harbor APIs have explicit ownership arbitration because several Kubernetes objects must not overwrite the same Harbor configuration silently.

## Source and generated boundaries

```text
Go API types and Kubebuilder markers
  ├─→ DeepCopy implementations
  ├─→ canonical CRDs under config/
  └─→ generated API reference

Controller RBAC markers
  └─→ canonical RBAC under config/

Canonical CRDs and RBAC under config/
  └─→ chart CRDs and RBAC
```

The Go types and markers are authoritative. Generated outputs are committed for delivery and review but are never edited directly. `make generate` rebuilds them, and `make verify-generated` protects the boundary against drift.

`hack/harbor-openapi.yaml` is different: it is a checked-in reference for Harbor semantics used while implementing the typed client and controllers. It is not input to code or manifest generation.

## Verification boundaries

- API and focused unit tests cover type behavior and deterministic helpers.
- Harbor client tests exercise HTTP requests, responses, pagination, and errors against test servers.
- Controller envtest suites exercise observable reconciliation against a real Kubernetes API server with simulated Harbor endpoints.
- The E2E suite deploys the operator and Harbor into Kind to validate the complete runtime and packaging path.

The canonical commands and CI mapping are documented in [Testing and Verification](contributing/testing.md). Consequential changes to this architecture belong in [architecture decision records](decisions/index.md); the current page should describe the resulting system, not preserve historical rationale.
