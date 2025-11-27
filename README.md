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

If you also want sample CRs applied:

```sh
make kind-up-with-samples
```

### 3. Working with Samples

Apply or remove sample CRs (in `config/samples`):

```sh
# Apply sample HarborConnection, Registry, Project, etc.
make samples-apply

# Delete all Harbor CRs (HarborConnection last) in all namespaces
make clean-samples
```

> `clean-samples` removes **all** custom resources in the
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

## Deploying to Another Cluster

For a non-Kind cluster, you typically:

1. Make sure an image of the operator is available to the cluster
   (e.g. build+push in CI to `ghcr.io/.../harbor-operator:<tag>`).

2. Install CRDs:

   ```sh
   make install
   ```

3. Configure the operator image in `config/manager/kustomization.yaml` or via:

   ```sh
   cd config/manager
   kustomize edit set image controller=<your-registry>/harbor-operator:<tag>
   ```

4. Deploy the operator:

   ```sh
   make deploy
   ```

5. Create a `HarborConnection` and any `Registry` / `Project` / `User` / `Member` CRs you need.

## Installer Bundle

If you want to ship a single `install.yaml` that contains CRDs plus the operator deployment:

```sh
make build-installer
```

This will:

- Set the image in `config/manager` to `harbor-operator:local` (or whatever `IMG_LOCAL` is)
- Build `config/default` with kustomize
- Write the result to `dist/install.yaml`

You can then install with:

```sh
kubectl apply -f dist/install.yaml
```

If you publish this file (e.g. in a GitHub release), users can install via a raw URL.

## Uninstalling

If you want to remove Harbor-managed resources, CRDs, and the operator:

```sh
# Remove Harbor CRs (HarborConnection last)
make clean-samples

# Remove CRDs for the harbor.harbor-operator.io API group
make uninstall

# Remove the operator deployment
make undeploy
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
