# Troubleshooting

## Start with the CR Status

Most problems surface on the custom resource itself through conditions and messages.

Check:

```sh
kubectl get <resource> <name> -o yaml
```

Look for the `Ready` condition and its `reason` and `message`.

## Common Failure Cases

## Harbor Authentication or Connectivity Failures

Typical causes:

- wrong credentials in the referenced secret
- wrong `baseURL`
- missing custom CA material
- Harbor not reachable from the operator namespace

Relevant resources:

- `HarborConnection`
- `ClusterHarborConnection`

## Singleton Conflict

If a singleton resource reports a conflict, another CR already owns that singleton API for the same Harbor instance.

Check for:

- another `Configuration`
- another `GCSchedule`
- another `PurgeAuditSchedule`
- another `ScanAllSchedule`

that resolves to the same Harbor base URL.

## Stuck Finalizer

If a resource is stuck in `Terminating`, check:

- whether its connection object still exists
- whether `spec.deletionPolicy` is `Delete`
- whether Harbor cleanup is still possible

If you explicitly want to remove the Kubernetes object without Harbor cleanup, switch to `deletionPolicy: Orphan`.

## Robot Secret Write Failures

If a `Robot` fails while writing its secret:

- verify that the destination secret name does not already belong to something unrelated
- verify RBAC for secrets in the target namespace

## Inspect Operator Logs

```sh
kubectl logs -n harbor-operator-system deploy/harbor-operator
```

For local development on Kind, `make kind-refresh` is usually the quickest way to redeploy after a fix.
