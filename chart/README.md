# ecp

Helm chart for the European Control Plane (ECP) API servers:

- **gateway-global** — serves the global SECA providers (`seca.region`,
  `seca.authorization`).
- **gateway-regional** — serves the regional SECA providers (`seca.workspace`,
  `seca.storage`, `seca.network`, `seca.compute`) for one region.

Reconciliation is not part of this chart: install a delegator chart alongside
the regional gateway — for Aruba Cloud that is
[`csp/aruba/chart`](../csp/aruba/chart).

## Topology: together or split?

Both. The gateways are independent servers (two subcommands of the same
binary) and the chart supports either layout via the `enabled` toggles:

- **All-in-one** (default, what the e2e stack uses): both gateways in one
  cluster.
- **Split** (the realistic production layout, as in the IONOS split demo —
  see `doc/PLUGINS.md`): install the chart once per cluster —
  `--set gatewayRegional.enabled=false` on the global cluster,
  `--set gatewayGlobal.enabled=false --set gatewayRegional.region=<region>`
  on each regional cluster.

## Installing

No public ECP images are published yet: build and push the images (see
`test/internal/build/`) and point `*.image.repository` at your registry.

```bash
# All-in-one
helm install ecp chart \
  --namespace ecp --create-namespace \
  --set gatewayRegional.region=itbg-bergamo

# Global cluster only
helm install ecp chart -n ecp --create-namespace \
  --set gatewayRegional.enabled=false

# Regional cluster only
helm install ecp chart -n ecp --create-namespace \
  --set gatewayGlobal.enabled=false \
  --set gatewayRegional.region=itbg-bergamo
```

## CRDs

[`crds/`](crds) is the output directory of `make generate-api`, so the chart
always ships the current generated CRDs. Helm installs the CRDs on first
`helm install` but (by design) never upgrades or deletes them; to upgrade
CRDs on an existing cluster apply them directly:

```bash
kubectl apply -f chart/crds/
```

## Authentication

Auth is **disabled by default**, mirroring the gateway binary's opt-in
default — the API is then unauthenticated, so do not expose it beyond the
cluster in that mode. Two authentication plugins exist (`auth.plugin`):

- **`dummy`** (default) — base64 JSON username/password tokens, no signature
  verification. Development and testing only.

  ```bash
  helm install ecp chart -n ecp --create-namespace \
    --set gatewayRegional.region=itbg-bergamo \
    --set auth.enabled=true \
    --set auth.dummyUsers.users.admin=some-password
  ```

  Or reference a pre-existing Secret carrying a `users.json` key with
  `auth.dummyUsers.existingSecret`.

- **`jwt`** — standard signed JWTs, verified against a configured key with
  the `alg` header pinned to `auth.jwt.signingMethod`:

  ```bash
  helm install ecp chart -n ecp --create-namespace \
    --set gatewayRegional.region=itbg-bergamo \
    --set auth.enabled=true \
    --set auth.plugin=jwt \
    --set-file auth.jwt.key=jwt-public-key.pem
  ```

  The key is a PEM public key — or, for HS\* methods, the raw HMAC secret,
  which can also *mint* tokens and must be guarded accordingly (it is always
  stored in a Secret; `auth.jwt.existingSecret` with a `jwt.pub` key works
  too). The plugin applies to **both** gateways — the binary registers the
  same auth flags on the global and the regional server.

See `doc/AUTH.md` for the token formats, down-scoping and the RBAC model.

## Values

See [values.yaml](values.yaml) for the full commented list. The notable ones:

| Key | Default | Notes |
|-----|---------|-------|
| `gatewayGlobal.enabled` | `true` | Deploy the global gateway |
| `gatewayRegional.enabled` | `true` | Deploy the regional gateway |
| `gatewayRegional.region` | `""` | **Required** when the regional gateway is enabled |
| `auth.enabled` | `false` | Bearer-token authn + SECA RBAC authz on both gateways |
| `auth.plugin` | `dummy` | Authenticator for both gateways: `dummy` or `jwt` |
| `auth.jwt.signingMethod` | `ES256` | Pinned JWT `alg` when `auth.plugin=jwt` |
| `auth.jwt.key` | `""` | PEM public key / raw HS\* secret (required for `jwt` unless `auth.jwt.existingSecret`) |
| `auth.authz.impl` | `cached` | `cached` (informer) or `direct` (per-request) checker |
| `auth.dummyUsers.users` | `{}` | username → password map (required when `auth.plugin=dummy`) |
| `*.image.repository` | `ghcr.io/eu-sovereign-cloud/ecp/...` | No public images yet — set to your registry |
| `*.service.type` / `*.ingress.enabled` | `ClusterIP` / `false` | How to expose each gateway |

`helm lint`/CI note: because `gatewayRegional.region` has no sane default,
lint with the CI values: `helm lint chart -f chart/ci/default-values.yaml`.
