# Aruba conformance (placeholder)

Unlike IONOS, aruba does **not** yet have a bespoke conformance harness — and it
doesn't need one to run. The aruba plugin is a delegator plugin set (loaded with
`PLUGIN=aruba`), so its conformance is the **generic two-phase flow**. This folder
is a placeholder that documents that flow and the one missing piece.

Why there's no `Makefile` here (yet), in contrast to [`../ionos-e2e/`](../ionos-e2e/):

- Aruba has **no standalone plugin binary** — it only runs inside the delegator
  (`../cmd/delegator`, `PLUGIN=aruba`). There is nothing separate to deploy.
- Aruba has **no in-repo backend deploy tooling**. Its backend is the third-party
  [`arubacloud-resource-operator`](https://github.com/Arubacloud/arubacloud-resource-operator)
  (the plugin writes its CRs); that operator plus Aruba credentials must be
  installed out of band. There is no `csp/aruba/deploy` analogous to
  `csp/ionos/deploy`.

## Running aruba conformance today

From the harness root (`test/`), it's the two-phase flow — deploy → provision
backend → run:

```shell
# 1. Deploy the stack (gateways + delegator) with the aruba plugin.
make conformance-deploy CONFORMANCE_PLUGIN=aruba     # or: make e2e-deploy E2E_PLUGIN=aruba

# 2. Install the arubacloud-resource-operator and its Aruba credentials in the
#    target cluster. (No tooling for this lives in the repo yet — see below.)

# 3. Run the suite against the deployed stack.
make conformance                                     # or: make e2e
```

## TODO: promote this to a real harness

When aruba gains backend deploy tooling — a `csp/aruba/deploy` mirroring
`csp/ionos/deploy` — add a `Makefile` here analogous to
[`../ionos-e2e/Makefile`](../ionos-e2e/Makefile) that:

1. creates the demo cluster(s),
2. installs the `arubacloud-resource-operator` (+ credentials),
3. deploys the delegator with `PLUGIN=aruba`,
4. runs `secatest` against the gateway,

and wire `conformance-aruba*` targets in `test/Makefile` delegating to it (as is
already done for `conformance-ionos*`).
