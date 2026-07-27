# Aruba Network and Compute Resources Mapping

This document describes the mapping implemented between SECA network and compute resources and Aruba resources. The mapping is not 1:1: Aruba models fewer objects than SECA does, so some SECA resources have no Aruba counterpart at all, and some Aruba resources (a key pair) have no SECA counterpart and are synthesised.

The Aruba resources are the custom resources of the [arubacloud-resource-operator](https://github.com/Arubacloud/arubacloud-resource-operator) `v1.1.1` (`arubacloud.com/v1alpha1`), which the plugin creates and then watches; the operator is what actually talks to the Aruba CMP.

## Mapping

| SECA Resource         | Aruba Resource      | Waits for                                                       |
|-----------------------|---------------------|-----------------------------------------------------------------|
| Network               | `VPC`               | Workspace active, an `InternetGateway` in the workspace, `Project` active |
| Subnet                | `Subnet`            | Workspace active, `Project` active, parent `VPC` active          |
| Public IP             | `ElasticIP`         | Workspace active, `Project` active                               |
| Route Table           | *none*              | —                                                               |
| Internet Gateway      | *none*              | —                                                               |
| Security Group        | *none directly* — materialised as `SecurityGroup` at instance attach | — |
| Security Group Rule   | *none directly* — materialised as `SecurityRule` at instance attach | — |
| NIC                   | *none*              | Aruba has no standalone NIC: NICs are attributes of a `CloudServer` |
| Network SKU           | not reconciled      | Read-only catalog                                                |
| Instance              | `CloudServer` (+ `KeyPair`, `SecurityGroup`, `SecurityRule`) | Workspace & `Project` active, its NICs present, their subnets active, an ssh key, a boot volume |
| Compute SKU           | *mapped, not created* | Its vCPU/RAM select the CloudServer `FlavorName` — see [SKU mapping](#sku-mapping) |

### Route Table and Internet Gateway have no Aruba counterpart

Aruba's network API (see the [SDK](https://github.com/Arubacloud/sdk-go) `client_network.go`) exposes VPCs, subnets, security groups and rules, elastic IPs, VPC peerings and VPN tunnels — there is no route table and no internet gateway object. Internet egress and intra-VPC routing are properties Aruba configures on the VPC itself.

Both SECA resources are therefore accepted and go active immediately without creating anything. They exist so the SECA model stays consistent across providers. The `InternetGateway` additionally gates the Network: an Aruba VPC always provides internet egress, so the SECA resource representing that egress must exist before the VPC is created. **The Route Table gates nothing** — see below.

### Why the Route Table is not a precondition

The Aruba API can auto-create a default subnet and security group together with a VPC, via the `preset` boolean in the create request (`VPCPropertiesInnerRequest.Preset` in the SDK). The operator hard-codes `NotDefault().WithoutPreset()` and its `VPCSpec` does not expose the field, so `preset` is always `false`: creating a VPC never derives a subnet or any routing object. Subnets are created independently afterwards, as their own custom resources. Because nothing is derived from the VPC, there is nothing for a Route Table to be a precondition of.

### Security groups are materialised at instance attach, not at creation

Aruba's `SecurityGroup` and `SecurityRule` both require a `VPCReference` (a rule additionally requires a `SecurityGroupReference`). A SECA `SecurityGroup` carries **no VPC**: it is workspace-scoped and only takes effect once a NIC or instance references it (via `securityGroupRefs`). That reference is what binds the group to a subnet, hence to a network, hence to an Aruba VPC. So the SECA `SecurityGroup` and `SecurityGroupRule` controllers create nothing on accept — they go active — and the real Aruba `SecurityGroup` + `SecurityRule` are created by the **compute-instance handler**, per VPC, when an instance attaches the group (see "Compute" below). A standalone SECA `SecurityGroupRule` is a reusable template that does nothing until a `SecurityGroup` pulls it in via `ruleRefs`; it is materialised as part of the group that references it.

Deleting the SECA `SecurityGroup` is what reaps the materialised Aruba resources: its `Delete` lists every Aruba `SecurityGroup` (and its `SecurityRule`s) labelled for that SECA group — one per VPC it was attached in — and removes them. Instance deletion leaves them (they may be shared), so the SECA group is the single owner that can clean them up.

Replicating a SECA security group into *every* VPC up front was considered and rejected: it would create dormant Aruba resources nothing uses, and would need the Network handler to back-fill security groups into VPCs created later, for no present benefit.

## Compute

A SECA `Instance` maps to an Aruba `CloudServer`. A CloudServer's required references — VPC, subnets, security groups, key pair, boot volume — are **none of them named directly** by the Instance; the handler resolves them from the instance's dependency graph and materialises what Aruba needs but SECA does not model as its own resource:

- **NICs → subnets → VPC.** The instance's `primaryNicRef`/`additionalNicRefs` are loaded; each NIC's `subnetRef`, `securityGroupRefs` and `publicIpRefs` are collected. A NIC reference carries no network, so the Aruba `Subnet` backing a SECA subnet name is located by workspace label across namespaces and matched by name; the first active match wins. The subnet's `VPCReference` fixes the CloudServer's VPC.
- **Security groups + rules** referenced by the NICs (and the instance's own `securityGroupRef`) are materialised in that VPC: one Aruba `SecurityGroup` per `(network, SECA group)`, named `<seca-group>-<network>`, plus its `SecurityRule`s built from the group's inline `rules` and its `ruleRefs`.
- **Key pair.** An Aruba CloudServer requires a `KeyPairReference`, but SECA has no key-pair resource: the public key travels inline in `Instance.Spec.SshKeys` (the field is documented as "references" but actually carries the key material, e.g. `ssh-rsa AAAA…`). The handler creates a `KeyPair` named `<instance>-key` from the first ssh key and deletes it with the instance.
- **Boot/data volumes** come from `BootVolume`/`DataVolumes` (Aruba `BlockStorage`, already reconciled by the storage plugin); **flavor** is resolved from the instance's compute SKU — its vCPU/RAM select an Aruba flavor (see [SKU mapping](#sku-mapping)), not the SKU name used verbatim; an optional **elastic IP** comes from the first NIC public IP.

Missing dependencies gate the create with `ErrStillProcessing` (the instance stays in `creating` and is retried): a NIC not created yet, a subnet not yet active, **no security group** (a CloudServer requires ≥1), or **no ssh key** (the required `KeyPairReference` cannot be built). `PowerOn`/`PowerOff` are **no-ops**: Aruba's `CloudServer` CRD exposes no power field, so power state lives only on the SECA side.

On instance delete the CloudServer is deleted and then the key pair; the **materialised security groups are left in place** — they may be shared with other instances and are reaped only when the SECA security group itself is deleted (see "Security groups are materialised at instance attach" above).

### Compute mapping ceilings

| Behaviour | Detail |
|---|---|
| Multiple ssh keys | Aruba `KeyPair` holds a single value; SECA allows up to 32 → the first is used. |
| Security-group rule changes | Rules are created once, at attach; later edits to the SECA group are not reconciled onto the Aruba side. |
| Multi-VPC instance | All of an instance's subnets are assumed to share one network/VPC; the first subnet's VPC wins. |
| Rule fan-out | One SECA rule expands to one Aruba `SecurityRule` per `(protocol × port × source)`: `tcp+udp` → TCP and UDP rules, a port list → one rule per port, each `sourceRef` → its own rule. |
| Rule targets | A `security-groups/<name>` source becomes a `SecurityGroup` target; anything else is treated as an IP/CIDR literal (`Ip` target). No source means all traffic → `0.0.0.0/0`. Instance/gateway sources have no Aruba target type and are mapped best-effort to `Ip`. |

## SKU mapping

SECA SKUs describe **capacity** (vCPU/RAM, IOPS); Aruba names a fixed **catalog** (flavors, storage tiers). [`pkg/adapter/skumap`](pkg/adapter/skumap) bridges the two. The catalog is **embedded**: the delegator holds no Aruba credentials (only the operator does), so it cannot query Aruba's live catalog — the source of truth is the `sdk-go` `CloudServerFlavor` / `BlockStorageType` enums, transcribed into `skumap`.

| SECA SKU | Mapped to | How |
|---|---|---|
| Compute (`InstanceSKU` — vCPU, RAM) | `CloudServer.FlavorName` (`CSO<cpu>A<ram>`) | **Exact** match on vCPU **and** RAM (GB): `{4, 8}` → `CSO4A8`. No matching flavor → the instance goes to `error` with a clear message (`no Aruba CloudServer flavor provides N vCPU / M GB RAM`) instead of a bare Aruba `400`. |
| Storage (`StorageSKU` — IOPS) | `BlockStorage.Type` (`Standard` / `Performance`) | IOPS at/above a threshold → `Performance`, else `Standard`. A coarse heuristic keyed on the objective metric (IOPS), because Aruba exposes only two tiers and does not publish their IOPS boundary; the SECA `Type` string (`local-durable`…) describes durability, not a perf tier. Safe: Aruba defaults the field when unset. |
| Network (`NetworkSKU` — bandwidth) | — | Aruba's `VPC`/`Subnet` have no bandwidth or SKU field; not mapped. |

Adding an Aruba flavor is a one-row change to `computeFlavors` in `skumap.go`. The SECA SKU catalog itself (the `InstanceSKU`/`StorageSKU` CRs) is provisioned separately — the region catalog in production, the test-data fixtures in tests — and only capacities Aruba actually offers will resolve to a flavor.

## What is not propagated

| SECA field | Why |
|---|---|
| `RouteTable.Spec.Routes` | Aruba has no route table. The nearest primitive is per-subnet DHCP static routes, which the operator's `Subnet` CRD does not expose (only `dhcp.enabled`). The SECA route-table plugin interface also has no `Update`, so route edits never reach the plugin. |
| `Network.Spec.CIDR`, `AdditionalCIDRs` | An Aruba `VPC` has no address range; addressing is defined per subnet. |
| `Network.Spec.SkuRef`, `Subnet.Spec.SkuRef` | No SKU concept on the Aruba side for these. |
| `Subnet.Spec.Zone`, `Subnet.Spec.RouteTableRef` | No equivalent field on the Aruba `Subnet` (see routes, above). |
| `InternetGateway.Spec.EgressOnly` | No gateway object to configure. |
| `Instance.Spec.UserData`, `AntiAffinityGroup` | No equivalent field on the Aruba `CloudServer`. |
| `Instance.Spec.PrimaryNicRef`/`AdditionalNicRefs` | Not propagated as NICs (Aruba has none); consumed to resolve the CloudServer's subnets and security groups. |

Some SECA specs are **rejected** rather than silently ignored, because honouring them partially would hand back something the user did not ask for:

- `PublicIp.Spec.Address` (BYOIP) — an Aruba `ElasticIP` has no address field; Aruba always allocates it.
- `PublicIp.Spec.Version: IPv6` — `ElasticIP` is IPv4-only.

An IPv6-only `Subnet` is likewise rejected: the Aruba `Subnet` CRD validates `cidr` against an IPv4 pattern. Dual-stack subnets are accepted, with the IPv6 range dropped.

## Defaults applied

| Field | Value | Why |
|---|---|---|
| Region | `ITBG-Bergamo` | When the SECA resource carries no region. |
| `Subnet.Spec.Type` | `Advanced` | Lets the subnet use the CIDR from the SECA spec; `Basic` would have Aruba choose the range and silently ignore it. |
| `Subnet.Spec.DHCP.Enabled` | `true` | The CRD requires the field; SECA has no knob for it. |
| `ElasticIP.Spec.BillingPeriod` | `Hour` | The CRD requires the field; SECA has no knob for it. |
| `CloudServer.Spec.Zone` | `ITBG-1` | When the SECA `Instance` carries no zone. |

## Status reporting limitation

While a resource waits for a dependency, the plugin returns `ErrStillProcessing`, which the framework turns into a plain requeue. The SECA resource stays in state `creating` with **no message naming the missing dependency** — the plugin interface has no channel for a message on an in-progress wait (returning a real error would surface a condition, but flip the state to `error`).

So a Network with no Internet Gateway, or an Instance with no ssh key, sits in `creating` indefinitely rather than reporting the reason. Surfacing that needs a framework change to the plugin contract.

## Namespaces and references

Aruba CRs are created in the same namespace as the SECA resource they mirror (hashed by scope, via `ComputeNamespace`/`ComputeNetworkNamespace` in `framework/backend/kubernetes/adapter.go`):

- `VPC`, `ElasticIP`, `CloudServer`, `KeyPair` and the materialised `SecurityGroup`/`SecurityRule` → workspace namespace, `hash(tenant/workspace)`.
- `Subnet` → its network's namespace, `hash(tenant/workspace/network)`.
- `Project` (from the SECA Workspace) → tenant namespace, `hash(tenant)`.

Every Aruba CR references its `Project` by `{name: <workspace>, namespace: hash(tenant)}`; the Aruba `Subnet`, `SecurityGroup`, `SecurityRule` and `CloudServer` additionally reference their `VPC` by `{name: <seca network>, namespace: hash(tenant/workspace)}`. The `CloudServer` references subnets in the network namespace and its key pair, security groups and volumes in the workspace namespace — cross-namespace references the operator supports.

## Testing the plugin

The plugin has its own integration suite in [`test/integration`](test/integration), run from this directory with:

```shell
make test-integration ARUBA_TENANT=<your-aruba-account>
```

Like [`csp/dummy/test/integration`](../dummy/test/integration) it exercises the plugin **directly**: it creates the SECA CRs and asserts the delegator reconciles them into `arubacloud.com` CRs — there is **no gateway** in the path. Unlike the dummy suite it does **not** stand up its own cluster, because the backend is the real third-party operator, not a self-contained simulation.

### Requirements

The suite connects to the **current kube-context** and expects the stack already deployed — the two-phase flow documented in [`test/conformance/aruba`](../../test/conformance/aruba):

1. **A running cluster.** For a local KIND cluster: `make -C ../../test kind-start`.
2. **The [`arubacloud-resource-operator`](https://github.com/Arubacloud/arubacloud-resource-operator) and its Aruba credentials** installed in that cluster. This is installed **out of band** — there is no in-repo tooling for it — and is what actually provisions resources against the Aruba CMP.
3. **The `delegator-aruba` deployed** with the SECA CRDs and the test-data fixtures (which provide the SKUs the suite references — `sku-1`, `network-sku-1`, `compute-sku-1`): `make -C ../../test kind-deploy-stack E2E_PLUGIN=aruba E2E_TENANT=<your-aruba-account>`.
4. **A real Aruba account in `ARUBA_TENANT`.** The operator provisions real cloud resources, so a resource only reaches `Active` against a genuine tenant (the default, `test-tenant`, will not provision). It **must match** the `E2E_TENANT` the stack was deployed with, so the fixture SKUs land in the same `hash(tenant)` namespace the plugin reads them from.

### What the test does

`TestArubaFlow` drives the whole dependency graph **in order** — not per-resource in parallel like the dummy suite, because aruba's resources depend on one another and are provisioned for real:

```
workspace          ─▶ Project
block-storage      ─▶ BlockStorage
internet-gateway   ─▶ (no-op)            ┐ gate: the VPC is created only
network            ─▶ VPC                ┘ once an internet-gateway exists
route-table        ─▶ (no-op)
subnet             ─▶ Subnet             (needs the VPC active)
public-ip          ─▶ ElasticIP
security-group     ─▶ (no-op)           ┐ the real Aruba SecurityGroup + rules
security-group-rule─▶ (no-op)           │ are materialised at instance attach
nic                ─▶ (no-op)           ┘
instance           ─▶ CloudServer        (+ per-VPC SecurityGroup, SecurityRule, KeyPair)
```

Each step asserts the SECA resource reaches `Active` **and** the matching `arubacloud.com` CR does too. Two behaviours are checked explicitly:

- **Internet-gateway gating** — the network is created *before* any internet-gateway, and the suite asserts **no VPC is created** while the gate holds; it then creates the internet-gateway and asserts the VPC appears and provisions.
- **Instance materialisation** — it asserts the per-VPC `SecurityGroup` (`<sg>-<network>`) and the `KeyPair` provision and that the `CloudServer` CR is created with them referenced. It does **not** assert the VM itself reaches `Active`: the CloudServer's `FlavorName` is the SECA compute-SKU name verbatim (`compute-sku-1` in the fixtures), which is not a real Aruba flavor — provisioning a running VM needs the compute-SKU catalog to carry real Aruba flavor names, a data concern outside the plugin's contract.

A `t.Cleanup` deletes every resource in reverse dependency order, so a run — even a failed one — does not leak real Aruba resources.

### Constraints the suite encodes

Real Aruba CMP validation, discovered against a live account, that test inputs must respect:

- **Resource names must be ≥ 4 characters** — Aruba rejects shorter names with a `400` validation error.
- **`Instance.Spec.SshKeys` must carry a well-formed public key** — the `KeyPair` fails validation otherwise. The suite ships a valid ed25519 key; override it with `ARUBA_SSH_KEY`.
