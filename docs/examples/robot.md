# Robot Credentials

Robot accounts are a good example of the operator owning generated output.

## Robot

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Robot
metadata:
  name: ci
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  level: project
  permissions:
    - kind: project
      namespace: library
      access:
        - resource: repository
          action: pull
          effect: allow
        - resource: repository
          action: push
          effect: allow
  secretRef:
    name: harbor-robot-ci
    key: secret
```

## Notes

- the operator creates and manages the output secret
- the secret is not treated as an input password source
- if the target secret already exists and is unrelated, reconciliation fails instead of silently adopting it
