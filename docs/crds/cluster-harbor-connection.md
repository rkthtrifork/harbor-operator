# ClusterHarborConnection CRD

The **ClusterHarborConnection** custom resource describes how the operator should
connect to a Harbor instance using a cluster-scoped connection object. Use it
when multiple namespaces should share the same Harbor endpoint and credentials.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ClusterHarborConnection
metadata:
  name: shared-harbor
spec:
  baseURL: "https://harbor.example.com"
  credentials:
    type: basic
    username: "platform-admin"
    passwordSecretRef:
      name: shared-harbor-credentials
      namespace: harbor-operator-system
      key: password
```

## Key Fields

- **spec.baseURL** (string, required)
  Harbor API base URL. Must include a scheme such as `https://`.

- **spec.credentials** (object, optional)
  Username plus a Secret reference containing the password or token. For a
  cluster-scoped connection, set `passwordSecretRef.namespace` explicitly.

- **spec.caBundle** / **spec.caBundleSecretRef** (optional)
  PEM-encoded CA material for validating Harbor TLS certificates.

## Notes

- Harbor-backed CRDs can reference this object via:
  `spec.harborConnectionRef.name: shared-harbor` and
  `spec.harborConnectionRef.kind: ClusterHarborConnection`.
- Updating a `ClusterHarborConnection` triggers reconciliation of Harbor-backed CRs that reference it.
- Use namespaced `HarborConnection` for tenant-local credentials and
  `ClusterHarborConnection` for shared platform-managed access.
