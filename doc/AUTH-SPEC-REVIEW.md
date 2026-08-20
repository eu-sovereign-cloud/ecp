# Auth Findings — Token Model & SECA Spec Alignment

Findings from reviewing the gateway auth middleware (`feat/gateway-auth-middleware`,
see [AUTH.md](AUTH.md)) against the SECA specification: what the bearer token must
carry, where authentication and authorization decisions belong, and where the current
implementation diverges from the spec.

> **Status — implemented** (`feat/gateway-auth-middleware`, [#316](https://github.com/eu-sovereign-cloud/ecp/pull/316); the
> signature-verifying authenticator followed in [#334](https://github.com/eu-sovereign-cloud/ecp/pull/334) and the §6
> checklist was finished after it). The final model:
> the bearer token carries the **subject** (`sub`), the issuer-asserted **`tenants`**
> membership gate and an optional **`scope`** down-scope
> (`tenants`/`regions`/`workspaces`) — nothing else. **Roles are resolved entirely from
> `RoleAssignment`/`Role` in the tenant namespace and are never read from the token.**
> The `scope` object can only *narrow* what a token may exercise (see [AUTH.md](AUTH.md)
> § Token down-scoping). Authentication verifies token authenticity against a **fixed,
> operator-configured endpoint** — any endpoint named inside the token is ignored, as are
> its `iss`/`aud` unless the operator configured what to expect; the shipped Dummy
> authenticator validates username/password. **Region is a tenant-less
> catalog resource by design** — upstream confirmed the current path shape is correct,
> so no spec-side correction is coming. Tenant-scoped RBAC cannot govern it: the gateway
> serves `seca.region` authn-only via the configurable `--authz-skip-providers` list
> (see [AUTH.md](AUTH.md) § Per-provider authorization skip). The sections below record
> the analysis that led to this model; where an older subsection weighs alternatives,
> this banner is the decision that was taken.

Spec sources (under `modules/go-sdk/spec/`):

- `website/docs/content/2. Conventions/03-api-security.md` — JWT authn; authorization independence
- `website/docs/content/3. Architecture/05-api-access-control.md` — request flow, RBAC model, status codes
- `website/docs/content/5. Other/05-tenant-and-token.md` — CSP-operated token issuance, tenant onboarding
- `website/docs/content/6. Examples/01. usage.md` — required claims, self-service RoleAssignment flow
- `spec/spec/schemas/rbac.yaml` — Role / RoleAssignment schemas

---

## 1. Full picture: CSP IdP + SECA gateway

**Key fact**: the IdP is the **CSP's own Authentication Service**, not an arbitrary
external provider. Tenant onboarding is contractual with the CSP and user accounts are
created in the CSP's identity system — so the issuer knows each user's tenant(s) by
construction and can stamp them into the token.

Onboarding (per `05-tenant-and-token.md`):

1. Tenant establishes a contractual agreement with the CSP.
2. Tenant creates user accounts via the CSP's mechanisms → **the IdP now knows user → tenant**.
3. SECA API is enabled/activated.
4. Users obtain JWTs via the CSP's designated method (OIDC recommended, not mandated).

Request flow:

```
User ──(credentials)──▶ CSP IdP ──(signed JWT)──▶ User
User ──(Bearer JWT)──▶ Gateway authn   : signature / expiry / revocation   → 401 on failure
                       Gateway authz   : tenant membership + RBAC lookup   → 403 on denial
                       (infrastructure error anywhere)                     → 500, never 403
                       Provider handler
```

Division of labor — each fact lives with its natural authority:

| Question | Answered by | Mechanism |
|---|---|---|
| Who is calling? | Token — `sub` | Signature verification against CSP IdP keys |
| Member of which tenant(s)? | Token — tenant claim | Membership gate vs URL tenant |
| What may this token exercise? | Token — `scope` (optional) | Attenuation cap; can only narrow |
| Which resources are targeted? | URL path | `{tenant}/{workspace}`, resource, verb extraction |
| Which region / API family? | Server config / route registration | Regional server knows its region; provider baked into route |
| What may the **user** do? | Gateway RBAC store | Role + RoleAssignment in the tenant's namespace |

There is **no user→tenant membership directory in the gateway**: membership is emergent.
Subject S "belongs to" tenant T exactly insofar as T's own policy store (a dedicated
per-tenant Kubernetes namespace, `sha3-224(tenant)`) contains a RoleAssignment naming S.
No assignment → empty match set → fail-closed 403. Choosing a tenant in the URL grants
nothing; it only selects which policy store gets to reject you.

---

## 2. Token requirements

The rule that decides what belongs in the token (**grant vs cap**):

> A claim that **grants** ability (roles, permissions) duplicates the RoleAssignment
> store's authority and creates a second source of truth — it conflicts with the model.
> A claim that **caps** (tenant, scope) can only *narrow* what the caller may do — it
> creates no second authority and is safe.

| Claim | Status | Notes |
|---|---|---|
| `sub` | **Required** | The only claim the spec's usage guide requires: *"The JWT must contain a `sub` claim that identifies you"*. Matched against `RoleAssignment.spec.subs` ("subject IDs (from JWT)"). |
| Signature, `iss`, `aud`, `exp` | **Required** | Standard JWT validation; spec also lists a *revocation check* (needs short TTLs or an introspection endpoint — offline verification alone cannot satisfy it). |
| `scope.tenants` | **Optional (cap)** | Part of the down-scope; a non-empty list caps which tenants the token may act in. Caller-asserted (narrows only); the issuer-asserted gate is the `tenants` claim below, and both are enforced. |
| `scope.regions` / `scope.workspaces` | **Optional (cap)** | The other down-scope dimensions, matched against the request's region/workspace. Absent = no restriction; skipped when the request has no value for that dimension. |
| `tenants` | **Optional (gate)** | Issuer-asserted tenant membership → `Identity.MemberTenants`. Non-empty ⇒ the request's tenant must be listed. Unlike `scope.tenants` the caller cannot omit it, so it is what makes a `subs: ["*"]` grant mean "every member of this tenant" (see §5). |
| `roles` | **Removed — never read** | Entitlement assertion — resolved solely from the RoleAssignment store. Any `roles` field in the token is ignored. See §4. |
| `permissions` | **Must NOT appear** | Worse than roles: inlines policy itself, bypassing even the Role indirection — the IdP would own *what a role means*, not just who has it. |

Supporting spec language:

- *"JWT's main function is to authenticate the user's identity … **not to directly handle
  authorization or dictate permissions**."* (`03-api-security.md`)
- Authorization is *"**independent of any data in an authentication token** like JWT."* (`03-api-security.md`)
- The spec's mention of *"claims embedded in the JWT (e.g. `scope`, `role`, or `permissions`)"*
  is an illustrative *"e.g."*, not a normative requirement — and only `scope` is coherent
  with the rest of the architecture.

---

## 3. Authentication & authorization placement

**Authentication (→ 401)** — validates the credential, produces identity:

- Verify signature / issuer / audience / expiry; revocation via short token TTLs (§6).
- Output: `Identity{Subject, MemberTenants, TokenScope}`. Nothing else is needed from the token.
- Tenant-mismatch placement depends on the IdP topology:
  - **Per-tenant issuer** (e.g. Keycloak realm per tenant): wrong-tenant token fails issuer
    validation → **401**, check lives in authn.
  - **Single shared issuer + tenant claim** (Entra `tid` model): signature is valid, the
    mismatch is policy → **403**, check lives in authz. Operationally simpler (one JWKS).

**Authorization (→ 403)** — all server-side, per request:

1. Tenant membership gate: token tenant(s) cover the URL tenant.
2. Load Role + RoleAssignment from the URL tenant's namespace (direct or informer-cached).
3. Evaluate: `subs` contains subject (or `*`) ∧ a scope entry covers
   tenant/region/workspace ∧ an assigned role has a permission matching
   provider/resource/verb.

**Technical failure (→ 500)** — RBAC store unreachable, cache read failure. Never
disguised as a denial; already implemented correctly in the current middleware.

Current implementation status: all of it. The chain, error categories and RBAC evaluation
match this placement; the `jwt` plugin verifies signature, `alg`, `exp` and the configured
`iss`/`aud`; the tenant claim reaches `Identity.MemberTenants` and is gated in the
evaluator (**403**, the single-shared-issuer placement — this deployment has one JWKS-less
configured key, not an issuer per tenant); and the roles-claim divergence below is gone.

---

## 4. Key conflict: roles in the token (duplication & drift)

**Current behavior**: the evaluator only honors a role that appears **both** in a
matching RoleAssignment **and** in the token (`ra.spec.roles ∩ token.roles`,
`gateway/internal/authz/seca/evaluator.go` → `claimHasRole`). With the dummy
authenticator this is free — clients hand-write their token JSON. With a real IdP it
becomes the main friction point:

- **Dual bookkeeping.** Role membership must be recorded twice: in RoleAssignments
  (gateway) *and* in IdP claim configuration. Both must agree for anything to work.
- **Fail-closed drift.** Any mismatch produces 403s that are hard to diagnose — the
  assignment looks correct, but the token silently lacks the role name.
- **Wrong data owner.** The IdP must hold *dynamic per-user SECA role membership* —
  runtime data it has no natural source for. (Contrast: a tenant claim is static
  onboarding data the IdP genuinely owns.)
- **Breaks the spec's own flows.** In the usage guide, a user creates a RoleAssignment
  (Step 0) and continues with the *same* token; *"creating a workspace automatically
  grants you admin permissions"* is a mid-session grant. Token-borne entitlements would
  be stale the moment either happens — requiring token reissue after every grant.
- **Breaks tenant self-service.** A tenant admin creating a RoleAssignment via the API
  achieves nothing until someone with IdP admin access also updates claim mappings.

**Design ladder** (most → least aligned with the spec):

1. ✅ **Subject-only token; RBAC fully server-side** — drop the intersection. Any
   standard OIDC issuer works; matches the usage guide exactly.
2. ✅ Subject + **optional** roles claim as attenuation: if present, intersect
   (down-scoping, like narrow OAuth scopes); if absent, no restriction.
   Careful: absent = unrestricted, present-but-empty = deny.
3. ⚠️ Subject + **mandatory** roles claim (current implementation) — operationally
   fragile for all the reasons above.
4. ❌ Role-only token — role names become bearer capabilities: no per-user granularity,
   no tenant isolation (System Roles exist in every tenant), no gateway-side revocation,
   and `subs` (mandatory in the schema) is silently bypassed.

**Decision (implemented)**: option (1) — the token is **subject-only for roles**; the
`∩ claim.Roles` intersection is gone and roles come solely from `RoleAssignment`. The
attenuation idea from (2) was kept but re-homed onto the **SECA scope dimensions** (the
token's optional `scope.tenants/regions/workspaces` cap), not onto roles — so a token can
still shrink its own blast radius without the IdP ever needing to know a caller's roles.

---

## 5. Other sharp edges

- **`subs: ["*"]` wildcard (resolved)**: schema says *"all users of the tenant scopes"*,
  but on its own the implementation grants it to **every authenticated principal** — the
  token's `scope` down-scope is *caller-asserted*, so it does not constrain the wildcard.
  The `tenants` claim closes this: an issuer that stamps membership on every token it
  mints turns `*` into "every member of this tenant", because the gate runs before the
  assignment loop. An issuer that stamps nothing leaves the old footgun, so `*` still
  wants care on a shared issuer. (The e2e fixtures used to need a wildcard assignment so
  every caller could list regions; serving `seca.region` authn-only removed that need,
  and the `ra-wildcard` fixture with it.)
- **Tenant-less routes**: an empty tenant makes `ComputeNamespace` return `""`, which
  would list RoleAssignments across **all namespaces**. The only current tenant-less path
  is the region catalog, whose shape upstream confirmed as correct — it no longer reaches
  the authz middleware at all (`--authz-skip-providers`, default `seca.region`). Keep this
  hazard in mind before wrapping any other tenant-less endpoint in the authz middleware
  instead of adding its provider to the skip list.
- **Global server vs region-scoped assignments**: the global server has an empty region,
  so an assignment restricted to specific `regions` never covers global-server requests;
  only region-unrestricted assignments do. The same holds for the token down-scope, which
  skips a dimension the request does not carry (so a `regions` cap does not block
  region-agnostic global-server calls).
- **Token `scope` vocabulary (resolved)**: the token down-scope reuses the **SECA scope
  dimensions** (`tenants`/`regions`/`workspaces`), matched against the request's
  tenant/region/workspace — deliberately *not* OAuth2 operation-family scopes, avoiding an
  overload with `RoleAssignmentScope`. Adding operation-family scopes later would need a
  separate, explicitly-defined claim.

---

## 6. Checklist

Done on `feat/gateway-auth-middleware`:

- [x] Remove the roles-claim intersection from the evaluator — roles come solely from
      `RoleAssignment` (`gateway/internal/authz/seca/evaluator.go`).
- [x] Define the token `scope` down-scope over the SECA dimensions
      (`tenants`/`regions`/`workspaces`) and enforce it in `Evaluate`.
- [x] Serve the tenant-less region catalog authn-only — authorization layer skipped per
      provider via the configurable `--authz-skip-providers` (default `seca.region`).

Done for the real (signature-verifying) `jwt` authenticator:

- [x] Verify the signature against the operator-configured key (`--jwt-secret`) with the
      `alg` pinned to `--jwt-signing-method`, plus mandatory `exp` and the configured
      `iss`/`aud` (`--jwt-issuer`, `--jwt-audience` — each enforced *and* required in the
      token when set). Nothing named inside the token selects how it is verified.
- [x] Subject claim: **`sub`**, platform-wide — it is what `RoleAssignment.spec.subs`
      lists, so an issuer identifying users by email puts the email in `sub`.
- [x] Signed, IdP-asserted tenant-membership gate: the `tenants` claim →
      `Identity.MemberTenants`, enforced in `Evaluate` before any assignment is
      considered, so it caps `subs: ["*"]` where the caller-asserted `scope.tenants`
      cannot. Both caps apply, so the effective set is their intersection.
- [x] Revocation strategy: **short-lived tokens + refresh at the issuer**, with `exp`
      mandatory so a token cannot opt out. Per-request introspection was rejected — a
      network round-trip on the hot path and a hard dependency on IdP availability. See
      [AUTH.md](AUTH.md) § Token lifetime and revocation.

Deliberately not implemented (no current deployment needs it):

- **JWKS / OIDC discovery.** The key is a mounted file, rotated by rotating the file. An
  issuer that rotates keys on its own schedule would want discovery plus a key cache.
- **A configurable name for the tenant-membership claim** (`tid`, `org_id`, …). Map it to
  `tenants` on the issuer instead.
