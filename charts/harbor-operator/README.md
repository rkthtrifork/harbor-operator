# Harbor Operator Helm Chart

This chart installs the harbor-operator controller into your cluster.

## Install (OCI)

```sh
helm registry login ghcr.io
helm install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator --version <chart-version>
```

## Template (render YAML)

```sh
helm template harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator --version <chart-version>
```

## Values

Default values live in `values.yaml`, and validation in `values.schema.json`.

Common overrides:

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set image.tag=v0.3.0 \\
  --set metrics.enabled=true
```

When `metrics.enabled=true` and `metrics.secure=true`, the chart also installs the delegated authentication RBAC needed for authenticated access to `/metrics` (`tokenreviews`, `subjectaccessreviews`, `system:auth-delegator`, and `extension-apiserver-authentication-reader`).

Note: set only one of `pdb.minAvailable` or `pdb.maxUnavailable`. If both are set, the chart will prefer `maxUnavailable`.

### Prometheus ServiceMonitor

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set metrics.enabled=true \\
  --set metrics.serviceMonitor.enabled=true
```

### Network Policy (metrics only)

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set networkPolicy.enabled=true \\
  --set networkPolicy.ingress.metrics.namespaces[0]=monitoring
```

For Cilium:

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set networkPolicy.enabled=true \\
  --set networkPolicy.type=cilium \\
  --set networkPolicy.ingress.metrics.namespaces[0]=monitoring
```

Egress defaults:
- kube‑api server (`networkPolicy.egress.kubeAPIPorts`, default `443`)
- kube‑dns (UDP/TCP 53)

For kind, the kube‑api server listens on `6443` by default:

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set networkPolicy.enabled=true \\
  --set networkPolicy.type=cilium \\
  --set networkPolicy.egress.kubeAPIPorts[0]=6443
```

To allow Harbor traffic, add one or more selectors:

```yaml
networkPolicy:
  enabled: true
  egress:
    harborSelectors:
      - namespace: harbor-system
        podSelector:
          matchLabels:
            app.kubernetes.io/component: core
            app.kubernetes.io/instance: harbor
```

## CRDs

CRDs are packaged in the chart under `crds/`. These are synced from `config/crd/bases`.
