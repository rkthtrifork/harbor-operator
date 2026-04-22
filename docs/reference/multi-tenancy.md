# Multi-Tenancy

This page describes the recommended multi-tenant operating model for
`harbor-operator`.

The short version is:

- use operator settings for runtime scope
- use Kubernetes object references between Harbor resources
- use admission policy such as Kyverno for tenant-specific naming rules

## Recommended Model

For a shared cluster, the cleanest setup is usually:

1. Scope the operator to the namespaces it should manage with `--watch-namespaces`.
2. Point that operator instance at a single shared Harbor instance with `--harbor-connection`.
3. Use `metadata.name` as the Harbor-side identity for named resources.
4. Use Kubernetes object references for relationships between Harbor resources.
5. Enforce tenant-specific naming conventions with admission policy such as Kyverno.

This keeps the operator focused on reconciliation while leaving tenant naming
policy to the cluster policy layer.

## Operator Controls

The operator exposes two runtime controls that are useful in multi-tenant
deployments.

### `--watch-namespaces`

Use `--watch-namespaces=team-a,team-b` to restrict the operator cache and
reconcilers to a fixed set of namespaces.

Use this when:

- one operator deployment should only manage a subset of the cluster
- you want to reduce blast radius
- you want clearer ownership boundaries between operator instances

If omitted, the operator watches all namespaces.

### `--harbor-connection`

Use `--harbor-connection=shared-harbor` to force all Harbor-backed resources to
use one `ClusterHarborConnection`.

In this mode:

- `spec.harborConnectionRef` becomes optional
- if `spec.harborConnectionRef` is still set, it must match the configured
  `ClusterHarborConnection`
- updates to that `ClusterHarborConnection` fan out reconciles to dependent
  Harbor-backed resources

This is useful when one operator instance is intended to manage exactly one
Harbor installation.

## API Shape for Tenant Safety

The operator API now leans toward a simpler, safer model:

- `metadata.name` is the Harbor identity for named resources such as `Project`,
  `Registry`, `User`, `UserGroup`, `Label`, `Robot`, `ReplicationPolicy`,
  `ScannerRegistration`, and `WebhookPolicy`
- relationships use Kubernetes object references such as `projectRef`,
  `registryRef`, `memberUser.userRef`, and `memberGroup.groupRef`
- CRDs do not expose raw Harbor ID selectors or `nameOrID` union fields

This improves tenant isolation because references resolve through Kubernetes
objects and their status rather than through free-form Harbor identifiers.

## What Belongs in Policy

Tenant-specific naming rules usually belong outside the operator.

Examples:

- requiring project names to start with a tenant prefix
- requiring user names to start with a tenant prefix
- requiring referenced object names to use the same prefix

Those are cluster governance concerns, not Harbor reconciliation concerns.

Kyverno or a similar admission policy engine is a good fit because:

- bad objects are rejected at admission time
- the policy can derive prefixes from namespace labels
- the rule can be changed per cluster without changing the operator

## Example Kyverno Policies

The examples below assume:

- tenant identity is stored on the namespace label
  `capsule.clastix.io/tenant`
- the tenant prefix is derived by removing the `-tenant` suffix

Adjust the label key and prefix logic to match your own platform conventions.

### Enforce Project Name Prefix

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: harbor-project-prefix
spec:
  validationFailureAction: Enforce
  background: false
  rules:
    - name: validate-project-metadata-name-prefix
      match:
        any:
          - resources:
              kinds:
                - harbor.harbor-operator.io/v1alpha1/Project
      context:
        - name: tenantName
          apiCall:
            urlPath: "/api/v1/namespaces/{{request.object.metadata.namespace}}"
            jmesPath: 'metadata.labels."capsule.clastix.io/tenant" || `""`'
        - name: tenantPrefix
          variable:
            value: "{{ replace_all('{{tenantName}}', '-tenant', '') }}"
            jmesPath: "to_string(@)"
      preconditions:
        all:
          - key: "{{ request.operation }}"
            operator: AnyIn
            value: ["CREATE", "UPDATE"]
          - key: "{{ tenantName }}"
            operator: NotEquals
            value: ""
      validate:
        message: "Harbor Project metadata.name must start with '{{tenantPrefix}}-'"
        deny:
          conditions:
            any:
              - key: "{{ regex_match('^{{tenantPrefix}}-.*', request.object.metadata.name) }}"
                operator: Equals
                value: false
```

### Enforce User Name Prefix

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: harbor-user-prefix
spec:
  validationFailureAction: Enforce
  background: false
  rules:
    - name: validate-user-metadata-name-prefix
      match:
        any:
          - resources:
              kinds:
                - harbor.harbor-operator.io/v1alpha1/User
      context:
        - name: tenantName
          apiCall:
            urlPath: "/api/v1/namespaces/{{request.object.metadata.namespace}}"
            jmesPath: 'metadata.labels."capsule.clastix.io/tenant" || `""`'
        - name: tenantPrefix
          variable:
            value: "{{ replace_all('{{tenantName}}', '-tenant', '') }}"
            jmesPath: "to_string(@)"
      preconditions:
        all:
          - key: "{{ request.operation }}"
            operator: AnyIn
            value: ["CREATE", "UPDATE"]
          - key: "{{ tenantName }}"
            operator: NotEquals
            value: ""
      validate:
        message: "Harbor User metadata.name must start with '{{tenantPrefix}}-'"
        deny:
          conditions:
            any:
              - key: "{{ regex_match('^{{tenantPrefix}}-.*', request.object.metadata.name) }}"
                operator: Equals
                value: false
```

### Enforce Member References Stay Within the Tenant Prefix

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: harbor-member-prefix
spec:
  validationFailureAction: Enforce
  background: false
  rules:
    - name: validate-member-project-ref-prefix
      match:
        any:
          - resources:
              kinds:
                - harbor.harbor-operator.io/v1alpha1/Member
      context:
        - name: tenantName
          apiCall:
            urlPath: "/api/v1/namespaces/{{request.object.metadata.namespace}}"
            jmesPath: 'metadata.labels."capsule.clastix.io/tenant" || `""`'
        - name: tenantPrefix
          variable:
            value: "{{ replace_all('{{tenantName}}', '-tenant', '') }}"
            jmesPath: "to_string(@)"
      preconditions:
        all:
          - key: "{{ request.operation }}"
            operator: AnyIn
            value: ["CREATE", "UPDATE"]
          - key: "{{ tenantName }}"
            operator: NotEquals
            value: ""
      validate:
        message: "Harbor Member spec.projectRef.name must start with '{{tenantPrefix}}-'"
        deny:
          conditions:
            any:
              - key: "{{ regex_match('^{{tenantPrefix}}-.*', request.object.spec.projectRef.name || '') }}"
                operator: Equals
                value: false
    - name: validate-member-user-ref-prefix
      match:
        any:
          - resources:
              kinds:
                - harbor.harbor-operator.io/v1alpha1/Member
      context:
        - name: tenantName
          apiCall:
            urlPath: "/api/v1/namespaces/{{request.object.metadata.namespace}}"
            jmesPath: 'metadata.labels."capsule.clastix.io/tenant" || `""`'
        - name: tenantPrefix
          variable:
            value: "{{ replace_all('{{tenantName}}', '-tenant', '') }}"
            jmesPath: "to_string(@)"
      preconditions:
        all:
          - key: "{{ request.operation }}"
            operator: AnyIn
            value: ["CREATE", "UPDATE"]
          - key: "{{ tenantName }}"
            operator: NotEquals
            value: ""
          - key: "{{ request.object.spec.memberUser.userRef.name || '' }}"
            operator: NotEquals
            value: ""
      validate:
        message: "Harbor Member spec.memberUser.userRef.name must start with '{{tenantPrefix}}-'"
        deny:
          conditions:
            any:
              - key: "{{ regex_match('^{{tenantPrefix}}-.*', request.object.spec.memberUser.userRef.name || '') }}"
                operator: Equals
                value: false
```

Extend the same pattern for `memberGroup.groupRef.name`, robot project-scoped
permission namespaces, or any other Harbor object names that are globally shared
within your Harbor deployment.

## Suggested Deployment Patterns

### Shared Harbor, Shared Operator

- one operator instance
- one `ClusterHarborConnection`
- `--harbor-connection` set
- `--watch-namespaces` set to the participating tenant namespaces
- Kyverno enforces naming prefixes

### Shared Harbor, Per-Tenant Operator Instances

- one operator instance per tenant or tenant group
- each instance uses `--watch-namespaces`
- each instance may use the same `--harbor-connection`
- Kyverno may still enforce name prefixes if Harbor object names are globally shared

### Tenant-Local Harbor Access

- use namespaced `HarborConnection`
- do not set `--harbor-connection`
- each namespace explicitly selects its Harbor connection

## Related Reading

- [Common Spec Fields](common-spec-fields.md)
- [Connection Patterns](connection-patterns.md)
- [Deletion and Ownership](deletion-and-ownership.md)
