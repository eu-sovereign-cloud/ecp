---
name: new-vertical-resource
description: >-
  Add or complete a vertical resource in the ECP control plane — a SECA-spec resource such as
  image, subnet, sku, public-ip, nic, instance, security-group, route-table, internet-gateway,
  or region. Covers the full slice: domain types, CRD code generation, conversion, controller,
  REST handler, the dummy CSP plugin, deployment/RBAC, tests, and docs, strictly following the
  vendored spec at modules/go-sdk/spec. Use when the user says "add/implement/scaffold/finish/
  wire up the <X> vertical/resource/controller/CRD", or invokes the planning command:
  `/plan new ['read-only'] ('global'|'regional-tenant'|'regional-workspace') resource [<group>]
  <resource_name> from <resource_spec_path>` — where `read-only` is present iff the spec exposes
  only list/get, `<group>` is present for regional resources and omitted for global ones, and
  `<resource_name>` is the resource's spec `name:`.
---

# Add a new vertical resource

## What this does

ECP models every cloud resource as a **vertical slice** through
`domain → backend (CR + controller + plugin) → frontend (REST)`, plus a CSP plugin, deployment
manifests, tests, and docs. This skill drives an agent through creating (or completing) one for
a resource defined in the **SECA spec**, using the three canonical verticals as exemplars by
role:

- `resource/workspace/v1/` — **regional-tenant**, read-write (full controller/plugin reference).
- `resource/storage/v1/block-storage/` — **regional-workspace**, read-write (full reference).
- `resource/region/v1/` — **global**, read-only (global scope + rest-only shape; no controller).

Every resource is already scaffolded from the go-sdk; the work is the **delta** between the
scaffold and a complete vertical, which varies per resource — so the guide specifies every step
and you skip the ones already done.

## When to use it

Trigger on requests to add, scaffold, complete, or wire up a resource / controller / CRD for a
named SECA resource, and on the `/plan new …` command in the `description`. Resolve the resource
from `modules/go-sdk/spec/spec/resources/<group>.v1.yaml`; the user's shorthand may differ from
the spec name (and the spec name may differ from the Go Kind/dir — e.g. spec `sku` → Kind
`StorageSKU`, dir `storage-sku`). If the resource is not in the spec, **stop and ask the user.**

## How to run it

**Follow [`templates/plans/NEW_RESOURCE.md`](templates/plans/NEW_RESOURCE.md) step by step.** It
is the authoritative, dependency-ordered checklist (types → CRD → conversion →
controller/plugin → REST → wiring → tests → docs). Start at its **§1 Preflight**, which parses
the `/plan` command, resolves identity from the spec, **validates the declared `read-only` flag
and scope against the spec** (notify the user on any mismatch), and computes the delta so you
only do the missing steps.

## Hard rules (the guide expands each)

1. **The spec is the source of truth.** Name, fields, requirements, and validations come from
   `modules/go-sdk/spec`. The resource must strictly abide by it.
2. **Never edit the submodules** `modules/go-sdk/` or `modules/go-sdk/spec/` — CI fails on manual
   edits. They are read-only inputs to code generation. (`.claude/` files stay local and are out
   of scope.)
3. **One possible Makefile edit:** ensure your slice is in the hardcoded list in
   `framework/backend/kubernetes/Makefile` `generate-crds`. Omit it and the CRD ships **silently
   missing its spec validations**. No other wiring is needed.
4. **Don't trust static tracing of code generation** — run the generation step and inspect the
   outputs.
5. **Follow [`doc/CONVENTIONS.md`](../doc/CONVENTIONS.md)** for all hand-written code.
6. **Verify only at the very end:** `make pre-commit-ctzd`, `make pre-merge-ctzd`, then the dummy
   plugin integration tests. **Docker and Kind are the only required tooling.** If unavailable,
   warn the user and mark the work **UNVERIFIED** — never claim checks that didn't run.
7. **Commit policy is the user's call:** ask before committing (auto-commit / don't / ask
   exactly), verify the branch and uncommitted work first, never auto-merge, never run anything
   that can erase work.

## Key references

| Topic | Path |
|---|---|
| Full step-by-step guide | [`templates/plans/NEW_RESOURCE.md`](templates/plans/NEW_RESOURCE.md) |
| Spec (source of truth) | [`modules/go-sdk/spec/spec/resources/`](../modules/go-sdk/spec/spec/resources/) |
| Canonical verticals | [`resource/workspace/v1/`](../resource/workspace/v1/), [`resource/storage/v1/block-storage/`](../resource/storage/v1/block-storage/), [`resource/region/v1/`](../resource/region/v1/) |
| Code generation | [`doc/CODEGEN.md`](../doc/CODEGEN.md), [`framework/backend/kubernetes/Makefile`](../framework/backend/kubernetes/Makefile) |
| Coding conventions | [`doc/CONVENTIONS.md`](../doc/CONVENTIONS.md) |
| Plugin system | [`doc/PLUGINS.md`](../doc/PLUGINS.md), [`csp/dummy/`](../csp/dummy/) |
