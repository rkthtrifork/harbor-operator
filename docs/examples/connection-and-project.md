# Connection and Project

This is the most basic end-to-end example: define a Harbor connection and then create a project through the operator.

## HarborConnection

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: harborconnection-sample
spec:
  baseURL: http://harbor-core.default.svc
  credentials:
    type: basic
    username: admin
    passwordSecretRef:
      name: harbor-core
      key: HARBOR_ADMIN_PASSWORD
```

## Project

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: project-sample
spec:
  harborConnectionRef:
    name: harborconnection-sample
    kind: HarborConnection
  public: false
```

## Notes

- `metadata.name` is the Harbor project name for `Project`
- `public` is required on the `Project` spec
- updating the connection object requeues the dependent `Project`
