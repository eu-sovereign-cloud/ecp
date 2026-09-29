# ecp

Helm chart for the European Control Plane (ECP) API servers:

- **gateway-global** — serves the global SECA providers (`seca.region`,
  `seca.authorization`).
- **gateway-regional** — serves the regional SECA providers (`seca.workspace`,
  `seca.storage`, `seca.network`, `seca.compute`) for the regions listed in
  `gatewayRegional.regions`; the first is the default.

Reconciliation is done by the [`delegator`](../delegator), which runs
alongside the regional gateway with `plugin` set to the CSP you run. It ships
here as an optional subchart (`ecp-delegator.enabled`, off by default) — or
install that chart standalone if you want to version and upgrade it
separately.

## Topology: together or split?

Both. The gateways are independent servers (two subcommands of the same
binary) and the chart supports either layout via the `enabled` toggles:

- **All-in-one** (default, what the e2e stack uses): both gateways in one
  cluster.
- **Split** (the realistic production layout, as in the IONOS split demo —
  see `doc/PLUGINS.md`): install the chart once per cluster —
  `--set gatewayRegional.enabled=false` on the global cluster,
  `--set gatewayGlobal.enabled=false --set 'gatewayRegional.regions={<region>}'`
  on each regional cluster.
- **Multi-region** (one self-installable cluster serving several regions): list them all,
  `--set 'gatewayRegional.regions={<region-a>,<region-b>}'`. One regional gateway then
  serves all of them, selected per request by a `/regions/<region>` path prefix; the
  **first** is the default for requests that name none. Advertise the prefixed URLs in each Region CR's `providers[].url` so
  clients discover the right base URL. A `Workspace` is identified by tenant **and** region, so
  a tenant can use the same workspace name in each region; every resource below a workspace is
  still keyed by tenant/workspace alone, so its name is taken in every region of one deployment,
  but only the region that created it can read, update or delete it. See
  [doc/ARCHITECTURE.md](../../doc/ARCHITECTURE.md#multi-region-gateways).

## Installing

Images are published to `ghcr.io/eu-sovereign-cloud/ecp/` on every `v*` tag
(see `.github/workflows/image-release.yaml`), and `*.image.tag` defaults to
the chart's appVersion — so the defaults resolve with no override.

Installing **from a checkout** needs the delegator dependency resolved first,
even though it is disabled by default — Helm materializes a declared
dependency before it evaluates the condition that switches it off, so without
this every command below fails with `missing in charts/ directory`:

```bash
helm dependency update charts/ecp
```

The resolved `charts/*.tgz` is gitignored and regenerated on demand; re-run it
after editing [`charts/delegator/`](../delegator). Installing from a packaged
release instead has the dependency already embedded and needs none of this.

```bash
# Global and Regional clusters
helm install ecp charts/ecp \
  --namespace ecp --create-namespace \
  --set 'gatewayRegional.regions={itbg-bergamo}'

# Global cluster only
helm install ecp charts/ecp -n ecp --create-namespace \
  --set gatewayRegional.enabled=false

# Regional cluster only
helm install ecp charts/ecp -n ecp --create-namespace \
  --set gatewayGlobal.enabled=false \
  --set 'gatewayRegional.regions={itbg-bergamo}'

# Global and Regional clusters, with the delegator as a subchart
helm install ecp charts/ecp -n ecp --create-namespace \
  --set 'gatewayRegional.regions={itbg-bergamo}' \
  --set ecp-delegator.enabled=true \
  --set ecp-delegator.plugin=aruba
```

`plugin=dummy` works the same way, but its image is never published — pass
`--set ecp-delegator.image.repository=<locally built image>` with it.

## CRDs

[`crds/`](crds) is the output directory of `make generate-api`, so the chart
always ships the current generated CRDs. Helm installs the CRDs on first
`helm install` but (by design) never upgrades or deletes them; to upgrade
CRDs on an existing cluster apply them directly:

```bash
kubectl apply -f charts/ecp/crds/
```

A `Workspace` must carry a `region` field, and it can never change. Every workspace created
by a release up to `v0.0.3-alpha` predates that field: it has only the internal
`secapi.cloud/region` label, and it lives in the tenant namespace rather than its per-region
one. Once the new CRD is applied, the API server refuses every update to such a workspace,
including the delegator releasing its finalizer. So **delete those workspaces before
upgrading**, while the old gateway and delegator can still tear them down:

```bash
kubectl get workspaces -A -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,REGION:.region'
```

One that is left over after the upgrade cannot be deleted normally. The upgraded delegator
reads it back with no region, and refuses to write a workspace without one, so the delete never
gets going and its finalizer is never released. Remove it by hand: request the delete first,
then, in one patch, add its region and drop its finalizers. The order matters: once the region
is set, a running delegator puts its finalizer back on a workspace that is not yet being deleted,
then addresses it in its per-region namespace, where it does not live, and never releases it.
Adding the field is allowed, because the immutability rule applies only to a region that is
already set. This skips the plugin's teardown, so whatever the CSP created for the workspace,
and the namespace it owned for its children, must be removed by hand too:

```bash
kubectl delete workspace <name> -n <ns> --wait=false
kubectl patch workspace <name> -n <ns> --type merge \
  -p '{"region":"<its secapi.cloud/region label>","metadata":{"finalizers":null}}'
```

## Authentication

Auth is **disabled by default**, mirroring the gateway binary's opt-in
default — the API is then unauthenticated, so do not expose it beyond the
cluster in that mode. Two authentication plugins exist (`auth.plugin`):

- **`dummy`** (default) — base64 JSON username/password tokens, no signature
  verification. Development and testing only.

  ```bash
  helm install ecp charts/ecp -n ecp --create-namespace \
    --set 'gatewayRegional.regions={itbg-bergamo}' \
    --set auth.enabled=true \
    --set auth.dummyUsers.users.admin=some-password
  ```

  Or reference a pre-existing Secret carrying a `users.json` key with
  `auth.dummyUsers.existingSecret`.

- **`jwt`** — standard signed JWTs, verified against a configured key with
  the `alg` header pinned to `auth.jwt.signingMethod`:

  ```bash
  helm install ecp charts/ecp -n ecp --create-namespace \
    --set 'gatewayRegional.regions={itbg-bergamo}' \
    --set auth.enabled=true \
    --set auth.plugin=jwt \
    --set-file auth.jwt.key=jwt-public-key.pem
  ```

  The key is a PEM public key — or, for HS\* methods, the raw HMAC secret,
  which can also *mint* tokens and must be guarded accordingly (it is always
  stored in a Secret; `auth.jwt.existingSecret` with a `jwt.pub` key works
  too). The plugin applies to **both** gateways — the binary registers the
  same auth flags on the global and the regional server.

  Set `auth.jwt.issuer` and `auth.jwt.audience` to your IdP's issuer URL and
  this API's identifier. Each is enforced only when set — and enforcing one
  also makes that claim mandatory — so a token minted for another service, or
  by another issuer sharing the key, is rejected rather than accepted on its
  signature alone.

Every auth value becomes a **command-line flag** on the gateway container: the
images are the bare binary, and it reads only `APP_ENV` from the environment (plus
`REGIONS` on the regional gateway, as a fallback for an unset `--regions`, which the chart
always sets).
Adding a knob to this chart therefore means adding it to `ecp.authArgs` in
[_helpers.tpl](templates/_helpers.tpl) — a value that renders into an env var
reaches nothing. `ci/scripts/chart-smoke.sh` guards that in CI by installing the
chart with `auth.enabled=true` and asserting an anonymous request is rejected.

See `doc/AUTH.md` for the token formats, down-scoping and the RBAC model.

## Values

See [values.yaml](values.yaml) for the full commented list. The notable ones:

| Key | Default | Notes |
|-----|---------|-------|
| `gatewayGlobal.enabled` | `true` | Deploy the global gateway |
| `gatewayRegional.enabled` | `true` | Deploy the regional gateway |
| `gatewayRegional.regions` | `[]` | **Required.** The regions served (`--regions`). The first is the default for a request that names none; any other is reachable only under its `/regions/<region>` path prefix. The single-region `gatewayRegional.region` was removed, and setting it fails the render |
| `auth.enabled` | `false` | Bearer-token authn + SECA RBAC authz on both gateways |
| `auth.plugin` | `dummy` | Authenticator for both gateways: `dummy` or `jwt` |
| `auth.jwt.signingMethod` | `ES256` | Pinned JWT `alg` when `auth.plugin=jwt` |
| `auth.jwt.key` | `""` | PEM public key / raw HS\* secret (required for `jwt` unless `auth.jwt.existingSecret`) |
| `auth.jwt.issuer` / `auth.jwt.audience` | `""` | Expected `iss` / `aud`; each is enforced (and required in the token) only when set |
| `auth.authz.impl` | `cached` | `cached` (informer) or `direct` (per-request) checker |
| `auth.dummyUsers.users` | `{}` | username → password map (required when `auth.plugin=dummy`) |
| `*.image.repository` | `ghcr.io/eu-sovereign-cloud/ecp/...` | Override only to mirror the images into your own registry |
| `*.service.type` / `*.ingress.enabled` | `ClusterIP` / `false` | How to expose each gateway |
| `*.service.nodePort` | `""` | Fixed node port, honoured only when `service.type=NodePort` (else auto-assigned) |
| `ecp-delegator.enabled` | `false` | Deploy the [delegator](../delegator) as a subchart. Its dependency is resolved either way — see Installing |
| `ecp-delegator.plugin` | `""` | **Required** when enabled — `aruba`, `dummy` or `ionos`; any other `ecp-delegator.*` value from that chart passes through |
| `ecp-delegator.regions` | `[]` | Regions the delegator reconciles; empty reconciles every one. See [charts/delegator](../delegator#regions) |

`helm lint`/CI note: because `gatewayRegional.regions` has no sane default,
lint with the CI values: `helm lint charts/ecp -f charts/ecp/ci/default-values.yaml`.
