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
  --set metrics.enabled=true \\
  --set harborConnection=shared-harbor \\
  --set watchNamespaces[0]=team-a
```

- `harborConnection` sets the operator-wide `ClusterHarborConnection` used for all Harbor-backed resources.
- `watchNamespaces` scopes the operator cache and reconcilers to a fixed list of namespaces.
- `defaultCreationPolicy` supplies `Create`, `Adopt`, or `CreateOrAdopt` when a resource omits `spec.creationPolicy`; it defaults to `Create`, and an explicit resource value takes precedence.
- `defaultDriftDetectionInterval` supplies the periodic reconciliation interval when a resource omits `spec.driftDetectionInterval`; it defaults to `0s` (disabled), while an explicit resource value, including `0s`, takes precedence.
- `harborRequestTimeout` limits each request to the Harbor API, defaults to `30s`, and must be greater than zero.

When `metrics.enabled=true` and `metrics.secure=true`, the endpoint uses HTTPS and Kubernetes token authentication and authorization. The chart binds the operator to the narrowly scoped token and subject-access review permissions it needs. It also creates a `*-metrics-reader` ClusterRole for `GET /metrics`, but does not bind that role because the chart cannot safely infer the Prometheus service account.

Note: set only one of `pdb.minAvailable` or `pdb.maxUnavailable`. If both are set, the chart will prefer `maxUnavailable`.

### Prometheus ServiceMonitor

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set metrics.enabled=true \\
  --set metrics.serviceMonitor.enabled=true \\
  --set metrics.readerRole.binding.create=true \\
  --set metrics.readerRole.binding.serviceAccount.name=prometheus \\
  --set metrics.readerRole.binding.serviceAccount.namespace=monitoring
```

The default ServiceMonitor uses the Prometheus pod's projected service-account token and accepts controller-runtime's generated development certificate. For production, provide a TLS Secret whose certificate covers the metrics Service DNS name and configure certificate verification:

```yaml
metrics:
  enabled: true
  tls:
    certificateSecret: harbor-operator-metrics-tls
  serviceMonitor:
    enabled: true
    tlsConfig:
      insecureSkipVerify: false
      ca:
        secret:
          name: harbor-operator-metrics-ca
          key: ca.crt
  readerRole:
    binding:
      create: true
      serviceAccount:
        name: prometheus
        namespace: monitoring
```

When certificate verification is enabled and `tlsConfig.serverName` is omitted, the chart uses the metrics Service DNS name. Use `metrics.serviceMonitor.endpointAdditionalProperties` for Prometheus Operator endpoint options not modeled directly by the chart. Set `metrics.serviceMonitor.bearerTokenFile` to an empty string if authentication is supplied another way.

To expose plain HTTP instead, explicitly set `metrics.secure=false`. This disables the authentication and authorization filter and is suitable only when access is protected by equivalent cluster controls.

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
