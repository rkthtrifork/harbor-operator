# harbor-operator

> [!IMPORTANT]
> This operator does **not** install Harbor in “real” clusters.  
> It assumes Harbor is already running and introduces a **HarborConnection** CRD
> that stores connection details for that instance.  
> All other CRDs reference a HarborConnection.

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
  All other CRs reference this.

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

### Optional: Start Kind with Cilium + Hubble

If you want Cilium to be the CNI (so Hubble can observe pod traffic), create
the cluster with Cilium from the start:

```sh
make kind-up-cilium
```

### 3. Working with Samples

Apply or remove sample CRs (in `config/samples`):

```sh
# Apply sample HarborConnection, Registry, Project, etc.
make apply-samples

# Delete all Harbor CRs (HarborConnection last) in all namespaces
make delete-crs
```

> `delete-crs` removes **all** custom resources in the
> `harbor.harbor-operator.io` API group, not just the manifests in `config/samples/`.

### 4. Rebuilding and Redeploying

If you change the operator code:

```sh
make kind-redeploy
```

This will:

- Delete all Harbor CRs managed by the operator (HarborConnection last)
- Remove the operator deployment and CRDs
- Rebuild the image
- Load it into Kind
- Reinstall CRDs and redeploy the operator

### 5. Tearing Down

To delete only the Kind cluster:

```sh
make kind-down
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

Create a `HarborConnection` and any `Registry` / `Project` / `User` / `Member` CRs you need.

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
# Remove Harbor CRs (HarborConnection last)
make delete-crs

# Remove the operator
helm uninstall harbor-operator -n harbor-operator-system

# Remove CRDs for the harbor.harbor-operator.io API group
make uninstall
```

In a Kind dev cluster, a full reset is just:

```sh
make kind-reset
# or
make kind-down
```

## Contributing

- Format and vet:

  ```sh
  make fmt vet
  ```

- Run tests:

  ```sh
  make test
  ```

- Run linters:

  ```sh
  make lint
  ```

Open a PR with a clear description of what you changed and why.
