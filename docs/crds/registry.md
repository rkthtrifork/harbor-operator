# Registry CRD

The **Registry** custom resource defines a registry to be managed by the operator on your Harbor instance. It references a HarborConnection for accessing the Harbor API and supports both creation and ongoing reconciliation.

## Quick Start

Define a Registry resource with the required connection and configuration details. When applied, the operator ensures that the corresponding registry is created (or updated) in Harbor. If a registry with the same name already exists in Harbor and `allowTakeover` is enabled, the operator will adopt it by updating the CR status with its Harbor ID and reconfiguring the registry to match the desired state.

> [!CAUTION]  
> Setting `allowTakeover` to true means that if a registry with the same name already exists in Harbor, the operator will take control of it and update its configuration to match your custom resource. Use this setting only when you are sure that you want the operator to manage (and potentially modify) that existing registry.

**Recommendation:**  
It is recommended to leave the `name` field empty. When omitted, the operator will default the registry name to the custom resource’s metadata name, ensuring that your cluster remains the single source of truth.

#### Example

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Registry
metadata:
  name: my-registry
spec:
  # Reference to the HarborConnection resource.
  harborConnectionRef: "my-harbor-connection"
  # The registry type, e.g., "github-ghcr".
  type: github-ghcr
  # Leave this field empty to default to the CR's metadata name.
  name: ""
  # The registry URL.
  url: "https://registry.example.com"
  # Set to true to bypass certificate verification if necessary.
  insecure: false
  # If true, the operator will search for and adopt an existing registry with the same name in Harbor.
  allowTakeover: true
  # Optional: interval for periodic drift detection (e.g., "5m" for five minutes).
  # Defaults to no drift detection if omitted or set to 0.
  driftDetectionInterval: 5m
  # Optional: update this value to force an immediate reconcile.
  reconcileNonce: "update-123"
```

## Field Descriptions

- **harborConnectionRef**:  
  The name of the HarborConnection resource that contains the connection details for your Harbor instance.

- **type**:  
  The type of registry (e.g., `github-ghcr`). Only supported types are accepted.

- **name**:  
  The registry’s name.

  - **Recommendation:** Leave this field empty so the operator defaults it to the custom resource’s metadata name. This ensures your cluster remains the single source of truth.

- **url**:  
  The URL of the registry. This value is validated to ensure it is a proper URL.

- **insecure**:  
  A boolean flag indicating whether to skip certificate verification.

- **allowTakeover**:  
  If set to true and a registry with the specified name already exists in Harbor, the operator will adopt it.

  - **Caution:** This means the operator will update the existing registry’s configuration to match the desired state defined in the custom resource. Use this option carefully.

- **driftDetectionInterval**:  
  Specifies the interval at which the operator will periodically check that the registry’s configuration in Harbor matches the desired state defined in the CR.

  - This field uses a duration type (e.g., `"5m"` for five minutes).
  - **Note:** The default is no drift detection (i.e. zero duration). To enable periodic checks, explicitly set a non-zero value.

- **reconcileNonce**:  
  An optional field that forces an immediate reconciliation when updated. This is useful for triggering manual updates without waiting for the next drift check.

## Advanced Details

- **Resource Adoption:**  
  If a registry with the specified name already exists in Harbor and `allowTakeover` is enabled, the operator will adopt the existing registry by updating the CR’s status with its Harbor ID and ensuring that its configuration is changed to match the desired state.

- **API Interactions:**  
  The operator communicates with Harbor using standard HTTP methods:

  - **POST:** To create a new registry when none exists.
  - **PUT:** To update an existing registry if differences are detected.
  - **GET:** To list and retrieve registry details for comparison.
  - **DELETE:** Triggered via a finalizer when a Registry resource is removed from Kubernetes.

- **Authentication:**  
  For all Harbor API calls, the operator uses credentials defined in the referenced HarborConnection. The password is retrieved from the Kubernetes Secret specified in `accessSecretRef`.

- **Drift Detection:**  
  When `driftDetectionInterval` is set to a non-zero duration, the operator requeues the registry for periodic checks. If the configuration in Harbor drifts from the desired state, an update is initiated.

- **Finalizer Usage:**  
  A finalizer is attached to the Registry resource to ensure that if the CR is deleted, the corresponding registry in Harbor is also removed. In deletion, the operator trusts the stored HarborRegistryID—if the registry is not found using that ID, it assumes the registry has already been deleted.
