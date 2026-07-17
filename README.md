# harbor-operator

> [!IMPORTANT]
> This operator does **not** install Harbor in “real” clusters.  
> It assumes Harbor is already running and introduces a **HarborConnection** CRD
> or **ClusterHarborConnection** CRD that stores connection details for that instance.  
> All other CRDs reference one of those connection objects.

**harbor-operator** is a Kubernetes operator that lets you manage Harbor resources
— registries, projects, users, and members — as Kubernetes Custom Resources (CRs).

Instead of clicking around in the Harbor UI, you describe your desired state in YAML.
The operator then reconciles that state with your Harbor instance via its API.

> [!NOTE]
> You may see `kubectl` warnings like `unrecognized format "int64"` when applying CRDs.
> These are client-side validation warnings from `kubectl` and are safe to ignore.

## Concepts

- **HarborConnection**  
  Connection details for an existing Harbor instance (base URL, optional credentials).
  Namespaced and intended for tenant-local use.

- **ClusterHarborConnection**
  Cluster-scoped connection details for a shared Harbor instance.
  Use this when multiple namespaces should reference the same Harbor endpoint and credentials.

- **Registry**  
  Represents a Harbor “target registry” (e.g. GHCR) and its configuration.

- **Project**  
  Represents a Harbor project, including metadata (public/private, auto-scan, etc.).

- **User**  
  Represents a Harbor user and (optionally) its lifecycle.

- **Member**  
  Represents membership of a user or group in a Harbor project with a given role.

- **Robot**  
  Represents a Harbor robot account (system or project level).

- **Configuration**  
  Represents Harbor system configuration (OIDC, robot defaults, etc.).

- **ReplicationPolicy**  
  Represents Harbor replication policies between registries.

- **WebhookPolicy**  
  Represents project-level Harbor webhook policies.

- **ImmutableTagRule**  
  Represents project-level immutable tag rules.

- **Label**  
  Represents Harbor labels (global or project-scoped).

- **UserGroup**  
  Represents Harbor user groups (LDAP/HTTP/OIDC).

- **ScannerRegistration**  
  Represents Harbor scanner registrations.

- **ScanAllSchedule**  
  Represents Harbor scan-all scheduling configuration.

- **Quota**  
  Represents Harbor project quota configuration.

## Getting Started

### Prerequisites

- Go
- Docker (or compatible container runtime)
- kubectl
- A Kubernetes cluster (Kind is fine for dev)

## Local Development with Kind

This repo ships with a Kind-based dev environment that:

- Creates a Kind cluster
- Installs Traefik
- Installs Harbor via the official Helm chart
- Builds and deploys harbor-operator into that cluster

> This is just for development convenience. In a real environment, you bring your own Harbor.

### 1. Hosts Entry (for Harbor ingress)

Add this line to `/etc/hosts` (or your platform’s hosts file):

```sh
127.0.0.1 core.harbor.domain
```

### 2. Start Kind + Harbor + Operator

```sh
make kind-up
```

This will:

- Create a Kind cluster using `hack/kind-configuration.yaml`
- Install Traefik (NodePorts 30080/30443 by default)
- Install Harbor via Helm
- Build a local `harbor-operator:local` image, load it into Kind, and deploy it

`make kind-up` uses Kind's default CNI and has the fastest startup time.

### Optional: Start Kind with Cilium + Hubble

If you want Cilium to be the CNI (so Hubble can observe pod traffic), create
the cluster with Cilium from the start:

```sh
make kind-up KIND_CNI=cilium
```

`KIND_CNI=cilium` installs Cilium and Hubble. Startup is slower than the default Kind CNI.

### 3. Working with Samples

Apply or remove sample CRs (in `config/samples`):

```sh
# Apply sample HarborConnection, Registry, Project, etc.
make apply-samples

# Delete all Harbor CRs (connection objects last) in all namespaces
make delete-harbor-crs
```

> `delete-harbor-crs` removes **all** custom resources in the
> `harbor.harbor-operator.io` API group, not just the manifests in `config/samples/`.

### 4. Normal Edit / Test Loop

For iterative development:

```sh
make kind-refresh
```

This will:

- Rebuild the operator image
- Load it into the existing Kind cluster
- Regenerate code and manifests
- Sync CRDs and RBAC into the Helm chart
- Apply the latest CRDs to the cluster
- Redeploy the operator with Helm

The controller runs inside the Kind cluster, so code changes take effect after
rebuilding and redeploying the operator image.

Use `make kind-refresh` after changing:

- controller logic
- Harbor client code
- CRD Go types
- generated CRDs
- RBAC
- Helm chart wiring for the operator

### 5. Full Reset / Clean Redeploy

If you want to wipe operator-managed CRs, remove the operator, remove CRDs, and
then install everything again from scratch:

```sh
make kind-redeploy
```

This will:

- Delete all Harbor CRs managed by the operator (connection objects last)
- Remove the operator deployment and CRDs
- Rebuild the image
- Load it into Kind
- Reinstall CRDs and redeploy the operator

Typical uses:

- you intentionally want a clean slate
- a CRD change is incompatible with existing CRs
- you want to re-test first-install behavior

### 6. Useful Low-Level Targets

The high-level workflow above is usually enough, but these targets are useful
when working on packaging and deployment details:

```sh
make generate-manifests
make sync-chart-assets
make apply-crds
make docker-build
```

- `generate-manifests` regenerates canonical CRDs and RBAC
- `sync-chart-assets` copies generated CRDs and RBAC into the Helm chart
- `apply-crds` applies the current chart CRDs to the cluster
- `docker-build` builds the operator image without deploying it

### 7. Tearing Down

To delete only the Kind cluster:

```sh
make kind-down
```

## Documentation

The repository ships two documentation layers:

- hand-written operator guides under `docs/crds/`
- generated schema reference under `docs/reference/api.md`

The documentation site is built with MkDocs Material and deployed from `main` to GitHub Pages.
In the repository settings, GitHub Pages should be configured to deploy from GitHub Actions.

Generate only the API reference with:

```sh
make generate-api-reference
```

Build the MkDocs site locally with:

```sh
make docs-build
```

Serve it locally with:

```sh
make docs-serve
```

## Helm Chart (OCI)

We publish an OCI Helm chart to GHCR.

```sh
helm registry login ghcr.io
helm install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator --version <chart-version>
```

To render locally:

```sh
helm template harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator --version <chart-version>
```

Values are documented in `charts/harbor-operator/values.yaml` and validated by `charts/harbor-operator/values.schema.json`.

Create either a `HarborConnection` or a `ClusterHarborConnection`, then point your Harbor CRs at it with `spec.harborConnectionRef`.

## Behavioral Notes

- Updates to `HarborConnection` and `ClusterHarborConnection` trigger reconciliation of Harbor-backed CRs that reference them.
- `spec.deletionPolicy` defaults to `Delete`. Use `Orphan` when you want Kubernetes deletion to proceed even if Harbor cleanup cannot be completed.
- `Configuration`, `GCSchedule`, `PurgeAuditSchedule`, and `ScanAllSchedule` are singleton-style Harbor APIs. Only one CR may manage each of those per Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.

## Metrics

The operator supports Prometheus metrics via controller-runtime. Metrics are **disabled by default**.

To enable via Helm:

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \
  --version <chart-version> \
  --set metrics.enabled=true
```

## Uninstalling

If you want to remove Harbor-managed resources, CRDs, and the operator:

```sh
# Remove Harbor CRs (connection objects last)
make delete-harbor-crs

# Remove the operator deployment while retaining its CRDs
make undeploy

# Remove CRDs for the harbor.harbor-operator.io API group
make delete-crds
```

In a Kind dev cluster, a full reset is just:

```sh
make kind-reset
# or
make kind-down
```

## Contributing

### New Contributor Workflow

If you are new to the repo, the shortest path to a working local environment is:

```sh
make kind-up
```

or, if you want Cilium/Hubble:

```sh
make kind-up KIND_CNI=cilium
```

Then iterate with:

```sh
make kind-refresh
```

And if you need a full reset:

```sh
make kind-redeploy
```

### Common Commands

- Apply automatic formatting and lint fixes:

  ```sh
  make fix
  ```

- Run tests:

  ```sh
  make test
  ```

- Run linters:

  ```sh
  make lint
  ```

- Run the normal non-E2E CI baseline (generated drift, lint, tests, and the docs build):

  ```sh
  make check
  ```

Open a PR with a clear description of what you changed and why.
