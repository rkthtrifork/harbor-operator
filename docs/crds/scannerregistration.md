# Scanner Registration CRD

A **ScannerRegistration** custom resource manages Harbor scanner registrations via
`/api/v2.0/scanners`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ScannerRegistration
metadata:
  name: trivy
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  name: trivy
  url: http://harbor-scanner-trivy:8080
  auth: Bearer
  accessCredential: "token"
  default: true
```

## Key Fields

- **spec.url** (string, required)
  Scanner adapter base URL.

- **spec.auth** (string, optional)
  Authentication type (Basic, Bearer, X-ScannerAdapter-API-Key).

- **spec.accessCredential** / **spec.accessCredentialSecretRef** (optional)
  Credential value for authentication (use secret reference for sensitive values).

- **spec.default** (bool, optional)
  When `true`, promotes this registration to the system default scanner. `false`
  or omission does not change Harbor's current default scanner assignment.

- **spec.skipCertVerify** (bool, optional)
  Defaults to `false`. If `true`, disables TLS certificate verification for
  scanner requests and should only be used when secure verification cannot be configured.

- **spec.useInternalAddr** (bool, optional)
  Controls whether the scanner uses Harbor's internal address. Defaults to `false`.

- **spec.disabled** (bool, optional)
  Disables the scanner registration. Defaults to `false`.

- **spec.creationPolicy** (string, optional)
  Controls whether the registration is created, adopted, or either. When omitted, uses the operator's default creation policy (`Create` unless configured otherwise).

## Common Fields

`ScannerRegistration` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates the scanner registration in Harbor.

- **Delete**
  Deletes the registration in Harbor when the CR is deleted.
