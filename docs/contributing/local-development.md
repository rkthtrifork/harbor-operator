# Local Development

## Start the Local Stack

```sh
make kind-up
```

Or with Cilium and Hubble:

```sh
make kind-up-cilium
```

## Normal Iteration Loop

```sh
make kind-refresh
```

That rebuilds the image, reloads it into Kind, regenerates assets, reapplies CRDs, and redeploys the operator.

## Reset the Operator State

```sh
make kind-redeploy
```

Use this when you want a clean slate for operator-managed resources and CRDs on an existing Kind cluster.
