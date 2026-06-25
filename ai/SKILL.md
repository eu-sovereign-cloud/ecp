---
name: new-vertical-resource
description: >-
  Add or complete a vertical resource in the ECP control plane — a SECA-spec resource such
  as image, network, subnet, public-ip, nic, instance, security-group, route-table, or
  internet-gateway. Covers the full slice: domain types, CRD code generation, controller,
  REST handler, the dummy CSP plugin, deployment/RBAC, tests, and docs, strictly following
  the vendored spec at modules/go-sdk/spec. Use when the user says things like "add a new
  resource", "implement/scaffold/finish the <X> vertical", "create the <X> controller",
  "add the <X> CRD", or "wire up <X> end to end".
---

# Add a new vertical resource

## What this does

ECP models every cloud resource as a **vertical slice** that cuts through
`domain → backend (CR + controller + plugin) → frontend (REST)`, with a CSP plugin,
deployment manifests, tests, and docs. This skill walks an agent through creating one for a
resource defined in the **SECA spec**, using the two complete reference verticals —
`resource/workspace/v1/` (regional-tenant-scoped) and `resource/storage/block-storage/v1/`
(regional-workspace-scoped) — as the pattern to copy.

## When to use it

Trigger on requests to add, scaffold, complete, or wire up a resource/controller/CRD for a
named SECA resource. The resource name is whatever the spec calls it — resolve it from
`modules/go-sdk/spec/spec/resources/<group>.v1.yaml` (top-level `name:`/`plural:`). User
shorthand may differ from the spec (e.g. "VPC" is the resource named `network`); always use
the spec's exact name.

## How to run it

**Follow [`templates/plans/NEW_RESOURCE.md`](templates/plans/NEW_RESOURCE.md) step by step.**
That guide is the authoritative, dependency-ordered checklist (types → CRD → controller →
plugin → REST → wiring → tests → docs). Start at its §1 Preflight: it tells you how to
resolve the name, classify the scope from the spec's `hierarchy:` field, decide whether you
extend an existing API group or create a new one, and — crucially — **compute the delta**
against the current tree so you only do the missing steps (a resource may be a bare
scaffold, partially done, or already complete).

## Hard rules (the guide expands each)

1. **Spec is the source of truth.** Name, fields, requirements, and validations come from
   `modules/go-sdk/spec`. The resource must strictly abide by it.
2. **Never edit the submodules** `modules/go-sdk/` or `modules/go-sdk/spec/` — CI fails on
   manual changes. They are read-only inputs to code generation. (Excluded from this skill:
   `.claude/` files, which are local to each developer.)
3. **One required Makefile edit:** add your slice to the hardcoded list in
   `framework/backend/kubernetes/Makefile` `generate-crds`. Omit it and the CRD ships
   **silently missing its spec validations**. No other wiring is needed.
4. **Don't trust static tracing of code generation** — run the generation step and inspect
   the outputs (one generator, `conditioned-gen`, has no visible invocation at all).
5. **Follow [`doc/CONVENTIONS.md`](../doc/CONVENTIONS.md)** for all hand-written code.
6. **Verify only at the very end:** `make pre-commit-ctzd` then `make pre-merge-ctzd`.
   **Docker is the only required toolchain.** If it is unavailable, warn the user and mark
   the work **UNVERIFIED** — never claim passing checks that didn't run.
7. **Do not auto-commit.** If a commit is unavoidable, use a separate branch and a one-line
   Conventional Commit — no agent attribution.

## Key references

| Topic | Path |
|---|---|
| Full step-by-step guide | [`templates/plans/NEW_RESOURCE.md`](templates/plans/NEW_RESOURCE.md) |
| Spec (source of truth) | [`modules/go-sdk/spec/spec/resources/`](../modules/go-sdk/spec/spec/resources/) |
| Complete reference verticals | [`resource/workspace/v1/`](../resource/workspace/v1/), [`resource/storage/block-storage/v1/`](../resource/storage/block-storage/v1/) |
| Code generation | [`doc/CODEGEN.md`](../doc/CODEGEN.md), [`framework/backend/kubernetes/Makefile`](../framework/backend/kubernetes/Makefile) |
| Coding conventions | [`doc/CONVENTIONS.md`](../doc/CONVENTIONS.md) |
| Plugin system | [`doc/PLUGINS.md`](../doc/PLUGINS.md), [`csp/dummy/`](../csp/dummy/) |
