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
  Sets this registration as the system default scanner.

## Common Fields

- **spec.harborConnectionRef** selects the Harbor connection object by `name` and optional `kind`.
- **spec.deletionPolicy** controls delete behavior when Harbor cleanup cannot be completed. Use `Delete` (default) for managed cleanup or `Orphan` as an explicit break-glass option.

## Behavior

- **Create / Update**
  Creates or updates the scanner registration in Harbor.

- **Delete**
  Deletes the registration in Harbor when the CR is deleted.
