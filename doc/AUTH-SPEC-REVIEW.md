# Auth Findings — Token Model & SECA Spec Alignment

Findings from reviewing the gateway auth middleware (`feat/gateway-auth-middleware`,
see [AUTH.md](AUTH.md)) against the SECA specification: what the bearer token must
carry, where authentication and authorization decisions belong, and where the current
implementation diverges from the spec.

> **Status — implemented on `feat/gateway-auth-middleware`.** The final model:
> the bearer token carries the **subject** (`subs`) plus an optional **`scope`**
> down-scope (`tenants`/`regions`/`workspaces`) only. **Roles are resolved entirely from
> `RoleAssignment`/`Role` in the tenant namespace and are never read from the token.**
> The `scope` object can only *narrow* what a token may exercise (see [AUTH.md](AUTH.md)
> § Token down-scoping). Authentication verifies token authenticity against a **fixed,
> operator-configured endpoint** — any endpoint named inside the token is ignored; the
> shipped Dummy authenticator validates username/password. **Region is treated as an
> ordinary tenant-scoped resource**; the special-case path correction comes upstream in
> the SECA spec, so it is intentionally not handled here. The sections below record the
> analysis that led to this model; where an older subsection weighs alternatives, this
> banner is the decision that was taken.

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
| `scope.tenants` | **Optional (cap)** | Part of the down-scope; a non-empty list caps which tenants the token may act in. Caller-asserted (narrows only). A *signed* IdP-asserted tenant membership gate remains future work (see §5). |
| `scope.regions` / `scope.workspaces` | **Optional (cap)** | The other down-scope dimensions, matched against the request's region/workspace. Absent = no restriction; skipped when the request has no value for that dimension. |
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

- Verify signature / issuer / audience / expiry; revocation strategy TBD.
- Output: `Identity{Subject, Tenant(s)}`. Nothing else is needed from the token.
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

Current implementation status: chain, error categories, and RBAC evaluation exist and
match this placement. Missing: real JWT verification (dummy authenticator only), tenant
claim in `Identity` + membership gate, and the roles-claim divergence below.

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

- **`subs: ["*"]` wildcard**: schema says *"all users of the tenant scopes"*, but the
  implementation grants it to **every authenticated principal**. The token's `scope`
  down-scope is *caller-asserted*, so it does **not** constrain the wildcard — a wildcard
  assignment genuinely grants everyone. A real membership gate would have to come from a
  signed, IdP-asserted tenant claim in the future authenticator; until then `*` is a
  footgun on a shared issuer. (This is why the e2e "nobody is denied" fixtures had to
  target a provider the wildcard role does not cover, once token roles were removed.)
- **Tenant-less routes**: an empty tenant makes `ComputeNamespace` return `""`, which
  would list RoleAssignments across **all namespaces**. The only current tenant-less path
  is the region catalog, which is the special case being corrected upstream (region
  becomes an ordinary `[tenant]` resource). Keep this in mind before wrapping any other
  tenant-less endpoint in the authz middleware.
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

Remaining for the real (signature-verifying) authenticator:

- [ ] Verify signature via the operator-configured verification endpoint / JWKS, plus
      `iss`, `aud`, `exp`; ignore any endpoint named inside the token.
- [ ] Decide the subject claim (`sub` vs `email`) — it must match the identifier
      convention used in `RoleAssignment.spec.subs` platform-wide.
- [ ] Optionally add a *signed*, IdP-asserted tenant-membership gate (distinct from the
      caller-asserted `scope.tenants` cap, which cannot constrain a `subs: ["*"]` grant).
- [ ] Pick a revocation strategy: short-lived tokens + refresh, or an introspection call.
